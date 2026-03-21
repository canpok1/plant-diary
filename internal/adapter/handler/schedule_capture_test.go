package handler

import (
	"database/sql"
	"testing"
	"time"
)

// TODO: capture_times_utc が NULL（空の NullString）→ false
// TODO: capture_times_utc が空文字列 → false
// TODO: 登録時刻を過ぎていて last_scheduled_capture_at が NULL → true
// TODO: 登録時刻を過ぎていて last_scheduled_capture_at が登録時刻より前 → true
// TODO: 登録時刻をまだ過ぎていない → false
// TODO: 登録時刻を過ぎているが last_scheduled_capture_at が登録時刻より後 → false
// TODO: 複数時刻のうち1つを過ぎていて未キャプチャ → true

func TestComputeShouldScheduleCapture_NullCaptureTimesUTC(t *testing.T) {
	captureTimesUTC := sql.NullString{Valid: false}
	lastScheduledCaptureAt := sql.NullTime{Valid: false}
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	got := computeShouldScheduleCapture(captureTimesUTC, lastScheduledCaptureAt, now)

	if got != false {
		t.Errorf("expected false, got %v", got)
	}
}
