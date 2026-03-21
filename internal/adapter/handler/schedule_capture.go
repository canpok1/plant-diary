package handler

import (
	"database/sql"
	"time"
)

// computeShouldScheduleCapture はスケジュール撮影が必要かどうかを判定する。
// 仕様:
//  1. capture_times_utc が NULL または空 → false
//  2. いずれかの登録時刻 T について:
//     - now > T かつ
//     - last_scheduled_capture_at IS NULL または last_scheduled_capture_at < T
//     → true
func computeShouldScheduleCapture(captureTimesUTC sql.NullString, lastScheduledCaptureAt sql.NullTime, now time.Time) bool {
	return false
}
