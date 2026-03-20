-- cameras テーブルを再作成して script_key に CHECK 制約を追加する
-- SQLite では ALTER TABLE で CHECK 制約を追加できないため、テーブルを再作成する
CREATE TABLE cameras_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    script_key TEXT NOT NULL UNIQUE CHECK(length(script_key) = 32),
    target_brightness REAL NOT NULL DEFAULT 0.475,
    brightness_tolerance REAL NOT NULL DEFAULT 0.175,
    max_adjust_retries INTEGER NOT NULL DEFAULT 5,
    book_id INTEGER NOT NULL REFERENCES books(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO cameras_new SELECT * FROM cameras;

DROP TABLE cameras;

ALTER TABLE cameras_new RENAME TO cameras;
