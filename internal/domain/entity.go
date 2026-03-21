package domain

import "time"

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
	BookID    *int
}

// Book は日記帳を表す構造体
type Book struct {
	ID        int
	UUID      string
	CreatorID int
	Name      string
	UploadKey string
	CreatedAt time.Time
}

// BookView はトップページ表示用の日記帳情報
type BookView struct {
	ID            int
	Name          string
	LatestDiaryAt time.Time // 最新日記の日時（ゼロ値は日記なし）
	HasDiaries    bool      // 日記が存在する場合true
}

// User はユーザーを表す構造体
type User struct {
	ID           int
	UUID         string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

// Session はセッションを表す構造体
type Session struct {
	ID        string
	UserID    int
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Camera はカメラ設定を表す構造体
type Camera struct {
	ID                   int
	Name                 string
	ScriptKey            string
	TargetBrightness     float64
	BrightnessTolerance  float64
	MaxAdjustRetries     int
	BookID               int
	CreatedAt            time.Time
	TestCaptureRequested bool
}
