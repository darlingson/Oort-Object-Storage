package services

import (
	"context"
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func TestCreateBucketSuccess(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresBucketRepository(db)

	service := NewBucketService(repo)

	bucket, err := service.CreateBucket(
		context.Background(),
		"documents",
	)

	if err != nil {
		t.Fatalf("expected no error")
	}

	if bucket.Name != "documents" {
		t.Fatalf("unexpected name")
	}
}

func TestDuplicateBucketRejected(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresBucketRepository(db)

	service := NewBucketService(repo)

	_, _ = service.CreateBucket(
		context.Background(),
		"documents",
	)

	_, err := service.CreateBucket(
		context.Background(),
		"documents",
	)

	if err != ErrBucketExists {
		t.Fatalf(
			"expected ErrBucketExists got %v",
			err,
		)
	}
}