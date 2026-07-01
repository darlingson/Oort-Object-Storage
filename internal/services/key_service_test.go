package services

import (
	"context"
	"testing"
	"time"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func TestCreateKeySuccess(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	service := NewKeyService(repo)

	key, rawKey, err := service.Create(
		context.Background(),
		"kyc-service",
		[]string{"kyc-uploads"},
		nil,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if key.Name != "kyc-service" {
		t.Fatalf(
			"expected kyc-service got %s",
			key.Name,
		)
	}

	if len(rawKey) == 0 {
		t.Fatalf("expected raw key")
	}

	if len(key.KeyHash) == 0 {
		t.Fatalf("expected hash to be set")
	}
}

func TestCreateKeyWithExpiry(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	service := NewKeyService(repo)

	expiry := time.Now().Add(24 * time.Hour)

	key, _, err := service.Create(
		context.Background(),
		"temp-key",
		[]string{"temp"},
		&expiry,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if key.ExpiresAt == nil {
		t.Fatalf("expected expiry to be set")
	}
}

func TestValidateKeySuccess(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	service := NewKeyService(repo)

	_, rawKey, _ := service.Create(
		context.Background(),
		"kyc-service",
		[]string{"kyc-uploads"},
		nil,
	)

	found, err := service.Validate(
		context.Background(),
		rawKey,
	)

	if err != nil {
		t.Fatalf(
			"expected valid key got %v",
			err,
		)
	}

	if found.Name != "kyc-service" {
		t.Fatalf(
			"expected kyc-service got %s",
			found.Name,
		)
	}
}

func TestValidateExpiredKey(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	service := NewKeyService(repo)

	expiry := time.Now().Add(-1 * time.Hour)

	_, rawKey, _ := service.Create(
		context.Background(),
		"expired-key",
		[]string{"temp"},
		&expiry,
	)

	_, err := service.Validate(
		context.Background(),
		rawKey,
	)

	if err != ErrKeyExpired {
		t.Fatalf(
			"expected ErrKeyExpired got %v",
			err,
		)
	}
}

func TestValidateInvalidKey(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	service := NewKeyService(repo)

	_, err := service.Validate(
		context.Background(),
		"oort_invalid_key_that_does_not_exist",
	)

	if err != ErrKeyNotFound {
		t.Fatalf(
			"expected ErrKeyNotFound got %v",
			err,
		)
	}
}

func TestCanAccessBucket(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	service := NewKeyService(repo)

	key, _, _ := service.Create(
		context.Background(),
		"kyc-service",
		[]string{"kyc-uploads"},
		nil,
	)

	if !service.CanAccessBucket(key, "kyc-uploads") {
		t.Fatalf(
			"expected access to kyc-uploads",
		)
	}

	if service.CanAccessBucket(key, "contracts") {
		t.Fatalf(
			"expected no access to contracts",
		)
	}
}

func TestWildcardAccess(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	service := NewKeyService(repo)

	key, _, _ := service.Create(
		context.Background(),
		"admin",
		[]string{"*"},
		nil,
	)

	if !service.CanAccessBucket(key, "anything") {
		t.Fatalf(
			"expected wildcard to allow any bucket",
		)
	}
}

func TestDeleteKey(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	service := NewKeyService(repo)

	key, rawKey, _ := service.Create(
		context.Background(),
		"to-delete",
		[]string{"temp"},
		nil,
	)

	err := service.Delete(
		context.Background(),
		key.ID,
	)

	if err != nil {
		t.Fatalf(
			"expected no error got %v",
			err,
		)
	}

	_, err = service.Validate(
		context.Background(),
		rawKey,
	)

	if err != ErrKeyNotFound {
		t.Fatalf(
			"expected ErrKeyNotFound after delete",
		)
	}
}
