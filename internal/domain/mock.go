package domain

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"plant-diary/internal/utils"
)

// MockDiaryRepository はメモリ上でデータを保持するモック実装
type MockDiaryRepository struct {
	mu          sync.RWMutex
	diaries     map[int]*Diary
	bookDiaries map[int][]int // bookID -> []diaryID
	nextID      int
}

// NewMockDiaryRepository は新しいMockDiaryRepositoryを生成する
func NewMockDiaryRepository() *MockDiaryRepository {
	return &MockDiaryRepository{
		diaries:     make(map[int]*Diary),
		bookDiaries: make(map[int][]int),
		nextID:      1,
	}
}

// GetAllDiaries は全ての日記を新着順で返す
func (r *MockDiaryRepository) GetAllDiaries() ([]Diary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Diary, 0, len(r.diaries))
	for _, d := range r.diaries {
		result = append(result, *d)
	}

	// 新着順（CreatedAt降順）でソート
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// GetDiaryByID は指定IDの日記を返す。見つからない場合はnilを返す
func (r *MockDiaryRepository) GetDiaryByID(id int) (*Diary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.diaries[id]
	if !ok {
		return nil, nil
	}

	copy := *d
	return &copy, nil
}

// UpdateDiaryContent は指定IDの日記のcontentを更新する。見つからない場合はエラーを返す
func (r *MockDiaryRepository) UpdateDiaryContent(id int, content string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.diaries[id]
	if !ok {
		return fmt.Errorf("diary %d not found", id)
	}
	d.Content = content
	return nil
}

// CreateDiaryForUser は指定ユーザーの新しい日記エントリを作成する（モックではuserIDを無視）
func (r *MockDiaryRepository) CreateDiaryForUser(userID int, imagePath, content string, createdAt time.Time) error {
	return r.CreateDiary(imagePath, content, createdAt)
}

// CreateDiaryForBook は指定日記帳・クリエイターの新しい日記エントリを作成する（モックではcreatorIDを無視）
func (r *MockDiaryRepository) CreateDiaryForBook(bookID, creatorID int, imagePath, content string, createdAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	bookIDCopy := bookID
	diary := &Diary{
		ID:        r.nextID,
		ImagePath: imagePath,
		Content:   content,
		CreatedAt: createdAt,
		BookID:    &bookIDCopy,
	}
	r.diaries[r.nextID] = diary
	r.bookDiaries[bookID] = append(r.bookDiaries[bookID], r.nextID)
	r.nextID++
	return nil
}

