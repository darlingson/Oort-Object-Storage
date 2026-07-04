package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/google/uuid"
)

func setupAuthTest(t *testing.T) *services.JWTService {
	t.Helper()

	t.Setenv("JWT_SECRET", "test-secret-for-middleware")
	config.Load()

	return services.NewJWTService()
}

func validJWT(t *testing.T, svc *services.JWTService) string {
	t.Helper()

	token, err := svc.Create(
		&models.User{
			ID:    uuid.New(),
			Email: "test@test.com",
		},
		[]string{"bucket:create"},
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	return token
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	svc := setupAuthTest(t)
	middleware := Auth(svc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_EmptyHeader(t *testing.T) {
	svc := setupAuthTest(t)
	middleware := Auth(svc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "")

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_WrongFormat(t *testing.T) {
	svc := setupAuthTest(t)
	middleware := Auth(svc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "not-bearer-token-format")

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_WrongScheme(t *testing.T) {
	svc := setupAuthTest(t)
	middleware := Auth(svc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0")

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	svc := setupAuthTest(t)
	middleware := Auth(svc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	svc := setupAuthTest(t)
	middleware := Auth(svc)

	token := validJWT(t, svc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				t.Error("expected user ID in context")
			}

			perms := GetPermissions(r.Context())
			if len(perms) == 0 {
				t.Error("expected permissions in context")
			}

			w.WriteHeader(http.StatusOK)
		},
	)

	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
