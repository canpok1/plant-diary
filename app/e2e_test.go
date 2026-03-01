//go:build e2e

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// setupE2EServer はE2Eテスト用のHTTPサーバーを設定して起動する
func setupE2EServer(t *testing.T) *httptest.Server {
	t.Helper()

	ts, _ := setupE2EServerWithDB(t)
	return ts
}

// setupE2EServerWithDB はE2Eテスト用のHTTPサーバーとDBを設定して起動する
func setupE2EServerWithDB(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()

	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)
	userRepo := NewSQLiteUserRepository(db)
	bookRepo := NewSQLiteBookRepository(db)
	sessionRepo := NewSQLiteSessionRepository(db)
	generator := &MockDiaryGenerator{}

	srv, err := NewServer(repo, userRepo, bookRepo, sessionRepo, generator, t.TempDir())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts, db
}

// loginAsUser はユーザーでログインしてセッションCookieを返す
func loginAsUser(t *testing.T, ts *httptest.Server, username, password string) []*http.Cookie {
	t.Helper()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("password", password)
	resp, err := client.PostForm(ts.URL+"/login", formData)
	if err != nil {
		t.Fatalf("POST /login failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("login failed with status %d", resp.StatusCode)
	}
	return resp.Cookies()
}

// TestE2E_GetIndex はGET /がHTMLを返すことを検証する
func TestE2E_GetIndex(t *testing.T) {
	ts := setupE2EServer(t)

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected Content-Type text/html, got %s", contentType)
	}
}

// TestE2E_GetLogin はGET /loginがログインフォームを返すことを検証する
func TestE2E_GetLogin(t *testing.T) {
	ts := setupE2EServer(t)

	resp, err := http.Get(ts.URL + "/login")
	if err != nil {
		t.Fatalf("GET /login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected Content-Type text/html, got %s", contentType)
	}
}

// TestE2E_PostApiUsers_CreateUser はPOST /api/usersでユーザーが作成できることを検証する
func TestE2E_PostApiUsers_CreateUser(t *testing.T) {
	ts := setupE2EServer(t)

	reqBody := `{"username": "testuser", "password": "testpass"}`
	resp, err := http.Post(ts.URL+"/api/users", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /api/users failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var result UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got '%s'", result.Username)
	}
	if result.Uuid == "" {
		t.Error("expected non-empty Uuid")
	}
}

// TestE2E_PostApiUsers_DuplicateUsername は重複ユーザー名でPOST /api/usersが400を返すことを検証する
func TestE2E_PostApiUsers_DuplicateUsername(t *testing.T) {
	ts := setupE2EServer(t)

	reqBody := `{"username": "testuser", "password": "testpass"}`

	// 1回目：成功
	resp1, err := http.Post(ts.URL+"/api/users", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("first POST /api/users failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first user creation failed with status %d", resp1.StatusCode)
	}

	// 2回目：重複エラー
	resp2, err := http.Post(ts.URL+"/api/users", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("second POST /api/users failed: %v", err)
	}
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for duplicate username, got %d", resp2.StatusCode)
	}
}

// TestE2E_Login_Success はログイン成功で/へリダイレクトされることを検証する
func TestE2E_Login_Success(t *testing.T) {
	ts := setupE2EServer(t)

	// ユーザー作成
	reqBody := `{"username": "testuser", "password": "testpass"}`
	createResp, err := http.Post(ts.URL+"/api/users", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /api/users failed: %v", err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("user creation failed with status %d", createResp.StatusCode)
	}

	// リダイレクトを追わないクライアント
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// ログイン
	formData := url.Values{}
	formData.Set("username", "testuser")
	formData.Set("password", "testpass")
	loginResp, err := client.PostForm(ts.URL+"/login", formData)
	if err != nil {
		t.Fatalf("POST /login failed: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusFound {
		t.Errorf("expected status 302, got %d", loginResp.StatusCode)
	}

	location := loginResp.Header.Get("Location")
	if location != "/" {
		t.Errorf("expected redirect to /, got %s", location)
	}

	// セッションCookieが設定されていることを確認
	cookies := loginResp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_id" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Error("expected session_id cookie to be set")
	}
}

// TestE2E_Login_InvalidCredentials は誤った認証情報でログインが失敗することを検証する
func TestE2E_Login_InvalidCredentials(t *testing.T) {
	ts := setupE2EServer(t)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	formData := url.Values{}
	formData.Set("username", "nonexistent")
	formData.Set("password", "wrongpass")
	resp, err := client.PostForm(ts.URL+"/login", formData)
	if err != nil {
		t.Fatalf("POST /login failed: %v", err)
	}
	defer resp.Body.Close()

	// ログイン失敗時は200でフォームを再表示
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 on login failure, got %d", resp.StatusCode)
	}
}

