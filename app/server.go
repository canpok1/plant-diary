package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Server はHTTPサーバーを表す構造体
type Server struct {
	repo      DiaryRepository
	photosDir string
	templates *template.Template
	mux       *http.ServeMux
}

// NewServer は新しいServerを生成する
func NewServer(repo DiaryRepository, photosDir string) (*Server, error) {
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
		repo:      repo,
		photosDir: photosDir,
		templates: tmpl,
		mux:       http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /diary/{id}", s.handleDiary)
	s.mux.HandleFunc("GET /photos/{filename}", s.handlePhoto)

	return s, nil
}

// ServeHTTP はhttp.Handlerインターフェースを実装する
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleIndex は日記一覧ページを表示する
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	diaries, err := s.repo.GetAllDiaries()
	if err != nil {
		log.Printf("ERROR: failed to get diaries: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// ImagePathをファイル名のみに変換
	for i := range diaries {
		diaries[i].ImagePath = filepath.Base(diaries[i].ImagePath)
	}

	data := map[string]interface{}{
		"Diaries": diaries,
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

	data := map[string]interface{}{
		"Diary": &diaryView,
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
