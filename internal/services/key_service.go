package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
)

var (
	ErrKeyNotFound = errors.New("api key not found")
	ErrKeyExpired  = errors.New("api key has expired")
)

type KeyService struct {
	repo repositories.KeyRepository
}

func NewKeyService(
	repo repositories.KeyRepository,
) *KeyService {

	return &KeyService{
		repo: repo,
	}
}

func (s *KeyService) Create(
	ctx context.Context,
	name string,
	buckets []string,
	expiresAt *time.Time,
) (*models.APIKey, string, error) {

	rawBytes := make([]byte, 32)

	_, err := rand.Read(rawBytes)

	if err != nil {
		return nil, "", err
	}

	rawKey := "oort_" + base64.RawURLEncoding.EncodeToString(rawBytes)

	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	var dbExpiresAt *time.Time
	if expiresAt != nil {
		u := expiresAt.UTC()
		dbExpiresAt = &u
	}

	key := &models.APIKey{
		ID:        uuid.New(),
		Name:      name,
		KeyHash:   keyHash,
		Buckets:   buckets,
		ExpiresAt: dbExpiresAt,
	}

	err = s.repo.Create(ctx, key)

	if err != nil {
		return nil, "", err
	}

	return key, rawKey, nil
}

func (s *KeyService) Validate(
	ctx context.Context,
	rawKey string,
) (*models.APIKey, error) {

	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	key, err := s.repo.FindByHash(ctx, keyHash)

	if err != nil {
		return nil, ErrKeyNotFound
	}

	if key.ExpiresAt != nil &&
		time.Now().After(*key.ExpiresAt) {

		return nil, ErrKeyExpired
	}

	return key, nil
}

func (s *KeyService) CanAccessBucket(
	key *models.APIKey,
	bucketName string,
) bool {

	for _, allowed := range key.Buckets {

		if allowed == "*" || allowed == bucketName {
			return true
		}
	}

	return false
}

func (s *KeyService) List(
	ctx context.Context,
) ([]models.APIKey, error) {

	return s.repo.List(ctx)
}

func (s *KeyService) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {

	return s.repo.Delete(ctx, id)
}
