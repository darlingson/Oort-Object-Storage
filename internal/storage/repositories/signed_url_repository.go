package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type SignedURLRepository interface {
	Create(ctx context.Context, signedURL *models.SignedURL) error
	FindByToken(ctx context.Context, token string) (*models.SignedURL, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
