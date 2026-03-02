-- systemユーザーを作成（ログイン不可、既存データ移行用）
INSERT OR IGNORE INTO users (uuid, username, password_hash, created_at)
VALUES ('00000000000000000000000000000000', 'system', 'DISABLED', CURRENT_TIMESTAMP);

-- user_idカラム追加（NULL許容）
ALTER TABLE diary ADD COLUMN user_id INTEGER REFERENCES users(id);

-- 既存データをsystemユーザーに割り当て
UPDATE diary SET user_id = (SELECT id FROM users WHERE username = 'system')
WHERE user_id IS NULL;
