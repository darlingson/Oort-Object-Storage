package services

import (
	"context"
	"testing"
	"time"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func TestSignUploadURL(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	result, err := svc.Sign(context.Background(), SignInput{
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Operation:  "upload",
		ExpiresIn:  1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Token == "" {
		t.Fatal("expected token to be non-empty")
	}
	if !time.Now().UTC().Before(result.ExpiresAt) {
		t.Fatal("expected expires_at to be in the future")
	}
}

func TestSignDownloadURL(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	result, err := svc.Sign(context.Background(), SignInput{
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Operation:  "download",
		ExpiresIn:  2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Token == "" {
		t.Fatal("expected token to be non-empty")
	}
}

func TestValidateValidToken(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	signed, err := svc.Sign(context.Background(), SignInput{
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Operation:  "upload",
		ExpiresIn:  1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	result, err := svc.Validate(
		context.Background(),
		signed.Token,
		"docs",
		"test.txt",
		"upload",
	)
	if err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}

	if result.BucketName != "docs" {
		t.Fatalf("expected docs, got %s", result.BucketName)
	}
	if result.ObjectKey != "test.txt" {
		t.Fatalf("expected test.txt, got %s", result.ObjectKey)
	}
	if result.Operation != "upload" {
		t.Fatalf("expected upload, got %s", result.Operation)
	}
}

func TestValidateExpiredToken(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	signed, err := svc.Sign(context.Background(), SignInput{
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Operation:  "upload",
		ExpiresIn:  -1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	_, err = svc.Validate(
		context.Background(),
		signed.Token,
		"docs",
		"test.txt",
		"upload",
	)
	if err != ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateWrongOperation(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	signed, err := svc.Sign(context.Background(), SignInput{
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Operation:  "upload",
		ExpiresIn:  1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	_, err = svc.Validate(
		context.Background(),
		signed.Token,
		"docs",
		"test.txt",
		"download",
	)
	if err != ErrTokenWrongOp {
		t.Fatalf("expected ErrTokenWrongOp, got %v", err)
	}
}

func TestValidateKeyMismatch(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	signed, err := svc.Sign(context.Background(), SignInput{
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Operation:  "upload",
		ExpiresIn:  1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	_, err = svc.Validate(
		context.Background(),
		signed.Token,
		"docs",
		"wrong.txt",
		"upload",
	)
	if err != ErrTokenKeyMismatch {
		t.Fatalf("expected ErrTokenKeyMismatch, got %v", err)
	}
}

func TestValidateBucketMismatch(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	signed, err := svc.Sign(context.Background(), SignInput{
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Operation:  "upload",
		ExpiresIn:  1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	_, err = svc.Validate(
		context.Background(),
		signed.Token,
		"other-bucket",
		"test.txt",
		"upload",
	)
	if err != ErrTokenKeyMismatch {
		t.Fatalf("expected ErrTokenKeyMismatch, got %v", err)
	}
}

func TestValidateTokenNotFound(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	_, err := svc.Validate(
		context.Background(),
		"oort_nonexistent",
		"docs",
		"test.txt",
		"upload",
	)
	if err != ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestConsumeToken(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresSignedURLRepository(db)
	svc := NewSignedURLService(repo)

	signed, err := svc.Sign(context.Background(), SignInput{
		BucketName: "docs",
		ObjectKey:  "test.txt",
		Operation:  "upload",
		ExpiresIn:  1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	result, err := svc.Validate(
		context.Background(),
		signed.Token,
		"docs",
		"test.txt",
		"upload",
	)
	if err != nil {
		t.Fatalf("expected valid token: %v", err)
	}

	err = svc.Consume(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("expected no error consuming: %v", err)
	}

	_, err = svc.Validate(
		context.Background(),
		signed.Token,
		"docs",
		"test.txt",
		"upload",
	)
	if err != ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound after consume, got %v", err)
	}
}
