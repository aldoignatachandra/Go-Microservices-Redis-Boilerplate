-- +migrate Up
CREATE TABLE IF NOT EXISTS user_activity_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(255) NOT NULL,
    entity VARCHAR(100),
    entity_id UUID,
    ip_address VARCHAR(45),
    user_agent TEXT,
    details JSONB
);

CREATE INDEX IF NOT EXISTS user_activity_logs_user_id_idx ON user_activity_logs(user_id);
CREATE INDEX IF NOT EXISTS user_activity_logs_action_idx ON user_activity_logs(action);
CREATE INDEX IF NOT EXISTS user_activity_logs_created_at_idx ON user_activity_logs(created_at);
CREATE INDEX IF NOT EXISTS user_activity_logs_deleted_at_idx ON user_activity_logs(deleted_at);

-- +migrate Down
DROP TABLE IF EXISTS user_activity_logs;
