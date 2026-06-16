package repository

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(dbPath, schemaPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open database connection: %w", err)
	}

	// Настройки для стабильной работы SQLite в многопоточном режиме (Gin)
	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
	`)

	if err != nil {
		return nil, fmt.Errorf("cannot exec pragma: %w", err)
	}

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("cannot find schema path: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return nil, fmt.Errorf("cannot exex schema: %w", err)
	}

	return &Repository{db: db}, nil
}
