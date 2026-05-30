package migrations

import (
	"database/sql"
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/database"
)

func setupTestDB(t *testing.T) *sql.DB {

	cfg := &config.Config{
		DBHost: "localhost",
		DBPort: "5432",
		DBUser: "postgres",
		DBPassword: "masterpassword",
		DBName: "oort_objects_test",
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		t.Fatalf("db connection failed: %v", err)
	}

	return db
}

func TestMigrationsRun(t *testing.T) {

	db := setupTestDB(t)
	defer db.Close()

	err := Run(db, "../../../migrations")

	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	var exists bool

	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'buckets'
		)
	`).Scan(&exists)

	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if !exists {
		t.Fatalf("expected buckets table to exist")
	}
}