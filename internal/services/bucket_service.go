package services

import (
	"context"
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

	existing, _ := s.repo.FindByName(ctx, name)

	if existing != nil {
		return nil, ErrBucketExists
	}

	bucket := &models.Bucket{
		ID:   uuid.New(),
		Name: name,
	}

	err := s.repo.Create(ctx, bucket)

	if err != nil {
		return nil, err
	}

	return bucket, nil
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