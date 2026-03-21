package sqlite

import (
	"database/sql"
	"testing"
	"time"

	"plant-diary/internal/domain"
)

func setupTestDBWithCameras(t *testing.T) *sql.DB {
	t.Helper()
	db := setupTestDB(t)
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cameras (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			script_key TEXT(32) NOT NULL UNIQUE,
			target_brightness REAL NOT NULL DEFAULT 0.475,
			brightness_tolerance REAL NOT NULL DEFAULT 0.175,
			max_adjust_retries INTEGER NOT NULL DEFAULT 5,
			book_id INTEGER NOT NULL REFERENCES books(id),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			test_capture_requested INTEGER NOT NULL DEFAULT 0,
			last_test_photo_path TEXT,
			last_test_photo_captured_at DATETIME,
			capture_times_utc TEXT,
			last_scheduled_capture_at DATETIME
		);
	`)
	if err != nil {
		t.Fatalf("failed to create cameras table: %v", err)
	}
	return db
}

func TestSQLiteCameraRepository_CreateCamera(t *testing.T) {
	db := setupTestDBWithCameras(t)
	userRepo := NewSQLiteUserRepository(db)
	bookRepo := NewSQLiteBookRepository(db)
	cameraRepo := NewSQLiteCameraRepository(db)

	if err := userRepo.CreateUser("uuid-001", "alice", "hash"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	alice, err := userRepo.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	book, err := bookRepo.CreateBook(alice.ID, "Alice's Garden")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}
	if camera == nil {
		t.Fatal("expected camera, got nil")
	}
	if camera.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if camera.Name != "テストカメラ" {
		t.Errorf("expected Name 'テストカメラ', got '%s'", camera.Name)
	}
	if len(camera.ScriptKey) != 32 {
		t.Errorf("expected ScriptKey length 32, got %d", len(camera.ScriptKey))
	}
	if camera.TargetBrightness != 0.475 {
		t.Errorf("expected TargetBrightness 0.475, got %f", camera.TargetBrightness)
	}
	if camera.BrightnessTolerance != 0.175 {
		t.Errorf("expected BrightnessTolerance 0.175, got %f", camera.BrightnessTolerance)
	}
	if camera.MaxAdjustRetries != 5 {
		t.Errorf("expected MaxAdjustRetries 5, got %d", camera.MaxAdjustRetries)
	}
	if camera.BookID != book.ID {
		t.Errorf("expected BookID %d, got %d", book.ID, camera.BookID)
	}
	if camera.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestSQLiteCameraRepository_GetAllCameras(t *testing.T) {
	db := setupTestDBWithCameras(t)
	userRepo := NewSQLiteUserRepository(db)
	bookRepo := NewSQLiteBookRepository(db)
	cameraRepo := NewSQLiteCameraRepository(db)

	// 空の場合は空スライスを返す
	cameras, err := cameraRepo.GetAllCameras()
	if err != nil {
		t.Fatalf("GetAllCameras failed: %v", err)
	}
	if len(cameras) != 0 {
		t.Errorf("expected 0 cameras, got %d", len(cameras))
	}

	if err := userRepo.CreateUser("uuid-001", "alice", "hash"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	alice, err := userRepo.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	book, err := bookRepo.CreateBook(alice.ID, "Alice's Garden")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	if _, err := cameraRepo.CreateCamera("カメラ1", book.ID); err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}
	if _, err := cameraRepo.CreateCamera("カメラ2", book.ID); err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	cameras, err = cameraRepo.GetAllCameras()
	if err != nil {
		t.Fatalf("GetAllCameras failed: %v", err)
	}
	if len(cameras) != 2 {
		t.Errorf("expected 2 cameras, got %d", len(cameras))
	}
}

func TestSQLiteCameraRepository_GetCameraByID(t *testing.T) {
	db := setupTestDBWithCameras(t)
	userRepo := NewSQLiteUserRepository(db)
	bookRepo := NewSQLiteBookRepository(db)
	cameraRepo := NewSQLiteCameraRepository(db)

	if err := userRepo.CreateUser("uuid-001", "alice", "hash"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	alice, err := userRepo.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	book, err := bookRepo.CreateBook(alice.ID, "Alice's Garden")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	created, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	camera, err := cameraRepo.GetCameraByID(created.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if camera == nil {
		t.Fatal("expected camera, got nil")
	}
	if camera.Name != "テストカメラ" {
		t.Errorf("expected Name 'テストカメラ', got '%s'", camera.Name)
	}

	// 存在しないIDはnilを返す
	notFound, err := cameraRepo.GetCameraByID(9999)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-existent ID, got %v", notFound)
	}
}

func TestSQLiteCameraRepository_GetCameraByScriptKey(t *testing.T) {
	db := setupTestDBWithCameras(t)
	userRepo := NewSQLiteUserRepository(db)
	bookRepo := NewSQLiteBookRepository(db)
	cameraRepo := NewSQLiteCameraRepository(db)

	if err := userRepo.CreateUser("uuid-001", "alice", "hash"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	alice, err := userRepo.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	book, err := bookRepo.CreateBook(alice.ID, "Alice's Garden")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	created, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	camera, err := cameraRepo.GetCameraByScriptKey(created.ScriptKey)
	if err != nil {
		t.Fatalf("GetCameraByScriptKey failed: %v", err)
	}
	if camera == nil {
		t.Fatal("expected camera, got nil")
	}
	if camera.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, camera.ID)
	}

	// 存在しないkeyはnilを返す
	notFound, err := cameraRepo.GetCameraByScriptKey("nonexistentkey00000000000000000")
	if err != nil {
		t.Fatalf("GetCameraByScriptKey failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-existent key, got %v", notFound)
	}
}

func TestSQLiteCameraRepository_UpdateCamera(t *testing.T) {
	db := setupTestDBWithCameras(t)
	userRepo := NewSQLiteUserRepository(db)
	bookRepo := NewSQLiteBookRepository(db)
	cameraRepo := NewSQLiteCameraRepository(db)

	if err := userRepo.CreateUser("uuid-001", "alice", "hash"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	alice, err := userRepo.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	book, err := bookRepo.CreateBook(alice.ID, "Alice's Garden")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	book2, err := bookRepo.CreateBook(alice.ID, "Book 2")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	created, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	if err := cameraRepo.UpdateCamera(created.ID, "更新カメラ", 0.5, 0.2, 10, book2.ID); err != nil {
		t.Fatalf("UpdateCamera failed: %v", err)
	}

	updated, err := cameraRepo.GetCameraByID(created.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if updated.Name != "更新カメラ" {
		t.Errorf("expected Name '更新カメラ', got '%s'", updated.Name)
	}
	if updated.TargetBrightness != 0.5 {
		t.Errorf("expected TargetBrightness 0.5, got %f", updated.TargetBrightness)
	}
	if updated.BrightnessTolerance != 0.2 {
		t.Errorf("expected BrightnessTolerance 0.2, got %f", updated.BrightnessTolerance)
	}
	if updated.MaxAdjustRetries != 10 {
		t.Errorf("expected MaxAdjustRetries 10, got %d", updated.MaxAdjustRetries)
	}
	if updated.BookID != book2.ID {
		t.Errorf("expected BookID %d, got %d", book2.ID, updated.BookID)
	}
}

func TestSQLiteCameraRepository_DeleteCamera(t *testing.T) {
	db := setupTestDBWithCameras(t)
	userRepo := NewSQLiteUserRepository(db)
	bookRepo := NewSQLiteBookRepository(db)
	cameraRepo := NewSQLiteCameraRepository(db)

	if err := userRepo.CreateUser("uuid-001", "alice", "hash"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	alice, err := userRepo.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	book, err := bookRepo.CreateBook(alice.ID, "Alice's Garden")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	created, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	if err := cameraRepo.DeleteCamera(created.ID); err != nil {
		t.Fatalf("DeleteCamera failed: %v", err)
	}

	// 削除後はnilを返す
	camera, err := cameraRepo.GetCameraByID(created.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if camera != nil {
		t.Errorf("expected nil after deletion, got %v", camera)
	}

	// 存在しないIDを削除するとエラー
	if err := cameraRepo.DeleteCamera(9999); err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestSQLiteCameraRepository_ImplementsInterface(t *testing.T) {
	db := setupTestDBWithCameras(t)
	// コンパイル時にインターフェースを満たすことを確認
	var _ domain.CameraRepository = NewSQLiteCameraRepository(db)
}

func setupTestCamera(t *testing.T, db *sql.DB) *domain.Camera {
	t.Helper()
	userRepo := NewSQLiteUserRepository(db)
	bookRepo := NewSQLiteBookRepository(db)
	cameraRepo := NewSQLiteCameraRepository(db)

	if err := userRepo.CreateUser("uuid-001", "alice", "hash"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	alice, err := userRepo.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	book, err := bookRepo.CreateBook(alice.ID, "Alice's Garden")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}
	return camera
}

func TestSQLiteCameraRepository_UpdateCameraTestCaptureRequested(t *testing.T) {
	db := setupTestDBWithCameras(t)
	camera := setupTestCamera(t, db)
	cameraRepo := NewSQLiteCameraRepository(db)

	// true に更新
	if err := cameraRepo.UpdateCameraTestCaptureRequested(camera.ID, true); err != nil {
		t.Fatalf("UpdateCameraTestCaptureRequested(true) failed: %v", err)
	}
	updated, err := cameraRepo.GetCameraByID(camera.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if updated.TestCaptureRequested != 1 {
		t.Errorf("expected TestCaptureRequested 1, got %d", updated.TestCaptureRequested)
	}

	// false に更新
	if err := cameraRepo.UpdateCameraTestCaptureRequested(camera.ID, false); err != nil {
		t.Fatalf("UpdateCameraTestCaptureRequested(false) failed: %v", err)
	}
	updated, err = cameraRepo.GetCameraByID(camera.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if updated.TestCaptureRequested != 0 {
		t.Errorf("expected TestCaptureRequested 0, got %d", updated.TestCaptureRequested)
	}

	// 存在しないIDはエラー
	if err := cameraRepo.UpdateCameraTestCaptureRequested(9999, true); err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestSQLiteCameraRepository_UpdateCameraAfterTestPhoto(t *testing.T) {
	db := setupTestDBWithCameras(t)
	camera := setupTestCamera(t, db)
	cameraRepo := NewSQLiteCameraRepository(db)

	photoPath := "/photos/test/photo_001.jpg"
	capturedAt := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	if err := cameraRepo.UpdateCameraAfterTestPhoto(camera.ID, photoPath, capturedAt); err != nil {
		t.Fatalf("UpdateCameraAfterTestPhoto failed: %v", err)
	}

	updated, err := cameraRepo.GetCameraByID(camera.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if !updated.LastTestPhotoPath.Valid {
		t.Error("expected LastTestPhotoPath to be valid")
	}
	if updated.LastTestPhotoPath.String != photoPath {
		t.Errorf("expected LastTestPhotoPath %q, got %q", photoPath, updated.LastTestPhotoPath.String)
	}
	if !updated.LastTestPhotoCapturedAt.Valid {
		t.Error("expected LastTestPhotoCapturedAt to be valid")
	}
	if !updated.LastTestPhotoCapturedAt.Time.Equal(capturedAt) {
		t.Errorf("expected LastTestPhotoCapturedAt %v, got %v", capturedAt, updated.LastTestPhotoCapturedAt.Time)
	}
	// TestCaptureRequested がクリアされること
	if updated.TestCaptureRequested != 0 {
		t.Errorf("expected TestCaptureRequested 0 after test photo, got %d", updated.TestCaptureRequested)
	}

	// 存在しないIDはエラー
	if err := cameraRepo.UpdateCameraAfterTestPhoto(9999, photoPath, capturedAt); err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestSQLiteCameraRepository_UpdateCameraScheduleConfig(t *testing.T) {
	db := setupTestDBWithCameras(t)
	camera := setupTestCamera(t, db)
	cameraRepo := NewSQLiteCameraRepository(db)

	captureTimesUTC := "03:00,09:00"

	if err := cameraRepo.UpdateCameraScheduleConfig(camera.ID, captureTimesUTC); err != nil {
		t.Fatalf("UpdateCameraScheduleConfig failed: %v", err)
	}

	updated, err := cameraRepo.GetCameraByID(camera.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if !updated.CaptureTimesUTC.Valid {
		t.Error("expected CaptureTimesUTC to be valid")
	}
	if updated.CaptureTimesUTC.String != captureTimesUTC {
		t.Errorf("expected CaptureTimesUTC %q, got %q", captureTimesUTC, updated.CaptureTimesUTC.String)
	}

	// 存在しないIDはエラー
	if err := cameraRepo.UpdateCameraScheduleConfig(9999, captureTimesUTC); err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestSQLiteCameraRepository_UpdateCameraAfterScheduledCapture(t *testing.T) {
	db := setupTestDBWithCameras(t)
	camera := setupTestCamera(t, db)
	cameraRepo := NewSQLiteCameraRepository(db)

	capturedAt := time.Date(2026, 3, 21, 3, 0, 0, 0, time.UTC)

	if err := cameraRepo.UpdateCameraAfterScheduledCapture(camera.ID, capturedAt); err != nil {
		t.Fatalf("UpdateCameraAfterScheduledCapture failed: %v", err)
	}

	updated, err := cameraRepo.GetCameraByID(camera.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if !updated.LastScheduledCaptureAt.Valid {
		t.Error("expected LastScheduledCaptureAt to be valid")
	}
	if !updated.LastScheduledCaptureAt.Time.Equal(capturedAt) {
		t.Errorf("expected LastScheduledCaptureAt %v, got %v", capturedAt, updated.LastScheduledCaptureAt.Time)
	}

	// 存在しないIDはエラー
	if err := cameraRepo.UpdateCameraAfterScheduledCapture(9999, capturedAt); err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestComputeShouldScheduleCapture(t *testing.T) {
	baseTime := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	captureTime := time.Date(2026, 3, 21, 9, 0, 0, 0, time.UTC) // "09:00" UTC

	tests := []struct {
		name            string
		captureTimesUTC sql.NullString
		lastCaptureAt   sql.NullTime
		now             time.Time
		want            bool
	}{
		{
			name:            "capture_times_utcがNULL→false",
			captureTimesUTC: sql.NullString{Valid: false},
			lastCaptureAt:   sql.NullTime{Valid: false},
			now:             baseTime,
			want:            false,
		},
		{
			name:            "capture_times_utcが空→false",
			captureTimesUTC: sql.NullString{String: "", Valid: true},
			lastCaptureAt:   sql.NullTime{Valid: false},
			now:             baseTime,
			want:            false,
		},
		{
			name:            "nowが登録時刻より前→false",
			captureTimesUTC: sql.NullString{String: "12:00", Valid: true},
			lastCaptureAt:   sql.NullTime{Valid: false},
			now:             time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			want:            false,
		},
		{
			name:            "lastCaptureAtがNULLかつnow>T→true",
			captureTimesUTC: sql.NullString{String: "09:00", Valid: true},
			lastCaptureAt:   sql.NullTime{Valid: false},
			now:             baseTime,
			want:            true,
		},
		{
			name:            "lastCaptureAt<Tかつnow>T→true",
			captureTimesUTC: sql.NullString{String: "09:00", Valid: true},
			lastCaptureAt:   sql.NullTime{Time: captureTime.Add(-time.Hour), Valid: true},
			now:             baseTime,
			want:            true,
		},
		{
			name:            "lastCaptureAt==T→false",
			captureTimesUTC: sql.NullString{String: "09:00", Valid: true},
			lastCaptureAt:   sql.NullTime{Time: captureTime, Valid: true},
			now:             baseTime,
			want:            false,
		},
		{
			name:            "lastCaptureAt>T→false",
			captureTimesUTC: sql.NullString{String: "09:00", Valid: true},
			lastCaptureAt:   sql.NullTime{Time: captureTime.Add(time.Hour), Valid: true},
			now:             baseTime,
			want:            false,
		},
		{
			name:            "複数時刻・いずれかが条件を満たす→true",
			captureTimesUTC: sql.NullString{String: "03:00,09:00", Valid: true},
			lastCaptureAt:   sql.NullTime{Valid: false},
			now:             baseTime,
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeShouldScheduleCapture(tt.captureTimesUTC, tt.lastCaptureAt, tt.now)
			if got != tt.want {
				t.Errorf("computeShouldScheduleCapture() = %v, want %v", got, tt.want)
			}
		})
	}
}
