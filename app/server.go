package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Server はHTTPサーバーを表す構造体
type Server struct {
	repo        DiaryRepository
	userRepo    UserRepository
	sessionRepo SessionRepository
	photosDir   string
	templates   *template.Template
	mux         *http.ServeMux
}

// NewServer は新しいServerを生成する
func NewServer(repo DiaryRepository, userRepo UserRepository, sessionRepo SessionRepository, photosDir string) (*Server, error) {
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
		sessionRepo: sessionRepo,
		photosDir:   photosDir,
		templates:   tmpl,
		mux:         http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /diary/{id}", s.handleDiary)
	s.mux.HandleFunc("GET /photos/{filename}", s.handlePhoto)
	s.mux.HandleFunc("GET /slideshow", s.handleSlideshow)
	s.mux.HandleFunc("GET /login", s.handleLoginGet)
	s.mux.HandleFunc("POST /login", s.handleLoginPost)
	s.mux.HandleFunc("POST /logout", s.handleLogout)

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

// handleIndex は日記一覧ページを表示する
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	keyword := r.URL.Query().Get("q")

	var diaries []Diary
	var err error
	selectedYear := 0
	selectedMonth := 0

	if yearStr != "" && monthStr != "" {
		year, yearErr := strconv.Atoi(yearStr)
		month, monthErr := strconv.Atoi(monthStr)
		if yearErr == nil && monthErr == nil && year > 0 && month >= 1 && month <= 12 {
			jst := time.FixedZone("Asia/Tokyo", 9*60*60)
			startDateJST := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, jst)
			endDateJST := startDateJST.AddDate(0, 1, 0).Add(-time.Nanosecond)
			diaries, err = s.repo.GetDiariesInDateRange(startDateJST.UTC(), endDateJST.UTC())
			selectedYear = year
			selectedMonth = month
		} else {
			if keyword != "" {
				diaries, err = s.repo.SearchDiaries(keyword)
			} else {
				diaries, err = s.repo.GetAllDiaries()
			}
		}
	} else {
		if keyword != "" {
			diaries, err = s.repo.SearchDiaries(keyword)
		} else {
			diaries, err = s.repo.GetAllDiaries()
		}
	}

	if err != nil {
		log.Printf("ERROR: failed to get diaries: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// 月別フィルタとキーワード検索を組み合わせた場合、インメモリでキーワード絞り込み
	// SQLite の LIKE と挙動を合わせるため大小文字を区別しない比較を行う
	if keyword != "" && selectedYear != 0 {
		filtered := make([]Diary, 0, len(diaries))
		kw := strings.ToLower(keyword)
		for _, d := range diaries {
			if strings.Contains(strings.ToLower(d.Content), kw) {
				filtered = append(filtered, d)
			}
		}
		diaries = filtered
	}

	availableMonths, err := s.repo.GetAvailableYearMonths()
	if err != nil {
		log.Printf("ERROR: failed to get available year months: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// フィルタ適用時はGetDiariesInDateRangeがASC順で返すため、全件表示と揃えてDESC順にソート
	if selectedYear != 0 {
		sort.Slice(diaries, func(i, j int) bool {
			return diaries[i].CreatedAt.After(diaries[j].CreatedAt)
		})
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
	if currentUser != nil {
		username = currentUser.Username
	}

	data := map[string]interface{}{
		"Diaries":         diaries,
		"AvailableMonths": availableMonths,
		"SelectedYear":    selectedYear,
		"SelectedMonth":   selectedMonth,
		"Keyword":         keyword,
		"LoggedIn":        loggedIn,
		"Username":        username,
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

// handleSlideshow はスライドショーページを表示する
func (s *Server) handleSlideshow(w http.ResponseWriter, r *http.Request) {
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
			// 終了日は当日の終わりまで含める
			to = t.Add(24*time.Hour - time.Nanosecond).UTC()
		}
	}

	diaries, err := s.repo.GetDiariesAsc(from, to)
	if err != nil {
		log.Printf("ERROR: failed to get diaries for slideshow: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// ImagePathをファイル名のみに変換し、JavaScript用データを準備
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
		"Diaries":    diaries,
		"From":       fromStr,
		"To":         toStr,
		"PhotosJSON": template.JS(photosJSON),
	}

	if err := s.templates.ExecuteTemplate(w, "slideshow.html", data); err != nil {
		log.Printf("ERROR: failed to render slideshow template: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
}

// PostApiUsers はユーザー作成APIのハンドラ（POST /api/users）
func (s *Server) PostApiUsers(w http.ResponseWriter, r *http.Request) {
	// UPLOAD_API_KEY が未設定の場合は 503
	apiKey := os.Getenv("UPLOAD_API_KEY")
	if apiKey == "" {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// X-API-Key ヘッダーの検証（タイミング攻撃防止のため定数時間比較を使用）
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-API-Key")), []byte(apiKey)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

// renderError はエラーページをレンダリングする
func (s *Server) renderError(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)

	var message string
	switch statusCode {
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
