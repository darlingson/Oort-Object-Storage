package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type UserService struct {
	userRepo repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) Create(
	ctx context.Context,
	email string,
	password string,
) (*models.User, error) {

	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return nil, errors.New("email is required")
	}

	if password == "" {
		return nil, errors.New("password is required")
	}

	existing, _ := s.userRepo.FindByEmail(ctx, email)
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(
	ctx context.Context,
	email string,
	password string,
) (*models.User, error) {

	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, ErrUserNotFound
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)
	if err != nil {
		return nil, ErrInvalidPassword
	}

	return user, nil
}

func (s *UserService) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.User, error) {

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *UserService) GetPermissions(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Permission, error) {

	return s.userRepo.GetEffectivePermissions(ctx, userID)
}

func (s *UserService) AssignPermission(
	ctx context.Context,
	userID uuid.UUID,
	permissionID uuid.UUID,
) error {

	return s.userRepo.AssignPermission(ctx, userID, permissionID)
}
