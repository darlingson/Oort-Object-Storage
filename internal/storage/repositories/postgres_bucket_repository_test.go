package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func TestCreateBucket(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresBucketRepository(db)

	bucket := &models.Bucket{
		ID:   uuid.New(),
		Name: "documents",
	}

	err := repo.Create(context.Background(), bucket)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestFindBucketByName(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresBucketRepository(db)

	bucket := &models.Bucket{
		ID:   uuid.New(),
		Name: "documents",
	}

	_ = repo.Create(context.Background(), bucket)

	found, err := repo.FindByName(
		context.Background(),
		"documents",
	)

	if err != nil {
		t.Fatalf("expected bucket")
	}

	if found.Name != "documents" {
		t.Fatalf("unexpected bucket name")
	}
}

func TestListBuckets(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresBucketRepository(db)

	_ = repo.Create(
		context.Background(),
		&models.Bucket{
			ID: uuid.New(),
			Name: "bucket1",
		},
	)

	_ = repo.Create(
		context.Background(),
		&models.Bucket{
			ID: uuid.New(),
			Name: "bucket2",
		},
	)

	buckets, err := repo.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error")
	}

	if len(buckets) != 2 {
		t.Fatalf(
			"expected 2 buckets got %d",
			len(buckets),
		)
	}
}