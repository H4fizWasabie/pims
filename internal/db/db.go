package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

//go:embed migration.sql
var migrationSQL string

func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Println("migrations applied")
	return nil
}
