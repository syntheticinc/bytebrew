-- name: InsertProcessedEvent :exec
INSERT INTO processed_stripe_events (event_id, event_type) VALUES ($1, $2);

-- name: IsEventProcessed :one
SELECT EXISTS(SELECT 1 FROM processed_stripe_events WHERE event_id = $1);
