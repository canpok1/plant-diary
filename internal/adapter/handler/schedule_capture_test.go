package handler

import (
	"database/sql"
	"testing"
	"time"
)

// DONE: capture_times_utc が NULL（空の NullString）→ false
// DONE: capture_times_utc が空文字列 → false
// DONE: 登録時刻を過ぎていて last_scheduled_capture_at が NULL → true
// DONE: 登録時刻を過ぎていて last_scheduled_capture_at が登録時刻より前 → true
// DONE: 登録時刻をまだ過ぎていない → false
// DONE: 登録時刻を過ぎているが last_scheduled_capture_at が登録時刻より後 → false
// DONE: 複数時刻のうち1つを過ぎていて未キャプチャ → true

// TestComputeShouldScheduleCapture_PastTimeAndNullLastCapture は、
// 登録時刻を過ぎていて last_scheduled_capture_at が NULL の場合 true を返すことを検証する
// TestComputeShouldScheduleCapture_PastTimeAndLastCaptureBeforeScheduled は、
// 登録時刻を過ぎていて last_scheduled_capture_at が登録時刻より前の場合 true を返すことを検証する
// TestComputeShouldScheduleCapture_FutureTime は、
// 登録時刻をまだ過ぎていない場合 false を返すことを検証する
// TestComputeShouldScheduleCapture_PastTimeButAlreadyCaptured は、
// 登録時刻を過ぎているが last_scheduled_capture_at が登録時刻より後の場合 false を返すことを検証する
// TestComputeShouldScheduleCapture_MultipleTimes_OnePastAndUncaptured は、
// 複数時刻のうち1つを過ぎていて未キャプチャの場合 true を返すことを検証する
func TestComputeShouldScheduleCapture_MultipleTimes_OnePastAndUncaptured(t *testing.T) {
	// now = 10:30 UTC
	// 登録時刻1 = 03:00 UTC → 過ぎている（未キャプチャ）
	// 登録時刻2 = 15:00 UTC → まだ過ぎていない
	captureTimesUTC := sql.NullString{Valid: true, String: "03:00,15:00"}
	lastScheduledCaptureAt := sql.NullTime{Valid: false}
	now := time.Date(2026, 3, 21, 10, 30, 0, 0, time.UTC)

	got := computeShouldScheduleCapture(captureTimesUTC, lastScheduledCaptureAt, now)

	if got != true {
		t.Errorf("expected true, got %v", got)
	}
}

func TestComputeShouldScheduleCapture_PastTimeButAlreadyCaptured(t *testing.T) {
	// now = 12:30 UTC、登録時刻 = 12:00 UTC → 過ぎている
	// last_scheduled_capture_at = 12:10 UTC → 登録時刻より後（撮影済み）
	captureTimesUTC := sql.NullString{Valid: true, String: "12:00"}
	lastScheduledCaptureAt := sql.NullTime{
		Valid: true,
		Time:  time.Date(2026, 3, 21, 12, 10, 0, 0, time.UTC),
	}
	now := time.Date(2026, 3, 21, 12, 30, 0, 0, time.UTC)

	got := computeShouldScheduleCapture(captureTimesUTC, lastScheduledCaptureAt, now)

	if got != false {
		t.Errorf("expected false, got %v", got)
	}
}

func TestComputeShouldScheduleCapture_FutureTime(t *testing.T) {
	// now = 11:30 UTC、登録時刻 = 12:00 UTC → まだ過ぎていない
	captureTimesUTC := sql.NullString{Valid: true, String: "12:00"}
	lastScheduledCaptureAt := sql.NullTime{Valid: false}
	now := time.Date(2026, 3, 21, 11, 30, 0, 0, time.UTC)

	got := computeShouldScheduleCapture(captureTimesUTC, lastScheduledCaptureAt, now)

	if got != false {
		t.Errorf("expected false, got %v", got)
	}
}

func TestComputeShouldScheduleCapture_PastTimeAndLastCaptureBeforeScheduled(t *testing.T) {
	// now = 12:30 UTC、登録時刻 = 12:00 UTC → 過ぎている
	// last_scheduled_capture_at = 11:00 UTC → 登録時刻より前
	captureTimesUTC := sql.NullString{Valid: true, String: "12:00"}
	lastScheduledCaptureAt := sql.NullTime{
		Valid: true,
		Time:  time.Date(2026, 3, 21, 11, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, 3, 21, 12, 30, 0, 0, time.UTC)

	got := computeShouldScheduleCapture(captureTimesUTC, lastScheduledCaptureAt, now)

	if got != true {
		t.Errorf("expected true, got %v", got)
	}
}

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
