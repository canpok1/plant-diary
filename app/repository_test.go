package main

import (
	"testing"
	"time"
)

func TestMockDiaryRepository_CreateDiary(t *testing.T) {
	repo := NewMockDiaryRepository()

	err := repo.CreateDiary("/path/to/image.jpg", "テスト日記")
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

	err := repo.CreateDiary("/path/to/image.jpg", "テスト日記")
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
	err := repo.CreateDiary("/path/1.jpg", "日記1")
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = repo.CreateDiary("/path/2.jpg", "日記2")
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = repo.CreateDiary("/path/3.jpg", "日記3")
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
	err = repo.CreateDiary("/path/to/new.jpg", "新しい日記")
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
