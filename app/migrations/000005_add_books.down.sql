-- book_idカラムを削除
ALTER TABLE diary DROP COLUMN book_id;

-- booksテーブルを削除
DROP TABLE IF EXISTS books;
