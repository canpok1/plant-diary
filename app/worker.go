package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
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
// contextがキャンセルされると停止し、doneチャネルを閉じる。
func (w *Worker) Start(ctx context.Context, done chan struct{}) {
	go func() {
		defer close(done)
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
	// 1. 最新日記の日付を取得
	latestCreatedAt, err := w.repo.GetLatestDiaryCreatedAt()
	if err != nil {
		log.Printf("ERROR: failed to get latest diary created_at: %v", err)
		return
	}

	// 2. 全ファイルをスキャン
	pattern := filepath.Join(w.photosDir, "*.jpg")
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("ERROR: failed to glob images: %v", err)
		return
	}

	// 3. 日付をパースし、フィルタリング
	type imageWithTime struct {
		path      string
		createdAt time.Time
	}
	var validImages []imageWithTime

	for _, file := range files {
		// 既に処理済みならスキップ
		processed, err := w.repo.IsImageProcessed(file)
		if err != nil {
			log.Printf("ERROR: failed to check image status for %s: %v", file, err)
			continue
		}
		if processed {
			continue
		}

		// ファイル名から日付をパース
		createdAt, err := parseCreatedAtFromFilename(file)
		if err != nil {
			log.Printf("WARN: skipping %s: %v", file, err)
			continue
		}

		// 最新日記より新しい画像のみ
		if createdAt.After(latestCreatedAt) {
			validImages = append(validImages, imageWithTime{file, createdAt})
		}
	}

	// 4. 日付順（昇順）にソート
	sort.Slice(validImages, func(i, j int) bool {
		return validImages[i].createdAt.Before(validImages[j].createdAt)
	})

	// 5. 順番に処理
	for _, img := range validImages {
		// 画像ファイルの読み込み検証
		_, err := ReadImageFile(img.path)
		if err != nil {
			log.Printf("ERROR: skipping image %s: %v", img.path, err)
			continue
		}

		// 日記を生成（リトライ付き）
		var content string
		retryErr := Retry(w.retryConfig, fmt.Sprintf("generate diary for %s", img.path), func() error {
			var genErr error
			content, genErr = w.generator.GenerateDiary(img.path)
			return genErr
		})
		if retryErr != nil {
			log.Printf("ERROR: skipping image %s: %v", img.path, retryErr)
			continue
		}

		// 日記を保存
		if err := w.repo.CreateDiary(img.path, content, img.createdAt); err != nil {
			log.Printf("ERROR: failed to save diary for %s: %v", img.path, err)
			continue
		}

		log.Printf("INFO: diary created for %s", img.path)
	}
}

// parseCreatedAtFromFilename はファイル名から撮影日時を抽出する。
// ファイル名形式: YYYYMMDD_HHMM_UTC.jpg
// 例: "20260216_1110_UTC.jpg" -> 2026-02-16 11:10:00 (UTC)
func parseCreatedAtFromFilename(imagePath string) (time.Time, error) {
	basename := filepath.Base(imagePath)
	nameWithoutExt := strings.TrimSuffix(basename, filepath.Ext(basename))

	// 新形式（YYYYMMDD_HHMM_UTC）をパース
	createdAt, err := time.Parse("20060102_1504_UTC", nameWithoutExt)
	if err != nil {
		// 旧形式（YYYYMMDD_HHMM）との後方互換性のため、フォールバック
		createdAt, err = time.Parse("20060102_1504", nameWithoutExt)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse filename %s: %w", basename, err)
		}
	}

	return createdAt, nil
}
