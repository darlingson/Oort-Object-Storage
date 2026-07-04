package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

func setupJWTTest(t *testing.T) *JWTService {
	t.Helper()

	t.Setenv("JWT_SECRET", "test-secret-for-jwt-testing")
	config.Load()

	return NewJWTService()
}

func TestJWTService_CreateAndValidate(t *testing.T) {
	svc := setupJWTTest(t)

	user := &models.User{
		ID:    uuid.New(),
		Email: "alice@test.com",
	}

	permissions := []string{"bucket:create", "object:upload"}

	token, err := svc.Create(user, permissions)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if claims.UserID != user.ID.String() {
		t.Fatalf("expected user id %s, got %s", user.ID.String(), claims.UserID)
	}

	if claims.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, claims.Email)
	}

	if len(claims.Permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(claims.Permissions))
	}
}

func TestJWTService_ValidateInvalidToken(t *testing.T) {
	svc := setupJWTTest(t)

	_, err := svc.Validate("this-is-not-a-valid-jwt")
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTService_ValidateWrongSecret(t *testing.T) {
	svc := setupJWTTest(t)

	user := &models.User{
		ID:    uuid.New(),
		Email: "bob@test.com",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		UserID:      user.ID.String(),
		Email:       user.Email,
		Permissions: []string{"bucket:create"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})

	signed, _ := token.SignedString([]byte("different-secret"))

	_, err := svc.Validate(signed)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for wrong secret, got %v", err)
	}
}

func TestJWTService_ValidateExpiredToken(t *testing.T) {
	svc := setupJWTTest(t)

	claims := &Claims{
		UserID:      uuid.New().String(),
		Email:       "expired@test.com",
		Permissions: []string{"bucket:create"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret-for-jwt-testing"))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = svc.Validate(tokenStr)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for expired token, got %v", err)
	}
}
