package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
	"github.com/google/uuid"
)

func setupObjectAccessTest(t *testing.T) (*services.KeyService, *services.JWTService, string, string) {
	t.Helper()

	db := tests.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	tests.TruncateTables(t, db)

	keyRepo := repositories.NewPostgresKeyRepository(db)
	keySvc := services.NewKeyService(keyRepo)

	_, rawKey, err := keySvc.Create(
		context.Background(),
		"test-service",
		[]string{"test-bucket"},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	_, adminRawKey, err := keySvc.Create(
		context.Background(),
		"admin-service",
		[]string{"*"},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create admin key: %v", err)
	}

	t.Setenv("JWT_SECRET", "test-secret-for-objectaccess")
	config.Load()

	jwtSvc := services.NewJWTService()

	return keySvc, jwtSvc, rawKey, adminRawKey
}

func objectAccessReq(method, bucket, key string) *http.Request {
	req := httptest.NewRequest(method, "/buckets/"+bucket+"/objects/test.txt", nil)

	if key != "" {
		req.Header.Set("X-API-Key", key)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucket)
	rctx.URLParams.Add("*", "test.txt")

	return req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, rctx),
	)
}

func objectAccessBearerReq(method, bucket, token string) *http.Request {
	req := httptest.NewRequest(method, "/buckets/"+bucket+"/objects/test.txt", nil)

	req.Header.Set("Authorization", "Bearer "+token)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucket)
	rctx.URLParams.Add("*", "test.txt")

	return req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, rctx),
	)
}

func TestObjectAccess_NoCredentials(t *testing.T) {
	keySvc, jwtSvc, _, _ := setupObjectAccessTest(t)
	middleware := ObjectAccess(keySvc, jwtSvc)

	rr := httptest.NewRecorder()
	req := objectAccessReq(http.MethodGet, "test-bucket", "")

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestObjectAccess_APIKey_ValidBucket(t *testing.T) {
	keySvc, jwtSvc, rawKey, _ := setupObjectAccessTest(t)
	middleware := ObjectAccess(keySvc, jwtSvc)

	rr := httptest.NewRecorder()
	req := objectAccessReq(http.MethodPut, "test-bucket", rawKey)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestObjectAccess_APIKey_WrongBucket(t *testing.T) {
	keySvc, jwtSvc, rawKey, _ := setupObjectAccessTest(t)
	middleware := ObjectAccess(keySvc, jwtSvc)

	rr := httptest.NewRecorder()
	req := objectAccessReq(http.MethodPut, "other-bucket", rawKey)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestObjectAccess_APIKey_InvalidKey(t *testing.T) {
	keySvc, jwtSvc, _, _ := setupObjectAccessTest(t)
	middleware := ObjectAccess(keySvc, jwtSvc)

	rr := httptest.NewRecorder()
	req := objectAccessReq(
		http.MethodPut, "test-bucket",
		"oort_invalid_key_that_does_not_exist",
	)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestObjectAccess_APIKey_Wildcard(t *testing.T) {
	keySvc, jwtSvc, _, adminRawKey := setupObjectAccessTest(t)
	middleware := ObjectAccess(keySvc, jwtSvc)

	rr := httptest.NewRecorder()
	req := objectAccessReq(http.MethodGet, "any-bucket", adminRawKey)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestObjectAccess_JWT_ValidPermission(t *testing.T) {
	_, jwtSvc, _, _ := setupObjectAccessTest(t)
	middleware := ObjectAccess(services.NewKeyService(nil), jwtSvc)

	token, err := jwtSvc.Create(
		&models.User{
			ID:    uuid.New(),
			Email: "user@test.com",
		},
		[]string{"object:upload"},
	)
	if err != nil {
		t.Fatalf("failed to create JWT: %v", err)
	}

	rr := httptest.NewRecorder()
	req := objectAccessBearerReq(http.MethodPut, "test-bucket", token)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestObjectAccess_JWT_MissingPermission(t *testing.T) {
	_, jwtSvc, _, _ := setupObjectAccessTest(t)
	middleware := ObjectAccess(services.NewKeyService(nil), jwtSvc)

	token, err := jwtSvc.Create(
		&models.User{
			ID:    uuid.New(),
			Email: "user@test.com",
		},
		[]string{"bucket:create"},
	)
	if err != nil {
		t.Fatalf("failed to create JWT: %v", err)
	}

	rr := httptest.NewRecorder()
	req := objectAccessBearerReq(http.MethodPut, "test-bucket", token)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestObjectAccess_JWT_InvalidToken(t *testing.T) {
	_, jwtSvc, _, _ := setupObjectAccessTest(t)
	middleware := ObjectAccess(services.NewKeyService(nil), jwtSvc)

	rr := httptest.NewRecorder()
	req := objectAccessBearerReq(
		http.MethodGet, "test-bucket",
		"invalid-jwt-token",
	)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestObjectAccess_MissingBucketParam(t *testing.T) {
	keySvc, jwtSvc, rawKey, _ := setupObjectAccessTest(t)
	middleware := ObjectAccess(keySvc, jwtSvc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", rawKey)

	rr := httptest.NewRecorder()
	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
