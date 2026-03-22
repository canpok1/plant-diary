package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"plant-diary/internal/domain"
	"plant-diary/internal/usecase"
)

// promptPreviewRequest はプロンプトプレビューAPIのリクエスト構造体
type promptPreviewRequest struct {
	Prompt         string `json:"prompt"`
	ImageDiaryID   int    `json:"image_diary_id"`
	ContextDiaryID *int   `json:"context_diary_id"`
}

// promptPreviewResponse はプロンプトプレビューAPIのレスポンス構造体
type promptPreviewResponse struct {
	ExpandedPrompt   string `json:"expanded_prompt"`
	GeneratedContent string `json:"generated_content"`
}

// handlePostBookPromptPreview は POST /api/books/{id}/prompt-preview のハンドラ
func (s *Server) handlePostBookPromptPreview(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	bookID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid book id: %s", idStr)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	var req promptPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Bad Request: prompt is required", http.StatusBadRequest)
		return
	}
	if req.ImageDiaryID <= 0 {
		http.Error(w, "Bad Request: image_diary_id must be positive", http.StatusBadRequest)
		return
	}

	book, err := s.bookRepo.GetBookByID(bookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", bookID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	if currentUser.ID != book.CreatorID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	imageDiary, err := s.repo.GetDiaryByID(req.ImageDiaryID)
	if err != nil {
		log.Printf("ERROR: failed to get diary %d: %v", req.ImageDiaryID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if imageDiary == nil {
		http.Error(w, "Not Found: image diary not found", http.StatusNotFound)
		return
	}
	if imageDiary.BookID == nil || *imageDiary.BookID != bookID {
		http.Error(w, "Not Found: image diary does not belong to this book", http.StatusNotFound)
		return
	}

	var pastDiaries []domain.Diary
	if req.ContextDiaryID != nil {
		contextDiary, err := s.repo.GetDiaryByID(*req.ContextDiaryID)
		if err != nil {
			log.Printf("ERROR: failed to get context diary %d: %v", *req.ContextDiaryID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if contextDiary == nil {
			http.Error(w, "Not Found: context diary not found", http.StatusNotFound)
			return
		}
		startOfDay := time.Date(imageDiary.CreatedAt.Year(), imageDiary.CreatedAt.Month(), imageDiary.CreatedAt.Day(), 0, 0, 0, 0, imageDiary.CreatedAt.Location())
		startDate := startOfDay.AddDate(0, -1, 0)
		endDate := time.Date(contextDiary.CreatedAt.Year(), contextDiary.CreatedAt.Month(), contextDiary.CreatedAt.Day(), 0, 0, 0, 0, contextDiary.CreatedAt.Location()).Add(-time.Nanosecond)
		pastDiaries, err = s.repo.GetDiariesInDateRange(bookID, startDate, endDate)
		if err != nil {
			log.Printf("WARN: failed to get past diaries: %v", err)
			pastDiaries = []domain.Diary{}
		}
	}

	expandedPrompt := expandPrompt(req.Prompt, book.Name, imageDiary.CreatedAt, pastDiaries)

	genWithPrompt, ok := s.generator.(usecase.DiaryGeneratorWithPrompt)
	if !ok {
		http.Error(w, "Internal Server Error: generator not available", http.StatusInternalServerError)
		return
	}

	diskImagePath := filepath.Join(s.photosDir, imageDiary.ImagePath)
	generatedContent, err := genWithPrompt.GenerateDiaryWithPrompt(diskImagePath, expandedPrompt)
	if err != nil {
		log.Printf("ERROR: failed to generate diary: %v", err)
		http.Error(w, "Internal Server Error: failed to generate diary", http.StatusInternalServerError)
		return
	}

	resp := promptPreviewResponse{
		ExpandedPrompt:   expandedPrompt,
		GeneratedContent: generatedContent,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR: failed to encode response: %v", err)
	}
}
