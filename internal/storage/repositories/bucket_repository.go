package repositories

import (
	"context"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type BucketRepository interface {
	Create(ctx context.Context, bucket *models.Bucket) error
	FindByName(ctx context.Context, name string) (*models.Bucket, error)
	List(ctx context.Context) ([]models.Bucket, error)
}