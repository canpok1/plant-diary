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

const (
	basePrompt = "この植物の写真を見て、成長の様子や変化を観察してください。親しみやすい口調で、200文字程度の観察日記を書いてください。"
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

		// 過去1ヶ月の日記を取得（対象日は除外）
		// 対象日の0時（日付境界）を計算
		startOfDay := time.Date(img.createdAt.Year(), img.createdAt.Month(), img.createdAt.Day(), 0, 0, 0, 0, img.createdAt.Location())
		oneMonthAgo := startOfDay.AddDate(0, -1, 0)
		endOfPrevDay := startOfDay.Add(-time.Nanosecond)
		pastDiaries, err := w.repo.GetDiariesInDateRange(oneMonthAgo, endOfPrevDay)
		if err != nil {
			log.Printf("WARN: failed to get past diaries for %s: %v, continuing with empty history", img.path, err)
			pastDiaries = []Diary{}
		}

		// 動的プロンプトを生成
		prompt := buildDiaryPrompt(pastDiaries)

		// 日記を生成（リトライ付き）
		var content string
		retryErr := Retry(w.retryConfig, fmt.Sprintf("generate diary for %s", img.path), func() error {
			var genErr error
			// 型アサーションで DiaryGeneratorWithPrompt をサポートしているか確認
			if genWithPrompt, ok := w.generator.(DiaryGeneratorWithPrompt); ok {
				content, genErr = genWithPrompt.GenerateDiaryWithPrompt(img.path, prompt)
			} else {
				content, genErr = w.generator.GenerateDiary(img.path)
			}
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

// buildDiaryPrompt は過去日記を含む動的プロンプトを生成する。
func buildDiaryPrompt(pastDiaries []Diary) string {
	if len(pastDiaries) == 0 {
		return basePrompt
	}

	// JSTに変換するためのヘルパー
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)

	var builder strings.Builder
	builder.WriteString(basePrompt)
	builder.WriteString("\n\n参考までに、過去1ヶ月の観察記録を以下に示します：\n\n")

	for _, diary := range pastDiaries {
		jstTime := diary.CreatedAt.In(jst)
		fmt.Fprintf(&builder, "【%s】\n%s\n\n", jstTime.Format("2006年01月02日"), diary.Content)
	}

	builder.WriteString("これまでの観察記録を踏まえて、今回の写真から見られる成長の変化や特徴を記述してください。")

	return builder.String()
}
