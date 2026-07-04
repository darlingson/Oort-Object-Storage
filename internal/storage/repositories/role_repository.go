package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	FindByName(ctx context.Context, name string) (*models.Role, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.Role, error)
	AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID) error
	ListPermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error)
	AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error
}
