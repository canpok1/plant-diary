package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plant-diary/internal/domain"
)

// stubDiaryGeneratorWithPrompt はプロンプト付き日記生成のスタブ実装
type stubDiaryGeneratorWithPrompt struct{}

func (g *stubDiaryGeneratorWithPrompt) GenerateDiary(_ string) (string, error) {
	return "スタブ日記テキスト", nil
}

func (g *stubDiaryGeneratorWithPrompt) GenerateDiaryWithPrompt(_ string, _ string) (string, error) {
	return "スタブ生成コンテンツ", nil
}

// setupServerForPromptPreview はプロンプトプレビューテスト用のサーバーとリポジトリを返す
func setupServerForPromptPreview(t *testing.T) (*Server, *domain.MockDiaryRepository, *domain.MockBookRepository, *domain.MockUserRepository, *domain.MockSessionRepository) {
	t.Helper()

	diaryRepo := domain.NewMockDiaryRepository()
	userRepo := domain.NewMockUserRepository()
	bookRepo := domain.NewMockBookRepository()
	sessionRepo := domain.NewMockSessionRepository()
	cameraRepo := domain.NewMockCameraRepository()
	generator := &stubDiaryGeneratorWithPrompt{}

	srv, err := NewServer(diaryRepo, userRepo, bookRepo, sessionRepo, generator, cameraRepo, "../../../templates", t.TempDir())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv, diaryRepo, bookRepo, userRepo, sessionRepo
}

// TODO: 正常系: 200レスポンスでexpanded_promptとgenerated_contentが返る
// TODO: 正常系: context_diary_idありで過去日記が展開される
// TODO: 異常系 401: 未認証
// TODO: 異常系 403: 日記帳の所有者でない
// TODO: 異常系 404: 日記帳が存在しない
// TODO: 異常系 404: 画像日記が存在しない
// TODO: 異常系 404: 画像日記が別の日記帳に属する
// TODO: 異常系 400: promptが空
// TODO: 異常系 400: image_diary_idが0以下

