package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func createTestSignedURL() *models.SignedURL {
	return &models.SignedURL{
		ID:         uuid.New(),
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Token:      "oort_" + uuid.New().String(),
		Operation:  "upload",
		ExpiresAt:  time.Now().UTC().Add(1 * time.Hour),
	}
}

func TestCreateSignedURL(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresSignedURLRepository(db)
	su := createTestSignedURL()

	err := repo.Create(context.Background(), su)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestFindSignedURLByToken(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresSignedURLRepository(db)
	su := createTestSignedURL()

	err := repo.Create(context.Background(), su)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	found, err := repo.FindByToken(context.Background(), su.Token)
	if err != nil {
		t.Fatalf("expected to find token, got %v", err)
	}

	if found.BucketName != "docs" {
		t.Fatalf("expected docs, got %s", found.BucketName)
	}
	if found.ObjectKey != "test.txt" {
		t.Fatalf("expected test.txt, got %s", found.ObjectKey)
	}
	if found.Operation != "upload" {
		t.Fatalf("expected upload, got %s", found.Operation)
	}
}

func TestFindSignedURLByTokenNotFound(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresSignedURLRepository(db)

	_, err := repo.FindByToken(context.Background(), "oort_nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent token")
	}
}

func TestDeleteSignedURL(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresSignedURLRepository(db)
	su := createTestSignedURL()

	err := repo.Create(context.Background(), su)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	err = repo.Delete(context.Background(), su.ID)
	if err != nil {
		t.Fatalf("expected no error deleting, got %v", err)
	}

	_, err = repo.FindByToken(context.Background(), su.Token)
	if err == nil {
		t.Fatal("expected token to be deleted")
	}
}
