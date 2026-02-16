package main

import (
	"testing"
	"time"
)

func TestMockDiaryRepository_CreateDiary(t *testing.T) {
	repo := NewMockDiaryRepository()

	err := repo.CreateDiary("/path/to/image.jpg", "テスト日記", time.Now())
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	diaries, err := repo.GetAllDiaries()
	if err != nil {
		t.Fatalf("GetAllDiaries failed: %v", err)
	}

	if len(diaries) != 1 {
		t.Errorf("expected 1 diary, got %d", len(diaries))
	}

	if diaries[0].ImagePath != "/path/to/image.jpg" {
		t.Errorf("expected ImagePath '/path/to/image.jpg', got '%s'", diaries[0].ImagePath)
	}

	if diaries[0].Content != "テスト日記" {
		t.Errorf("expected Content 'テスト日記', got '%s'", diaries[0].Content)
	}
}

func TestMockDiaryRepository_GetDiaryByID(t *testing.T) {
	repo := NewMockDiaryRepository()

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

	// 存在しないIDを取得
	diary, err = repo.GetDiaryByID(999)
	if err != nil {
		t.Fatalf("GetDiaryByID failed: %v", err)
	}

	if diary != nil {
		t.Errorf("expected nil for non-existent ID, got %v", diary)
	}
}

func TestMockDiaryRepository_GetAllDiaries_Order(t *testing.T) {
	repo := NewMockDiaryRepository()

	// 複数の日記を作成（時間をずらす）
	err := repo.CreateDiary("/path/1.jpg", "日記1", time.Now())
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = repo.CreateDiary("/path/2.jpg", "日記2", time.Now())
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = repo.CreateDiary("/path/3.jpg", "日記3", time.Now())
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	diaries, err := repo.GetAllDiaries()
	if err != nil {
		t.Fatalf("GetAllDiaries failed: %v", err)
	}

	if len(diaries) != 3 {
		t.Errorf("expected 3 diaries, got %d", len(diaries))
	}

	// 新着順（CreatedAt降順）であることを確認
	if diaries[0].Content != "日記3" {
		t.Errorf("expected first diary to be '日記3', got '%s'", diaries[0].Content)
	}

	if diaries[2].Content != "日記1" {
		t.Errorf("expected last diary to be '日記1', got '%s'", diaries[2].Content)
	}
}

func TestMockDiaryRepository_IsImageProcessed(t *testing.T) {
	repo := NewMockDiaryRepository()

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

func TestMockDiaryRepository_GetDiariesInDateRange(t *testing.T) {
	repo := NewMockDiaryRepository()

	// 複数の日記を作成
	time1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	time4 := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)

	err := repo.CreateDiary("/path/1.jpg", "日記1", time1)
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

	err = repo.CreateDiary("/path/4.jpg", "日記4", time4)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	// 日付範囲内の日記を取得
	startDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)

	diaries, err := repo.GetDiariesInDateRange(startDate, endDate)
	if err != nil {
		t.Fatalf("GetDiariesInDateRange failed: %v", err)
	}

	// 2件（日記2と日記3）が取得されることを確認
	if len(diaries) != 2 {
		t.Fatalf("expected 2 diaries, got %d", len(diaries))
	}

	// 古い順（created_at ASC）であることを確認
	if diaries[0].Content != "日記2" {
		t.Errorf("expected first diary to be '日記2', got '%s'", diaries[0].Content)
	}

	if diaries[1].Content != "日記3" {
		t.Errorf("expected second diary to be '日記3', got '%s'", diaries[1].Content)
	}
}

func TestMockDiaryRepository_GetDiariesInDateRange_Empty(t *testing.T) {
	repo := NewMockDiaryRepository()

	// 日記を1件作成
	time1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	err := repo.CreateDiary("/path/1.jpg", "日記1", time1)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	// 範囲外の日付で検索
	startDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)

	diaries, err := repo.GetDiariesInDateRange(startDate, endDate)
	if err != nil {
		t.Fatalf("GetDiariesInDateRange failed: %v", err)
	}

	// 結果が空であることを確認
	if len(diaries) != 0 {
		t.Errorf("expected 0 diaries, got %d", len(diaries))
	}
}
