package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"plant-diary/internal/domain"
)

// handleGetCameras は GET /cameras のハンドラ（カメラ一覧）
func (s *Server) handleGetCameras(w http.ResponseWriter, r *http.Request) {
	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	allCameras, err := s.cameraRepo.GetAllCameras()
	if err != nil {
		log.Printf("ERROR: failed to get all cameras: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// 現在のユーザーが所有するカメラのみに絞り込む
	var cameras []domain.Camera
	for _, cam := range allCameras {
		book, err := s.bookRepo.GetBookByID(cam.BookID)
		if err != nil {
			log.Printf("ERROR: failed to get book %d for camera %d: %v", cam.BookID, cam.ID, err)
			continue
		}
		if book == nil || book.CreatorID != user.ID {
			continue
		}
		cameras = append(cameras, cam)
	}

	data := map[string]interface{}{
		"Cameras":  cameras,
		"LoggedIn": true,
		"Username": user.Username,
	}

	if err := s.templates.ExecuteTemplate(w, "cameras.html", data); err != nil {
		log.Printf("ERROR: failed to render cameras template: %v", err)
		s.renderError(w, http.StatusInternalServerError)
	}
}

// handleGetCamerasNew は GET /cameras/new のハンドラ（カメラ追加フォーム）
func (s *Server) handleGetCamerasNew(w http.ResponseWriter, r *http.Request) {
	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
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

	if err := s.templates.ExecuteTemplate(w, "cameras_new.html", data); err != nil {
		log.Printf("ERROR: failed to render cameras_new template: %v", err)
		s.renderError(w, http.StatusInternalServerError)
	}
}

// handlePostCameras は POST /cameras のハンドラ（カメラ登録）
func (s *Server) handlePostCameras(w http.ResponseWriter, r *http.Request) {
	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
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

	bookIDStr := r.FormValue("book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil || bookID <= 0 {
		s.renderError(w, http.StatusBadRequest)
		return
	}

	// 指定した book が現在のユーザーのものであることを確認
	book, err := s.bookRepo.GetBookByID(bookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", bookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil || book.CreatorID != user.ID {
		s.renderError(w, http.StatusNotFound)
		return
	}

	camera, err := s.cameraRepo.CreateCamera(name, bookID)
	if err != nil {
		log.Printf("ERROR: failed to create camera: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cameras/%d/settings", camera.ID), http.StatusFound)
}

// handleGetCameraSettings は GET /cameras/{id}/settings のハンドラ（カメラ設定フォーム）
func (s *Server) handleGetCameraSettings(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid camera id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	camera, err := s.cameraRepo.GetCameraByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get camera %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if camera == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// カメラが現在のユーザーの book に属していることを確認
	book, err := s.bookRepo.GetBookByID(camera.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", camera.BookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil || book.CreatorID != user.ID {
		s.renderError(w, http.StatusNotFound)
		return
	}

	books, err := s.bookRepo.GetBooksByCreatorID(user.ID)
	if err != nil {
		log.Printf("ERROR: failed to get books for user %d: %v", user.ID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Camera":   camera,
		"Books":    books,
		"LoggedIn": true,
		"Username": user.Username,
		"Success":  r.URL.Query().Get("success") == "1",
	}

	if err := s.templates.ExecuteTemplate(w, "camera_settings.html", data); err != nil {
		log.Printf("ERROR: failed to render camera_settings template for camera %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
	}
}

// handlePostCameraSettings は POST /cameras/{id}/settings のハンドラ（カメラ設定更新）
func (s *Server) handlePostCameraSettings(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid camera id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	camera, err := s.cameraRepo.GetCameraByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get camera %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if camera == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// カメラが現在のユーザーの book に属していることを確認
	cameraBook, err := s.bookRepo.GetBookByID(camera.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", camera.BookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if cameraBook == nil || cameraBook.CreatorID != user.ID {
		s.renderError(w, http.StatusNotFound)
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

	targetBrightnessStr := r.FormValue("target_brightness")
	targetBrightness, err := strconv.ParseFloat(targetBrightnessStr, 64)
	if err != nil {
		s.renderError(w, http.StatusBadRequest)
		return
	}
	if math.IsNaN(targetBrightness) || math.IsInf(targetBrightness, 0) || targetBrightness < 0 || targetBrightness > 1 {
		s.renderError(w, http.StatusBadRequest)
		return
	}

	brightnessToleranceStr := r.FormValue("brightness_tolerance")
	brightnessTolerance, err := strconv.ParseFloat(brightnessToleranceStr, 64)
	if err != nil {
		s.renderError(w, http.StatusBadRequest)
		return
	}
	if math.IsNaN(brightnessTolerance) || math.IsInf(brightnessTolerance, 0) || brightnessTolerance < 0 || brightnessTolerance > 1 {
		s.renderError(w, http.StatusBadRequest)
		return
	}

	maxAdjustRetriesStr := r.FormValue("max_adjust_retries")
	maxAdjustRetries, err := strconv.Atoi(maxAdjustRetriesStr)
	if err != nil || maxAdjustRetries < 1 {
		s.renderError(w, http.StatusBadRequest)
		return
	}

	bookIDStr := r.FormValue("book_id")
	bookID, err := strconv.Atoi(bookIDStr)
	if err != nil || bookID <= 0 {
		s.renderError(w, http.StatusBadRequest)
		return
	}

	// 更新先 book も現在のユーザーのものであることを確認
	newBook, err := s.bookRepo.GetBookByID(bookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", bookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if newBook == nil || newBook.CreatorID != user.ID {
		s.renderError(w, http.StatusNotFound)
		return
	}

	if err := s.cameraRepo.UpdateCamera(id, name, targetBrightness, brightnessTolerance, maxAdjustRetries, bookID); err != nil {
		log.Printf("ERROR: failed to update camera %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cameras/%d/settings?success=1", id), http.StatusFound)
}

// patchCameraRequest は PATCH /cameras/{id} のリクエストボディ
type patchCameraRequest struct {
	TestCaptureRequested bool `json:"test_capture_requested"`
}

// handlePatchCamera は PATCH /cameras/{id} のハンドラ（テスト撮影リクエスト）
func (s *Server) handlePatchCamera(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid camera id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	var reqBody patchCameraRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		s.renderError(w, http.StatusBadRequest)
		return
	}

	camera, err := s.cameraRepo.GetCameraByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get camera %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if camera == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if user == nil {
		s.renderError(w, http.StatusUnauthorized)
		return
	}

	// カメラが現在のユーザーの book に属していることを確認
	book, err := s.bookRepo.GetBookByID(camera.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", camera.BookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil || book.CreatorID != user.ID {
		s.renderError(w, http.StatusForbidden)
		return
	}

	if err := s.cameraRepo.UpdateCameraTestCaptureRequested(id, reqBody.TestCaptureRequested); err != nil {
		log.Printf("ERROR: failed to update camera %d test_capture_requested: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("ERROR: failed to encode response: %v", err)
	}
}

// handleGetCameraTestPhoto は GET /cameras/{id}/test-photo のハンドラ（テスト写真表示）
func (s *Server) handleGetCameraTestPhoto(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid camera id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	camera, err := s.cameraRepo.GetCameraByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get camera %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if camera == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// カメラが現在のユーザーの book に属していることを確認
	book, err := s.bookRepo.GetBookByID(camera.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", camera.BookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil || book.CreatorID != user.ID {
		s.renderError(w, http.StatusNotFound)
		return
	}

	if !camera.LastTestPhotoPath.Valid {
		s.renderError(w, http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(filepath.Dir(s.photosDir), camera.LastTestPhotoPath.String)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		s.renderError(w, http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullPath)
}

// handlePostCameraDelete は POST /cameras/{id}/delete のハンドラ（カメラ削除）
func (s *Server) handlePostCameraDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("ERROR: invalid camera id: %s", idStr)
		s.renderError(w, http.StatusNotFound)
		return
	}

	camera, err := s.cameraRepo.GetCameraByID(id)
	if err != nil {
		log.Printf("ERROR: failed to get camera %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if camera == nil {
		s.renderError(w, http.StatusNotFound)
		return
	}

	user, err := s.getCurrentUser(r)
	if err != nil {
		log.Printf("ERROR: failed to get current user: %v", err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	// カメラが現在のユーザーの book に属していることを確認
	book, err := s.bookRepo.GetBookByID(camera.BookID)
	if err != nil {
		log.Printf("ERROR: failed to get book %d: %v", camera.BookID, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}
	if book == nil || book.CreatorID != user.ID {
		s.renderError(w, http.StatusNotFound)
		return
	}

	if err := s.cameraRepo.DeleteCamera(id); err != nil {
		log.Printf("ERROR: failed to delete camera %d: %v", id, err)
		s.renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/cameras", http.StatusFound)
}
