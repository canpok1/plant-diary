package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"plant-diary/internal/domain"
	"plant-diary/internal/usecase"
	"plant-diary/internal/utils"
)

// handlePostScheduledPhoto は POST /api/scheduled-photo のハンドラ
func (s *Server) handlePostScheduledPhoto(w http.ResponseWriter, r *http.Request) {
	scriptKey, ok := extractBearerToken(r)
	if !ok {
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

	// book を解決
	book, err := s.bookRepo.GetBookByID(camera.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book by id: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if book == nil {
		log.Printf("ERROR: book %d not found for camera %d", camera.BookID, camera.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// リクエストボディのサイズ制限（最大32MB）
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)

	// multipart/form-data のパース（最大32MB）
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if err.Error() == "http: request body too large" {
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// サーバー受信時刻（スケジュール更新に使用）
	receivedAt := time.Now().UTC()

	// captured_at の解析（省略時はサーバー受信時刻）
	capturedAt := receivedAt
	if capturedAtStr := r.FormValue("captured_at"); capturedAtStr != "" {
		t, err := time.Parse(time.RFC3339, capturedAtStr)
		if err != nil {
			http.Error(w, "Bad Request: invalid captured_at format", http.StatusBadRequest)
			return
		}
		capturedAt = t.UTC()
	}

	// 写真ファイルの取得
	file, _, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Bad Request: photo field is required", http.StatusBadRequest)
		return
	}
	defer file.Close() //nolint:errcheck

	// 保存先ディレクトリの作成（book.UUID ベース）
	bookDir := filepath.Join(s.photosDir, book.UUID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		log.Printf("ERROR: failed to create book photo dir %s: %v", bookDir, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ファイル名生成（YYYYMMDD_HHMMSS_UUID_UTC.jpg、衝突防止のためUUIDを付加）
	filename := capturedAt.Format("20060102_150405") + "_" + uuid.New().String() + "_UTC.jpg"
	diskImagePath := filepath.Join(bookDir, filename)
	dbImagePath := filepath.Join(book.UUID, filename)

	// ファイルの保存（O_EXCLで衝突時はエラー）
	dst, err := os.OpenFile(diskImagePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		log.Printf("ERROR: failed to create photo file %s: %v", diskImagePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(dst, file); err != nil {
		if closeErr := dst.Close(); closeErr != nil {
			log.Printf("WARN: failed to close photo file %s: %v", diskImagePath, closeErr)
		}
		if removeErr := os.Remove(diskImagePath); removeErr != nil {
			log.Printf("WARN: failed to remove photo file %s: %v", diskImagePath, removeErr)
		}
		log.Printf("ERROR: failed to save photo file %s: %v", diskImagePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := dst.Close(); err != nil {
		log.Printf("ERROR: failed to close photo file %s: %v", diskImagePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// カメラの更新（受信時刻を使用してクライアントによる改ざんを防止）
	if err := s.cameraRepo.UpdateCameraAfterScheduledCapture(camera.ID, receivedAt); err != nil {
		log.Printf("ERROR: failed to update camera %d after scheduled capture: %v", camera.ID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// goroutineで非同期に日記を生成・保存
	if s.generator != nil {
		go func() {
			startOfDay := time.Date(capturedAt.Year(), capturedAt.Month(), capturedAt.Day(), 0, 0, 0, 0, capturedAt.Location())
			oneMonthAgo := startOfDay.AddDate(0, -1, 0)
			endOfPrevDay := startOfDay.Add(-time.Nanosecond)
			pastDiaries, err := s.repo.GetDiariesInDateRange(book.ID, oneMonthAgo, endOfPrevDay)
			if err != nil {
				log.Printf("WARN: failed to get past diaries for %s: %v, continuing with empty history", diskImagePath, err)
				pastDiaries = []domain.Diary{}
			}

			prompt := expandPrompt(book.Prompt, book.Name, capturedAt, pastDiaries)

			var content string
			retryErr := utils.Retry(utils.DefaultRetryConfig(), fmt.Sprintf("generate diary for %s", diskImagePath), func() error {
				var genErr error
				if genWithPrompt, ok := s.generator.(usecase.DiaryGeneratorWithPrompt); ok {
					content, genErr = genWithPrompt.GenerateDiaryWithPrompt(diskImagePath, prompt)
				} else {
					content, genErr = s.generator.GenerateDiary(diskImagePath)
				}
				return genErr
			})
			if retryErr != nil {
				log.Printf("ERROR: failed to generate diary for %s: %v", diskImagePath, retryErr)
				return
			}

			if err := s.repo.CreateDiaryForBook(book.ID, book.CreatorID, dbImagePath, content, capturedAt); err != nil {
				log.Printf("ERROR: failed to save diary for %s: %v", diskImagePath, err)
				return
			}

			log.Printf("INFO: diary created for %s (scheduled capture)", diskImagePath)
		}()
	}

	w.WriteHeader(http.StatusOK)
}
