package database

import (
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
)

func TestNewPostgresConnection(t *testing.T) {

	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "masterpassword",
		DBName:     "oort_objects_test",
	}

	db, err := NewPostgres(cfg)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	defer db.Close()

	if err = db.Ping(); err != nil {
		t.Fatalf("database not reachable: %v", err)
	}
}