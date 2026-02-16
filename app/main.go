package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("INFO: Starting Plant Diary System...")

	// コンテキストの作成（シグナルハンドリング）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// DiaryGenerator の初期化
	var generator DiaryGenerator
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		geminiGen, err := NewGeminiDiaryGenerator()
		if err != nil {
			log.Fatalf("FATAL: failed to initialize Gemini API: %v", err)
		}
		generator = geminiGen
		log.Println("INFO: Using GeminiDiaryGenerator")
	} else {
		generator = &MockDiaryGenerator{}
		log.Println("INFO: Using MockDiaryGenerator (GEMINI_API_KEY not set)")
	}

	// DiaryRepository の初期化
	// TODO: Issue #12 完了後、SQLiteDB実装に切り替え
	repo := NewMockDiaryRepository()
	log.Println("INFO: Using MockDiaryRepository")

	// Worker の初期化と起動
	photosDir := "data/photos"
	worker := NewWorker(repo, generator, photosDir)
	worker.Start(ctx)
	log.Printf("INFO: Worker started. Polling %s every 1 minute...", photosDir)

	// HTTPサーバーの初期化と起動
	srv, err := NewServer(repo, photosDir)
	if err != nil {
		log.Fatalf("FATAL: failed to initialize server: %v", err)
	}

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: srv,
	}

	go func() {
		log.Println("INFO: HTTP server starting on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("FATAL: HTTP server error: %v", err)
		}
	}()

	// シグナルハンドリング（Ctrl+C で終了）
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("INFO: Shutting down...")
	cancel()

	// HTTPサーバーのGraceful Shutdown（最大5秒待機）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: HTTP server shutdown error: %v", err)
	}

	log.Println("INFO: Plant Diary System stopped")
}
