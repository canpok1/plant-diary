package main

import (
	"testing"
	"time"
)

func TestMockBookRepository_CreateBook(t *testing.T) {
	repo := NewMockBookRepository()

	book, err := repo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	if book == nil {
		t.Fatal("expected book, got nil")
	}
	if book.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if book.CreatorID != 1 {
		t.Errorf("expected CreatorID 1, got %d", book.CreatorID)
	}
	if book.Name != "Test Book" {
		t.Errorf("expected Name 'Test Book', got '%s'", book.Name)
	}
	if book.UUID == "" {
		t.Error("expected non-empty UUID")
	}
	if book.UploadKey == "" {
		t.Error("expected non-empty UploadKey")
	}
}

func TestMockBookRepository_GetBooksByCreatorID(t *testing.T) {
	repo := NewMockBookRepository()

	if _, err := repo.CreateBook(1, "Alice Book 1"); err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	if _, err := repo.CreateBook(1, "Alice Book 2"); err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	if _, err := repo.CreateBook(2, "Bob Book 1"); err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	aliceBooks, err := repo.GetBooksByCreatorID(1)
	if err != nil {
		t.Fatalf("GetBooksByCreatorID failed: %v", err)
	}
	if len(aliceBooks) != 2 {
		t.Errorf("expected 2 books for creator 1, got %d", len(aliceBooks))
	}

	bobBooks, err := repo.GetBooksByCreatorID(2)
	if err != nil {
		t.Fatalf("GetBooksByCreatorID failed: %v", err)
	}
	if len(bobBooks) != 1 {
		t.Errorf("expected 1 book for creator 2, got %d", len(bobBooks))
	}
}

func TestMockBookRepository_GetBookByID(t *testing.T) {
	repo := NewMockBookRepository()

	created, err := repo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	book, err := repo.GetBookByID(created.ID)
	if err != nil {
		t.Fatalf("GetBookByID failed: %v", err)
	}
	if book == nil {
		t.Fatal("expected book, got nil")
	}
	if book.Name != "Test Book" {
		t.Errorf("expected Name 'Test Book', got '%s'", book.Name)
	}

	// 存在しないIDを取得
	notFound, err := repo.GetBookByID(9999)
	if err != nil {
		t.Fatalf("GetBookByID failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-existent ID, got %v", notFound)
	}
}

func TestMockBookRepository_GetBookByUploadKey(t *testing.T) {
	repo := NewMockBookRepository()

	created, err := repo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	book, err := repo.GetBookByUploadKey(created.UploadKey)
	if err != nil {
		t.Fatalf("GetBookByUploadKey failed: %v", err)
	}
	if book == nil {
		t.Fatal("expected book, got nil")
	}
	if book.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, book.ID)
	}

	// 存在しないキーを取得
	notFound, err := repo.GetBookByUploadKey("nonexistent")
	if err != nil {
		t.Fatalf("GetBookByUploadKey failed: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-existent key, got %v", notFound)
	}
}

func TestMockBookRepository_DeleteBook(t *testing.T) {
	repo := NewMockBookRepository()

	created, err := repo.CreateBook(1, "Test Book")
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	if err := repo.DeleteBook(created.ID); err != nil {
		t.Fatalf("DeleteBook failed: %v", err)
	}

	// 削除後に取得するとnilを返す
	book, err := repo.GetBookByID(created.ID)
	if err != nil {
		t.Fatalf("GetBookByID failed: %v", err)
	}
	if book != nil {
		t.Errorf("expected nil after deletion, got %v", book)
	}

	// 存在しないIDを削除するとエラー
	if err := repo.DeleteBook(9999); err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestMockBookRepository_GetAllBooks(t *testing.T) {
	repo := NewMockBookRepository()

	// 日記帳が存在しない場合
	books, err := repo.GetAllBooks()
	if err != nil {
		t.Fatalf("GetAllBooks failed: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected 0 books, got %d", len(books))
	}

	// 日記帳を作成
	if _, err := repo.CreateBook(1, "Book 1"); err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}
	if _, err := repo.CreateBook(2, "Book 2"); err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	books, err = repo.GetAllBooks()
	if err != nil {
		t.Fatalf("GetAllBooks failed: %v", err)
	}
	if len(books) != 2 {
		t.Errorf("expected 2 books, got %d", len(books))
	}
}

func TestMockBookRepository_ImplementsInterface(t *testing.T) {
	// コンパイル時にインターフェースを満たすことを確認
	var _ BookRepository = NewMockBookRepository()
}

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

