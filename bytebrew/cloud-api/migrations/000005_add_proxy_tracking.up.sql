ALTER TABLE subscriptions
    ADD COLUMN proxy_steps_used INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN proxy_steps_limit INTEGER NOT NULL DEFAULT 300,
    ADD COLUMN byok_enabled BOOLEAN NOT NULL DEFAULT true;
