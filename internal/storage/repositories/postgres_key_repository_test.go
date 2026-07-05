package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func createTestKey(name string, buckets []string) *models.APIKey {

	now := time.Now()

	return &models.APIKey{
		ID:        uuid.New(),
		Name:      name,
		KeyHash:   uuid.New().String(),
		Buckets:   buckets,
		ExpiresAt: &now,
	}
}

func TestCreateKey(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresKeyRepository(db)

	key := createTestKey(
		"kyc-service",
		[]string{"kyc-uploads"},
	)

	err := repo.Create(
		context.Background(),
		key,
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}
}

func TestFindKeyByHash(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresKeyRepository(db)

	key := createTestKey(
		"kyc-service",
		[]string{"kyc-uploads"},
	)

	_ = repo.Create(
		context.Background(),
		key,
	)

	found, err := repo.FindByHash(
		context.Background(),
		key.KeyHash,
	)

	if err != nil {
		t.Fatalf("expected key")
	}

	if found.Name != "kyc-service" {
		t.Fatalf(
			"expected kyc-service got %s",
			found.Name,
		)
	}

	if len(found.Buckets) != 1 ||
		found.Buckets[0] != "kyc-uploads" {
		t.Fatalf("unexpected buckets")
	}
}

func TestListKeys(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresKeyRepository(db)

	_ = repo.Create(
		context.Background(),
		createTestKey(
			"key-one",
			[]string{"bucket-a"},
		),
	)

	_ = repo.Create(
		context.Background(),
		createTestKey(
			"key-two",
			[]string{"bucket-b", "bucket-c"},
		),
	)

	keys, err := repo.List(context.Background())

	if err != nil {
		t.Fatalf("expected no error")
	}

	if len(keys) != 2 {
		t.Fatalf(
			"expected 2 keys got %d",
			len(keys),
		)
	}
}

func TestWildcardKey(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresKeyRepository(db)

	key := createTestKey(
		"admin",
		[]string{"*"},
	)

	err := repo.Create(
		context.Background(),
		key,
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	found, err := repo.FindByHash(
		context.Background(),
		key.KeyHash,
	)

	if err != nil {
		t.Fatalf("expected key")
	}

	if len(found.Buckets) != 1 ||
		found.Buckets[0] != "*" {
		t.Fatalf("expected wildcard bucket")
	}
}

func TestDeleteKey(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := NewPostgresKeyRepository(db)

	key := createTestKey(
		"to-delete",
		[]string{"temp"},
	)

	_ = repo.Create(context.Background(), key)

	err := repo.Delete(
		context.Background(),
		key.ID,
	)

	if err != nil {
		t.Fatalf(
			"expected no error got %v",
			err,
		)
	}

	_, err = repo.FindByHash(
		context.Background(),
		key.KeyHash,
	)

	if err == nil {
		t.Fatal(
			"expected key to be deleted",
		)
	}
}
