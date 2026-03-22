package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// scriptConfigResponse は GET /api/script-config のレスポンス
type scriptConfigResponse struct {
	TargetBrightness      float64 `json:"target_brightness"`
	BrightnessTolerance   float64 `json:"brightness_tolerance"`
	MaxAdjustRetries      int     `json:"max_adjust_retries"`
	ShouldTestCapture     bool    `json:"should_test_capture"`
	ShouldScheduleCapture bool    `json:"should_schedule_capture"`
}

// handleGetScriptConfig は GET /api/script-config のハンドラ
func (s *Server) handleGetScriptConfig(w http.ResponseWriter, r *http.Request) {
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

	resp := scriptConfigResponse{
		TargetBrightness:      camera.TargetBrightness,
		BrightnessTolerance:   camera.BrightnessTolerance,
		MaxAdjustRetries:      camera.MaxAdjustRetries,
		ShouldTestCapture:     camera.TestCaptureRequested,
		ShouldScheduleCapture: computeShouldScheduleCapture(camera.CaptureTimesUTC, camera.LastScheduledCaptureAt, time.Now().UTC()),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR: failed to encode response: %v", err)
	}
}
