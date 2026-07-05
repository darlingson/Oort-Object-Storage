package tests

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/database"
	"github.com/darlingson/Oort-Object-Storage/internal/database/migrations"
)

func findMigrationsDir() string {
	wd, _ := os.Getwd()
	dir := wd
	for i := 0; i < 10; i++ {
		path := filepath.Join(dir, "migrations")
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(wd, "migrations")
}

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

	migDir := findMigrationsDir()
	err = migrations.Run(db, migDir)
	if err != nil {
		t.Fatalf("failed running migrations on test db: %v", err)
	}

	return db
}

func TruncateTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE TABLE
			signed_urls,
			user_permissions,
			user_roles,
			role_permissions,
			permissions,
			roles,
			users,
			objects,
			buckets,
			api_keys
		RESTART IDENTITY CASCADE
	`)

	if err != nil {
		t.Fatalf(
			"failed truncating tables: %v",
			err,
		)
	}
}