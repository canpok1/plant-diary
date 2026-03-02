package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

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

	diaryView := *diary

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
		if diary.BookID != nil {
			book, err := s.bookRepo.GetBookByID(*diary.BookID)
			if err != nil {
				log.Printf("ERROR: failed to get book %d: %v", *diary.BookID, err)
				s.renderError(w, http.StatusInternalServerError)
				return
			}
			if book != nil {
				isOwner = book.CreatorID == currentUser.ID
			}
		}
	}

	data := map[string]interface{}{
		"Diary":    &diaryView,
		"LoggedIn": loggedIn,
		"Username": username,
		"IsOwner":  isOwner,
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

	// 日記帳の作成者チェック
	if diary.BookID == nil {
		s.renderError(w, http.StatusForbidden)
		return
	}
	book, err := s.bookRepo.GetBookByID(*diary.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", *diary.BookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil || currentUser == nil || book.CreatorID != currentUser.ID {
		s.renderError(w, http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Diary":    diary,
		"LoggedIn": true,
		"Username": currentUser.Username,
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

	currentUser, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// 日記帳の作成者チェック
	if diary.BookID == nil {
		s.renderError(w, http.StatusForbidden)
		return
	}
	book, err := s.bookRepo.GetBookByID(*diary.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", *diary.BookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil || currentUser == nil || book.CreatorID != currentUser.ID {
		s.renderError(w, http.StatusForbidden)
		return
	}

	if err := s.repo.UpdateDiaryContent(id, content); err != nil {
		log.Printf("ERROR: failed to update diary %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/diary/%d", id), http.StatusFound)
}
