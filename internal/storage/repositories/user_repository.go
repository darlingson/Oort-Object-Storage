package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	AssignPermission(ctx context.Context, userID, permissionID uuid.UUID) error
	GetEffectivePermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error)
}
