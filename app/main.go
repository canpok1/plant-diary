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

	// DB初期化とマイグレーション実行
	dbPath := "data/plant_log.db"
	migrationsPath := "migrations"
	db, err := InitDB(dbPath, migrationsPath)
	if err != nil {
		log.Fatalf("FATAL: failed to initialize database: %v", err)
	}
	defer db.Close()

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

	// DiaryRepository の初期化（SQLite実装）
	repo := NewSQLiteDiaryRepository(db)
	log.Println("INFO: Using SQLiteDiaryRepository")

	// UserRepository の初期化（SQLite実装）
	userRepo := NewSQLiteUserRepository(db)

	// SessionRepository の初期化（SQLite実装）
	sessionRepo := NewSQLiteSessionRepository(db)

	// HTTPサーバーの初期化と起動
	photosDir := "data/photos"
	srv, err := NewServer(repo, userRepo, sessionRepo, generator, photosDir)
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

	// HTTPサーバーのGraceful Shutdown（最大5秒待機）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: HTTP server shutdown error: %v", err)
	}

	log.Println("INFO: Plant Diary System stopped")
}
