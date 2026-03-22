-- name: UpsertStripeCustomer :exec
INSERT INTO stripe_customers (user_id, customer_id)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET customer_id = $2;

-- name: GetStripeCustomerByUserID :one
SELECT user_id, customer_id, created_at
FROM stripe_customers
WHERE user_id = $1;

-- name: GetUserIDByStripeCustomerID :one
SELECT user_id
FROM stripe_customers
WHERE customer_id = $1;
