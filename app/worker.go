package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"
)

// Worker は未処理画像を監視し、日記生成処理を実行する。
type Worker struct {
	repo        DiaryRepository
	generator   DiaryGenerator
	photosDir   string
	retryConfig RetryConfig
}

// NewWorker は新しいWorkerを生成する。
func NewWorker(repo DiaryRepository, generator DiaryGenerator, photosDir string) *Worker {
	return &Worker{
		repo:        repo,
		generator:   generator,
		photosDir:   photosDir,
		retryConfig: DefaultRetryConfig(),
	}
}

// Start は1分ごとにポーリングするGoroutineを起動する。
// contextがキャンセルされると停止する。
func (w *Worker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		// 起動直後に1回実行
		w.processNewImages()

		for {
			select {
			case <-ctx.Done():
				log.Println("INFO: Worker stopped")
				return
			case <-ticker.C:
				w.processNewImages()
			}
		}
	}()
}

// processNewImages は未処理画像を検出し、1枚ずつ順次処理する。
func (w *Worker) processNewImages() {
	// data/photos/*.jpg の一覧を取得
	pattern := filepath.Join(w.photosDir, "*.jpg")
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("ERROR: failed to glob images: %v", err)
		return
	}

	for _, file := range files {
		// 未処理画像かどうかを判定
		processed, err := w.repo.IsImageProcessed(file)
		if err != nil {
			log.Printf("ERROR: failed to check image status: %v", err)
			continue
		}
		if processed {
			continue
		}

		// 画像ファイルの読み込み検証
		_, err = ReadImageFile(file)
		if err != nil {
			log.Printf("ERROR: skipping image %s: %v", file, err)
			continue
		}

		// 日記を生成（リトライ付き）
		var content string
		retryErr := Retry(w.retryConfig, fmt.Sprintf("generate diary for %s", file), func() error {
			var genErr error
			content, genErr = w.generator.GenerateDiary(file)
			return genErr
		})
		if retryErr != nil {
			log.Printf("ERROR: skipping image %s: %v", file, retryErr)
			continue
		}

		// 日記を保存
		if err := w.repo.CreateDiary(file, content); err != nil {
			log.Printf("ERROR: failed to save diary for %s: %v", file, err)
			continue
		}

		log.Printf("INFO: diary created for %s", file)
	}
}
