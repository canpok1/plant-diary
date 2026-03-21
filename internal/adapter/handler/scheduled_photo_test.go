package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPostApiScheduledPhoto_NoAuthHeader はAuthorizationヘッダーなしで401を返すことを検証する
func TestPostApiScheduledPhoto_NoAuthHeader(t *testing.T) {
	srv, _, _ := setupServerWithCameraRepo(t)

	var body bytes.Buffer
	w2 := multipart.NewWriter(&body)
	fw, err := w2.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = fw.Write([]byte("fake image data"))
	w2.Close() //nolint:errcheck

	req := httptest.NewRequest("POST", "/api/scheduled-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestPostApiScheduledPhoto_InvalidScriptKey は無効なscript_keyで401を返すことを検証する
func TestPostApiScheduledPhoto_InvalidScriptKey(t *testing.T) {
	srv, _, _ := setupServerWithCameraRepo(t)

	var body bytes.Buffer
	w2 := multipart.NewWriter(&body)
	fw, err := w2.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = fw.Write([]byte("fake image data"))
	w2.Close() //nolint:errcheck

	req := httptest.NewRequest("POST", "/api/scheduled-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer invalidkeyxxxxxxxxxxxxxxxxxxxxxxx")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestPostApiScheduledPhoto_NoPhotoField はphotoフィールドなしで400を返すことを検証する
func TestPostApiScheduledPhoto_NoPhotoField(t *testing.T) {
	srv, cameraRepo, bookRepo := setupServerWithCameraRepo(t)

	book, err := bookRepo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	var body bytes.Buffer
	w2 := multipart.NewWriter(&body)
	w2.Close() //nolint:errcheck

	req := httptest.NewRequest("POST", "/api/scheduled-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestPostApiScheduledPhoto_ValidRequest は有効なscript_keyとphotoファイルで200 OKを返すことを検証する
func TestPostApiScheduledPhoto_ValidRequest(t *testing.T) {
	srv, cameraRepo, bookRepo := setupServerWithCameraRepo(t)

	book, err := bookRepo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	var body bytes.Buffer
	w2 := multipart.NewWriter(&body)
	fw, err := w2.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = fw.Write([]byte("fake image data"))
	w2.Close() //nolint:errcheck

	req := httptest.NewRequest("POST", "/api/scheduled-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestPostApiScheduledPhoto_UpdatesLastScheduledCaptureAt はUpdateCameraAfterScheduledCaptureが呼ばれることを検証する
func TestPostApiScheduledPhoto_UpdatesLastScheduledCaptureAt(t *testing.T) {
	srv, cameraRepo, bookRepo := setupServerWithCameraRepo(t)

	book, err := bookRepo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	var body bytes.Buffer
	w2 := multipart.NewWriter(&body)
	fw, err := w2.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = fw.Write([]byte("fake image data"))
	w2.Close() //nolint:errcheck

	req := httptest.NewRequest("POST", "/api/scheduled-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// カメラの更新確認
	updatedCamera, err := cameraRepo.GetCameraByID(camera.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if !updatedCamera.LastScheduledCaptureAt.Valid {
		t.Error("expected LastScheduledCaptureAt to be set")
	}
}
