DROP INDEX IF EXISTS idx_users_google_id;

-- Restore NOT NULL constraint (delete Google-only users first)
DELETE FROM users WHERE password_hash = '' OR password_hash IS NULL;

ALTER TABLE users
    ALTER COLUMN password_hash SET NOT NULL,
    DROP COLUMN IF EXISTS google_id;
