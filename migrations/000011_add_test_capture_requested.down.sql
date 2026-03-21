-- SQLite does not support DROP COLUMN in older versions; recreate table without the column
CREATE TABLE cameras_backup AS SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at FROM cameras;
DROP TABLE cameras;
ALTER TABLE cameras_backup RENAME TO cameras;
