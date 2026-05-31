package handlers

import (
	"encoding/json"
	"net/http"
	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

type BucketHandler struct {
	service *services.BucketService
}

func NewBucketHandler(
	service *services.BucketService,
) *BucketHandler {

	return &BucketHandler{
		service: service,
	}
}

type CreateBucketRequest struct {
	Name string `json:"name"`
}
func (h *BucketHandler) CreateBucket(
	w http.ResponseWriter,
	r *http.Request,
) {

	var req CreateBucketRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	bucket, err := h.service.CreateBucket(
		r.Context(),
		req.Name,
	)

	if err != nil {

		if err == services.ErrBucketExists {
			http.Error(
				w,
				err.Error(),
				http.StatusConflict,
			)
			return
		}

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

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(bucket)
}

func (h *BucketHandler) ListBuckets(
	w http.ResponseWriter,
	r *http.Request,
) {

	buckets, err := h.service.ListBuckets(
		r.Context(),
	)

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

	json.NewEncoder(w).Encode(buckets)
}

func (h *BucketHandler) GetBucket(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := chi.URLParam(r, "name")

	bucket, err := h.service.GetBucket(
		r.Context(),
		name,
	)

	if err != nil {
		http.Error(
			w,
			"bucket not found",
			http.StatusNotFound,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(bucket)
}