// GetDiariesByBookID は指定日記帳IDの日記を新着順で返す
func (r *MockDiaryRepository) GetDiariesByBookID(bookID int) ([]Diary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	diaryIDs, ok := r.bookDiaries[bookID]
	if !ok {
		return nil, nil
	}

	result := make([]Diary, 0, len(diaryIDs))
	for _, id := range diaryIDs {
		if d, ok := r.diaries[id]; ok {
			result = append(result, *d)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

// CreateDiary は新しい日記エントリを作成する
func (r *MockDiaryRepository) CreateDiary(imagePath, content string, createdAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	diary := &Diary{
		ID:        r.nextID,
		ImagePath: imagePath,
		Content:   content,
		CreatedAt: createdAt,
	}
	r.diaries[r.nextID] = diary
	r.nextID++

	return nil
}

// IsImageProcessed は指定画像パスが既に処理済みかどうかを返す
func (r *MockDiaryRepository) IsImageProcessed(imagePath string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, d := range r.diaries {
		if d.ImagePath == imagePath {
			return true, nil
		}
	}

	return false, nil
}

// GetLatestDiaryCreatedAt は最新の日記の作成日時を返す。日記が存在しない場合はゼロ値を返す
func (r *MockDiaryRepository) GetLatestDiaryCreatedAt() (time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.diaries) == 0 {
		return time.Time{}, nil
	}

	var latest time.Time
	for _, d := range r.diaries {
		if d.CreatedAt.After(latest) {
			latest = d.CreatedAt
		}
	}

	return latest, nil
}

// GetAvailableYearMonths は日記が存在する年月一覧をJST基準で新しい順に返す
func (r *MockDiaryRepository) GetAvailableYearMonths() ([]YearMonth, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	seen := make(map[string]bool)
	var result []YearMonth
	for _, d := range r.diaries {
		t := d.CreatedAt.In(jst)
		key := fmt.Sprintf("%d-%02d", t.Year(), int(t.Month()))
		if !seen[key] {
			seen[key] = true
			result = append(result, YearMonth{Year: t.Year(), Month: int(t.Month())})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Year != result[j].Year {
			return result[i].Year > result[j].Year
		}
		return result[i].Month > result[j].Month
	})

	return result, nil
}

// SearchDiaries はキーワードを含む日記を新着順で返す
func (r *MockDiaryRepository) SearchDiaries(keyword string) ([]Diary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Diary, 0)
	for _, d := range r.diaries {
		if strings.Contains(d.Content, keyword) {
			result = append(result, *d)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

// GetDiariesByBookIDAsc は指定日記帳IDの日記を古い順で返す。from/toがゼロ値の場合はその条件を無視する
func (r *MockDiaryRepository) GetDiariesByBookIDAsc(bookID int, from, to time.Time) ([]Diary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	diaryIDs, ok := r.bookDiaries[bookID]
	if !ok {
		return nil, nil
	}

	result := make([]Diary, 0)
	for _, id := range diaryIDs {
		if d, ok := r.diaries[id]; ok {
			if from.IsZero() || (d.CreatedAt.Equal(from) || d.CreatedAt.After(from)) {
				if to.IsZero() || (d.CreatedAt.Equal(to) || d.CreatedAt.Before(to)) {
					result = append(result, *d)
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

// GetDiariesInDateRange は指定日記帳・日付範囲内の日記を古い順で返す
func (r *MockDiaryRepository) GetDiariesInDateRange(bookID int, startDate, endDate time.Time) ([]Diary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	diaryIDs, ok := r.bookDiaries[bookID]
	if !ok {
		return []Diary{}, nil
	}

	result := make([]Diary, 0)
	for _, id := range diaryIDs {
		if d, ok := r.diaries[id]; ok {
			if (d.CreatedAt.Equal(startDate) || d.CreatedAt.After(startDate)) &&
				(d.CreatedAt.Equal(endDate) || d.CreatedAt.Before(endDate)) {
				result = append(result, *d)
			}
		}
	}

	// 古い順（CreatedAt昇順）でソート
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

// MockBookRepository はメモリ上でデータを保持するモック実装
type MockBookRepository struct {
	mu     sync.RWMutex
	books  map[int]*Book
	nextID int
}

// NewMockBookRepository は新しいMockBookRepositoryを生成する
func NewMockBookRepository() *MockBookRepository {
	return &MockBookRepository{
		books:  make(map[int]*Book),
		nextID: 1,
	}
}

// GetAllBooks は全ての日記帳をBookView形式で返す（モックではLatestDiaryAtはnil）
func (r *MockBookRepository) GetAllBooks() ([]BookView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]BookView, 0, len(r.books))
	for _, b := range r.books {
		result = append(result, BookView{
			ID:   b.ID,
			Name: b.Name,
		})
	}
	return result, nil
}

// CreateBook は新しい日記帳を作成する
func (r *MockBookRepository) CreateBook(creatorID int, name string) (*Book, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	uuid, err := utils.GenerateUUID()
	if err != nil {
		return nil, err
	}
	uploadKey, err := utils.GenerateUUID()
	if err != nil {
		return nil, err
	}

	book := &Book{
		ID:        r.nextID,
		UUID:      uuid,
		CreatorID: creatorID,
		Name:      name,
		UploadKey: uploadKey,
		CreatedAt: time.Now(),
	}
	r.books[r.nextID] = book
	r.nextID++

	copy := *book
	return &copy, nil
}

// GetBooksByCreatorID は指定クリエイターIDの日記帳一覧を返す
func (r *MockBookRepository) GetBooksByCreatorID(creatorID int) ([]Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Book, 0)
	for _, b := range r.books {
		if b.CreatorID == creatorID {
			result = append(result, *b)
		}
	}
	return result, nil
}

// GetBookByID は指定IDの日記帳を返す。見つからない場合はnilを返す
func (r *MockBookRepository) GetBookByID(id int) (*Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.books[id]
	if !ok {
		return nil, nil
	}
	copy := *b
	return &copy, nil
}

// GetBookByUploadKey は指定upload_keyの日記帳を返す。見つからない場合はnilを返す
func (r *MockBookRepository) GetBookByUploadKey(uploadKey string) (*Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, b := range r.books {
		if b.UploadKey == uploadKey {
			copy := *b
			return &copy, nil
		}
	}
	return nil, nil
}

// UpdateBookName は指定IDの日記帳の名前を更新する
func (r *MockBookRepository) UpdateBookName(id int, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.books[id]
	if !ok {
		return fmt.Errorf("book %d not found", id)
	}
	b.Name = name
	return nil
}

// DeleteBook は指定IDの日記帳を削除する
func (r *MockBookRepository) DeleteBook(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.books[id]; !ok {
		return fmt.Errorf("book %d not found", id)
	}
	delete(r.books, id)
	return nil
}
