-- SQLiteはDROP COLUMNをサポートしているが、互換性のためテーブル再作成で対応
CREATE TABLE cameras_backup AS SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at FROM cameras;
DROP TABLE cameras;
ALTER TABLE cameras_backup RENAME TO cameras;
