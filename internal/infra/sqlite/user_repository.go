package sqlite

import (
	"database/sql"

	"plant-diary/internal/domain"
)

// SQLiteUserRepository はSQLiteを使用したUserRepositoryの実装
type SQLiteUserRepository struct {
	db *sql.DB
}

// NewSQLiteUserRepository は新しいSQLiteUserRepositoryを生成する
func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

// CreateUser は新しいユーザーを作成する
func (r *SQLiteUserRepository) CreateUser(uuid, username, passwordHash string) error {
	_, err := r.db.Exec(
		"INSERT INTO users (uuid, username, password_hash) VALUES (?, ?, ?)",
		uuid, username, passwordHash,
	)
	return err
}

// GetUserByUsername はusernameからユーザーを取得する。見つからない場合はnilを返す
func (r *SQLiteUserRepository) GetUserByUsername(username string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(
		"SELECT id, uuid, username, password_hash, created_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.UUID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID はIDからユーザーを取得する。見つからない場合はnilを返す
func (r *SQLiteUserRepository) GetUserByID(id int) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(
		"SELECT id, uuid, username, password_hash, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.UUID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByUUID はUUIDからユーザーを取得する。見つからない場合はnilを返す
func (r *SQLiteUserRepository) GetUserByUUID(uuid string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(
		"SELECT id, uuid, username, password_hash, created_at FROM users WHERE uuid = ?",
		uuid,
	).Scan(&u.ID, &u.UUID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
