package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"plant-diary/internal/domain"
)

// SQLiteDiaryRepository はSQLiteを使用したDiaryRepositoryの実装
type SQLiteDiaryRepository struct {
	db *sql.DB
}

// NewSQLiteDiaryRepository は新しいSQLiteDiaryRepositoryを生成する
func NewSQLiteDiaryRepository(db *sql.DB) *SQLiteDiaryRepository {
	return &SQLiteDiaryRepository{db: db}
}

// GetAllDiaries は全ての日記を新着順（created_at DESC）で返す
func (r *SQLiteDiaryRepository) GetAllDiaries() ([]domain.Diary, error) {
	rows, err := r.db.Query("SELECT id, image_path, content, created_at FROM diary ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []domain.Diary
	for rows.Next() {
		var d domain.Diary
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
func (r *SQLiteDiaryRepository) GetDiaryByID(id int) (*domain.Diary, error) {
	var d domain.Diary
	err := r.db.QueryRow("SELECT id, image_path, content, created_at, book_id FROM diary WHERE id = ?", id).
		Scan(&d.ID, &d.ImagePath, &d.Content, &d.CreatedAt, &d.BookID)
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

// CreateDiaryForUser は指定ユーザーの新しい日記エントリを作成する
func (r *SQLiteDiaryRepository) CreateDiaryForUser(userID int, imagePath, content string, createdAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO diary (image_path, content, created_at, user_id) VALUES (?, ?, ?, ?)",
		imagePath, content, createdAt, userID,
	)
	return err
}

// CreateDiaryForBook は指定日記帳・クリエイターの新しい日記エントリを作成する
func (r *SQLiteDiaryRepository) CreateDiaryForBook(bookID, creatorID int, imagePath, content string, createdAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO diary (image_path, content, created_at, user_id, book_id) VALUES (?, ?, ?, ?, ?)",
		imagePath, content, createdAt, creatorID, bookID,
	)
	return err
}

// UpdateDiaryContent は指定IDの日記のcontentを更新し、updated_atも現在時刻に更新する
func (r *SQLiteDiaryRepository) UpdateDiaryContent(id int, content string) error {
	result, err := r.db.Exec(
		"UPDATE diary SET content = ?, updated_at = ? WHERE id = ?",
		content, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("diary %d not found", id)
	}
	return nil
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
func (r *SQLiteDiaryRepository) GetAvailableYearMonths() ([]domain.YearMonth, error) {
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

	var result []domain.YearMonth
	for rows.Next() {
		var ym domain.YearMonth
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
func (r *SQLiteDiaryRepository) SearchDiaries(keyword string) ([]domain.Diary, error) {
	rows, err := r.db.Query(
		"SELECT id, image_path, content, created_at FROM diary WHERE content LIKE ? ORDER BY created_at DESC",
		"%"+keyword+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []domain.Diary
	for rows.Next() {
		var d domain.Diary
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

// GetDiariesByBookID は指定日記帳IDの日記を新着順（created_at DESC）で返す
func (r *SQLiteDiaryRepository) GetDiariesByBookID(bookID int) ([]domain.Diary, error) {
	rows, err := r.db.Query(
		"SELECT id, image_path, content, created_at FROM diary WHERE book_id = ? ORDER BY created_at DESC",
		bookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []domain.Diary
	for rows.Next() {
		var d domain.Diary
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

// GetDiariesByBookIDAsc は指定日記帳IDの日記を古い順（created_at ASC）で返す。from/toがゼロ値の場合はその条件を無視する
func (r *SQLiteDiaryRepository) GetDiariesByBookIDAsc(bookID int, from, to time.Time) ([]domain.Diary, error) {
	var rows *sql.Rows
	var err error

	switch {
	case from.IsZero() && to.IsZero():
		rows, err = r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE book_id = ? ORDER BY created_at ASC", bookID)
	case from.IsZero():
		rows, err = r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE book_id = ? AND created_at <= ? ORDER BY created_at ASC", bookID, to)
	case to.IsZero():
		rows, err = r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE book_id = ? AND created_at >= ? ORDER BY created_at ASC", bookID, from)
	default:
		rows, err = r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE book_id = ? AND created_at >= ? AND created_at <= ? ORDER BY created_at ASC", bookID, from, to)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []domain.Diary
	for rows.Next() {
		var d domain.Diary
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

// GetDiariesInDateRange は指定日記帳・日付範囲内の日記を古い順（created_at ASC）で返す
func (r *SQLiteDiaryRepository) GetDiariesInDateRange(bookID int, startDate, endDate time.Time) ([]domain.Diary, error) {
	rows, err := r.db.Query("SELECT id, image_path, content, created_at FROM diary WHERE book_id = ? AND created_at >= ? AND created_at <= ? ORDER BY created_at ASC", bookID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []domain.Diary
	for rows.Next() {
		var d domain.Diary
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
