-- +migrate Up
-- Add username and name fields to users table, remove is_active
ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(50) NOT NULL UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE users DROP COLUMN IF EXISTS is_active;

-- Create index on username for faster lookups
CREATE INDEX IF NOT EXISTS users_username_idx ON users(username);

-- +migrate Down
-- Reverse: remove username and name, add back is_active
ALTER TABLE users DROP COLUMN IF EXISTS username;
ALTER TABLE users DROP COLUMN IF EXISTS name;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE NOT NULL;

DROP INDEX IF EXISTS users_username_idx;
