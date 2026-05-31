package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

type ObjectHandler struct {
	service *services.ObjectService
}

func NewObjectHandler(
	service *services.ObjectService,
) *ObjectHandler {
	return &ObjectHandler{
		service: service,
	}
}
func (h *ObjectHandler) UploadObject(
	w http.ResponseWriter,
	r *http.Request,
) {

	bucket := chi.URLParam(r, "bucket")
	key := chi.URLParam(r, "key")

	r.ParseMultipartForm(10 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	obj, err := h.service.UploadObject(
		r.Context(),
		bucket,
		key,
		header.Header.Get("Content-Type"),
		header.Size,
		file,
	)

	if err != nil {
		if err == services.ErrBucketNotFound {
			http.Error(w, "bucket not found", http.StatusNotFound)
			return
		}

		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(obj)
}