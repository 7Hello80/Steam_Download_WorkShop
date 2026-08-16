package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Connection pool settings (SQLite works best with single writer)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Enable WAL mode and foreign keys
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}

func RunMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			password_hash TEXT,
			github_id TEXT UNIQUE,
			avatar_url TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id)`,

		`CREATE TABLE IF NOT EXISTS steam_accounts (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			steam_username TEXT NOT NULL,
			encrypted_password TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(user_id, steam_username)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_steam_accounts_user_id ON steam_accounts(user_id)`,

		`CREATE TABLE IF NOT EXISTS download_tasks (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			app_id INTEGER NOT NULL,
			pubfile_id INTEGER NOT NULL,
			steam_username TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			queue_position INTEGER,
			output_dir TEXT,
			zip_path TEXT,
			zip_filename TEXT,
			error_message TEXT,
			file_size INTEGER DEFAULT 0,
			login_id INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			started_at DATETIME,
			completed_at DATETIME,
			expires_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON download_tasks(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON download_tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_expires_at ON download_tasks(expires_at)`,

		`CREATE TABLE IF NOT EXISTS announcements (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_by TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_announcements_active ON announcements(is_active)`,

		`CREATE TABLE IF NOT EXISTS payment_records (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			task_id TEXT NOT NULL REFERENCES download_tasks(id) ON DELETE CASCADE,
			amount REAL NOT NULL,
			currency TEXT NOT NULL DEFAULT 'CNY',
			method TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			transaction_id TEXT,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			paid_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payment_records(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_task_id ON payment_records(task_id)`,

		`CREATE TABLE IF NOT EXISTS pending_registrations (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			verification_code TEXT NOT NULL,
			code_expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pending_email ON pending_registrations(email)`,

		`CREATE TABLE IF NOT EXISTS sponsors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			method TEXT NOT NULL DEFAULT 'wechat',
			amount TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			is_visible INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sponsors_visible ON sponsors(is_visible)`,
	}

	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}

	// Add email verification columns (idempotent — ignore errors if columns already exist)
	alterMigrations := []string{
		`ALTER TABLE users ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN verification_code TEXT`,
		`ALTER TABLE users ADD COLUMN verification_code_expires_at DATETIME`,
		`ALTER TABLE users ADD COLUMN banned INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN banned_at DATETIME`,
	}
	for _, m := range alterMigrations {
		db.Exec(m) // ignore error — column may already exist
	}

	return nil
}
