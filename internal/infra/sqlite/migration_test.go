package sqlite

import (
	"database/sql"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// setupMigrate はテスト用のインメモリSQLiteとmigrateインスタンスを作成する
func setupMigrate(t *testing.T) (*sql.DB, *migrate.Migrate) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		db.Close()
		t.Fatalf("failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://../../../migrations", "sqlite3", driver)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create migrate instance: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	return db, m
}

// TestMigrations_AllUp は空のインメモリSQLiteに対して全マイグレーションをUp適用し、
// エラーなく完了することを確認する
func TestMigrations_AllUp(t *testing.T) {
	_, m := setupMigrate(t)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Up failed: %v", err)
	}
}

// TestMigrations_AllDown は全Up適用後に全Down適用し、エラーなく完了すること（テーブルが全て削除されること）を確認する
func TestMigrations_AllDown(t *testing.T) {
	db, m := setupMigrate(t)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Up failed: %v", err)
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Down failed: %v", err)
	}

	// テーブルが全て削除されていることを確認
	for _, table := range []string{"diary", "users", "sessions", "books"} {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			t.Fatalf("failed to query sqlite_master for table %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("table %s still exists after Down", table)
		}
	}
}

// TestMigrations_StepByStep は各マイグレーションを1ステップずつUp→Downし、
// 各ステップが正しく動作することを確認する
func TestMigrations_StepByStep(t *testing.T) {
	_, m := setupMigrate(t)

	const numMigrations = 6

	// 1ステップずつUpを適用
	for step := 1; step <= numMigrations; step++ {
		if err := m.Steps(1); err != nil {
			t.Fatalf("Steps(1) at migration %d failed: %v", step, err)
		}
	}

	// 1ステップずつDownを適用
	for step := numMigrations; step >= 1; step-- {
		if err := m.Steps(-1); err != nil {
			t.Fatalf("Steps(-1) at migration %d failed: %v", step, err)
		}
	}
}

// TestMigrations_UpDownUp は全Up → 全Down → 全Up のサイクルがエラーなく完了することを確認する（冪等性の確認）
func TestMigrations_UpDownUp(t *testing.T) {
	_, m := setupMigrate(t)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("first Up failed: %v", err)
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Down failed: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("second Up failed: %v", err)
	}
}

// TestMigrations_FinalSchema は全マイグレーション適用後に期待するテーブル・カラム・インデックスが存在することを確認する
func TestMigrations_FinalSchema(t *testing.T) {
	db, m := setupMigrate(t)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Up failed: %v", err)
	}

	// テーブルの存在確認
	for _, table := range []string{"diary", "users", "sessions", "books"} {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			t.Fatalf("failed to query sqlite_master for table %s: %v", table, err)
		}
		if count == 0 {
			t.Errorf("table %s does not exist", table)
		}
	}

	// カラムの存在確認
	tableColumns := map[string][]string{
		"diary":    {"id", "image_path", "content", "created_at", "updated_at", "user_id", "book_id"},
		"users":    {"id", "uuid", "username", "password_hash", "created_at"},
		"sessions": {"id", "user_id", "created_at", "expires_at"},
		"books":    {"id", "uuid", "creator_id", "name", "upload_key", "created_at"},
	}
	for table, expectedCols := range tableColumns {
		existingCols := migrationGetColumns(t, db, table)
		for _, col := range expectedCols {
			if !migrationContains(existingCols, col) {
				t.Errorf("column %s.%s does not exist", table, col)
			}
		}
	}

	// インデックスの存在確認
	for _, idx := range []string{"idx_created_at", "idx_sessions_expires_at"} {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx,
		).Scan(&count); err != nil {
			t.Fatalf("failed to query sqlite_master for index %s: %v", idx, err)
		}
		if count == 0 {
			t.Errorf("index %s does not exist", idx)
		}
	}
}

// migrationGetColumns はテーブルのカラム名一覧を返す
func migrationGetColumns(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("failed to get table info for %s: %v", table, err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("failed to scan table info for %s: %v", table, err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error for table %s: %v", table, err)
	}
	return columns
}

// migrationContains はスライスに指定の要素が含まれるかを返す
func migrationContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
