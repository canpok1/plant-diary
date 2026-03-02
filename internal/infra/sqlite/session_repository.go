package sqlite

import (
	"database/sql"
	"log"
	"time"

	"plant-diary/internal/domain"
)

// SQLiteSessionRepository はSQLiteを使用したSessionRepositoryの実装
type SQLiteSessionRepository struct {
	db *sql.DB
}

// NewSQLiteSessionRepository は新しいSQLiteSessionRepositoryを生成する
func NewSQLiteSessionRepository(db *sql.DB) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{db: db}
}

// CreateSession は新しいセッションを作成する（期限切れセッションも同時に削除）
func (r *SQLiteSessionRepository) CreateSession(id string, userID int, expiresAt time.Time) error {
	if _, err := r.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now()); err != nil {
		log.Printf("WARN: failed to delete expired sessions: %v", err)
	}
	_, err := r.db.Exec(
		"INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)",
		id, userID, expiresAt,
	)
	return err
}

// GetSessionByID はセッションIDからセッションを取得する。見つからない場合や期限切れの場合はnilを返す
func (r *SQLiteSessionRepository) GetSessionByID(id string) (*domain.Session, error) {
	var s domain.Session
	err := r.db.QueryRow(
		"SELECT id, user_id, created_at, expires_at FROM sessions WHERE id = ? AND expires_at > ?",
		id, time.Now(),
	).Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteSession はセッションを削除する
func (r *SQLiteSessionRepository) DeleteSession(id string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}
