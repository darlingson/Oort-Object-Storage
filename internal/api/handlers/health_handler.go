package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

type HealthHandler struct {
	healthService *services.HealthService
}

func NewHealthHandler(healthService *services.HealthService) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

type readinessStatus struct {
	Status    string `json:"status"`
	Database  string `json:"database"`
	Storage   string `json:"storage"`
	Timestamp string `json:"timestamp"`
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := readinessStatus{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	dbErr := h.healthService.CheckDB(ctx)
	if dbErr != nil {
		status.Database = "unhealthy"
	} else {
		status.Database = "ok"
	}

	storageErr := h.healthService.CheckStorage()
	if storageErr != nil {
		status.Storage = "unhealthy"
	} else {
		status.Storage = "ok"
	}

	if status.Database == "ok" && status.Storage == "ok" {
		status.Status = "ok"
		w.WriteHeader(http.StatusOK)
	} else {
		status.Status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"service": "Oort Object Storage API",
		"status":  "running",
	})
}
