package domain

import (
	"database/sql"
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

// MockUserRepository はメモリ上でデータを保持するユーザーモック実装
type MockUserRepository struct {
	mu     sync.RWMutex
	users  map[int]*User
	nextID int
}

// NewMockUserRepository は新しいMockUserRepositoryを生成する
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[int]*User),
		nextID: 1,
	}
}

// CreateUser は新しいユーザーを作成する
func (r *MockUserRepository) CreateUser(uuid, username, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.users {
		if u.Username == username {
			return fmt.Errorf("username %s already exists", username)
		}
	}

	user := &User{
		ID:           r.nextID,
		UUID:         uuid,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	r.users[r.nextID] = user
	r.nextID++
	return nil
}

// GetUserByUsername は指定usernameのユーザーを返す。見つからない場合はnilを返す
func (r *MockUserRepository) GetUserByUsername(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.users {
		if u.Username == username {
			copy := *u
			return &copy, nil
		}
	}
	return nil, nil
}

// GetUserByID は指定IDのユーザーを返す。見つからない場合はnilを返す
func (r *MockUserRepository) GetUserByID(id int) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	copy := *u
	return &copy, nil
}

// GetUserByUUID は指定UUIDのユーザーを返す。見つからない場合はnilを返す
func (r *MockUserRepository) GetUserByUUID(uuid string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.users {
		if u.UUID == uuid {
			copy := *u
			return &copy, nil
		}
	}
	return nil, nil
}

// MockSessionRepository はメモリ上でデータを保持するセッションモック実装
type MockSessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMockSessionRepository は新しいMockSessionRepositoryを生成する
func NewMockSessionRepository() *MockSessionRepository {
	return &MockSessionRepository{
		sessions: make(map[string]*Session),
	}
}

// CreateSession は新しいセッションを作成する
func (r *MockSessionRepository) CreateSession(id string, userID int, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[id] = &Session{
		ID:        id,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	return nil
}

// GetSessionByID は指定IDのセッションを返す。見つからない場合はnilを返す
func (r *MockSessionRepository) GetSessionByID(id string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.sessions[id]
	if !ok {
		return nil, nil
	}
	if s.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}
	copy := *s
	return &copy, nil
}

// DeleteSession は指定IDのセッションを削除する
func (r *MockSessionRepository) DeleteSession(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessions, id)
	return nil
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

// MockCameraRepository はメモリ上でデータを保持するモック実装
type MockCameraRepository struct {
	mu      sync.RWMutex
	cameras map[int]*Camera
	nextID  int
}

// NewMockCameraRepository は新しいMockCameraRepositoryを生成する
func NewMockCameraRepository() *MockCameraRepository {
	return &MockCameraRepository{
		cameras: make(map[int]*Camera),
		nextID:  1,
	}
}

// CreateCamera は新しいカメラを作成する
func (r *MockCameraRepository) CreateCamera(name string, bookID int) (*Camera, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	scriptKey, err := utils.GenerateUUID()
	if err != nil {
		return nil, err
	}

	camera := &Camera{
		ID:                  r.nextID,
		Name:                name,
		ScriptKey:           scriptKey,
		TargetBrightness:    0.475,
		BrightnessTolerance: 0.175,
		MaxAdjustRetries:    5,
		BookID:              bookID,
		CreatedAt:           time.Now(),
	}
	r.cameras[r.nextID] = camera
	r.nextID++

	copy := *camera
	return &copy, nil
}

// GetAllCameras は全てのカメラを返す
func (r *MockCameraRepository) GetAllCameras() ([]Camera, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Camera, 0, len(r.cameras))
	for _, c := range r.cameras {
		result = append(result, *c)
	}
	return result, nil
}

// GetCameraByID は指定IDのカメラを返す。見つからない場合はnilを返す
func (r *MockCameraRepository) GetCameraByID(id int) (*Camera, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.cameras[id]
	if !ok {
		return nil, nil
	}
	copy := *c
	return &copy, nil
}

// GetCameraByScriptKey は指定script_keyのカメラを返す。見つからない場合はnilを返す
func (r *MockCameraRepository) GetCameraByScriptKey(scriptKey string) (*Camera, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.cameras {
		if c.ScriptKey == scriptKey {
			copy := *c
			return &copy, nil
		}
	}
	return nil, nil
}

// UpdateCamera は指定IDのカメラ設定を更新する
func (r *MockCameraRepository) UpdateCamera(id int, name string, targetBrightness, brightnessTolerance float64, maxAdjustRetries, bookID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.cameras[id]
	if !ok {
		return fmt.Errorf("camera %d not found", id)
	}
	c.Name = name
	c.TargetBrightness = targetBrightness
	c.BrightnessTolerance = brightnessTolerance
	c.MaxAdjustRetries = maxAdjustRetries
	c.BookID = bookID
	return nil
}

// UpdateCameraTestCaptureRequested は指定IDのカメラのtest_capture_requestedを更新する
func (r *MockCameraRepository) UpdateCameraTestCaptureRequested(id int, requested bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.cameras[id]
	if !ok {
		return fmt.Errorf("camera %d not found", id)
	}
	c.TestCaptureRequested = requested
	return nil
}

// UpdateCameraAfterTestPhoto は指定IDのカメラのテスト写真情報を更新する
func (r *MockCameraRepository) UpdateCameraAfterTestPhoto(id int, lastTestPhotoPath string, capturedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.cameras[id]
	if !ok {
		return fmt.Errorf("camera %d not found", id)
	}
	c.TestCaptureRequested = false
	c.LastTestPhotoPath = sql.NullString{String: lastTestPhotoPath, Valid: true}
	c.LastTestPhotoCapturedAt = sql.NullTime{Time: capturedAt, Valid: true}
	return nil
}

// UpdateCameraAfterScheduledCapture は指定IDのカメラのスケジュール撮影情報を更新する
func (r *MockCameraRepository) UpdateCameraAfterScheduledCapture(id int, capturedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.cameras[id]
	if !ok {
		return fmt.Errorf("camera %d not found", id)
	}
	c.LastScheduledCaptureAt = sql.NullTime{Time: capturedAt, Valid: true}
	return nil
}

// DeleteCamera は指定IDのカメラを削除する
func (r *MockCameraRepository) DeleteCamera(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.cameras[id]; !ok {
		return fmt.Errorf("camera %d not found", id)
	}
	delete(r.cameras, id)
	return nil
}

// UpdateCameraScheduleConfig はスケジュール撮影の時刻設定を更新する
func (r *MockCameraRepository) UpdateCameraScheduleConfig(id int, captureTimesUTC string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.cameras[id]
	if !ok {
		return fmt.Errorf("camera %d not found", id)
	}
	c.CaptureTimesUTC = sql.NullString{String: captureTimesUTC, Valid: true}
	return nil
}
