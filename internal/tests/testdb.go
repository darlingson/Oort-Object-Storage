package tests

import (
	"database/sql"
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/database"
)

func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "masterpassword",
		DBName:     "oort_objects_test",
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		t.Fatalf("failed connecting test db: %v", err)
	}

	return db
}

func TruncateTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE TABLE
			objects,
			buckets
		RESTART IDENTITY CASCADE
	`)

	if err != nil {
		t.Fatalf(
			"failed truncating tables: %v",
			err,
		)
	}
}