package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func createTestBucket(
	t *testing.T,
	repo *PostgresBucketRepository,
) *models.Bucket {

	t.Helper()

	bucket := &models.Bucket{
		ID:   uuid.New(),
		Name: "documents",
	}

	err := repo.Create(
		context.Background(),
		bucket,
	)

	if err != nil {
		t.Fatalf("failed creating bucket: %v", err)
	}

	return bucket
}

func createTestObject(
	bucketID uuid.UUID,
) *models.Object {

	return &models.Object{
		ID:          uuid.New(),
		BucketID:    bucketID,
		ObjectKey:   "contracts/file.pdf",
		StoragePath: "ab/cd/12345",
		ContentType: "application/pdf",
		SizeBytes:   1024,
		Checksum:    "abc123",
	}
}

func TestCreateObject(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	bucketRepo := NewPostgresBucketRepository(db)
	objectRepo := NewPostgresObjectRepository(db)

	bucket := createTestBucket(
		t,
		bucketRepo,
	)

	object := createTestObject(
		bucket.ID,
	)

	err := objectRepo.Create(
		context.Background(),
		object,
	)

	if err != nil {
		t.Fatalf(
			"expected no error got %v",
			err,
		)
	}
}

func TestFindObjectByKey(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	bucketRepo := NewPostgresBucketRepository(db)
	objectRepo := NewPostgresObjectRepository(db)

	bucket := createTestBucket(
		t,
		bucketRepo,
	)

	object := createTestObject(
		bucket.ID,
	)

	err := objectRepo.Create(
		context.Background(),
		object,
	)

	if err != nil {
		t.Fatal(err)
	}

	found, err := objectRepo.FindByKey(
		context.Background(),
		bucket.ID,
		"contracts/file.pdf",
	)

	if err != nil {
		t.Fatalf(
			"expected object got %v",
			err,
		)
	}

	if found.ObjectKey != object.ObjectKey {
		t.Fatalf(
			"expected %s got %s",
			object.ObjectKey,
			found.ObjectKey,
		)
	}

	if found.StoragePath != object.StoragePath {
		t.Fatalf(
			"expected %s got %s",
			object.StoragePath,
			found.StoragePath,
		)
	}
}

func TestListObjectsByBucket(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	bucketRepo := NewPostgresBucketRepository(db)
	objectRepo := NewPostgresObjectRepository(db)

	bucket := createTestBucket(
		t,
		bucketRepo,
	)

	for i := 0; i < 3; i++ {

		object := &models.Object{
			ID:          uuid.New(),
			BucketID:    bucket.ID,
			ObjectKey:   uuid.New().String(),
			StoragePath: uuid.New().String(),
			ContentType: "application/pdf",
			SizeBytes:   1024,
			Checksum:    uuid.New().String(),
		}

		err := objectRepo.Create(
			context.Background(),
			object,
		)

		if err != nil {
			t.Fatal(err)
		}
	}

	objects, err := objectRepo.ListByBucket(
		context.Background(),
		bucket.ID,
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(objects) != 3 {
		t.Fatalf(
			"expected 3 objects got %d",
			len(objects),
		)
	}
}

func TestDeleteObject(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	bucketRepo := NewPostgresBucketRepository(db)
	objectRepo := NewPostgresObjectRepository(db)

	bucket := createTestBucket(
		t,
		bucketRepo,
	)

	object := createTestObject(
		bucket.ID,
	)

	err := objectRepo.Create(
		context.Background(),
		object,
	)

	if err != nil {
		t.Fatal(err)
	}

	err = objectRepo.Delete(
		context.Background(),
		object.ID,
	)

	if err != nil {
		t.Fatal(err)
	}

	_, err = objectRepo.FindByKey(
		context.Background(),
		bucket.ID,
		object.ObjectKey,
	)

	if err == nil {
		t.Fatal(
			"expected object to be deleted",
		)
	}
}