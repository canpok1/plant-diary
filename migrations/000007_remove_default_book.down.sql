-- デフォルト日記帳を再作成する（ロールバック用）
INSERT INTO books (uuid, creator_id, name, upload_key, created_at)
SELECT
    '00000000000000000000000000000001',
    id,
    'デフォルト',
    '00000000000000000000000000000002',
    CURRENT_TIMESTAMP
FROM users WHERE username = 'system';
