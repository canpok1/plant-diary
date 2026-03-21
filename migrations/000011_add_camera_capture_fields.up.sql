-- テスト撮影
ALTER TABLE cameras ADD COLUMN test_capture_requested INTEGER NOT NULL DEFAULT 0;
ALTER TABLE cameras ADD COLUMN last_test_photo_path TEXT;
ALTER TABLE cameras ADD COLUMN last_test_photo_captured_at DATETIME;  -- UTC

-- スケジュール撮影
ALTER TABLE cameras ADD COLUMN capture_times_utc TEXT;
ALTER TABLE cameras ADD COLUMN last_scheduled_capture_at DATETIME;  -- UTC
