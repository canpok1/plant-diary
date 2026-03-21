ALTER TABLE cameras ADD COLUMN capture_times_utc TEXT;
ALTER TABLE cameras ADD COLUMN last_scheduled_capture_at DATETIME;
