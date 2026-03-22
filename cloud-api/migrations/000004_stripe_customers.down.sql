ALTER TABLE subscriptions ADD COLUMN stripe_customer_id VARCHAR(255);

UPDATE subscriptions s
SET stripe_customer_id = sc.customer_id
FROM stripe_customers sc
WHERE s.user_id = sc.user_id;

CREATE INDEX idx_subscriptions_stripe_customer_id ON subscriptions(stripe_customer_id);

DROP TABLE stripe_customers;
