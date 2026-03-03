package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"plant-diary/internal/adapter"
	"plant-diary/internal/domain"
	"plant-diary/internal/usecase"
)

// Server はHTTPサーバーを表す構造体
type Server struct {
	repo        domain.DiaryRepository
	userRepo    domain.UserRepository
	bookRepo    domain.BookRepository
	sessionRepo domain.SessionRepository
	generator   usecase.DiaryGenerator
	photosDir   string
	templates   *template.Template
	mux         *http.ServeMux
}

// NewServer は新しいServerを生成する
func NewServer(repo domain.DiaryRepository, userRepo domain.UserRepository, bookRepo domain.BookRepository, sessionRepo domain.SessionRepository, generator usecase.DiaryGenerator, templatesDir string, photosDir string) (*Server, error) {
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

	// テンプレートディレクトリが存在するか確認
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("templates directory not found: %s", templatesDir)
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(filepath.Join(templatesDir, "*.html"))
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
	s.mux.HandleFunc("GET /books/{id}/settings", s.requireLogin(s.handleBookSettings))
	s.mux.HandleFunc("GET /books/{id}/slideshow", s.handleBookSlideshow)

	adapter.HandlerFromMux(s, s.mux)

	return s, nil
}

// ServeHTTP はhttp.Handlerインターフェースを実装する
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
