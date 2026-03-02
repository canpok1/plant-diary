package handler

import (
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"plant-diary/internal/utils"
)

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

	sessionID, err := utils.GenerateUUID()
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
