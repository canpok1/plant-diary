package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plant-diary/internal/adapter/handler"
	"plant-diary/internal/infra/gemini"
	"plant-diary/internal/infra/sqlite"
	"plant-diary/internal/usecase"
)

func main() {
	log.Println("INFO: Starting Plant Diary System...")

	// DB初期化とマイグレーション実行
	dbPath := "data/plant_log.db"
	migrationsPath := "migrations"
	db, err := sqlite.InitDB(dbPath, migrationsPath)
	if err != nil {
		log.Fatalf("FATAL: failed to initialize database: %v", err)
	}
	defer db.Close() //nolint:errcheck

	// DiaryGenerator の初期化
	var generator usecase.DiaryGenerator
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		geminiGen, err := gemini.NewGeminiDiaryGenerator()
		if err != nil {
			log.Fatalf("FATAL: failed to initialize Gemini API: %v", err)
		}
		generator = geminiGen
		log.Println("INFO: Using GeminiDiaryGenerator")
	} else {
		generator = &gemini.MockDiaryGenerator{}
		log.Println("INFO: Using MockDiaryGenerator (GEMINI_API_KEY not set)")
	}

	// DiaryRepository の初期化（SQLite実装）
	repo := sqlite.NewSQLiteDiaryRepository(db)
	log.Println("INFO: Using SQLiteDiaryRepository")

	// UserRepository の初期化（SQLite実装）
	userRepo := sqlite.NewSQLiteUserRepository(db)

	// BookRepository の初期化（SQLite実装）
	bookRepo := sqlite.NewSQLiteBookRepository(db)

	// SessionRepository の初期化（SQLite実装）
	sessionRepo := sqlite.NewSQLiteSessionRepository(db)

	// HTTPサーバーの初期化と起動
	templatesDir := "templates"
	photosDir := "data/photos"
	srv, err := handler.NewServer(repo, userRepo, bookRepo, sessionRepo, generator, templatesDir, photosDir)
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
