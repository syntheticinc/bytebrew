ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN verification_token VARCHAR(64);
ALTER TABLE users ADD COLUMN verification_expires_at TIMESTAMPTZ;

-- Mark existing users as verified (they registered before this feature)
UPDATE users SET email_verified = true;
