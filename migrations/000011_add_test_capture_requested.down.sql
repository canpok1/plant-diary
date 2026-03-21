-- SQLite does not support DROP COLUMN in older versions; recreate table without the column
CREATE TABLE cameras_backup (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    script_key TEXT NOT NULL UNIQUE CHECK(length(script_key) = 32),
    target_brightness REAL NOT NULL DEFAULT 0.475,
    brightness_tolerance REAL NOT NULL DEFAULT 0.175,
    max_adjust_retries INTEGER NOT NULL DEFAULT 5,
    book_id INTEGER NOT NULL REFERENCES books(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO cameras_backup (id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at)
    SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at FROM cameras;
DROP TABLE cameras;
ALTER TABLE cameras_backup RENAME TO cameras;
