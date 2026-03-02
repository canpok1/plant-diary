package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

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
