-- name: CreateSubscription :one
INSERT INTO subscriptions (user_id, tier, status, proxy_steps_limit, byok_enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, tier, status, current_period_start, current_period_end,
          stripe_subscription_id, proxy_steps_used, proxy_steps_limit, byok_enabled,
          created_at, updated_at;

-- name: GetSubscriptionByUserID :one
SELECT id, user_id, tier, status, current_period_start, current_period_end,
       stripe_subscription_id, proxy_steps_used, proxy_steps_limit, byok_enabled,
       created_at, updated_at
FROM subscriptions
WHERE user_id = $1;

-- name: UpdateSubscriptionTier :exec
UPDATE subscriptions
SET tier = $1, status = $2, current_period_start = $3,
    current_period_end = $4, updated_at = NOW()
WHERE user_id = $5;

-- name: UpdateStripeSubscriptionID :exec
UPDATE subscriptions SET stripe_subscription_id = $1, updated_at = NOW() WHERE user_id = $2;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE user_id = $2;

-- name: UpdateSubscriptionFull :exec
UPDATE subscriptions SET tier = $1, status = $2, current_period_start = $3,
  current_period_end = $4, stripe_subscription_id = $5, proxy_steps_limit = $6,
  updated_at = NOW()
WHERE user_id = $7;

-- name: ResetProxyStepsUsed :exec
UPDATE subscriptions SET proxy_steps_used = 0, updated_at = NOW() WHERE user_id = $1;

-- name: IncrementProxySteps :exec
UPDATE subscriptions SET proxy_steps_used = proxy_steps_used + 1, updated_at = NOW() WHERE user_id = $1;
