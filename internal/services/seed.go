package services

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
)

type SeedService struct {
	permRepo repositories.PermissionRepository
	roleRepo repositories.RoleRepository
	userRepo repositories.UserRepository
}

func NewSeedService(
	permRepo repositories.PermissionRepository,
	roleRepo repositories.RoleRepository,
	userRepo repositories.UserRepository,
) *SeedService {

	return &SeedService{
		permRepo: permRepo,
		roleRepo: roleRepo,
		userRepo: userRepo,
	}
}

func (s *SeedService) SeedIfNeeded(ctx context.Context) {
	s.seedPermissions(ctx)

	adminExists := true
	_, err := s.userRepo.FindByEmail(ctx, config.Get().AdminEmail)
	if err != nil {
		adminExists = false
	}

	if !adminExists {
		log.Println("seeding initial data")
		s.seedAdminRole(ctx)
	}
}

func (s *SeedService) seedPermissions(ctx context.Context) {

	for _, name := range models.AllPermissionNames {

		existing, err := s.permRepo.FindByName(ctx, name)
		if err == nil && existing != nil {
			continue
		}

		perm := &models.Permission{
			ID:   uuid.New(),
			Name: name,
		}

		err = s.permRepo.Create(ctx, perm)
		if err != nil {
			log.Printf("failed to seed permission %s: %v", name, err)
			continue
		}

		log.Printf("seeded permission: %s", name)
	}
}

func (s *SeedService) seedAdminRole(ctx context.Context) {

	role := &models.Role{
		ID:   uuid.New(),
		Name: "admin",
	}

	err := s.roleRepo.Create(ctx, role)
	if err != nil {
		log.Printf("failed to create admin role: %v", err)
		return
	}

	permissions, err := s.permRepo.List(ctx)
	if err != nil {
		log.Printf("failed to list permissions: %v", err)
		return
	}

	for _, perm := range permissions {
		err = s.roleRepo.AssignPermission(ctx, role.ID, perm.ID)
		if err != nil {
			log.Printf(
				"failed to assign %s to admin role: %v",
				perm.Name,
				err,
			)
		}
	}

	cfg := config.Get()

	userSvc := NewUserService(s.userRepo)

	user, err := userSvc.Create(
		ctx,
		cfg.AdminEmail,
		cfg.AdminPassword,
	)
	if err != nil {
		log.Printf("failed to create admin user: %v", err)
		return
	}

	err = s.roleRepo.AssignToUser(ctx, user.ID, role.ID)
	if err != nil {
		log.Printf("failed to assign admin role to user: %v", err)
		return
	}

	log.Printf("seeded admin user: %s", cfg.AdminEmail)
}
