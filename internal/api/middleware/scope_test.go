package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
	"github.com/go-chi/chi/v5"
)

func passHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func setupScopeTest(t *testing.T) (*services.KeyService, string, string) {
	t.Helper()

	db := tests.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	svc := services.NewKeyService(repo)

	_, rawKey, err := svc.Create(
		context.Background(),
		"kyc-service",
		[]string{"kyc-uploads"},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	_, adminRawKey, err := svc.Create(
		context.Background(),
		"admin",
		[]string{"*"},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create admin key: %v", err)
	}

	return svc, rawKey, adminRawKey
}

func requestWithBucket(method, bucket string) *http.Request {
	req := httptest.NewRequest(method, "/buckets/"+bucket+"/objects/test.txt", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucket)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestBucketScope_MissingKey(t *testing.T) {
	svc, _, _ := setupScopeTest(t)
	middleware := BucketScope(svc)

	rr := httptest.NewRecorder()
	req := requestWithBucket(http.MethodGet, "kyc-uploads")

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestBucketScope_InvalidKey(t *testing.T) {
	svc, _, _ := setupScopeTest(t)
	middleware := BucketScope(svc)

	rr := httptest.NewRecorder()
	req := requestWithBucket(http.MethodGet, "kyc-uploads")
	req.Header.Set("X-API-Key", "oort_invalid_key_that_does_not_exist")

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestBucketScope_ExpiredKey(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()
	tests.TruncateTables(t, db)

	repo := repositories.NewPostgresKeyRepository(db)
	svc := services.NewKeyService(repo)

	expiry := time.Now().Add(-1 * time.Hour)
	_, rawKey, err := svc.Create(
		context.Background(),
		"expired-key",
		[]string{"kyc-uploads"},
		&expiry,
	)
	if err != nil {
		t.Fatalf("failed to create expired key: %v", err)
	}

	middleware := BucketScope(svc)

	rr := httptest.NewRecorder()
	req := requestWithBucket(http.MethodGet, "kyc-uploads")
	req.Header.Set("X-API-Key", rawKey)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for expired key, got %d", rr.Code)
	}
}

func TestBucketScope_WrongBucket(t *testing.T) {
	svc, rawKey, _ := setupScopeTest(t)
	middleware := BucketScope(svc)

	rr := httptest.NewRecorder()
	req := requestWithBucket(http.MethodGet, "contracts")
	req.Header.Set("X-API-Key", rawKey)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for wrong bucket, got %d", rr.Code)
	}
}

func TestBucketScope_ValidKeyCorrectBucket(t *testing.T) {
	svc, rawKey, _ := setupScopeTest(t)
	middleware := BucketScope(svc)

	rr := httptest.NewRecorder()
	req := requestWithBucket(http.MethodGet, "kyc-uploads")
	req.Header.Set("X-API-Key", rawKey)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
}

func TestBucketScope_WildcardKey(t *testing.T) {
	svc, _, adminRawKey := setupScopeTest(t)
	middleware := BucketScope(svc)

	rr := httptest.NewRecorder()
	req := requestWithBucket(http.MethodGet, "any-bucket-whatsoever")
	req.Header.Set("X-API-Key", adminRawKey)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for wildcard key, got %d", rr.Code)
	}
}

func TestBucketScope_MissingBucketParam(t *testing.T) {
	svc, rawKey, _ := setupScopeTest(t)
	middleware := BucketScope(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", rawKey)

	rr := httptest.NewRecorder()
	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400 Bad Request for missing bucket, got %d: %s",
			rr.Code,
			rr.Body.String(),
		)
	}
}
