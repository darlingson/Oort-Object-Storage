package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

type KeyHandler struct {
	service *services.KeyService
}

func NewKeyHandler(
	service *services.KeyService,
) *KeyHandler {

	return &KeyHandler{
		service: service,
	}
}

type CreateKeyRequest struct {
	Name      string     `json:"name"`
	Buckets   []string   `json:"buckets"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type CreateKeyResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Buckets   []string   `json:"buckets"`
	RawKey    string     `json:"raw_key"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (h *KeyHandler) CreateKey(
	w http.ResponseWriter,
	r *http.Request,
) {

	var req CreateKeyRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	if req.Name == "" {
		http.Error(
			w,
			"name is required",
			http.StatusBadRequest,
		)
		return
	}

	if len(req.Buckets) == 0 {
		http.Error(
			w,
			"at least one bucket is required",
			http.StatusBadRequest,
		)
		return
	}

	key, rawKey, err := h.service.Create(
		r.Context(),
		req.Name,
		req.Buckets,
		req.ExpiresAt,
	)

	if err != nil {
		http.Error(
			w,
			"failed to create key",
			http.StatusInternalServerError,
		)
		return
	}

	resp := CreateKeyResponse{
		ID:        key.ID,
		Name:      key.Name,
		Buckets:   key.Buckets,
		RawKey:    rawKey,
		ExpiresAt: key.ExpiresAt,
		CreatedAt: key.CreatedAt,
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(resp)
}

func (h *KeyHandler) ListKeys(
	w http.ResponseWriter,
	r *http.Request,
) {

	keys, err := h.service.List(r.Context())

	if err != nil {
		http.Error(
			w,
			"internal error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	json.NewEncoder(w).Encode(keys)
}

func (h *KeyHandler) DeleteKey(
	w http.ResponseWriter,
	r *http.Request,
) {

	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)

	if err != nil {
		http.Error(
			w,
			"invalid key id",
			http.StatusBadRequest,
		)
		return
	}

	err = h.service.Delete(r.Context(), id)

	if err != nil {
		http.Error(
			w,
			"delete failed",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
