package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"plant-diary/internal/adapter"
	"plant-diary/internal/domain"
	"plant-diary/internal/usecase"
	"plant-diary/internal/utils"
)

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

// handlePhotoWithUserUUID はユーザーUUID配下の画像ファイルを配信する
func (s *Server) handlePhotoWithUserUUID(w http.ResponseWriter, r *http.Request) {
	userUUID := r.PathValue("user_uuid")
	filename := r.PathValue("filename")

	// ディレクトリトラバーサル防止
	if userUUID == "" || filename == "" ||
		userUUID == "." || userUUID == ".." ||
		filename == "." || filename == ".." ||
		strings.Contains(userUUID, "/") || strings.Contains(userUUID, "\\") ||
		strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		s.renderError(w, http.StatusNotFound)
		return
	}

	filePath := filepath.Join(s.photosDir, userUUID, filename)
	http.ServeFile(w, r, filePath)
}

// PostApiPhotos は写真アップロードAPIのハンドラ（POST /api/photos）
func (s *Server) PostApiPhotos(w http.ResponseWriter, r *http.Request) {
	// multipart/form-data のパース（最大32MB）
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// upload_key からBookを取得して認証
	uploadKey := r.FormValue("upload_key")
	if uploadKey == "" {
		http.Error(w, "Bad Request: upload_key is required", http.StatusBadRequest)
		return
	}

	book, err := s.bookRepo.GetBookByUploadKey(uploadKey)
	if err != nil {
		log.Printf("ERROR: failed to get book by upload key: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// captured_at の解析（省略時はサーバー受信時刻）
	capturedAt := time.Now().UTC()
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
	defer file.Close()

	// 保存先ディレクトリの作成（book.UUID ベース）
	bookDir := filepath.Join(s.photosDir, book.UUID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		log.Printf("ERROR: failed to create book photo dir %s: %v", bookDir, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ファイル名生成（YYYYMMDD_HHMMSS_UTC.jpg）秒単位で衝突を回避
	filename := capturedAt.Format("20060102_150405") + "_UTC.jpg"
	diskImagePath := filepath.Join(bookDir, filename) // ファイルシステム上のパス（AI生成・ファイル保存用）
	dbImagePath := filepath.Join(book.UUID, filename) // DB保存用相対パス（{Book.UUID}/{filename}）

	// ファイルの保存
	dst, err := os.Create(diskImagePath)
	if err != nil {
		log.Printf("ERROR: failed to create photo file %s: %v", diskImagePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		os.Remove(diskImagePath)
		log.Printf("ERROR: failed to save photo file %s: %v", diskImagePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	dst.Close()

	// ジョブIDを生成
	jobID, err := utils.GenerateUUID()
	if err != nil {
		log.Printf("ERROR: failed to generate job ID: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// goroutineで非同期に日記を生成・保存
	go func() {
		startOfDay := time.Date(capturedAt.Year(), capturedAt.Month(), capturedAt.Day(), 0, 0, 0, 0, capturedAt.Location())
		oneMonthAgo := startOfDay.AddDate(0, -1, 0)
		endOfPrevDay := startOfDay.Add(-time.Nanosecond)
		pastDiaries, err := s.repo.GetDiariesInDateRange(book.ID, oneMonthAgo, endOfPrevDay)
		if err != nil {
			log.Printf("WARN: failed to get past diaries for %s: %v, continuing with empty history", diskImagePath, err)
			pastDiaries = []domain.Diary{}
		}

		prompt := buildDiaryPrompt(pastDiaries)

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

		log.Printf("INFO: diary created for %s (job_id: %s)", diskImagePath, jobID)
	}()

	// 202 Accepted を返す
	resp := adapter.UploadPhotoResponse{JobId: jobID}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR: failed to encode response: %v", err)
	}
}

// PostApiUsers はユーザー作成APIのハンドラ（POST /api/users）
func (s *Server) PostApiUsers(w http.ResponseWriter, r *http.Request) {
	// リクエストボディの解析
	var req adapter.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 重複ユーザー名の確認
	existing, err := s.userRepo.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("ERROR: failed to check existing user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// パスワードのハッシュ化
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: failed to hash password: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// UUID生成（ハイフンなし32文字）
	uuid, err := utils.GenerateUUID()
	if err != nil {
		log.Printf("ERROR: failed to generate UUID: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ユーザー作成
	if err := s.userRepo.CreateUser(uuid, req.Username, string(hash)); err != nil {
		log.Printf("ERROR: failed to create user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := adapter.UserResponse{
		Uuid:     uuid,
		Username: req.Username,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR: failed to encode response: %v", err)
	}
}
