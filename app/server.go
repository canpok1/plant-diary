package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Server はHTTPサーバーを表す構造体
type Server struct {
	repo        DiaryRepository
	userRepo    UserRepository
	bookRepo    BookRepository
	sessionRepo SessionRepository
	generator   DiaryGenerator
	photosDir   string
	templates   *template.Template
	mux         *http.ServeMux
}

// NewServer は新しいServerを生成する
func NewServer(repo DiaryRepository, userRepo UserRepository, bookRepo BookRepository, sessionRepo SessionRepository, generator DiaryGenerator, photosDir string) (*Server, error) {
	// カスタムテンプレート関数を登録
	funcMap := template.FuncMap{
		"truncate": func(s string, length int) string {
			runes := []rune(s)
			if len(runes) <= length {
				return s
			}
			return string(runes[:length]) + "..."
		},
		"toJST": func(t time.Time) time.Time {
			jst := time.FixedZone("Asia/Tokyo", 9*60*60)
			return t.In(jst)
		},
		"weekdayJP": func(t time.Time) string {
			weekdays := []string{"日", "月", "火", "水", "木", "金", "土"}
			return weekdays[t.Weekday()]
		},
	}

	// テンプレートディレクトリの自動検出
	templatesPath := "templates"
	if _, err := os.Stat(templatesPath); os.IsNotExist(err) {
		templatesPath = "app/templates"
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(filepath.Join(templatesPath, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	s := &Server{
		repo:        repo,
		userRepo:    userRepo,
		bookRepo:    bookRepo,
		sessionRepo: sessionRepo,
		generator:   generator,
		photosDir:   photosDir,
		templates:   tmpl,
		mux:         http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /diary/{id}", s.handleDiary)
	s.mux.HandleFunc("GET /diary/{id}/edit", s.requireLogin(s.handleDiaryEditGet))
	s.mux.HandleFunc("POST /diary/{id}/edit", s.requireLogin(s.handleDiaryEditPost))
	s.mux.HandleFunc("GET /photos/{filename}", s.handlePhoto)
	s.mux.HandleFunc("GET /photos/{user_uuid}/{filename}", s.handlePhotoWithUserUUID)
	s.mux.HandleFunc("GET /login", s.handleLoginGet)
	s.mux.HandleFunc("POST /login", s.handleLoginPost)
	s.mux.HandleFunc("POST /logout", s.handleLogout)
	s.mux.HandleFunc("GET /books", s.requireLogin(s.handleGetBooks))
	s.mux.HandleFunc("POST /books", s.requireLogin(s.handlePostBooks))
	s.mux.HandleFunc("GET /books/{id}", s.handleGetBook)
	s.mux.HandleFunc("GET /books/{id}/slideshow", s.handleBookSlideshow)

	HandlerFromMux(s, s.mux)

	return s, nil
}

// ServeHTTP はhttp.Handlerインターフェースを実装する
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// getCurrentUser はリクエストのセッションCookieからログイン中のユーザーを返す。未ログインの場合はnilを返す
func (s *Server) getCurrentUser(r *http.Request) (*User, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, nil
	}
	session, err := s.sessionRepo.GetSessionByID(cookie.Value)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	return s.userRepo.GetUserByID(session.UserID)
}

// requireLogin は未ログイン時に /login へリダイレクトするミドルウェア
func (s *Server) requireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := s.getCurrentUser(r)
		if err != nil {
			log.Printf("ERROR: failed to get current user: %v", err)
			s.renderError(w, http.StatusInternalServerError)
			return
		}
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// handleLoginGet はログインフォームページを表示する
func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Error": "",
	}
	if err := s.templates.ExecuteTemplate(w, "login.html", data); err != nil {
		log.Printf("ERROR: failed to render login template: %v", err)
		s.renderError(w, http.StatusInternalServerError)
	}
}

// handleLoginPost は認証を行い、成功時はセッションを作成して / へリダイレクトする
func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	renderLoginError := func(msg string) {
		data := map[string]interface{}{"Error": msg}
		if err := s.templates.ExecuteTemplate(w, "login.html", data); err != nil {
			log.Printf("ERROR: failed to render login template: %v", err)
			s.renderError(w, http.StatusInternalServerError)
		}
	}

	user, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		log.Printf("ERROR: failed to get user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if user == nil {
		renderLoginError("ユーザー名またはパスワードが間違っています")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		renderLoginError("ユーザー名またはパスワードが間違っています")
		return
	}

	sessionID, err := generateUUID()
	if err != nil {
		log.Printf("ERROR: failed to generate session ID: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := s.sessionRepo.CreateSession(sessionID, user.ID, expiresAt); err != nil {
		log.Printf("ERROR: failed to create session: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleLogout はセッションを削除して / へリダイレクトする
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		if err := s.sessionRepo.DeleteSession(cookie.Value); err != nil {
			log.Printf("ERROR: failed to delete session: %v", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleIndex は日記帳一覧ページを表示する
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	books, err := s.bookRepo.GetAllBooks()
	if err != nil {
		log.Printf("ERROR: failed to get all books: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	currentUser, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	loggedIn := currentUser != nil
	username := ""
	if currentUser != nil {
		username = currentUser.Username
	}

	data := map[string]interface{}{
		"Books":    books,
		"LoggedIn": loggedIn,
		"Username": username,
	}

	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("ERROR: failed to render index template: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
}

// handleDiary は日記詳細ページを表示する
func (s *Server) handleDiary(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid diary id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	diary, err := s.repo.GetDiaryByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get diary %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if diary == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	// ImagePathからファイル名のみを抽出（表示用コピー）
	diaryView := *diary
	diaryView.ImagePath = filepath.Base(diary.ImagePath)

	currentUser, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	loggedIn := currentUser != nil
	username := ""
	if currentUser != nil {
		username = currentUser.Username
	}

	data := map[string]interface{}{
		"Diary":    &diaryView,
		"LoggedIn": loggedIn,
		"Username": username,
	}

	if err := s.templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		log.Printf("ERROR: failed to render detail template for diary %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
}

// handleDiaryEditGet は日記編集フォームページを表示する
func (s *Server) handleDiaryEditGet(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid diary id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	diary, err := s.repo.GetDiaryByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get diary %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if diary == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	currentUser, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	username := ""
	if currentUser != nil {
		username = currentUser.Username
	}

	data := map[string]interface{}{
		"Diary":    diary,
		"LoggedIn": true,
		"Username": username,
	}

	if err := s.templates.ExecuteTemplate(w, "edit.html", data); err != nil {
		log.Printf("ERROR: failed to render edit template for diary %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
}

// handleDiaryEditPost は日記のcontentを更新して詳細ページへリダイレクトする
func (s *Server) handleDiaryEditPost(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid diary id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")

	diary, err := s.repo.GetDiaryByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get diary %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if diary == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	if err := s.repo.UpdateDiaryContent(id, content); err != nil {
		log.Printf("ERROR: failed to update diary %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/diary/%d", id), http.StatusFound)
}

// handlePhoto は画像ファイルを配信する
func (s *Server) handlePhoto(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")

	// ディレクトリトラバーサル防止
	if filename == "" || filename == "." || filename == ".." ||
		strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		s.renderError(w, http.StatusNotFound)
		return
	}

	filePath := filepath.Join(s.photosDir, filename)
	http.ServeFile(w, r, filePath)
}

// handlePhotoWithUserUUID はユーザーUUID配下の画像ファイルを配信する
func (s *Server) handlePhotoWithUserUUID(w http.ResponseWriter, r *http.Request) {
	userUUID := r.PathValue("user_uuid")
	filename := r.PathValue("filename")

	// ディレクトリトラバーサル防止
	if userUUID == "" || filename == "" ||
		userUUID == "." || userUUID == ".." ||
		filename == "." || filename == ".." ||
		strings.Contains(userUUID, "/") || strings.Contains(userUUID, "\\") ||
		strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		s.renderError(w, http.StatusNotFound)
		return
	}

	filePath := filepath.Join(s.photosDir, userUUID, filename)
	http.ServeFile(w, r, filePath)
}

// handleBookSlideshow は日記帳別スライドショーページを表示する
func (s *Server) handleBookSlideshow(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid book id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	book, err := s.bookRepo.GetBookByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var from, to time.Time
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)

	if fromStr != "" {
		t, err := time.ParseInLocation("2006-01-02", fromStr, jst)
		if err == nil {
			from = t.UTC()
		}
	}
	if toStr != "" {
		t, err := time.ParseInLocation("2006-01-02", toStr, jst)
		if err == nil {
			to = t.Add(24*time.Hour - time.Nanosecond).UTC()
		}
	}

	diaries, err := s.repo.GetDiariesByBookIDAsc(id, from, to)
	if err != nil {
		log.Printf("ERROR: failed to get diaries for book slideshow %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	jstZone := time.FixedZone("Asia/Tokyo", 9*60*60)
	weekdays := []string{"日", "月", "火", "水", "木", "金", "土"}
	type photoItem struct {
		URL      string `json:"url"`
		DateTime string `json:"dateTime"`
		DiaryID  int    `json:"diaryId"`
	}
	photos := make([]photoItem, 0, len(diaries))
	for i := range diaries {
		diaries[i].ImagePath = filepath.Base(diaries[i].ImagePath)
		t := diaries[i].CreatedAt.In(jstZone)
		dateTime := fmt.Sprintf("%d年%d月%d日（%s）%s",
			t.Year(), int(t.Month()), t.Day(),
			weekdays[t.Weekday()],
			t.Format("15:04"),
		)
		photos = append(photos, photoItem{
			URL:      "/photos/" + diaries[i].ImagePath,
			DateTime: dateTime,
			DiaryID:  diaries[i].ID,
		})
	}
	photosJSON, err := json.Marshal(photos)
	if err != nil {
		log.Printf("ERROR: failed to marshal photos JSON: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Diaries":       diaries,
		"From":          fromStr,
		"To":            toStr,
		"PhotosJSON":    template.JS(photosJSON),
		"BookName":      book.Name,
		"BackURL":       fmt.Sprintf("/books/%d", id),
		"FilterBaseURL": fmt.Sprintf("/books/%d/slideshow", id),
	}

	if err := s.templates.ExecuteTemplate(w, "slideshow.html", data); err != nil {
		log.Printf("ERROR: failed to render slideshow template for book %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
	}
}

// PostApiPhotos は写真アップロードAPIのハンドラ（POST /api/photos）
func (s *Server) PostApiPhotos(w http.ResponseWriter, r *http.Request) {
	// multipart/form-data のパース（最大32MB）
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// upload_key からBookを取得して認証
	uploadKey := r.FormValue("upload_key")
	if uploadKey == "" {
		http.Error(w, "Bad Request: upload_key is required", http.StatusBadRequest)
		return
	}

	book, err := s.bookRepo.GetBookByUploadKey(uploadKey)
	if err != nil {
		log.Printf("ERROR: failed to get book by upload key: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// captured_at の解析（省略時はサーバー受信時刻）
	capturedAt := time.Now().UTC()
	if capturedAtStr := r.FormValue("captured_at"); capturedAtStr != "" {
		t, err := time.Parse(time.RFC3339, capturedAtStr)
		if err != nil {
			http.Error(w, "Bad Request: invalid captured_at format", http.StatusBadRequest)
			return
		}
		capturedAt = t.UTC()
	}

	// 写真ファイルの取得
	file, _, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Bad Request: photo field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 保存先ディレクトリの作成（book.UUID ベース）
	bookDir := filepath.Join(s.photosDir, book.UUID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		log.Printf("ERROR: failed to create book photo dir %s: %v", bookDir, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ファイル名生成（YYYYMMDD_HHMMSS_UTC.jpg）秒単位で衝突を回避
	filename := capturedAt.Format("20060102_150405") + "_UTC.jpg"
	imagePath := filepath.Join(bookDir, filename)

	// ファイルの保存
	dst, err := os.Create(imagePath)
	if err != nil {
		log.Printf("ERROR: failed to create photo file %s: %v", imagePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		os.Remove(imagePath)
		log.Printf("ERROR: failed to save photo file %s: %v", imagePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	dst.Close()

	// ジョブIDを生成
	jobID, err := generateUUID()
	if err != nil {
		log.Printf("ERROR: failed to generate job ID: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// goroutineで非同期に日記を生成・保存
	go func() {
		startOfDay := time.Date(capturedAt.Year(), capturedAt.Month(), capturedAt.Day(), 0, 0, 0, 0, capturedAt.Location())
		oneMonthAgo := startOfDay.AddDate(0, -1, 0)
		endOfPrevDay := startOfDay.Add(-time.Nanosecond)
		pastDiaries, err := s.repo.GetDiariesInDateRange(oneMonthAgo, endOfPrevDay)
		if err != nil {
			log.Printf("WARN: failed to get past diaries for %s: %v, continuing with empty history", imagePath, err)
			pastDiaries = []Diary{}
		}

		prompt := buildDiaryPrompt(pastDiaries)

		var content string
		retryErr := Retry(DefaultRetryConfig(), fmt.Sprintf("generate diary for %s", imagePath), func() error {
			var genErr error
			if genWithPrompt, ok := s.generator.(DiaryGeneratorWithPrompt); ok {
				content, genErr = genWithPrompt.GenerateDiaryWithPrompt(imagePath, prompt)
			} else {
				content, genErr = s.generator.GenerateDiary(imagePath)
			}
			return genErr
		})
		if retryErr != nil {
			log.Printf("ERROR: failed to generate diary for %s: %v", imagePath, retryErr)
			return
		}

		if err := s.repo.CreateDiaryForBook(book.ID, book.CreatorID, imagePath, content, capturedAt); err != nil {
			log.Printf("ERROR: failed to save diary for %s: %v", imagePath, err)
			return
		}

		log.Printf("INFO: diary created for %s (job_id: %s)", imagePath, jobID)
	}()

	// 202 Accepted を返す
	resp := UploadPhotoResponse{JobId: jobID}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR: failed to encode response: %v", err)
	}
}

// PostApiUsers はユーザー作成APIのハンドラ（POST /api/users）
func (s *Server) PostApiUsers(w http.ResponseWriter, r *http.Request) {
	// リクエストボディの解析
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 重複ユーザー名の確認
	existing, err := s.userRepo.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("ERROR: failed to check existing user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// パスワードのハッシュ化
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: failed to hash password: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// UUID生成（ハイフンなし32文字）
	uuid, err := generateUUID()
	if err != nil {
		log.Printf("ERROR: failed to generate UUID: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ユーザー作成
	if err := s.userRepo.CreateUser(uuid, req.Username, string(hash)); err != nil {
		log.Printf("ERROR: failed to create user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := UserResponse{
		Uuid:     uuid,
		Username: req.Username,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR: failed to encode response: %v", err)
	}
}

// handleGetBooks はログインユーザーの日記帳一覧ページを表示する
func (s *Server) handleGetBooks(w http.ResponseWriter, r *http.Request) {
	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	books, err := s.bookRepo.GetBooksByCreatorID(user.ID)
	if err != nil {
		log.Printf("ERROR: failed to get books for user %d: %v", user.ID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Books":    books,
		"LoggedIn": true,
		"Username": user.Username,
	}

	if err := s.templates.ExecuteTemplate(w, "books.html", data); err != nil {
		log.Printf("ERROR: failed to render books template: %v", err)
		s.renderError(w, http.StatusInternalServerError)
	}
}

// handlePostBooks はフォームから name を受け取り日記帳を作成して詳細ページへリダイレクトする
func (s *Server) handlePostBooks(w http.ResponseWriter, r *http.Request) {
	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		s.renderError(w, http.StatusBadRequest)
		return
	}

	book, err := s.bookRepo.CreateBook(user.ID, name)
	if err != nil {
		log.Printf("ERROR: failed to create book for user %d: %v", user.ID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/books/%d", book.ID), http.StatusFound)
}

// handleGetBook は日記帳詳細ページを表示する
func (s *Server) handleGetBook(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid book id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	book, err := s.bookRepo.GetBookByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	diaries, err := s.repo.GetDiariesByBookID(id)
	if err != nil {
		log.Printf("ERROR: failed to get diaries for book %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// ImagePathをファイル名のみに変換
	for i := range diaries {
		diaries[i].ImagePath = filepath.Base(diaries[i].ImagePath)
	}

	currentUser, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	loggedIn := currentUser != nil
	username := ""
	isOwner := false
	if currentUser != nil {
		username = currentUser.Username
		isOwner = currentUser.ID == book.CreatorID
	}

	data := map[string]interface{}{
		"Book":     book,
		"Diaries":  diaries,
		"LoggedIn": loggedIn,
		"Username": username,
		"IsOwner":  isOwner,
	}

	if err := s.templates.ExecuteTemplate(w, "book_detail.html", data); err != nil {
		log.Printf("ERROR: failed to render book_detail template for book %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
	}
}

// renderError はエラーページをレンダリングする
func (s *Server) renderError(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)

	var message string
	switch statusCode {
	case http.StatusForbidden:
		message = "アクセスが拒否されました"
	case http.StatusNotFound:
		message = "ページが見つかりません"
	case http.StatusInternalServerError:
		message = "サーバーエラーが発生しました"
	default:
		message = "エラーが発生しました"
	}

	data := map[string]interface{}{
		"StatusCode": statusCode,
		"Message":    message,
	}

	// エラーテンプレートが存在しない場合はプレーンテキストで返す
	if err := s.templates.ExecuteTemplate(w, "error.html", data); err != nil {
		http.Error(w, message, statusCode)
	}
}
