package services

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func TestUserService_Create(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresUserRepository(db)
	svc := NewUserService(repo)

	user, err := svc.Create(
		context.Background(),
		"alice@test.com",
		"secure-pass-123",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID == uuid.Nil {
		t.Fatal("expected non-nil user ID")
	}

	if user.Email != "alice@test.com" {
		t.Fatalf("expected alice@test.com, got %s", user.Email)
	}

	if user.PasswordHash == "" {
		t.Fatal("expected password hash to be set")
	}
}

func TestUserService_CreateDuplicateEmail(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresUserRepository(db)
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"dup@test.com",
		"password1",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = svc.Create(
		context.Background(),
		"dup@test.com",
		"password2",
	)
	if err != ErrEmailAlreadyExists {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestUserService_CreateEmptyEmail(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	repo := repositories.NewPostgresUserRepository(db)
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"",
		"password1",
	)
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestUserService_CreateEmptyPassword(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	repo := repositories.NewPostgresUserRepository(db)
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"test@test.com",
		"",
	)
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestUserService_LoginSuccess(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresUserRepository(db)
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"bob@test.com",
		"correct-password",
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user, err := svc.Login(
		context.Background(),
		"bob@test.com",
		"correct-password",
	)
	if err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}

	if user.Email != "bob@test.com" {
		t.Fatalf("expected bob@test.com, got %s", user.Email)
	}
}

func TestUserService_LoginWrongPassword(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresUserRepository(db)
	svc := NewUserService(repo)

	_, err := svc.Create(
		context.Background(),
		"bob@test.com",
		"correct-password",
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = svc.Login(
		context.Background(),
		"bob@test.com",
		"wrong-password",
	)
	if err != ErrInvalidPassword {
		t.Fatalf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestUserService_LoginUserNotFound(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresUserRepository(db)
	svc := NewUserService(repo)

	_, err := svc.Login(
		context.Background(),
		"nobody@test.com",
		"any-password",
	)
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_GetPermissions(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	userRepo := repositories.NewPostgresUserRepository(db)
	permRepo := repositories.NewPostgresPermissionRepository(db)

	svc := NewUserService(userRepo)

	user, err := svc.Create(
		context.Background(),
		"perm-test@test.com",
		"password",
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	perm := &models.Permission{
		ID:   uuid.New(),
		Name: "bucket:create",
	}

	err = permRepo.Create(context.Background(), perm)
	if err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	err = userRepo.AssignPermission(
		context.Background(),
		user.ID,
		perm.ID,
	)
	if err != nil {
		t.Fatalf("failed to assign permission: %v", err)
	}

	perms, err := svc.GetPermissions(
		context.Background(),
		user.ID,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(perms) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(perms))
	}

	if perms[0].Name != "bucket:create" {
		t.Fatalf(
			"expected bucket:create, got %s",
			perms[0].Name,
		)
	}
}
