package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequirePermission_HasPermission(t *testing.T) {
	middleware := RequirePermission("bucket:create")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := context.WithValue(
		req.Context(),
		PermissionsContextKey,
		[]string{"bucket:create", "object:upload"},
	)
	req = req.WithContext(ctx)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestRequirePermission_MissingPermission(t *testing.T) {
	middleware := RequirePermission("user:create")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := context.WithValue(
		req.Context(),
		PermissionsContextKey,
		[]string{"bucket:create", "object:upload"},
	)
	req = req.WithContext(ctx)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestRequirePermission_EmptyPermissions(t *testing.T) {
	middleware := RequirePermission("bucket:create")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := context.WithValue(
		req.Context(),
		PermissionsContextKey,
		[]string{},
	)
	req = req.WithContext(ctx)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestRequirePermission_NoContext(t *testing.T) {
	middleware := RequirePermission("bucket:create")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	middleware(passHandler()).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
