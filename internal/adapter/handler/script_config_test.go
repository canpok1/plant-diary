package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"plant-diary/internal/domain"
)

// setupServerWithCameraRepo はcameraRepoを含むテスト用のServerを生成する
func setupServerWithCameraRepo(t *testing.T) (*Server, *domain.MockCameraRepository, *domain.MockBookRepository) {
	t.Helper()

	diaryRepo := domain.NewMockDiaryRepository()
	userRepo := domain.NewMockUserRepository()
	bookRepo := domain.NewMockBookRepository()
	sessionRepo := domain.NewMockSessionRepository()
	cameraRepo := domain.NewMockCameraRepository()

	srv, err := NewServer(diaryRepo, userRepo, bookRepo, sessionRepo, nil, cameraRepo, "../../../templates", t.TempDir())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv, cameraRepo, bookRepo
}

// TestGetApiScriptConfig_ValidScriptKey は有効なscript_keyで200とJSONを返すことを検証する
func TestGetApiScriptConfig_ValidScriptKey(t *testing.T) {
	srv, cameraRepo, bookRepo := setupServerWithCameraRepo(t)

	// ユーザーと日記帳を作成
	book, err := bookRepo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/script-config", nil)
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["target_brightness"] == nil {
		t.Error("expected target_brightness in response")
	}
	if resp["brightness_tolerance"] == nil {
		t.Error("expected brightness_tolerance in response")
	}
	if resp["max_adjust_retries"] == nil {
		t.Error("expected max_adjust_retries in response")
	}
	if _, ok := resp["upload_key"]; ok {
		t.Error("upload_key should not be in response")
	}

	// should_test_capture が含まれること（デフォルトは false）
	if _, ok := resp["should_test_capture"]; !ok {
		t.Error("expected should_test_capture in response")
	}
	if resp["should_test_capture"] != false {
		t.Errorf("expected should_test_capture false, got %v", resp["should_test_capture"])
	}
}

// TestGetApiScriptConfig_InvalidScriptKey は無効なscript_keyで401を返すことを検証する
func TestGetApiScriptConfig_InvalidScriptKey(t *testing.T) {
	srv, _, _ := setupServerWithCameraRepo(t)

	req := httptest.NewRequest("GET", "/api/script-config", nil)
	req.Header.Set("Authorization", "Bearer invalidkey000000000000000000000")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestGetApiScriptConfig_ShouldTestCapture_True はTestCaptureRequestedがtrueのとき should_test_capture=true を返すことを検証する
func TestGetApiScriptConfig_ShouldTestCapture_True(t *testing.T) {
	srv, cameraRepo, bookRepo := setupServerWithCameraRepo(t)

	book, err := bookRepo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	// TestCaptureRequested を true に設定
	if err := cameraRepo.UpdateCameraTestCaptureRequested(camera.ID, true); err != nil {
		t.Fatalf("UpdateCameraTestCaptureRequested failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/script-config", nil)
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["should_test_capture"] != true {
		t.Errorf("expected should_test_capture true, got %v", resp["should_test_capture"])
	}
}

// TestGetApiScriptConfig_ShouldScheduleCapture_InResponse は should_schedule_capture がレスポンスに含まれることを検証する
func TestGetApiScriptConfig_ShouldScheduleCapture_InResponse(t *testing.T) {
	srv, cameraRepo, bookRepo := setupServerWithCameraRepo(t)

	book, err := bookRepo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/script-config", nil)
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := resp["should_schedule_capture"]; !ok {
		t.Error("expected should_schedule_capture in response")
	}
}

// TestGetApiScriptConfig_NoBearerToken はBearerトークンなしで401を返すことを検証する
func TestGetApiScriptConfig_NoBearerToken(t *testing.T) {
	srv, _, _ := setupServerWithCameraRepo(t)

	req := httptest.NewRequest("GET", "/api/script-config", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}
