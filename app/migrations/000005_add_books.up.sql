-- booksテーブルを作成
CREATE TABLE IF NOT EXISTS books (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid       TEXT NOT NULL UNIQUE,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    name       TEXT NOT NULL,
    upload_key TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- diaryテーブルにbook_idカラムを追加
ALTER TABLE diary ADD COLUMN book_id INTEGER REFERENCES books(id);

-- systemユーザーのデフォルト日記帳を作成
INSERT INTO books (uuid, creator_id, name, upload_key, created_at)
SELECT
    '00000000000000000000000000000001',
    id,
    'デフォルト',
    '00000000000000000000000000000002',
    CURRENT_TIMESTAMP
FROM users WHERE username = 'system';

-- 既存のdiaryをデフォルト日記帳に紐付け
UPDATE diary SET book_id = (
    SELECT books.id FROM books
    JOIN users ON books.creator_id = users.id
    WHERE users.username = 'system'
    LIMIT 1
)
WHERE book_id IS NULL;
