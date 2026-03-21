package sqlite

import (
	"database/sql"
	"testing"

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
			test_capture_requested INTEGER NOT NULL DEFAULT 0
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
