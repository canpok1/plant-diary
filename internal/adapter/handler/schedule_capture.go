package handler

import (
	"database/sql"
	"strings"
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
	if !captureTimesUTC.Valid || captureTimesUTC.String == "" {
		return false
	}

	// カンマ区切りの時刻リストを解析
	times := splitCaptureTimesUTC(captureTimesUTC.String)
	nowDate := now.Format("2006-01-02")

	for _, t := range times {
		// "HH:MM" 形式を当日の時刻に変換
		scheduledTime, err := time.ParseInLocation("2006-01-02 15:04", nowDate+" "+t, time.UTC)
		if err != nil {
			continue
		}

		// now > scheduledTime かつ last_scheduled_capture_at が NULL または scheduledTime より前
		if now.After(scheduledTime) {
			if !lastScheduledCaptureAt.Valid || lastScheduledCaptureAt.Time.Before(scheduledTime) {
				return true
			}
		}
	}

	return false
}

// splitCaptureTimesUTC はカンマ区切りの時刻文字列を分割してトリムする
func splitCaptureTimesUTC(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
