ALTER TABLE users
    ADD COLUMN google_id VARCHAR(255),
    ALTER COLUMN password_hash DROP NOT NULL;

CREATE UNIQUE INDEX idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;
