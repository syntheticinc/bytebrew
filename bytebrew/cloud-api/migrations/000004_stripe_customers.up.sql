CREATE TABLE stripe_customers (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    customer_id VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO stripe_customers (user_id, customer_id, created_at)
SELECT user_id, stripe_customer_id, NOW()
FROM subscriptions
WHERE stripe_customer_id IS NOT NULL;

ALTER TABLE subscriptions DROP COLUMN stripe_customer_id;
