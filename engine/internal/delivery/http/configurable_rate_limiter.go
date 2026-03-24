package http

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// RateLimitRule defines a rate limiting rule based on request headers.
type RateLimitRule struct {
	Name        string                   `mapstructure:"name" yaml:"name" json:"name"`
	KeyHeader   string                   `mapstructure:"key_header" yaml:"key_header" json:"key_header"`
	TierHeader  string                   `mapstructure:"tier_header" yaml:"tier_header" json:"tier_header"`
	Tiers       map[string]RateLimitTier `mapstructure:"tiers" yaml:"tiers" json:"tiers"`
	DefaultTier string                   `mapstructure:"default_tier" yaml:"default_tier" json:"default_tier"`
}

// RateLimitTier defines rate limit parameters for a specific tier.
type RateLimitTier struct {
	Requests  int    `mapstructure:"requests" yaml:"requests" json:"requests"`
	Window    string `mapstructure:"window" yaml:"window" json:"window"`
	Unlimited bool   `mapstructure:"unlimited" yaml:"unlimited" json:"unlimited"`
}

// ParseWindow parses the window duration string (e.g. "1h", "24h", "1m", "30s").
func (t RateLimitTier) ParseWindow() (time.Duration, error) {
	if t.Window == "" {
		return time.Hour, nil
	}
	return time.ParseDuration(t.Window)
}

// slidingWindow tracks request timestamps for a single key within a rule.
type slidingWindow struct {
	mu     sync.Mutex
	times  []time.Time
	window time.Duration
	limit  int
}

// allow checks whether a new request is within limits.
// If allowed, it records the request timestamp and returns (remaining, resetTime, true).
// If denied, it returns (0, resetTime, false) without recording.
func (sw *slidingWindow) allow(now time.Time) (remaining int, resetAt time.Time, allowed bool) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	cutoff := now.Add(-sw.window)

	// Prune expired entries
	valid := sw.times[:0]
	for _, t := range sw.times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	sw.times = valid

	// Calculate reset time: earliest entry expiry or now+window if empty
	if len(sw.times) > 0 {
		resetAt = sw.times[0].Add(sw.window)
	} else {
		resetAt = now.Add(sw.window)
	}

	if len(sw.times) >= sw.limit {
		return 0, resetAt, false
	}

	sw.times = append(sw.times, now)
	remaining = sw.limit - len(sw.times)
	return remaining, resetAt, true
}

// count returns the number of requests in the current window without recording.
func (sw *slidingWindow) count(now time.Time) (used int, resetAt time.Time) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	cutoff := now.Add(-sw.window)
	count := 0
	for _, t := range sw.times {
		if t.After(cutoff) {
			count++
		}
	}

	if len(sw.times) > 0 {
		// Find earliest valid entry for reset time
		for _, t := range sw.times {
			if t.After(cutoff) {
				resetAt = t.Add(sw.window)
				break
			}
		}
	}
	if resetAt.IsZero() {
		resetAt = now.Add(sw.window)
	}

	return count, resetAt
}

// windowKey uniquely identifies a sliding window: "ruleName:headerValue".
type windowKey struct {
	rule string
	key  string
}

// ConfigurableRateLimiter applies rate limits based on request context headers.
// EE only: if no valid EE license is present, all requests pass through.
type ConfigurableRateLimiter struct {
	rules   []RateLimitRule
	license *atomic.Pointer[domain.LicenseInfo]
	windows sync.Map // windowKey → *slidingWindow
}

// NewConfigurableRateLimiter creates a new ConfigurableRateLimiter.
func NewConfigurableRateLimiter(rules []RateLimitRule, license *atomic.Pointer[domain.LicenseInfo]) *ConfigurableRateLimiter {
	return &ConfigurableRateLimiter{
		rules:   rules,
		license: license,
	}
}

