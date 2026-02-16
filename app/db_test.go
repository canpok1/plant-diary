package main

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB はテスト用のインメモリSQLiteデータベースを作成する
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS diary (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			image_path TEXT NOT NULL UNIQUE,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_created_at ON diary(created_at DESC);
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteDiaryRepository_CreateDiary(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	err := repo.CreateDiary("/path/to/image.jpg", "テスト日記", time.Now())
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	diaries, err := repo.GetAllDiaries()
	if err != nil {
		t.Fatalf("GetAllDiaries failed: %v", err)
	}

	if len(diaries) != 1 {
		t.Fatalf("expected 1 diary, got %d", len(diaries))
	}

	if diaries[0].ImagePath != "/path/to/image.jpg" {
		t.Errorf("expected ImagePath '/path/to/image.jpg', got '%s'", diaries[0].ImagePath)
	}

	if diaries[0].Content != "テスト日記" {
		t.Errorf("expected Content 'テスト日記', got '%s'", diaries[0].Content)
	}

	if diaries[0].ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestSQLiteDiaryRepository_CreateDiary_DuplicateImagePath(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	err := repo.CreateDiary("/path/to/image.jpg", "日記1", time.Now())
	if err != nil {
		t.Fatalf("first CreateDiary failed: %v", err)
	}

	// 同じimage_pathで再度作成するとUNIQUE制約エラーになる
	err = repo.CreateDiary("/path/to/image.jpg", "日記2", time.Now())
	if err == nil {
		t.Error("expected error for duplicate image_path, got nil")
	}
}

func TestSQLiteDiaryRepository_GetDiaryByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	err := repo.CreateDiary("/path/to/image.jpg", "テスト日記", time.Now())
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	diary, err := repo.GetDiaryByID(1)
	if err != nil {
		t.Fatalf("GetDiaryByID failed: %v", err)
	}

	if diary == nil {
		t.Fatal("expected diary, got nil")
	}

	if diary.ID != 1 {
		t.Errorf("expected ID 1, got %d", diary.ID)
	}

	if diary.ImagePath != "/path/to/image.jpg" {
		t.Errorf("expected ImagePath '/path/to/image.jpg', got '%s'", diary.ImagePath)
	}

	if diary.Content != "テスト日記" {
		t.Errorf("expected Content 'テスト日記', got '%s'", diary.Content)
	}
}

func TestSQLiteDiaryRepository_GetDiaryByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	diary, err := repo.GetDiaryByID(999)
	if err != nil {
		t.Fatalf("GetDiaryByID failed: %v", err)
	}

	if diary != nil {
		t.Errorf("expected nil for non-existent ID, got %v", diary)
	}
}

func TestSQLiteDiaryRepository_GetAllDiaries_Order(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	// 明示的にcreated_atを指定して挿入（順序を保証）
	_, err := db.Exec("INSERT INTO diary (image_path, content, created_at) VALUES (?, ?, ?)",
		"/path/1.jpg", "日記1", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO diary (image_path, content, created_at) VALUES (?, ?, ?)",
		"/path/2.jpg", "日記2", time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO diary (image_path, content, created_at) VALUES (?, ?, ?)",
		"/path/3.jpg", "日記3", time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	diaries, err := repo.GetAllDiaries()
	if err != nil {
		t.Fatalf("GetAllDiaries failed: %v", err)
	}

	if len(diaries) != 3 {
		t.Fatalf("expected 3 diaries, got %d", len(diaries))
	}

	// 新着順（created_at DESC）であることを確認
	if diaries[0].Content != "日記3" {
		t.Errorf("expected first diary to be '日記3', got '%s'", diaries[0].Content)
	}

	if diaries[1].Content != "日記2" {
		t.Errorf("expected second diary to be '日記2', got '%s'", diaries[1].Content)
	}

	if diaries[2].Content != "日記1" {
		t.Errorf("expected last diary to be '日記1', got '%s'", diaries[2].Content)
	}
}

func TestSQLiteDiaryRepository_GetAllDiaries_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	diaries, err := repo.GetAllDiaries()
	if err != nil {
		t.Fatalf("GetAllDiaries failed: %v", err)
	}

	if diaries != nil {
		t.Errorf("expected nil for empty result, got %v", diaries)
	}
}

func TestSQLiteDiaryRepository_IsImageProcessed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	// 未処理の画像
	processed, err := repo.IsImageProcessed("/path/to/new.jpg")
	if err != nil {
		t.Fatalf("IsImageProcessed failed: %v", err)
	}

	if processed {
		t.Error("expected false for unprocessed image, got true")
	}

	// 画像を処理
	err = repo.CreateDiary("/path/to/new.jpg", "新しい日記", time.Now())
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	// 処理済みの画像
	processed, err = repo.IsImageProcessed("/path/to/new.jpg")
	if err != nil {
		t.Fatalf("IsImageProcessed failed: %v", err)
	}

	if !processed {
		t.Error("expected true for processed image, got false")
	}
}

func TestSQLiteDiaryRepository_ImplementsInterface(t *testing.T) {
	db := setupTestDB(t)
	// コンパイル時にインターフェースを満たすことを確認
	var _ DiaryRepository = NewSQLiteDiaryRepository(db)
}

func TestSQLiteDiaryRepository_CreateDiary_CustomCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	// カスタム日時を指定してdiary作成
	customTime := time.Date(2026, 2, 16, 11, 10, 0, 0, time.UTC)
	err := repo.CreateDiary("/path/to/20260216_1110_UTC.jpg", "UTC日時テスト", customTime)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	// 日記を取得して日時を確認
	diaries, err := repo.GetAllDiaries()
	if err != nil {
		t.Fatalf("GetAllDiaries failed: %v", err)
	}

	if len(diaries) != 1 {
		t.Fatalf("expected 1 diary, got %d", len(diaries))
	}

	// created_atがカスタム日時と一致することを確認
	if !diaries[0].CreatedAt.Equal(customTime) {
		t.Errorf("expected CreatedAt %v, got %v", customTime, diaries[0].CreatedAt)
	}
}

func TestSQLiteDiaryRepository_GetLatestDiaryCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteDiaryRepository(db)

	// 日記が存在しない場合はゼロ値を返す
	latest, err := repo.GetLatestDiaryCreatedAt()
	if err != nil {
		t.Fatalf("GetLatestDiaryCreatedAt failed: %v", err)
	}

	if !latest.IsZero() {
		t.Errorf("expected zero time for empty diary, got %v", latest)
	}

	// 複数の日記を作成
	time1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)

	err = repo.CreateDiary("/path/1.jpg", "日記1", time1)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	err = repo.CreateDiary("/path/2.jpg", "日記2", time2)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	err = repo.CreateDiary("/path/3.jpg", "日記3", time3)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	// 最新の日記の日時を取得
	latest, err = repo.GetLatestDiaryCreatedAt()
	if err != nil {
		t.Fatalf("GetLatestDiaryCreatedAt failed: %v", err)
	}

	// 最新の日時（time3）と一致することを確認
	if !latest.Equal(time3) {
		t.Errorf("expected latest time %v, got %v", time3, latest)
	}
}
