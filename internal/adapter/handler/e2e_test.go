//go:build e2e

package handler

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

	"plant-diary/internal/adapter"
	"plant-diary/internal/infra/gemini"
	"plant-diary/internal/infra/sqlite"

	_ "github.com/mattn/go-sqlite3"
)

// setupE2ETestDB はE2Eテスト用のインメモリSQLiteデータベースを作成する
func setupE2ETestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid          TEXT NOT NULL UNIQUE,
			username      TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS books (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid       TEXT NOT NULL UNIQUE,
			creator_id INTEGER NOT NULL REFERENCES users(id),
			name       TEXT NOT NULL,
			upload_key TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS cameras (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			script_key TEXT NOT NULL UNIQUE,
			target_brightness REAL NOT NULL DEFAULT 0.475,
			brightness_tolerance REAL NOT NULL DEFAULT 0.175,
			max_adjust_retries INTEGER NOT NULL DEFAULT 5,
			book_id INTEGER NOT NULL REFERENCES books(id),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS diary (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			image_path TEXT NOT NULL UNIQUE,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			user_id INTEGER REFERENCES users(id),
			book_id INTEGER REFERENCES books(id)
		);
		CREATE INDEX IF NOT EXISTS idx_created_at ON diary(created_at DESC);
		CREATE TABLE IF NOT EXISTS sessions (
			id         TEXT PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// setupE2EServer はE2Eテスト用のHTTPサーバーを設定して起動する
func setupE2EServer(t *testing.T) *httptest.Server {
	t.Helper()

	ts, _ := setupE2EServerWithDB(t)
	return ts
}

// setupE2EServerWithDB はE2Eテスト用のHTTPサーバーとDBを設定して起動する
func setupE2EServerWithDB(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()

	db := setupE2ETestDB(t)
	repo := sqlite.NewSQLiteDiaryRepository(db)
	userRepo := sqlite.NewSQLiteUserRepository(db)
	bookRepo := sqlite.NewSQLiteBookRepository(db)
	sessionRepo := sqlite.NewSQLiteSessionRepository(db)
	cameraRepo := sqlite.NewSQLiteCameraRepository(db)
	generator := &gemini.MockDiaryGenerator{}

	srv, err := NewServer(repo, userRepo, bookRepo, sessionRepo, generator, cameraRepo, "../../../templates", t.TempDir())
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

	var result adapter.UserResponse
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

// TestE2E_GetBooksNew_Unauthorized は未ログイン時にGET /books/newが/loginへリダイレクトすることを検証する
func TestE2E_GetBooksNew_Unauthorized(t *testing.T) {
	ts := setupE2EServer(t)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(ts.URL + "/books/new")
	if err != nil {
		t.Fatalf("GET /books/new failed: %v", err)
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

// TestE2E_GetBooksNew_Authorized はログイン後にGET /books/newが200を返すことを検証する
func TestE2E_GetBooksNew_Authorized(t *testing.T) {
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

	// セッションCookieを使って /books/new にアクセス
	cookies := loginResp.Cookies()
	req, err := http.NewRequest("GET", ts.URL+"/books/new", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	booksResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /books/new failed: %v", err)
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

// setupBookForOwner はオーナーユーザーと日記帳を作成してbookIDを返す
func setupBookForOwner(t *testing.T, ts *httptest.Server, db *sql.DB, ownerUsername, otherUsername string) int {
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
	userRepo := sqlite.NewSQLiteUserRepository(db)
	ownerUser, err := userRepo.GetUserByUsername(ownerUsername)
	if err != nil || ownerUser == nil {
		t.Fatalf("failed to get owner user: %v", err)
	}

	// 日記帳を作成
	bookRepo := sqlite.NewSQLiteBookRepository(db)
	book, err := bookRepo.CreateBook(ownerUser.ID, "Test Book")
	if err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	return book.ID
}

// TestE2E_BookSettings_UpdateName_Success はオーナーが名前を変更できることを検証する
func TestE2E_BookSettings_UpdateName_Success(t *testing.T) {
	ts, db := setupE2EServerWithDB(t)

	bookID := setupBookForOwner(t, ts, db, "owner", "other")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	cookies := loginAsUser(t, ts, "owner", "password")

	formData := url.Values{}
	formData.Set("name", "新しい日記帳名")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/books/%d/settings", ts.URL, bookID), strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /books/{id}/settings failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	expected := fmt.Sprintf("/books/%d/settings?success=1", bookID)
	if location != expected {
		t.Errorf("expected redirect to %s, got %s", expected, location)
	}
}

// TestE2E_BookSettings_UpdateName_EmptyName は空文字で送信するとエラーになることを検証する
func TestE2E_BookSettings_UpdateName_EmptyName(t *testing.T) {
	ts, db := setupE2EServerWithDB(t)

	bookID := setupBookForOwner(t, ts, db, "owner", "other")

	cookies := loginAsUser(t, ts, "owner", "password")

	formData := url.Values{}
	formData.Set("name", "")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/books/%d/settings", ts.URL, bookID), strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /books/{id}/settings failed: %v", err)
	}
	defer resp.Body.Close()

	// バリデーションエラーは設定画面を再表示（200）
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for empty name, got %d", resp.StatusCode)
	}
}

// TestE2E_BookSettings_UpdateName_TooLongName は51文字以上の名前で送信するとエラーになることを検証する
func TestE2E_BookSettings_UpdateName_TooLongName(t *testing.T) {
	ts, db := setupE2EServerWithDB(t)

	bookID := setupBookForOwner(t, ts, db, "owner", "other")

	cookies := loginAsUser(t, ts, "owner", "password")

	// 51文字の名前
	longName := strings.Repeat("あ", 51)
	formData := url.Values{}
	formData.Set("name", longName)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/books/%d/settings", ts.URL, bookID), strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /books/{id}/settings failed: %v", err)
	}
	defer resp.Body.Close()

	// バリデーションエラーは設定画面を再表示（200）
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for too long name, got %d", resp.StatusCode)
	}
}

// TestE2E_BookSettings_UpdateName_NonOwnerForbidden は非オーナーがPOSTすると403になることを検証する
func TestE2E_BookSettings_UpdateName_NonOwnerForbidden(t *testing.T) {
	ts, db := setupE2EServerWithDB(t)

	bookID := setupBookForOwner(t, ts, db, "owner", "other")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	cookies := loginAsUser(t, ts, "other", "password")

	formData := url.Values{}
	formData.Set("name", "乗っ取り")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/books/%d/settings", ts.URL, bookID), strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /books/{id}/settings failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
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
	userRepo := sqlite.NewSQLiteUserRepository(db)
	ownerUser, err := userRepo.GetUserByUsername(ownerUsername)
	if err != nil || ownerUser == nil {
		t.Fatalf("failed to get owner user: %v", err)
	}

	// 日記帳を作成
	bookRepo := sqlite.NewSQLiteBookRepository(db)
	book, err := bookRepo.CreateBook(ownerUser.ID, "Test Book")
	if err != nil {
		t.Fatalf("failed to create book: %v", err)
	}

	// 日記を作成
	diaryRepo := sqlite.NewSQLiteDiaryRepository(db)
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

// TODO: PATCH /cameras/{id} - 正常系: 200 と {"status":"ok"} を返す
// TODO: PATCH /cameras/{id} - リクエストボディが不正な場合 400 を返す

// setupCameraForOwner はオーナーユーザー・別ユーザー・日記帳・カメラを作成してcameraIDを返す
func setupCameraForOwner(t *testing.T, ts *httptest.Server, db *sql.DB, ownerUsername, otherUsername string) int {
	t.Helper()

	bookID := setupBookForOwner(t, ts, db, ownerUsername, otherUsername)

	// カメラを作成
	cameraRepo := sqlite.NewSQLiteCameraRepository(db)
	camera, err := cameraRepo.CreateCamera("Test Camera", bookID)
	if err != nil {
		t.Fatalf("failed to create camera: %v", err)
	}

	return camera.ID
}

// TestE2E_PatchCamera_NotFound は存在しないカメラIDでPATCH /cameras/{id}が404を返すことを検証する
func TestE2E_PatchCamera_NotFound(t *testing.T) {
	ts := setupE2EServer(t)

	// ユーザーを作成してログイン
	resp, err := http.Post(ts.URL+"/api/users", "application/json", strings.NewReader(`{"username": "user1", "password": "pass"}`))
	if err != nil {
		t.Fatalf("POST /api/users failed: %v", err)
	}
	resp.Body.Close()

	cookies := loginAsUser(t, ts, "user1", "pass")

	body := strings.NewReader(`{"test_capture_requested": true}`)
	req, err := http.NewRequest("PATCH", ts.URL+"/cameras/9999", body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH /cameras/9999 failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp2.StatusCode)
	}
}

// TestE2E_PatchCamera_Forbidden は別ユーザーのカメラにPATCH /cameras/{id}が403を返すことを検証する
func TestE2E_PatchCamera_Forbidden(t *testing.T) {
	ts, db := setupE2EServerWithDB(t)

	cameraID := setupCameraForOwner(t, ts, db, "owner", "other")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 別ユーザー（other）でログイン
	cookies := loginAsUser(t, ts, "other", "password")

	body := strings.NewReader(`{"test_capture_requested": true}`)
	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/cameras/%d", ts.URL, cameraID), body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PATCH /cameras/{id} failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

// TestE2E_PatchCamera_Unauthorized は未ログイン時にPATCH /cameras/{id}が/loginへリダイレクトすることを検証する
func TestE2E_PatchCamera_Unauthorized(t *testing.T) {
	ts := setupE2EServer(t)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	body := strings.NewReader(`{"test_capture_requested": true}`)
	req, err := http.NewRequest("PATCH", ts.URL+"/cameras/1", body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PATCH /cameras/1 failed: %v", err)
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
