package sqlite

import (
	"database/sql"
	"fmt"

	"plant-diary/internal/domain"
	"plant-diary/internal/utils"
)

// SQLiteBookRepository はSQLiteを使用したBookRepositoryの実装
type SQLiteBookRepository struct {
	db *sql.DB
}

// NewSQLiteBookRepository は新しいSQLiteBookRepositoryを生成する
func NewSQLiteBookRepository(db *sql.DB) *SQLiteBookRepository {
	return &SQLiteBookRepository{db: db}
}

// CreateBook は新しい日記帳を作成する
func (r *SQLiteBookRepository) CreateBook(creatorID int, name string) (*domain.Book, error) {
	uuid, err := utils.GenerateUUID()
	if err != nil {
		return nil, err
	}
	uploadKey, err := utils.GenerateUUID()
	if err != nil {
		return nil, err
	}

	result, err := r.db.Exec(
		"INSERT INTO books (uuid, creator_id, name, upload_key) VALUES (?, ?, ?, ?)",
		uuid, creatorID, name, uploadKey,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetBookByID(int(id))
}

// GetBooksByCreatorID は指定クリエイターIDの日記帳一覧を返す
func (r *SQLiteBookRepository) GetBooksByCreatorID(creatorID int) ([]domain.Book, error) {
	rows, err := r.db.Query(
		"SELECT id, uuid, creator_id, name, upload_key, created_at FROM books WHERE creator_id = ?",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []domain.Book
	for rows.Next() {
		var b domain.Book
		if err := rows.Scan(&b.ID, &b.UUID, &b.CreatorID, &b.Name, &b.UploadKey, &b.CreatedAt); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

// GetBookByID は指定IDの日記帳を返す。見つからない場合はnilを返す
func (r *SQLiteBookRepository) GetBookByID(id int) (*domain.Book, error) {
	var b domain.Book
	err := r.db.QueryRow(
		"SELECT id, uuid, creator_id, name, upload_key, created_at FROM books WHERE id = ?",
		id,
	).Scan(&b.ID, &b.UUID, &b.CreatorID, &b.Name, &b.UploadKey, &b.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// GetAllBooks は全ての日記帳をBookView形式で最新日記日時とともに返す
func (r *SQLiteBookRepository) GetAllBooks() ([]domain.BookView, error) {
	rows, err := r.db.Query(`
		SELECT b.id, b.name,
			(SELECT created_at FROM diary WHERE book_id = b.id ORDER BY created_at DESC LIMIT 1) as latest_diary_at
		FROM books b
		ORDER BY b.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.BookView
	for rows.Next() {
		var bv domain.BookView
		var latestDiaryAt sql.NullTime
		if err := rows.Scan(&bv.ID, &bv.Name, &latestDiaryAt); err != nil {
			return nil, err
		}
		if latestDiaryAt.Valid {
			bv.LatestDiaryAt = latestDiaryAt.Time
			bv.HasDiaries = true
		}

		result = append(result, bv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetBookByUploadKey は指定upload_keyの日記帳を返す。見つからない場合はnilを返す
func (r *SQLiteBookRepository) GetBookByUploadKey(uploadKey string) (*domain.Book, error) {
	var b domain.Book
	err := r.db.QueryRow(
		"SELECT id, uuid, creator_id, name, upload_key, created_at FROM books WHERE upload_key = ?",
		uploadKey,
	).Scan(&b.ID, &b.UUID, &b.CreatorID, &b.Name, &b.UploadKey, &b.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// UpdateBookName は指定IDの日記帳の名前を更新する
func (r *SQLiteBookRepository) UpdateBookName(id int, name string) error {
	result, err := r.db.Exec("UPDATE books SET name = ? WHERE id = ?", name, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("book %d not found", id)
	}
	return nil
}

// DeleteBook は指定IDの日記帳を削除する
func (r *SQLiteBookRepository) DeleteBook(id int) error {
	result, err := r.db.Exec("DELETE FROM books WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("book %d not found", id)
	}
	return nil
}
