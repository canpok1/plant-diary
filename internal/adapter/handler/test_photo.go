package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"plant-diary/internal/domain"
)

// handlePostTestPhoto は POST /api/test-photo のハンドラ
func (s *Server) handlePostTestPhoto(w http.ResponseWriter, r *http.Request) {
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

	// multipart/form-data のパース（最大32MB）
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 写真ファイルの取得
	file, _, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Bad Request: photo field is required", http.StatusBadRequest)
		return
	}
	defer file.Close() //nolint:errcheck

	// 保存先ディレクトリの作成（data/test-photos/）
	testPhotosDir := filepath.Join(filepath.Dir(s.photosDir), "test-photos")
	if err := os.MkdirAll(testPhotosDir, 0755); err != nil {
		log.Printf("ERROR: failed to create test-photos dir %s: %v", testPhotosDir, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ファイル名: {camera_id}_latest.jpg
	filename := domain.TestPhotoFilename(camera.ID)
	filePath := filepath.Join(testPhotosDir, filename)
	relPath := fmt.Sprintf("test-photos/%s", filename)

	// ファイルの保存（上書き）
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("ERROR: failed to create test photo file %s: %v", filePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(dst, file); err != nil {
		if closeErr := dst.Close(); closeErr != nil {
			log.Printf("WARN: failed to close test photo file %s: %v", filePath, closeErr)
		}
		if removeErr := os.Remove(filePath); removeErr != nil {
			log.Printf("WARN: failed to remove test photo file %s: %v", filePath, removeErr)
		}
		log.Printf("ERROR: failed to save test photo file %s: %v", filePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := dst.Close(); err != nil {
		log.Printf("ERROR: failed to close test photo file %s: %v", filePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// カメラの更新
	capturedAt := time.Now().UTC()
	if err := s.cameraRepo.UpdateCameraAfterTestPhoto(camera.ID, relPath, capturedAt); err != nil {
		log.Printf("ERROR: failed to update camera %d after test photo: %v", camera.ID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
