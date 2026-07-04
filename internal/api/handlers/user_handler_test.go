package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func setupUserHandlerTest(t *testing.T) *UserHandler {
	t.Helper()

	db := tests.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	tests.TruncateTables(t, db)

	userRepo := repositories.NewPostgresUserRepository(db)
	userSvc := services.NewUserService(userRepo)

	return NewUserHandler(userSvc)
}

func TestUserHandler_CreateUserSuccess(t *testing.T) {
	handler := setupUserHandlerTest(t)

	body := `{"email":"newuser@test.com","password":"secure-pass"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.CreateUser(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp UserResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Fatal("expected non-empty user ID")
	}

	if resp.Email != "newuser@test.com" {
		t.Fatalf("expected newuser@test.com, got %s", resp.Email)
	}
}

func TestUserHandler_CreateUserDuplicate(t *testing.T) {
	handler := setupUserHandlerTest(t)

	body := `{"email":"dup@test.com","password":"password"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.CreateUser(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on first create, got %d", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(body),
	)
	req2.Header.Set("Content-Type", "application/json")

	handler.CreateUser(rr2, req2)

	if rr2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestUserHandler_CreateUserInvalidBody(t *testing.T) {
	handler := setupUserHandlerTest(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader("not-json"),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.CreateUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
