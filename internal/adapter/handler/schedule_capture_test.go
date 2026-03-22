package handler

import (
	"database/sql"
	"testing"
	"time"
)

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
