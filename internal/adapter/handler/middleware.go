package handler

import (
	"log"
	"net/http"

	"plant-diary/internal/domain"
)

// getCurrentUser はリクエストのセッションCookieからログイン中のユーザーを返す。未ログインの場合はnilを返す
func (s *Server) getCurrentUser(r *http.Request) (*domain.User, error) {
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
