package domain

import "time"

// BookRepository は日記帳データへのアクセスを定義するインターフェース
type BookRepository interface {
	CreateBook(creatorID int, name string) (*Book, error)
	GetAllBooks() ([]BookView, error)
	GetBooksByCreatorID(creatorID int) ([]Book, error)
	GetBookByID(id int) (*Book, error)
	UpdateBookName(id int, name string) error
	DeleteBook(id int) error
}

// UserRepository はユーザーデータへのアクセスを定義するインターフェース
type UserRepository interface {
	CreateUser(uuid, username, passwordHash string) error
	GetUserByUsername(username string) (*User, error)
	GetUserByID(id int) (*User, error)
	GetUserByUUID(uuid string) (*User, error)
}

// SessionRepository はセッションデータへのアクセスを定義するインターフェース
type SessionRepository interface {
	CreateSession(id string, userID int, expiresAt time.Time) error
	GetSessionByID(id string) (*Session, error)
	DeleteSession(id string) error
}

// CameraRepository はカメラ設定データへのアクセスを定義するインターフェース
type CameraRepository interface {
	CreateCamera(name string, bookID int) (*Camera, error)
	GetAllCameras() ([]Camera, error)
	GetCameraByID(id int) (*Camera, error)
	GetCameraByScriptKey(scriptKey string) (*Camera, error)
	UpdateCamera(id int, name string, targetBrightness, brightnessTolerance float64, maxAdjustRetries, bookID int) error
	UpdateCameraTestCaptureRequested(id int, requested bool) error
	UpdateCameraAfterTestPhoto(id int, lastTestPhotoPath string, capturedAt time.Time) error
	UpdateCameraAfterScheduledCapture(id int, capturedAt time.Time) error
	DeleteCamera(id int) error
	UpdateCameraScheduleConfig(id int, captureTimesUTC string) error
}

// DiaryRepository は日記データへのアクセスを定義するインターフェース
type DiaryRepository interface {
	GetAllDiaries() ([]Diary, error)
	GetDiaryByID(id int) (*Diary, error)
	GetDiariesByBookID(bookID int) ([]Diary, error)
	CreateDiary(imagePath, content string, createdAt time.Time) error
	CreateDiaryForUser(userID int, imagePath, content string, createdAt time.Time) error
	CreateDiaryForBook(bookID, creatorID int, imagePath, content string, createdAt time.Time) error
	UpdateDiaryContent(id int, content string) error
	IsImageProcessed(imagePath string) (bool, error)
	GetLatestDiaryCreatedAt() (time.Time, error)
	GetDiariesInDateRange(bookID int, startDate, endDate time.Time) ([]Diary, error)
	GetAvailableYearMonths() ([]YearMonth, error)
	SearchDiaries(keyword string) ([]Diary, error)
	GetDiariesByBookIDAsc(bookID int, from, to time.Time) ([]Diary, error)
}
