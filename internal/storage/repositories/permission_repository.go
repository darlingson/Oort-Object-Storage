package repositories

import (
	"context"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type PermissionRepository interface {
	Create(ctx context.Context, permission *models.Permission) error
	FindByName(ctx context.Context, name string) (*models.Permission, error)
	List(ctx context.Context) ([]models.Permission, error)
}
