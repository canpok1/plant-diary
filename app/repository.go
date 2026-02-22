package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// YearMonth は年月を表す構造体
type YearMonth struct {
	Year  int
	Month int
}

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
	GetDiariesInDateRange(startDate, endDate time.Time) ([]Diary, error)
	GetAvailableYearMonths() ([]YearMonth, error)
	SearchDiaries(keyword string) ([]Diary, error)
	GetDiariesAsc(from, to time.Time) ([]Diary, error)
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

// GetDiariesAsc は全日記または指定期間の日記を古い順で返す。from/toがゼロ値の場合は全件取得
func (r *MockDiaryRepository) GetDiariesAsc(from, to time.Time) ([]Diary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Diary, 0)
	for _, d := range r.diaries {
		if from.IsZero() || (d.CreatedAt.Equal(from) || d.CreatedAt.After(from)) {
			if to.IsZero() || (d.CreatedAt.Equal(to) || d.CreatedAt.Before(to)) {
				result = append(result, *d)
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

// GetDiariesInDateRange は指定日付範囲内の日記を古い順で返す
func (r *MockDiaryRepository) GetDiariesInDateRange(startDate, endDate time.Time) ([]Diary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Diary, 0)
	for _, d := range r.diaries {
		if (d.CreatedAt.Equal(startDate) || d.CreatedAt.After(startDate)) &&
			(d.CreatedAt.Equal(endDate) || d.CreatedAt.Before(endDate)) {
			result = append(result, *d)
		}
	}

	// 古い順（CreatedAt昇順）でソート
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.Before(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}
