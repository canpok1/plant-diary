-- CHECK 制約を除いた元の cameras テーブルに戻す
CREATE TABLE cameras_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    script_key TEXT(32) NOT NULL UNIQUE,
    target_brightness REAL NOT NULL DEFAULT 0.475,
    brightness_tolerance REAL NOT NULL DEFAULT 0.175,
    max_adjust_retries INTEGER NOT NULL DEFAULT 5,
    book_id INTEGER NOT NULL REFERENCES books(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO cameras_old SELECT * FROM cameras;

DROP TABLE cameras;

ALTER TABLE cameras_old RENAME TO cameras;