// TestE2E_GetBooks_Unauthorized は未ログイン時にGET /booksが/loginへリダイレクトすることを検証する
func TestE2E_GetBooks_Unauthorized(t *testing.T) {
	ts := setupE2EServer(t)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(ts.URL + "/books")
	if err != nil {
		t.Fatalf("GET /books failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

// TestE2E_GetBooks_Authorized はログイン後にGET /booksが200を返すことを検証する
func TestE2E_GetBooks_Authorized(t *testing.T) {
	ts := setupE2EServer(t)

	// ユーザー作成
	reqBody := `{"username": "testuser", "password": "testpass"}`
	createResp, err := http.Post(ts.URL+"/api/users", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /api/users failed: %v", err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("user creation failed with status %d", createResp.StatusCode)
	}

	// リダイレクトを追わないクライアント
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// ログイン
	formData := url.Values{}
	formData.Set("username", "testuser")
	formData.Set("password", "testpass")
	loginResp, err := client.PostForm(ts.URL+"/login", formData)
	if err != nil {
		t.Fatalf("POST /login failed: %v", err)
	}
	loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusFound {
		t.Fatalf("login failed with status %d", loginResp.StatusCode)
	}

	// セッションCookieを使って /books にアクセス
	cookies := loginResp.Cookies()
	req, err := http.NewRequest("GET", ts.URL+"/books", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	booksResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /books failed: %v", err)
	}
	defer booksResp.Body.Close()

	if booksResp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", booksResp.StatusCode)
	}
}

// TestE2E_PostApiPhotos_Unauthorized は不正なupload_keyでPOST /api/photosが401を返すことを検証する
func TestE2E_PostApiPhotos_Unauthorized(t *testing.T) {
	ts := setupE2EServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("upload_key", "invalid-key"); err != nil {
		t.Fatalf("failed to write field: %v", err)
	}
	part, err := writer.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("fake image data")); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	writer.Close()

	resp, err := http.Post(ts.URL+"/api/photos", writer.FormDataContentType(), body)
	if err != nil {
		t.Fatalf("POST /api/photos failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

// setupDiaryForOwner はオーナーユーザーと日記帳・日記を作成してdiaryIDを返す
func setupDiaryForOwner(t *testing.T, ts *httptest.Server, db *sql.DB, ownerUsername, otherUsername string) int {
	t.Helper()

	// ユーザーを作成
	for _, username := range []string{ownerUsername, otherUsername} {
		body := fmt.Sprintf(`{"username": %q, "password": "password"}`, username)
		resp, err := http.Post(ts.URL+"/api/users", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST /api/users failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("user creation failed with status %d", resp.StatusCode)
		}
	}

	// オーナーユーザーを取得
	userRepo := NewSQLiteUserRepository(db)
	ownerUser, err := userRepo.GetUserByUsername(ownerUsername)
	if err != nil || ownerUser == nil {
		t.Fatalf("failed to get owner user: %v", err)
	}

	// 日記帳を作成
	bookRepo := NewSQLiteBookRepository(db)
	book, err := bookRepo.CreateBook(ownerUser.ID, "Test Book")
	if err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	// 日記を作成
	diaryRepo := NewSQLiteDiaryRepository(db)
	if err := diaryRepo.CreateDiaryForBook(book.ID, ownerUser.ID, "/photos/test.jpg", "テスト日記", time.Now()); err != nil {
		t.Fatalf("failed to create diary: %v", err)
	}

	// 日記IDを取得
	diaries, err := diaryRepo.GetDiariesByBookID(book.ID)
	if err != nil || len(diaries) == 0 {
		t.Fatalf("failed to get diaries: %v", err)
	}
	return diaries[0].ID
}

// TestE2E_DiaryEdit_OwnerCanEdit は日記帳作成者が日記を編集できることを検証する
func TestE2E_DiaryEdit_OwnerCanEdit(t *testing.T) {
	ts, db := setupE2EServerWithDB(t)

	diaryID := setupDiaryForOwner(t, ts, db, "owner", "other")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	cookies := loginAsUser(t, ts, "owner", "password")

	// GET /diary/{id}/edit が200を返すことを確認
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/diary/%d/edit", ts.URL, diaryID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /diary/{id}/edit failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for owner, got %d", resp.StatusCode)
	}
}

// TestE2E_DiaryEdit_NonOwnerForbidden は日記帳作成者以外が編集できないことを検証する
func TestE2E_DiaryEdit_NonOwnerForbidden(t *testing.T) {
	ts, db := setupE2EServerWithDB(t)

	diaryID := setupDiaryForOwner(t, ts, db, "owner", "other")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	cookies := loginAsUser(t, ts, "other", "password")

	// GET /diary/{id}/edit が403を返すことを確認
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/diary/%d/edit", ts.URL, diaryID), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /diary/{id}/edit failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403 for non-owner, got %d", resp.StatusCode)
	}

	// POST /diary/{id}/edit が403を返すことを確認
	formData := url.Values{}
	formData.Set("content", "改ざんされた内容")
	postReq, err := http.NewRequest("POST", fmt.Sprintf("%s/diary/%d/edit", ts.URL, diaryID), strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		postReq.AddCookie(c)
	}
	postResp, err := client.Do(postReq)
	if err != nil {
		t.Fatalf("POST /diary/{id}/edit failed: %v", err)
	}
	postResp.Body.Close()

	if postResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403 for non-owner POST, got %d", postResp.StatusCode)
	}
}
