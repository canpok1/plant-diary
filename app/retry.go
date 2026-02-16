package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

// RetryConfig はリトライ処理の設定を保持する。
type RetryConfig struct {
	MaxRetries int
	Intervals  []time.Duration
	SleepFunc  func(time.Duration) // テスト用に差し替え可能
}

// DefaultRetryConfig はデフォルトのリトライ設定（最大3回、指数バックオフ: 1秒, 2秒, 4秒）を返す。
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		Intervals: []time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
		},
		SleepFunc: time.Sleep,
	}
}

// Retry は指定された関数をリトライ付きで実行する。
// 最初の試行が失敗した場合、設定に基づいて最大MaxRetries回リトライする。
func Retry(config RetryConfig, operation string, fn func() error) error {
	if config.MaxRetries < 0 {
		return fmt.Errorf("MaxRetries must be >= 0")
	}
	if config.SleepFunc == nil {
		config.SleepFunc = time.Sleep
	}
	if len(config.Intervals) < config.MaxRetries {
		return fmt.Errorf("Intervals length (%d) is less than MaxRetries (%d)", len(config.Intervals), config.MaxRetries)
	}
	totalAttempts := 1 + config.MaxRetries
	for attempt := 1; attempt <= totalAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		if attempt < totalAttempts {
			interval := config.Intervals[attempt-1]
			log.Printf("ERROR: %s failed (attempt %d/%d): %v. Retrying in %v...", operation, attempt, totalAttempts, err, interval)
			config.SleepFunc(interval)
		} else {
			log.Printf("ERROR: %s failed (attempt %d/%d): %v. No more retries.", operation, attempt, totalAttempts, err)
			return fmt.Errorf("%s failed after %d attempts: %w", operation, totalAttempts, err)
		}
	}

	return nil
}

// ReadImageFile は画像ファイルを読み込み、バイトデータを返す。
// ファイルが存在しない場合や読み込めない場合はエラーを返す。
func ReadImageFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access image file %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not an image file: %s", path)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("image file is empty: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file %s: %w", path, err)
	}
	return data, nil
}
