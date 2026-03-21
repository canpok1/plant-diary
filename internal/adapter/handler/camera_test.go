package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"plant-diary/internal/domain"
)

// setupServerWithFullMockRepos はすべてのモックリポジトリを含むテスト用Serverを生成する
func setupServerWithFullMockRepos(t *testing.T) (*Server, *domain.MockCameraRepository, *domain.MockBookRepository, *domain.MockUserRepository, *domain.MockSessionRepository) {
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
	return srv, cameraRepo, bookRepo, userRepo, sessionRepo
}

// createTestSessionCookie はテスト用セッションを作成してCookieを返す
func createTestSessionCookie(t *testing.T, userRepo *domain.MockUserRepository, sessionRepo *domain.MockSessionRepository) (*domain.User, *http.Cookie) {
	t.Helper()

	if err := userRepo.CreateUser("test-uuid", "testuser", "hash"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	user, err := userRepo.GetUserByUsername("testuser")
	if err != nil || user == nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	sessionID := "test-session-id-12345"
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := sessionRepo.CreateSession(sessionID, user.ID, expiresAt); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	cookie := &http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	}
	return user, cookie
}

// TestGetCameraTestPhoto_NoSession はセッションなしで302リダイレクトを返すことを検証する
func TestGetCameraTestPhoto_NoSession(t *testing.T) {
	srv, _, _, _, _ := setupServerWithFullMockRepos(t)

	req := httptest.NewRequest("GET", "/cameras/1/test-photo", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected status 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("expected redirect to /login, got %s", loc)
	}
}

// TestGetCameraTestPhoto_InvalidCameraID は無効なIDで404を返すことを検証する
func TestGetCameraTestPhoto_InvalidCameraID(t *testing.T) {
	srv, _, _, userRepo, sessionRepo := setupServerWithFullMockRepos(t)
	_, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	req := httptest.NewRequest("GET", "/cameras/invalid/test-photo", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestGetCameraTestPhoto_CameraNotFound は存在しないカメラIDで404を返すことを検証する
func TestGetCameraTestPhoto_CameraNotFound(t *testing.T) {
	srv, _, _, userRepo, sessionRepo := setupServerWithFullMockRepos(t)
	_, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	req := httptest.NewRequest("GET", "/cameras/999/test-photo", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestGetCameraTestPhoto_NoTestPhoto はテスト写真未撮影時に404を返すことを検証する
func TestGetCameraTestPhoto_NoTestPhoto(t *testing.T) {
	srv, cameraRepo, bookRepo, userRepo, sessionRepo := setupServerWithFullMockRepos(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/cameras/"+itoa(camera.ID)+"/test-photo", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestGetCameraTestPhoto_FileNotFound はtest_photo_pathが設定されているがファイルが存在しない場合に404を返すことを検証する
func TestGetCameraTestPhoto_FileNotFound(t *testing.T) {
	srv, cameraRepo, bookRepo, userRepo, sessionRepo := setupServerWithFullMockRepos(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	// 存在しないファイルパスを設定
	if err := cameraRepo.UpdateCameraAfterTestPhoto(camera.ID, "test-photos/nonexistent.jpg", time.Now()); err != nil {
		t.Fatalf("UpdateCameraAfterTestPhoto failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/cameras/"+itoa(camera.ID)+"/test-photo", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestGetCameraTestPhoto_Success はテスト写真が存在する場合に200とファイル内容を返すことを検証する
func TestGetCameraTestPhoto_Success(t *testing.T) {
	srv, cameraRepo, bookRepo, userRepo, sessionRepo := setupServerWithFullMockRepos(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("テストカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	// テスト写真ファイルを作成
	testPhotosDir := filepath.Join(filepath.Dir(srv.photosDir), "test-photos")
	if err := os.MkdirAll(testPhotosDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	filename := domain.TestPhotoFilename(camera.ID)
	filePath := filepath.Join(testPhotosDir, filename)
	imageData := []byte("fake jpeg data")
	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	relPath := filepath.Join("test-photos", filename)
	if err := cameraRepo.UpdateCameraAfterTestPhoto(camera.ID, relPath, time.Now()); err != nil {
		t.Fatalf("UpdateCameraAfterTestPhoto failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/cameras/"+itoa(camera.ID)+"/test-photo", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestGetCameraTestPhoto_OtherUserCamera は他ユーザーのカメラへのアクセスで404を返すことを検証する
func TestGetCameraTestPhoto_OtherUserCamera(t *testing.T) {
	srv, cameraRepo, bookRepo, userRepo, sessionRepo := setupServerWithFullMockRepos(t)
	_, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	// 別ユーザー（ID=999）のbookにカメラを作成
	book, err := bookRepo.CreateBook(999, "Other User Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	camera, err := cameraRepo.CreateCamera("他ユーザーのカメラ", book.ID)
	if err != nil {
		t.Fatalf("CreateCamera failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/cameras/"+itoa(camera.ID)+"/test-photo", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// itoa は int を string に変換するヘルパー
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
