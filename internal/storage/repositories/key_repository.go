package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type KeyRepository interface {
	Create(ctx context.Context, key *models.APIKey) error
	FindByHash(ctx context.Context, hash string) (*models.APIKey, error)
	List(ctx context.Context) ([]models.APIKey, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID, at time.Time) error
}