// Middleware returns an HTTP middleware that enforces configurable rate limits.
func (rl *ConfigurableRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.isEEActive() {
			next.ServeHTTP(w, r)
			return
		}

		rc := domain.GetRequestContext(r.Context())

		// Evaluate all rules, collecting windows to increment on success.
		type pendingIncrement struct {
			sw *slidingWindow
		}
		var pending []pendingIncrement
		var firstDenyRule *RateLimitRule
		var denyLimit int
		var denyRemaining int
		var denyResetAt time.Time

		now := time.Now()

		for i := range rl.rules {
			rule := &rl.rules[i]

			keyValue := rl.getHeaderValue(rc, r, rule.KeyHeader)
			if keyValue == "" {
				continue
			}

			tier := rl.resolveTier(rc, r, rule)
			if tier.Unlimited {
				continue
			}

			windowDur, err := tier.ParseWindow()
			if err != nil {
				continue
			}

			sw := rl.getOrCreateWindow(rule.Name, keyValue, windowDur, tier.Requests)

			remaining, resetAt, allowed := sw.allow(now)
			if !allowed {
				firstDenyRule = rule
				denyLimit = tier.Requests
				denyRemaining = remaining
				denyResetAt = resetAt
				break
			}

			// Mark for rollback-free tracking — already recorded in allow()
			_ = pending // windows already incremented by allow()
			// Set rate limit headers for the first matching rule
			if len(pending) == 0 {
				setRateLimitHeaders(w, tier.Requests, remaining, resetAt)
			}
			pending = append(pending, pendingIncrement{sw: sw})
		}

		if firstDenyRule != nil {
			setRateLimitHeaders(w, denyLimit, denyRemaining, denyResetAt)

			retryAfter := int(math.Ceil(time.Until(denyResetAt).Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
				"error":       "Rate limit exceeded",
				"retry_after": retryAfter,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Rules returns the configured rate limit rules (for usage handler).
func (rl *ConfigurableRateLimiter) Rules() []RateLimitRule {
	return rl.rules
}

// Usage returns current usage for a specific rule, key header value.
func (rl *ConfigurableRateLimiter) Usage(ruleName, keyValue string, now time.Time) (used int, limit int, window time.Duration, resetAt time.Time, tierName string, found bool) {
	for i := range rl.rules {
		rule := &rl.rules[i]
		if rule.Name != ruleName {
			continue
		}

		// Find the tier — we need to check all tiers since we don't have request context here.
		// Look up the window directly.
		wk := windowKey{rule: ruleName, key: keyValue}
		val, ok := rl.windows.Load(wk)
		if !ok {
			// No window exists — return defaults from first/default tier
			tierName = rule.DefaultTier
			tier, tierOK := rule.Tiers[tierName]
			if !tierOK {
				return 0, 0, 0, time.Time{}, "", false
			}
			windowDur, err := tier.ParseWindow()
			if err != nil {
				return 0, 0, 0, time.Time{}, "", false
			}
			return 0, tier.Requests, windowDur, now.Add(windowDur), tierName, true
		}

		sw := val.(*slidingWindow)
		used, resetAt = sw.count(now)
		// Determine tier name by matching limit
		for name, tier := range rule.Tiers {
			if tier.Requests == sw.limit {
				tierName = name
				break
			}
		}
		windowDur, _ := time.ParseDuration("1h")
		if tierName != "" {
			if t, ok := rule.Tiers[tierName]; ok {
				windowDur, _ = t.ParseWindow()
			}
		}
		return used, sw.limit, windowDur, resetAt, tierName, true
	}
	return 0, 0, 0, time.Time{}, "", false
}

// isEEActive returns true if a valid (non-blocked) EE license is present.
func (rl *ConfigurableRateLimiter) isEEActive() bool {
	info := rl.license.Load()
	if info == nil {
		return false
	}
	return info.Status != domain.LicenseBlocked
}

// getHeaderValue reads a header from RequestContext first, then falls back to the HTTP request.
func (rl *ConfigurableRateLimiter) getHeaderValue(rc *domain.RequestContext, r *http.Request, header string) string {
	if rc != nil {
		if val := rc.Get(header); val != "" {
			return val
		}
	}
	return r.Header.Get(header)
}

// resolveTier determines which rate limit tier to use for a request.
func (rl *ConfigurableRateLimiter) resolveTier(rc *domain.RequestContext, r *http.Request, rule *RateLimitRule) RateLimitTier {
	tierName := rule.DefaultTier
	if rule.TierHeader != "" {
		if val := rl.getHeaderValue(rc, r, rule.TierHeader); val != "" {
			tierName = val
		}
	}

	tier, ok := rule.Tiers[tierName]
	if !ok {
		// Fall back to default tier
		tier, ok = rule.Tiers[rule.DefaultTier]
		if !ok {
			// No matching tier at all — unlimited by default
			return RateLimitTier{Unlimited: true}
		}
	}
	return tier
}

// getOrCreateWindow returns or creates a sliding window for the given rule+key combination.
func (rl *ConfigurableRateLimiter) getOrCreateWindow(ruleName, keyValue string, window time.Duration, limit int) *slidingWindow {
	wk := windowKey{rule: ruleName, key: keyValue}

	if val, ok := rl.windows.Load(wk); ok {
		sw := val.(*slidingWindow)
		// Update limit/window if tier changed (e.g. tier upgrade)
		sw.mu.Lock()
		sw.limit = limit
		sw.window = window
		sw.mu.Unlock()
		return sw
	}

	sw := &slidingWindow{
		window: window,
		limit:  limit,
	}
	actual, _ := rl.windows.LoadOrStore(wk, sw)
	return actual.(*slidingWindow)
}

// setRateLimitHeaders writes standard rate limit response headers.
func setRateLimitHeaders(w http.ResponseWriter, limit, remaining int, resetAt time.Time) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))
}