func TestMockDiaryRepository_GetAvailableYearMonths(t *testing.T) {
	repo := NewMockDiaryRepository()

	// 日記が存在しない場合は空を返す
	months, err := repo.GetAvailableYearMonths()
	if err != nil {
		t.Fatalf("GetAvailableYearMonths failed: %v", err)
	}
	if len(months) != 0 {
		t.Errorf("expected 0 months for empty repo, got %d", len(months))
	}

	// 複数月にまたがる日記を作成（JST基準で異なる年月）
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	time1 := time.Date(2026, 1, 10, 12, 0, 0, 0, jst)
	time2 := time.Date(2026, 1, 20, 12, 0, 0, 0, jst)
	time3 := time.Date(2026, 2, 5, 12, 0, 0, 0, jst)
	time4 := time.Date(2025, 12, 15, 12, 0, 0, 0, jst)

	if err := repo.CreateDiary("/path/1.jpg", "日記1", time1); err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}
	if err := repo.CreateDiary("/path/2.jpg", "日記2", time2); err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}
	if err := repo.CreateDiary("/path/3.jpg", "日記3", time3); err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}
	if err := repo.CreateDiary("/path/4.jpg", "日記4", time4); err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	months, err = repo.GetAvailableYearMonths()
	if err != nil {
		t.Fatalf("GetAvailableYearMonths failed: %v", err)
	}

	// 3つの年月（2026/02, 2026/01, 2025/12）が返ることを確認
	if len(months) != 3 {
		t.Fatalf("expected 3 months, got %d", len(months))
	}

	// 新しい順（降順）であることを確認
	if months[0].Year != 2026 || months[0].Month != 2 {
		t.Errorf("expected first month to be 2026/2, got %d/%d", months[0].Year, months[0].Month)
	}
	if months[1].Year != 2026 || months[1].Month != 1 {
		t.Errorf("expected second month to be 2026/1, got %d/%d", months[1].Year, months[1].Month)
	}
	if months[2].Year != 2025 || months[2].Month != 12 {
		t.Errorf("expected third month to be 2025/12, got %d/%d", months[2].Year, months[2].Month)
	}
}

func TestMockDiaryRepository_SearchDiaries(t *testing.T) {
	repo := NewMockDiaryRepository()

	time1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)

	err := repo.CreateDiary("/path/1.jpg", "葉が青くなってきた", time1)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}
	err = repo.CreateDiary("/path/2.jpg", "花が咲いた", time2)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}
	err = repo.CreateDiary("/path/3.jpg", "葉が黄色に変化した", time3)
	if err != nil {
		t.Fatalf("CreateDiary failed: %v", err)
	}

	// キーワードで検索
	diaries, err := repo.SearchDiaries("葉")
	if err != nil {
		t.Fatalf("SearchDiaries failed: %v", err)
	}

	if len(diaries) != 2 {
		t.Fatalf("expected 2 diaries, got %d", len(diaries))
	}

	// 新着順であることを確認
	if diaries[0].Content != "葉が黄色に変化した" {
		t.Errorf("expected first diary to be '葉が黄色に変化した', got '%s'", diaries[0].Content)
	}
	if diaries[1].Content != "葉が青くなってきた" {
		t.Errorf("expected second diary to be '葉が青くなってきた', got '%s'", diaries[1].Content)
	}

	// マッチしないキーワード
	diaries, err = repo.SearchDiaries("実")
	if err != nil {
		t.Fatalf("SearchDiaries failed: %v", err)
	}
	if len(diaries) != 0 {
		t.Errorf("expected 0 diaries, got %d", len(diaries))
	}

	// キーワードなし（全件）
	diaries, err = repo.SearchDiaries("")
	if err != nil {
		t.Fatalf("SearchDiaries failed: %v", err)
	}
	if len(diaries) != 3 {
		t.Errorf("expected 3 diaries for empty keyword, got %d", len(diaries))
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

func TestMockDiaryRepository_GetDiariesByBookID(t *testing.T) {
	repo := NewMockDiaryRepository()

	// 存在しないbookIDを指定した場合はnilを返す
	diaries, err := repo.GetDiariesByBookID(1)
	if err != nil {
		t.Fatalf("GetDiariesByBookID failed: %v", err)
	}
	if diaries != nil {
		t.Errorf("expected nil for non-existent bookID, got %v", diaries)
	}

	// 複数の日記帳に日記を作成
	time1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)

	if err := repo.CreateDiaryForBook(1, 100, "/path/1.jpg", "日記1", time1); err != nil {
		t.Fatalf("CreateDiaryForBook failed: %v", err)
	}
	if err := repo.CreateDiaryForBook(1, 100, "/path/2.jpg", "日記2", time2); err != nil {
		t.Fatalf("CreateDiaryForBook failed: %v", err)
	}
	if err := repo.CreateDiaryForBook(2, 100, "/path/3.jpg", "日記3", time3); err != nil {
		t.Fatalf("CreateDiaryForBook failed: %v", err)
	}

	// bookID=1の日記を取得（2件）
	diaries, err = repo.GetDiariesByBookID(1)
	if err != nil {
		t.Fatalf("GetDiariesByBookID failed: %v", err)
	}
	if len(diaries) != 2 {
		t.Fatalf("expected 2 diaries for bookID 1, got %d", len(diaries))
	}

	// 新着順（created_at DESC）であることを確認
	if diaries[0].Content != "日記2" {
		t.Errorf("expected first diary to be '日記2', got '%s'", diaries[0].Content)
	}
	if diaries[1].Content != "日記1" {
		t.Errorf("expected second diary to be '日記1', got '%s'", diaries[1].Content)
	}

	// bookID=2の日記を取得（1件）
	diaries, err = repo.GetDiariesByBookID(2)
	if err != nil {
		t.Fatalf("GetDiariesByBookID failed: %v", err)
	}
	if len(diaries) != 1 {
		t.Errorf("expected 1 diary for bookID 2, got %d", len(diaries))
	}
}
