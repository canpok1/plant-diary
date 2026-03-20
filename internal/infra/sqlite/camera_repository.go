package sqlite

import (
	"database/sql"
	"fmt"

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
		"SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at FROM cameras ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var cameras []domain.Camera
	for rows.Next() {
		var c domain.Camera
		if err := rows.Scan(&c.ID, &c.Name, &c.ScriptKey, &c.TargetBrightness, &c.BrightnessTolerance, &c.MaxAdjustRetries, &c.BookID, &c.CreatedAt); err != nil {
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
		"SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at FROM cameras WHERE id = ?",
		id,
	).Scan(&c.ID, &c.Name, &c.ScriptKey, &c.TargetBrightness, &c.BrightnessTolerance, &c.MaxAdjustRetries, &c.BookID, &c.CreatedAt)
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
		"SELECT id, name, script_key, target_brightness, brightness_tolerance, max_adjust_retries, book_id, created_at FROM cameras WHERE script_key = ?",
		scriptKey,
	).Scan(&c.ID, &c.Name, &c.ScriptKey, &c.TargetBrightness, &c.BrightnessTolerance, &c.MaxAdjustRetries, &c.BookID, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
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
