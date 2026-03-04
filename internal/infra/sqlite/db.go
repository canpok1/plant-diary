package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
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
		db.Close() //nolint:errcheck
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// WALモード有効化（読み取り並行性向上）
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close() //nolint:errcheck
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// SQLiteバージョン確認（ALTER TABLE DROP COLUMNはSQLite 3.35.0以上が必要）
	if err := checkSQLiteVersion(db); err != nil {
		db.Close() //nolint:errcheck
		return nil, err
	}

	// マイグレーション実行
	if err := runMigrations(db, migrationsPath); err != nil {
		db.Close() //nolint:errcheck
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("INFO: Database initialized successfully")
	return db, nil
}

// checkSQLiteVersion はSQLiteのバージョンが3.35.0以上であることを確認する。
// ALTER TABLE DROP COLUMNがSQLite 3.35.0以上でのみサポートされているため。
func checkSQLiteVersion(db *sql.DB) error {
	var version string
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&version); err != nil {
		return fmt.Errorf("failed to get SQLite version: %w", err)
	}
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return fmt.Errorf("unexpected SQLite version format: %s", version)
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	if major < 3 || (major == 3 && minor < 35) {
		return fmt.Errorf("SQLite %s is not supported; 3.35.0 or later is required", version)
	}
	log.Printf("INFO: SQLite version %s confirmed", version)
	return nil
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
