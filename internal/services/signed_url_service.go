package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
)

var (
	ErrTokenNotFound    = errors.New("token not found")
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenWrongOp     = errors.New("token operation mismatch")
	ErrTokenKeyMismatch = errors.New("token bucket/key mismatch")
)

type SignedURLService struct {
	repo repositories.SignedURLRepository
}

func NewSignedURLService(
	repo repositories.SignedURLRepository,
) *SignedURLService {

	return &SignedURLService{
		repo: repo,
	}
}

type SignInput struct {
	BucketName string
	ObjectKey  string
	Operation  string
	ExpiresIn  time.Duration
}

type SignResult struct {
	Token     string
	ExpiresAt time.Time
}

func (s *SignedURLService) Sign(
	ctx context.Context,
	input SignInput,
) (*SignResult, error) {

	rawBytes := make([]byte, 32)
	_, err := rand.Read(rawBytes)
	if err != nil {
		return nil, err
	}

	token := "oort_" + base64.RawURLEncoding.EncodeToString(rawBytes)
	expiresAt := time.Now().UTC().Add(input.ExpiresIn)

	signedURL := &models.SignedURL{
		ID:         uuid.New(),
		BucketName: input.BucketName,
		ObjectKey:  input.ObjectKey,
		Token:      token,
		Operation:  input.Operation,
		ExpiresAt:  expiresAt,
	}

	err = s.repo.Create(ctx, signedURL)
	if err != nil {
		return nil, err
	}

	return &SignResult{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

type ConsumeResult struct {
	ID         uuid.UUID
	BucketName string
	ObjectKey  string
	Operation  string
}

func (s *SignedURLService) Validate(
	ctx context.Context,
	token string,
	bucketName string,
	objectKey string,
	operation string,
) (*ConsumeResult, error) {

	su, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, ErrTokenNotFound
	}

	if time.Now().UTC().After(su.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	if su.Operation != operation {
		return nil, ErrTokenWrongOp
	}

	if su.BucketName != bucketName || su.ObjectKey != objectKey {
		return nil, ErrTokenKeyMismatch
	}

	return &ConsumeResult{
		ID:         su.ID,
		BucketName: su.BucketName,
		ObjectKey:  su.ObjectKey,
		Operation:  su.Operation,
	}, nil
}

func (s *SignedURLService) Consume(
	ctx context.Context,
	id uuid.UUID,
) error {

	return s.repo.Delete(ctx, id)
}
