package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"plant-diary/internal/domain"
)

// TODO: 正常系: 有効なscript_keyとphotoファイルで200 OKを返す
// TODO: 異常系: Authorizationヘッダーなしで401を返す
// TODO: 異常系: 無効なscript_keyで401を返す
// TODO: 異常系: photoフィールドなしで400を返す
// TODO: 正常系: ファイルが data/test-photos/{camera_id}_latest.jpg に保存される
// TODO: 正常系: UpdateCameraAfterTestPhoto が呼ばれる（test_capture_requested=false, last_test_photo_path, last_test_photo_captured_at が更新される）

// TestPostApiTestPhoto_NoAuthHeader はAuthorizationヘッダーなしで401を返すことを検証する
func TestPostApiTestPhoto_NoAuthHeader(t *testing.T) {
	srv, _, _ := setupServerWithCameraRepo(t)

	var body bytes.Buffer
	w2 := multipart.NewWriter(&body)
	fw, err := w2.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = fw.Write([]byte("fake image data"))
	w2.Close() //nolint:errcheck

	req := httptest.NewRequest("POST", "/api/test-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestPostApiTestPhoto_InvalidScriptKey は無効なscript_keyで401を返すことを検証する
func TestPostApiTestPhoto_InvalidScriptKey(t *testing.T) {
	srv, _, _ := setupServerWithCameraRepo(t)

	var body bytes.Buffer
	w2 := multipart.NewWriter(&body)
	fw, err := w2.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = fw.Write([]byte("fake image data"))
	w2.Close() //nolint:errcheck

	req := httptest.NewRequest("POST", "/api/test-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer invalidkeyxxxxxxxxxxxxxxxxxxxxxxx")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestPostApiTestPhoto_NoPhotoField はphotoフィールドなしで400を返すことを検証する
func TestPostApiTestPhoto_NoPhotoField(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/test-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestPostApiTestPhoto_ValidRequest は有効なscript_keyとphotoファイルで200 OKを返すことを検証する
func TestPostApiTestPhoto_ValidRequest(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/test-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestPostApiTestPhoto_UpdatesCameraAfterTestPhoto はUpdateCameraAfterTestPhotoが呼ばれることを検証する
func TestPostApiTestPhoto_UpdatesCameraAfterTestPhoto(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/test-photo", &body)
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
	if updatedCamera.TestCaptureRequested {
		t.Error("expected TestCaptureRequested to be false after upload")
	}
	if !updatedCamera.LastTestPhotoPath.Valid {
		t.Error("expected LastTestPhotoPath to be set")
	}
	if !updatedCamera.LastTestPhotoCapturedAt.Valid {
		t.Error("expected LastTestPhotoCapturedAt to be set")
	}
}

// TestPostApiTestPhoto_SavesFile はファイルが適切なパスに保存されることを検証する
func TestPostApiTestPhoto_SavesFile(t *testing.T) {
	srv, cameraRepo, bookRepo := setupServerWithCameraRepo(t)

	book, err := bookRepo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	imageData := []byte("fake image data for file check")

	var body bytes.Buffer
	w2 := multipart.NewWriter(&body)
	fw, err := w2.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = fw.Write(imageData)
	w2.Close() //nolint:errcheck

	req := httptest.NewRequest("POST", "/api/test-photo", &body)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+camera.ScriptKey)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// ファイルパスの確認
	updatedCamera, err := cameraRepo.GetCameraByID(camera.ID)
	if err != nil {
		t.Fatalf("GetCameraByID failed: %v", err)
	}
	if !updatedCamera.LastTestPhotoPath.Valid {
		t.Fatal("expected LastTestPhotoPath to be set")
	}

	// パスパターンの確認: {camera_id}_latest.jpg を含むこと
	expectedSuffix := domain.TestPhotoFilename(camera.ID)
	if !bytes.Contains([]byte(updatedCamera.LastTestPhotoPath.String), []byte(expectedSuffix)) {
		t.Errorf("expected path to contain %q, got %q", expectedSuffix, updatedCamera.LastTestPhotoPath.String)
	}
}
