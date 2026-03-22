package domain

import (
	"database/sql"
	"fmt"
	"time"
)

// DefaultBookPrompt は日記帳のデフォルトプロンプト
const DefaultBookPrompt = `{{book_name}}の写真を見て、成長の様子や変化を観察してください。親しみやすい口調で、200文字程度の観察日記を書いてください。
現在は{{datetime}}です。

{{past_diaries}}

これまでの観察記録を踏まえて、今回の写真から見られる成長の変化や特徴を記述してください。`

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
	Prompt    string
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
	ID                      int
	Name                    string
	ScriptKey               string
	TargetBrightness        float64
	BrightnessTolerance     float64
	MaxAdjustRetries        int
	BookID                  int
	CreatedAt               time.Time
	TestCaptureRequested    bool
	LastTestPhotoPath       sql.NullString
	LastTestPhotoCapturedAt sql.NullTime
	CaptureTimesUTC         sql.NullString // "03:00,09:00" 形式（UTC HH:MM）
	LastScheduledCaptureAt  sql.NullTime
}

// TestPhotoFilename はカメラIDに対応するテスト写真ファイル名を返す
func TestPhotoFilename(cameraID int) string {
	return fmt.Sprintf("%d_latest.jpg", cameraID)
}
