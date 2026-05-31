package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
)

var ErrBucketExists = errors.New("bucket already exists")

type BucketService struct {
	repo repositories.BucketRepository
}

func NewBucketService(
	repo repositories.BucketRepository,
) *BucketService {

	return &BucketService{
		repo: repo,
	}
}

func (s *BucketService) CreateBucket(
	ctx context.Context,
	name string,
) (*models.Bucket, error) {

	bucket, err := s.repo.FindByName(ctx, name)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if bucket != nil {
		return nil, ErrBucketExists
	}

	newBucket := &models.Bucket{
		ID:   uuid.New(),
		Name: name,
	}

	err = s.repo.Create(ctx, newBucket)

	if err != nil {
		return nil, err
	}

	return newBucket, nil
}

func (s *BucketService) GetBucket(
	ctx context.Context,
	name string,
) (*models.Bucket, error) {

	return s.repo.FindByName(ctx, name)
}

func (s *BucketService) ListBuckets(
	ctx context.Context,
) ([]models.Bucket, error) {

	return s.repo.List(ctx)
}