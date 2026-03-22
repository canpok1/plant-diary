-- books テーブルに upload_key カラムを追加して元に戻す
-- SQLite では ALTER TABLE DROP COLUMN が直接サポートされていないため、テーブルを再作成する

-- 外部キー制約を一時無効化
PRAGMA foreign_keys = OFF;

-- upload_key を含む books_old テーブルを作成
CREATE TABLE books_old (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid       TEXT NOT NULL UNIQUE,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    name       TEXT NOT NULL CHECK(length(name) <= 50),
    upload_key TEXT NOT NULL UNIQUE DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 既存データを books から books_old にコピー（upload_key は空文字でデフォルト）
INSERT INTO books_old (id, uuid, creator_id, name, upload_key, created_at)
SELECT id, uuid, creator_id, name, '', created_at FROM books;

-- 旧 books テーブルを削除
DROP TABLE books;

-- books_old を books にリネーム
ALTER TABLE books_old RENAME TO books;

-- 外部キー制約を再有効化
PRAGMA foreign_keys = ON;
