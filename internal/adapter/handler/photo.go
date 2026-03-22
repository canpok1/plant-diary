package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"plant-diary/internal/adapter"
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
