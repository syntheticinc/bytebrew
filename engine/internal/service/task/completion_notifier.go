package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// CompletionPayload is the JSON body sent to the on-complete webhook.
type CompletionPayload struct {
	TaskID     string `json:"task_id"`
	Status     string `json:"status"`
	Result     string `json:"result"`
	DurationMs int64  `json:"duration_ms"`
	TriggerID  string `json:"trigger_id"`
	AgentName  string `json:"agent_name"`
	Timestamp  string `json:"timestamp"`
}

// CompletionNotifier sends task results to configured webhook URLs.
type CompletionNotifier struct {
	httpClient  *http.Client
	maxRetries  int
	backoffBase time.Duration // base for exponential backoff; defaults to 1s
}

// NewCompletionNotifier creates a CompletionNotifier with sensible defaults.
func NewCompletionNotifier() *CompletionNotifier {
	return &CompletionNotifier{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		maxRetries:  3,
		backoffBase: time.Second,
	}
}

// NewCompletionNotifierWithOptions creates a CompletionNotifier with explicit
// HTTP client timeout, retry count, and backoff base duration. Intended for
// callers that need to tune delivery behaviour (e.g. high-throughput or test
// environments). Pass zero for backoffBase to use the default (1s).
func NewCompletionNotifierWithOptions(clientTimeout time.Duration, maxRetries int, backoffBase time.Duration) *CompletionNotifier {
	if backoffBase == 0 {
		backoffBase = time.Second
	}
	return &CompletionNotifier{
		httpClient:  &http.Client{Timeout: clientTimeout},
		maxRetries:  maxRetries,
		backoffBase: backoffBase,
	}
}

// Notify sends the completion payload to the given webhook URL.
// It retries on 5xx and network errors with exponential backoff (1s, 2s, 4s).
// 4xx responses are not retried (client error).
func (n *CompletionNotifier) Notify(ctx context.Context, webhookURL string, headers map[string]string, payload CompletionPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal completion payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < n.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * n.backoffBase
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		lastErr = n.doRequest(ctx, webhookURL, headers, body)
		if lastErr == nil {
			return nil
		}

		slog.WarnContext(ctx, "completion webhook attempt failed",
			"attempt", attempt+1,
			"max_retries", n.maxRetries,
			"url", webhookURL,
			"task_id", payload.TaskID,
			"error", lastErr,
		)

		// Check if it's a non-retryable error (4xx)
		if isNonRetryable(lastErr) {
			return lastErr
		}
	}

	return fmt.Errorf("completion webhook exhausted retries: %w", lastErr)
}

func (n *CompletionNotifier) doRequest(ctx context.Context, url string, headers map[string]string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return &retryableError{err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return &nonRetryableError{statusCode: resp.StatusCode}
	}

	return &retryableError{err: fmt.Errorf("server error: status %d", resp.StatusCode)}
}

// retryableError signals that the request can be retried.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) Unwrap() error {
	return e.err
}

// nonRetryableError signals a client error (4xx) that should not be retried.
type nonRetryableError struct {
	statusCode int
}

func (e *nonRetryableError) Error() string {
	return fmt.Sprintf("client error: status %d", e.statusCode)
}

func isNonRetryable(err error) bool {
	_, ok := err.(*nonRetryableError)
	return ok
}
