-- +migrate Up
-- Rename refresh_token to token in sessions table
ALTER TABLE sessions RENAME COLUMN refresh_token TO token;

-- Add last_used_at field
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();

-- Add device_type field
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS device_type VARCHAR(50);

-- Add deleted_at for soft delete support
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Create indexes
CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions(user_id);
CREATE INDEX IF NOT EXISTS sessions_deleted_at_idx ON sessions(deleted_at);

-- +migrate Down
-- Reverse: rename token back to refresh_token
ALTER TABLE sessions RENAME COLUMN token TO refresh_token;

-- Remove added columns
ALTER TABLE sessions DROP COLUMN IF EXISTS last_used_at;
ALTER TABLE sessions DROP COLUMN IF EXISTS device_type;
ALTER TABLE sessions DROP COLUMN IF EXISTS deleted_at;

-- Drop indexes
DROP INDEX IF EXISTS sessions_user_id_idx;
DROP INDEX IF EXISTS sessions_deleted_at_idx;
