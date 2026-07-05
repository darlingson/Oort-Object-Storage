package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func setupAuthHandlerTest(t *testing.T) *AuthHandler {
	t.Helper()

	db := tests.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	tests.TruncateTables(t, db)

	userRepo := repositories.NewPostgresUserRepository(db)
	userSvc := services.NewUserService(userRepo)

	_, err := userSvc.Create(
		context.Background(),
		"admin@test.com",
		"admin-pass",
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Setenv("JWT_SECRET", "test-secret-auth-handler")
	config.Load()

	jwtSvc := services.NewJWTService()

	return NewAuthHandler(userSvc, jwtSvc)
}

func TestAuthHandler_LoginSuccess(t *testing.T) {
	handler := setupAuthHandlerTest(t)

	body := `{"email":"admin@test.com","password":"admin-pass"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp LoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}

	if resp.User.Email != "admin@test.com" {
		t.Fatalf("expected admin@test.com, got %s", resp.User.Email)
	}
}

func TestAuthHandler_LoginWrongPassword(t *testing.T) {
	handler := setupAuthHandlerTest(t)

	body := `{"email":"admin@test.com","password":"wrong-pass"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthHandler_LoginInvalidBody(t *testing.T) {
	handler := setupAuthHandlerTest(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader("not-json"),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
