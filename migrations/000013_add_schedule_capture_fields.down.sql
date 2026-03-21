-- SQLiteはDROP COLUMNをサポートしているが、互換性のためテーブル再作成で対応
CREATE TABLE cameras_backup (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    script_key TEXT NOT NULL UNIQUE CHECK(length(script_key) = 32),
    target_brightness REAL NOT NULL DEFAULT 0.475,
    brightness_tolerance REAL NOT NULL DEFAULT 0.175,
    max_adjust_retries INTEGER NOT NULL DEFAULT 5,
    book_id INTEGER NOT NULL REFERENCES books(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    test_capture_requested INTEGER NOT NULL DEFAULT 0,
    last_test_photo_path TEXT,
    last_test_photo_captured_at DATETIME
);
INSERT INTO cameras_backup (id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at, test_capture_requested, last_test_photo_path, last_test_photo_captured_at)
    SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at, test_capture_requested, last_test_photo_path, last_test_photo_captured_at FROM cameras;
DROP TABLE cameras;
ALTER TABLE cameras_backup RENAME TO cameras;
