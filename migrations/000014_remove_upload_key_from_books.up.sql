-- books テーブルから upload_key カラムを削除する
-- SQLite では ALTER TABLE DROP COLUMN が直接サポートされていないため、テーブルを再作成する

-- 外部キー制約を一時無効化
PRAGMA foreign_keys = OFF;

-- upload_key を除いた books_new テーブルを作成
CREATE TABLE books_new (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid       TEXT NOT NULL UNIQUE,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    name       TEXT NOT NULL CHECK(length(name) <= 50),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 既存データを books から books_new にコピー（upload_key を除く）
INSERT INTO books_new (id, uuid, creator_id, name, created_at)
SELECT id, uuid, creator_id, name, created_at FROM books;

-- 旧 books テーブルを削除
DROP TABLE books;

-- books_new を books にリネーム
ALTER TABLE books_new RENAME TO books;

-- 外部キー制約を再有効化
PRAGMA foreign_keys = ON;
