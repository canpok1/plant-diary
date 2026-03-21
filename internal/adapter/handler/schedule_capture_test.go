package handler

import (
	"database/sql"
	"testing"
	"time"
)

// DONE: capture_times_utc が NULL（空の NullString）→ false
// DONE: capture_times_utc が空文字列 → false
// TODO: 登録時刻を過ぎていて last_scheduled_capture_at が NULL → true
// TODO: 登録時刻を過ぎていて last_scheduled_capture_at が登録時刻より前 → true
// TODO: 登録時刻をまだ過ぎていない → false
// TODO: 登録時刻を過ぎているが last_scheduled_capture_at が登録時刻より後 → false
// TODO: 複数時刻のうち1つを過ぎていて未キャプチャ → true

// TestComputeShouldScheduleCapture_PastTimeAndNullLastCapture は、
// 登録時刻を過ぎていて last_scheduled_capture_at が NULL の場合 true を返すことを検証する
func TestComputeShouldScheduleCapture_PastTimeAndNullLastCapture(t *testing.T) {
	// now = 12:30 UTC、登録時刻 = 12:00 UTC → 過ぎている
	captureTimesUTC := sql.NullString{Valid: true, String: "12:00"}
	lastScheduledCaptureAt := sql.NullTime{Valid: false}
	now := time.Date(2026, 3, 21, 12, 30, 0, 0, time.UTC)

	got := computeShouldScheduleCapture(captureTimesUTC, lastScheduledCaptureAt, now)

	if got != true {
		t.Errorf("expected true, got %v", got)
	}
}

func TestComputeShouldScheduleCapture_EmptyCaptureTimesUTC(t *testing.T) {
	captureTimesUTC := sql.NullString{Valid: true, String: ""}
	lastScheduledCaptureAt := sql.NullTime{Valid: false}
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	got := computeShouldScheduleCapture(captureTimesUTC, lastScheduledCaptureAt, now)

	if got != false {
		t.Errorf("expected false, got %v", got)
	}
}

func TestComputeShouldScheduleCapture_NullCaptureTimesUTC(t *testing.T) {
	captureTimesUTC := sql.NullString{Valid: false}
	lastScheduledCaptureAt := sql.NullTime{Valid: false}
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	got := computeShouldScheduleCapture(captureTimesUTC, lastScheduledCaptureAt, now)

	if got != false {
		t.Errorf("expected false, got %v", got)
	}
}
