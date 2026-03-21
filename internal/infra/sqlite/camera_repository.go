package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"plant-diary/internal/domain"
	"plant-diary/internal/utils"
)

// SQLiteCameraRepository はSQLiteを使用したCameraRepositoryの実装
type SQLiteCameraRepository struct {
	db *sql.DB
}

// NewSQLiteCameraRepository は新しいSQLiteCameraRepositoryを生成する
func NewSQLiteCameraRepository(db *sql.DB) *SQLiteCameraRepository {
	return &SQLiteCameraRepository{db: db}
}

// CreateCamera は新しいカメラを作成する。script_keyはサーバー側で自動生成される
func (r *SQLiteCameraRepository) CreateCamera(name string, bookID int) (*domain.Camera, error) {
	scriptKey, err := utils.GenerateUUID()
	if err != nil {
		return nil, err
	}

	result, err := r.db.Exec(
		"INSERT INTO cameras (name, script_key, book_id) VALUES (?, ?, ?)",
		name, scriptKey, bookID,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetCameraByID(int(id))
}

// GetAllCameras は全てのカメラを返す
func (r *SQLiteCameraRepository) GetAllCameras() ([]domain.Camera, error) {
	rows, err := r.db.Query(
		"SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at, test_capture_requested, last_test_photo_path, last_test_photo_captured_at, capture_times_utc, last_scheduled_capture_at FROM cameras ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var cameras []domain.Camera
	for rows.Next() {
		var c domain.Camera
		if err := rows.Scan(&c.ID, &c.Name, &c.ScriptKey, &c.TargetBrightness, &c.BrightnessTolerance, &c.MaxAdjustRetries, &c.BookID, &c.CreatedAt, &c.TestCaptureRequested, &c.LastTestPhotoPath, &c.LastTestPhotoCapturedAt, &c.CaptureTimesUTC, &c.LastScheduledCaptureAt); err != nil {
			return nil, err
		}
		cameras = append(cameras, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if cameras == nil {
		return []domain.Camera{}, nil
	}

	return cameras, nil
}

// GetCameraByID は指定IDのカメラを返す。見つからない場合はnilを返す
func (r *SQLiteCameraRepository) GetCameraByID(id int) (*domain.Camera, error) {
	var c domain.Camera
	err := r.db.QueryRow(
		"SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at, test_capture_requested, last_test_photo_path, last_test_photo_captured_at, capture_times_utc, last_scheduled_capture_at FROM cameras WHERE id = ?",
		id,
	).Scan(&c.ID, &c.Name, &c.ScriptKey, &c.TargetBrightness, &c.BrightnessTolerance, &c.MaxAdjustRetries, &c.BookID, &c.CreatedAt, &c.TestCaptureRequested, &c.LastTestPhotoPath, &c.LastTestPhotoCapturedAt, &c.CaptureTimesUTC, &c.LastScheduledCaptureAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetCameraByScriptKey は指定script_keyのカメラを返す。見つからない場合はnilを返す
func (r *SQLiteCameraRepository) GetCameraByScriptKey(scriptKey string) (*domain.Camera, error) {
	var c domain.Camera
	err := r.db.QueryRow(
		"SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at, test_capture_requested, last_test_photo_path, last_test_photo_captured_at, capture_times_utc, last_scheduled_capture_at FROM cameras WHERE script_key = ?",
		scriptKey,
	).Scan(&c.ID, &c.Name, &c.ScriptKey, &c.TargetBrightness, &c.BrightnessTolerance, &c.MaxAdjustRetries, &c.BookID, &c.CreatedAt, &c.TestCaptureRequested, &c.LastTestPhotoPath, &c.LastTestPhotoCapturedAt, &c.CaptureTimesUTC, &c.LastScheduledCaptureAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateCameraTestCaptureRequested は指定IDのカメラのtest_capture_requestedを更新する
func (r *SQLiteCameraRepository) UpdateCameraTestCaptureRequested(id int, requested bool) error {
	result, err := r.db.Exec(
		"UPDATE cameras SET test_capture_requested = ? WHERE id = ?",
		requested, id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("camera %d not found", id)
	}
	return nil
}

// UpdateCamera は指定IDのカメラ設定を更新する
func (r *SQLiteCameraRepository) UpdateCamera(id int, name string, targetBrightness, brightnessTolerance float64, maxAdjustRetries, bookID int) error {
	result, err := r.db.Exec(
		"UPDATE cameras SET name = ?, target_brightness = ?, brightness_tolerance = ?, max_adjust_retries = ?, book_id = ? WHERE id = ?",
		name, targetBrightness, brightnessTolerance, maxAdjustRetries, bookID, id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("camera %d not found", id)
	}
	return nil
}

// UpdateCameraAfterTestPhoto は指定IDのカメラのテスト写真情報を更新する
func (r *SQLiteCameraRepository) UpdateCameraAfterTestPhoto(id int, lastTestPhotoPath string, capturedAt time.Time) error {
	result, err := r.db.Exec(
		"UPDATE cameras SET test_capture_requested = 0, last_test_photo_path = ?, last_test_photo_captured_at = ? WHERE id = ?",
		lastTestPhotoPath, capturedAt.UTC(), id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("camera %d not found", id)
	}
	return nil
}

// UpdateCameraAfterScheduledCapture は指定IDのカメラのスケジュール撮影情報を更新する
func (r *SQLiteCameraRepository) UpdateCameraAfterScheduledCapture(id int, capturedAt time.Time) error {
	result, err := r.db.Exec(
		"UPDATE cameras SET last_scheduled_capture_at = ? WHERE id = ?",
		capturedAt.UTC(), id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("camera %d not found", id)
	}
	return nil
}

// DeleteCamera は指定IDのカメラを削除する
func (r *SQLiteCameraRepository) DeleteCamera(id int) error {
	result, err := r.db.Exec("DELETE FROM cameras WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("camera %d not found", id)
	}
	return nil
}

// UpdateCameraScheduleConfig はスケジュール撮影の時刻設定を更新する
func (r *SQLiteCameraRepository) UpdateCameraScheduleConfig(id int, captureTimesUTC string) error {
	result, err := r.db.Exec(
		"UPDATE cameras SET capture_times_utc = ? WHERE id = ?",
		captureTimesUTC, id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("camera %d not found", id)
	}
	return nil
}

// UpdateCameraAfterScheduledCapture はスケジュール撮影後の最終撮影時刻を更新する
func (r *SQLiteCameraRepository) UpdateCameraAfterScheduledCapture(id int, capturedAt time.Time) error {
	result, err := r.db.Exec(
		"UPDATE cameras SET last_scheduled_capture_at = ? WHERE id = ?",
		capturedAt.UTC(), id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("camera %d not found", id)
	}
	return nil
}

// computeShouldScheduleCapture は capture_times_utc と last_scheduled_capture_at を元に
// スケジュール撮影が必要かどうかを判定する
func computeShouldScheduleCapture(captureTimesUTC sql.NullString, lastCaptureAt sql.NullTime, now time.Time) bool {
	if !captureTimesUTC.Valid || captureTimesUTC.String == "" {
		return false
	}
	for _, part := range strings.Split(captureTimesUTC.String, ",") {
		part = strings.TrimSpace(part)
		if len(part) != 5 {
			continue
		}
		// HH:MM をパース
		var h, m int
		n, _ := fmt.Sscanf(part, "%d:%d", &h, &m)
		if n != 2 || h < 0 || h > 23 || m < 0 || m > 59 {
			continue
		}
		t := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, time.UTC)
		if now.After(t) {
			if !lastCaptureAt.Valid || lastCaptureAt.Time.Before(t) {
				return true
			}
		}
	}
	return false
}
