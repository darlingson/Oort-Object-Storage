package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

func TestHealthCheck(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	HealthCheck(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp["status"] != "running" {
		t.Fatalf("expected running, got %s", resp["status"])
	}
}

func TestLiveEndpoint(t *testing.T) {
	handler := NewHealthHandler(nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	handler.Live(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp["status"] != "ok" {
		t.Fatalf("expected ok, got %s", resp["status"])
	}
}

func TestReadyEndpoint_Healthy(t *testing.T) {
	storageDir := t.TempDir()

	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=masterpassword dbname=oort_objects_test sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	healthSvc := services.NewHealthService(db, storageDir)
	handler := NewHealthHandler(healthSvc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.Ready(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp readinessStatus
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("expected ok, got %s", resp.Status)
	}

	if resp.Database != "ok" {
		t.Fatalf("expected ok, got %s", resp.Database)
	}

	if resp.Storage != "ok" {
		t.Fatalf("expected ok, got %s", resp.Storage)
	}
}

func TestReadyEndpoint_StorageFailure(t *testing.T) {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=masterpassword dbname=oort_objects_test sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	healthSvc := services.NewHealthService(db, "/nonexistent-directory-12345")
	handler := NewHealthHandler(healthSvc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.Ready(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}

	var resp readinessStatus
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Status != "degraded" {
		t.Fatalf("expected degraded, got %s", resp.Status)
	}

	if resp.Storage != "unhealthy" {
		t.Fatalf("expected unhealthy, got %s", resp.Storage)
	}
}

func TestReadyEndpoint_StorageIsFile(t *testing.T) {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=masterpassword dbname=oort_objects_test sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	tmpFile := t.TempDir() + "/file.txt"
	os.WriteFile(tmpFile, []byte("test"), 0644)

	healthSvc := services.NewHealthService(db, tmpFile)
	handler := NewHealthHandler(healthSvc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.Ready(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}

	var resp readinessStatus
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Storage != "unhealthy" {
		t.Fatalf("expected unhealthy, got %s", resp.Storage)
	}
}
