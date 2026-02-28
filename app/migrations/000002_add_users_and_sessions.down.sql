DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP TABLE IF EXISTS sessions;

DROP INDEX IF EXISTS idx_users_uuid;
DROP TABLE IF EXISTS users;

ALTER TABLE diary DROP COLUMN updated_at;
