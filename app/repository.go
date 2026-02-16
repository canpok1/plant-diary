package main

import (
	"sync"
	"time"
)

// Diary は日記エントリを表す構造体
type Diary struct {
	ID        int
	ImagePath string
	Content   string
	CreatedAt time.Time
}

// DiaryRepository は日記データへのアクセスを定義するインターフェース
type DiaryRepository interface {
	GetAllDiaries() ([]Diary, error)
	GetDiaryByID(id int) (*Diary, error)
	CreateDiary(imagePath, content string, createdAt time.Time) error
	IsImageProcessed(imagePath string) (bool, error)
	GetLatestDiaryCreatedAt() (time.Time, error)
}

// MockDiaryRepository はメモリ上でデータを保持するモック実装
type MockDiaryRepository struct {
	mu      sync.RWMutex
	diaries map[int]*Diary
	nextID  int
}

// NewMockDiaryRepository は新しいMockDiaryRepositoryを生成する
func NewMockDiaryRepository() *MockDiaryRepository {
	return &MockDiaryRepository{
		diaries: make(map[int]*Diary),
		nextID:  1,
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
