package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// scriptConfigResponse は GET /api/script-config のレスポンス
type scriptConfigResponse struct {
	TargetBrightness    float64 `json:"target_brightness"`
	BrightnessTolerance float64 `json:"brightness_tolerance"`
	MaxAdjustRetries    int     `json:"max_adjust_retries"`
	UploadKey           string  `json:"upload_key"`
}

// handleGetScriptConfig は GET /api/script-config のハンドラ
func (s *Server) handleGetScriptConfig(w http.ResponseWriter, r *http.Request) {
	// Authorization: Bearer <script_key> の解析
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scriptKey := strings.TrimPrefix(authHeader, "Bearer ")
	if scriptKey == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// script_key でカメラを検索
	camera, err := s.cameraRepo.GetCameraByScriptKey(scriptKey)
	if err != nil {
		log.Printf("ERROR: failed to get camera by script_key: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if camera == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// camera.book_id に対応する books.upload_key を取得
	book, err := s.bookRepo.GetBookByID(camera.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", camera.BookID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if book == nil {
		log.Printf("ERROR: book %d not found for camera %d", camera.BookID, camera.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := scriptConfigResponse{
		TargetBrightness:    camera.TargetBrightness,
		BrightnessTolerance: camera.BrightnessTolerance,
		MaxAdjustRetries:    camera.MaxAdjustRetries,
		UploadKey:           book.UploadKey,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR: failed to encode response: %v", err)
	}
}
