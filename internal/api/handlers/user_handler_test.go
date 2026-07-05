package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func setupUserHandlerTest(t *testing.T) (*UserHandler, *services.UserService) {
	t.Helper()

	db := tests.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	tests.TruncateTables(t, db)

	userRepo := repositories.NewPostgresUserRepository(db)
	permRepo := repositories.NewPostgresPermissionRepository(db)
	userSvc := services.NewUserService(userRepo)

	return NewUserHandler(userSvc, permRepo), userSvc
}

func grantPermissionRequest(userID string, body string) *http.Request {
	req := httptest.NewRequest(
		http.MethodPost,
		"/users/"+userID+"/permissions",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", userID)
	return req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, rctx),
	)
}

func createTestUser(
	t *testing.T,
	userSvc *services.UserService,
	email string,
) string {
	t.Helper()

	user, err := userSvc.Create(
		context.Background(),
		email,
		"password",
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user.ID.String()
}

func TestUserHandler_CreateUserSuccess(t *testing.T) {
	handler, _ := setupUserHandlerTest(t)

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
	handler, _ := setupUserHandlerTest(t)

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
	handler, _ := setupUserHandlerTest(t)

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

func TestUserHandler_GrantPermissionSuccess(t *testing.T) {
	handler, userSvc := setupUserHandlerTest(t)

	perm := &models.Permission{
		ID:   uuid.New(),
		Name: "bucket:create",
	}
	err := repositories.NewPostgresPermissionRepository(
		tests.NewTestDB(t),
	).Create(context.Background(), perm)
	if err != nil {
		t.Fatalf("failed to seed permission: %v", err)
	}

	userID := createTestUser(t, userSvc, "target@test.com")

	rr := httptest.NewRecorder()
	handler.GrantPermission(
		rr,
		grantPermissionRequest(userID, `{"permission":"bucket:create"}`),
	)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	parsedID, _ := uuid.Parse(userID)
	perms, err := userSvc.GetPermissions(context.Background(), parsedID)
	if err != nil {
		t.Fatalf("failed to get permissions: %v", err)
	}

	if len(perms) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(perms))
	}

	if perms[0].Name != "bucket:create" {
		t.Fatalf("expected bucket:create, got %s", perms[0].Name)
	}
}

func TestUserHandler_GrantPermissionUserNotFound(t *testing.T) {
	handler, _ := setupUserHandlerTest(t)

	rr := httptest.NewRecorder()
	handler.GrantPermission(
		rr,
		grantPermissionRequest(uuid.New().String(), `{"permission":"bucket:create"}`),
	)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestUserHandler_GrantPermissionNotFound(t *testing.T) {
	handler, userSvc := setupUserHandlerTest(t)

	userID := createTestUser(t, userSvc, "target@test.com")

	rr := httptest.NewRecorder()
	handler.GrantPermission(
		rr,
		grantPermissionRequest(userID, `{"permission":"nonexistent-permission"}`),
	)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
