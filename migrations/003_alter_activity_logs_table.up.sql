-- +migrate Up
-- Rename activity_logs to user_activity_logs
ALTER TABLE activity_logs RENAME TO user_activity_logs;

-- Rename columns
ALTER TABLE user_activity_logs RENAME COLUMN resource TO entity;
ALTER TABLE user_activity_logs RENAME COLUMN resource_id TO entity_id;
ALTER TABLE user_activity_logs RENAME COLUMN metadata TO details;

-- Change action field size
ALTER TABLE user_activity_logs ALTER COLUMN action TYPE VARCHAR(255);

-- Add deleted_at for soft delete support
ALTER TABLE user_activity_logs ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Create indexes
CREATE INDEX IF NOT EXISTS user_activity_logs_user_id_idx ON user_activity_logs(user_id);
CREATE INDEX IF NOT EXISTS user_activity_logs_action_idx ON user_activity_logs(action);
CREATE INDEX IF NOT EXISTS user_activity_logs_created_at_idx ON user_activity_logs(created_at);
CREATE INDEX IF NOT EXISTS user_activity_logs_deleted_at_idx ON user_activity_logs(deleted_at);

-- +migrate Down
-- Reverse: rename columns back
ALTER TABLE user_activity_logs RENAME COLUMN entity TO resource;
ALTER TABLE user_activity_logs RENAME COLUMN entity_id TO resource_id;
ALTER TABLE user_activity_logs RENAME COLUMN details TO metadata;

-- Change action field size back
ALTER TABLE user_activity_logs ALTER COLUMN action TYPE VARCHAR(100);

-- Remove deleted_at
ALTER TABLE user_activity_logs DROP COLUMN IF EXISTS deleted_at;

-- Drop indexes
DROP INDEX IF EXISTS user_activity_logs_user_id_idx;
DROP INDEX IF EXISTS user_activity_logs_action_idx;
DROP INDEX IF EXISTS user_activity_logs_created_at_idx;
DROP INDEX IF EXISTS user_activity_logs_deleted_at_idx;

-- Rename table back
ALTER TABLE user_activity_logs RENAME TO activity_logs;
