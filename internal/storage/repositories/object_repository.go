package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type ObjectRepository interface {
	Create(
		ctx context.Context,
		object *models.Object,
	) error
	Upsert(
		ctx context.Context,
		object *models.Object,
	) error
	FindByKey(
		ctx context.Context,
		bucketID uuid.UUID,
		key string,
	) (*models.Object, error)

	Delete(
		ctx context.Context,
		id uuid.UUID,
	) error

	DeleteByKey(
		ctx context.Context,
		bucketID uuid.UUID,
		key string,
	) error

	ListByBucket(
		ctx context.Context,
		bucketID uuid.UUID,
	) ([]models.Object, error)
}