// TestPostBookPromptPreview_Unauthorized は未認証で401を返すことを検証する
func TestPostBookPromptPreview_Unauthorized(t *testing.T) {
	srv, _, bookRepo, _, _ := setupServerForPromptPreview(t)

	book, err := bookRepo.CreateBook(1, "テスト日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	reqBody := map[string]interface{}{
		"prompt":         "テストプロンプト",
		"image_diary_id": 1,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestPostBookPromptPreview_BookNotFound は存在しない日記帳IDで404を返すことを検証する
func TestPostBookPromptPreview_BookNotFound(t *testing.T) {
	srv, _, _, userRepo, sessionRepo := setupServerForPromptPreview(t)
	_, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	reqBody := map[string]interface{}{
		"prompt":         "テストプロンプト",
		"image_diary_id": 1,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/9999/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestPostBookPromptPreview_Forbidden は日記帳の所有者でない場合に403を返すことを検証する
func TestPostBookPromptPreview_Forbidden(t *testing.T) {
	srv, _, bookRepo, userRepo, sessionRepo := setupServerForPromptPreview(t)

	// 別のユーザーが所有する日記帳を作成
	book, err := bookRepo.CreateBook(999, "他人の日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	// ログインユーザー（ID=1）でリクエスト
	_, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	reqBody := map[string]interface{}{
		"prompt":         "テストプロンプト",
		"image_diary_id": 1,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

// TestPostBookPromptPreview_EmptyPrompt はpromptが空の場合に400を返すことを検証する
func TestPostBookPromptPreview_EmptyPrompt(t *testing.T) {
	srv, diaryRepo, bookRepo, userRepo, sessionRepo := setupServerForPromptPreview(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "テスト日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	// 日記を作成
	imageDiary := createTestDiaryForBook(t, diaryRepo, book.ID)

	reqBody := map[string]interface{}{
		"prompt":         "",
		"image_diary_id": imageDiary.ID,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestPostBookPromptPreview_InvalidImageDiaryID はimage_diary_idが0以下の場合に400を返すことを検証する
func TestPostBookPromptPreview_InvalidImageDiaryID(t *testing.T) {
	srv, _, bookRepo, userRepo, sessionRepo := setupServerForPromptPreview(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "テスト日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	reqBody := map[string]interface{}{
		"prompt":         "テストプロンプト",
		"image_diary_id": 0,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestPostBookPromptPreview_ImageDiaryNotFound は画像日記が存在しない場合に404を返すことを検証する
func TestPostBookPromptPreview_ImageDiaryNotFound(t *testing.T) {
	srv, _, bookRepo, userRepo, sessionRepo := setupServerForPromptPreview(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "テスト日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	reqBody := map[string]interface{}{
		"prompt":         "テストプロンプト",
		"image_diary_id": 9999,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestPostBookPromptPreview_ImageDiaryBelongsToDifferentBook は画像日記が別の日記帳に属する場合に404を返すことを検証する
func TestPostBookPromptPreview_ImageDiaryBelongsToDifferentBook(t *testing.T) {
	srv, diaryRepo, bookRepo, userRepo, sessionRepo := setupServerForPromptPreview(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "テスト日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	// 別の日記帳の日記を作成
	otherBook, err := bookRepo.CreateBook(user.ID, "別の日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	otherDiary := createTestDiaryForBook(t, diaryRepo, otherBook.ID)

	reqBody := map[string]interface{}{
		"prompt":         "テストプロンプト",
		"image_diary_id": otherDiary.ID,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestPostBookPromptPreview_ContextDiaryNotFound はcontext_diary_idが存在しない場合に404を返すことを検証する
func TestPostBookPromptPreview_ContextDiaryNotFound(t *testing.T) {
	srv, diaryRepo, bookRepo, userRepo, sessionRepo := setupServerForPromptPreview(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "テスト日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	imageDiary := createTestDiaryForBook(t, diaryRepo, book.ID)

	contextDiaryID := 9999
	reqBody := map[string]interface{}{
		"prompt":           "テストプロンプト",
		"image_diary_id":   imageDiary.ID,
		"context_diary_id": contextDiaryID,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestPostBookPromptPreview_Success は正常系で200レスポンスを返すことを検証する
func TestPostBookPromptPreview_Success(t *testing.T) {
	srv, diaryRepo, bookRepo, userRepo, sessionRepo := setupServerForPromptPreview(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "テスト日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	imageDiary := createTestDiaryForBook(t, diaryRepo, book.ID)

	reqBody := map[string]interface{}{
		"prompt":         "テストプロンプト {{book_name}}",
		"image_diary_id": imageDiary.ID,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["expanded_prompt"]; !ok {
		t.Error("expected expanded_prompt in response")
	}
	if _, ok := resp["generated_content"]; !ok {
		t.Error("expected generated_content in response")
	}

	// book_name プレースホルダーが展開されていることを確認
	expandedPrompt, _ := resp["expanded_prompt"].(string)
	if expandedPrompt == "" {
		t.Error("expected non-empty expanded_prompt")
	}
}

// TestPostBookPromptPreview_SuccessWithContextDiary はcontext_diary_idありで200レスポンスを返すことを検証する
func TestPostBookPromptPreview_SuccessWithContextDiary(t *testing.T) {
	srv, diaryRepo, bookRepo, userRepo, sessionRepo := setupServerForPromptPreview(t)
	user, cookie := createTestSessionCookie(t, userRepo, sessionRepo)

	book, err := bookRepo.CreateBook(user.ID, "テスト日記帳")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	// 過去日記を作成
	pastDiary := createTestDiaryForBookAt(t, diaryRepo, book.ID, time.Now().Add(-10*24*time.Hour))
	// 画像日記を作成（contextDiaryより後）
	imageDiary := createTestDiaryForBookAt(t, diaryRepo, book.ID, time.Now())

	contextDiaryID := pastDiary.ID
	reqBody := map[string]interface{}{
		"prompt":           "テストプロンプト",
		"image_diary_id":   imageDiary.ID,
		"context_diary_id": contextDiaryID,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/books/"+itoa(book.ID)+"/prompt-preview", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// createTestDiaryForBook はテスト用の日記を作成して返す
func createTestDiaryForBook(t *testing.T, diaryRepo *domain.MockDiaryRepository, bookID int) *domain.Diary {
	t.Helper()
	return createTestDiaryForBookAt(t, diaryRepo, bookID, time.Now())
}

// createTestDiaryForBookAt はテスト用の日記を指定時刻で作成して返す
func createTestDiaryForBookAt(t *testing.T, diaryRepo *domain.MockDiaryRepository, bookID int, createdAt time.Time) *domain.Diary {
	t.Helper()

	if err := diaryRepo.CreateDiaryForBook(bookID, 1, "test/image.jpg", "テスト日記コンテンツ", createdAt); err != nil {
		t.Fatalf("CreateDiaryForBook failed: %v", err)
	}

	diaries, err := diaryRepo.GetDiariesByBookID(bookID)
	if err != nil || len(diaries) == 0 {
		t.Fatalf("GetDiariesByBookID failed: %v", err)
	}

	// 最新の日記を返す（CreateDiaryForBookAtで作成した日記）
	for i := range diaries {
		if diaries[i].CreatedAt.Equal(createdAt) {
			d := diaries[i]
			return &d
		}
	}
	d := diaries[0]
	return &d
}
