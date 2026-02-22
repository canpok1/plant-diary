package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// InitDB はデータベース接続を初期化し、接続プールを設定し、マイグレーションを実行する。
func InitDB(dbPath, migrationsPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 接続プール設定
	db.SetMaxOpenConns(1) // SQLiteは単一書き込みのため
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// 接続確認
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// WALモード有効化（読み取り並行性向上）
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// マイグレーション実行
	if err := runMigrations(db, migrationsPath); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("INFO: Database initialized successfully")
	return db, nil
}

// runMigrations はgolang-migrate/migrateを使用してマイグレーションを実行する。
func runMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	// Note: WithInstanceを使用した場合、データベース接続は呼び出し側が管理する。
	// m.Close()を呼び出すとデータベース接続が閉じられる可能性があるため、呼び出さない。
	// マイグレーションは起動時に1回だけ実行されるため、ソースを閉じなくても問題ない。

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("INFO: Database migrations completed successfully")
	return nil
}

// SQLiteDiaryRepository はSQLiteを使用したDiaryRepositoryの実装
type SQLiteDiaryRepository struct {
	db *sql.DB
}

// NewSQLiteDiaryRepository は新しいSQLiteDiaryRepositoryを生成する
func NewSQLiteDiaryRepository(db *sql.DB) *SQLiteDiaryRepository {
	return &SQLiteDiaryRepository{db: db}
}

// GetAllDiaries は全ての日記を新着順（created_at DESC）で返す
func (r *SQLiteDiaryRepository) GetAllDiaries() ([]Diary, error) {
	rows, err := r.db.Query("SELECT id, image_path, content, created_at FROM diary ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []Diary
	for rows.Next() {
		var d Diary
		if err := rows.Scan(&d.ID, &d.ImagePath, &d.Content, &d.CreatedAt); err != nil {
			return nil, err
		}
		diaries = append(diaries, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return diaries, nil
}

// GetDiaryByID は指定IDの日記を返す。見つからない場合はnilを返す
func (r *SQLiteDiaryRepository) GetDiaryByID(id int) (*Diary, error) {
	var d Diary
	err := r.db.QueryRow("SELECT id, image_path, content, created_at FROM diary WHERE id = ?", id).
		Scan(&d.ID, &d.ImagePath, &d.Content, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// CreateDiary は新しい日記エントリを作成する
func (r *SQLiteDiaryRepository) CreateDiary(imagePath, content string, createdAt time.Time) error {
	_, err := r.db.Exec("INSERT INTO diary (image_path, content, created_at) VALUES (?, ?, ?)", imagePath, content, createdAt)
	return err
}

// IsImageProcessed は指定画像パスが既に処理済みかどうかを返す
func (r *SQLiteDiaryRepository) IsImageProcessed(imagePath string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM diary WHERE image_path = ? LIMIT 1)", imagePath).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetLatestDiaryCreatedAt は最新の日記の作成日時を返す。日記が存在しない場合はゼロ値を返す
func (r *SQLiteDiaryRepository) GetLatestDiaryCreatedAt() (time.Time, error) {
	var createdAt time.Time
	err := r.db.QueryRow("SELECT created_at FROM diary ORDER BY created_at DESC LIMIT 1").Scan(&createdAt)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return createdAt, nil
}

// GetAvailableYearMonths は日記が存在する年月一覧をJST基準で新しい順に返す
func (r *SQLiteDiaryRepository) GetAvailableYearMonths() ([]YearMonth, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT
			CAST(strftime('%Y', datetime(created_at, '+9 hours')) AS INTEGER),
			CAST(strftime('%m', datetime(created_at, '+9 hours')) AS INTEGER)
		FROM diary
		ORDER BY 1 DESC, 2 DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []YearMonth
	for rows.Next() {
		var ym YearMonth
		if err := rows.Scan(&ym.Year, &ym.Month); err != nil {
			return nil, err
		}
		result = append(result, ym)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// SearchDiaries はキーワードを含む日記を新着順（created_at DESC）で返す
func (r *SQLiteDiaryRepository) SearchDiaries(keyword string) ([]Diary, error) {
	rows, err := r.db.Query(
		"SELECT id, image_path, content, created_at FROM diary WHERE content LIKE ? ORDER BY created_at DESC",
		"%"+keyword+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []Diary
	for rows.Next() {
		var d Diary
		if err := rows.Scan(&d.ID, &d.ImagePath, &d.Content, &d.CreatedAt); err != nil {
			return nil, err
		}
		diaries = append(diaries, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return diaries, nil
}

// GetDiariesAsc は全日記または指定期間の日記を古い順（created_at ASC）で返す。from/toがゼロ値の場合はその条件を無視する
func (r *SQLiteDiaryRepository) GetDiariesAsc(from, to time.Time) ([]Diary, error) {
	var rows *sql.Rows
	var err error

	switch {
	case from.IsZero() && to.IsZero():
		rows, err = r.db.Query("SELECT id, image_path, content, created_at FROM diary ORDER BY created_at ASC")
	case from.IsZero():
		rows, err = r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE created_at <= ? ORDER BY created_at ASC", to)
	case to.IsZero():
		rows, err = r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE created_at >= ? ORDER BY created_at ASC", from)
	default:
		rows, err = r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE created_at >= ? AND created_at <= ? ORDER BY created_at ASC", from, to)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []Diary
	for rows.Next() {
		var d Diary
		if err := rows.Scan(&d.ID, &d.ImagePath, &d.Content, &d.CreatedAt); err != nil {
			return nil, err
		}
		diaries = append(diaries, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return diaries, nil
}

// GetDiariesInDateRange は指定日付範囲内の日記を古い順（created_at ASC）で返す
func (r *SQLiteDiaryRepository) GetDiariesInDateRange(startDate, endDate time.Time) ([]Diary, error) {
	rows, err := r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE created_at >= ? AND created_at <= ? ORDER BY created_at ASC", startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []Diary
	for rows.Next() {
		var d Diary
		if err := rows.Scan(&d.ID, &d.ImagePath, &d.Content, &d.CreatedAt); err != nil {
			return nil, err
		}
		diaries = append(diaries, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return diaries, nil
}
