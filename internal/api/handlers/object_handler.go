package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/config"
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
		config.AppLogger.Printf("upload error: %v", err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(obj)
}

func (h *ObjectHandler) DownloadObject(
	w http.ResponseWriter,
	r *http.Request,
) {

	bucket := chi.URLParam(r, "bucket")
	key := chi.URLParam(r, "key")

	obj, err := h.service.GetObject(
		r.Context(),
		bucket,
		key,
	)

	if err != nil {
		http.Error(w, "object not found", http.StatusNotFound)
		return
	}

	file, err := h.service.OpenObjectFile(obj.StoragePath)
	if err != nil {
		http.Error(w, "file missing", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(obj.SizeBytes, 10))
	w.Header().Set("Content-Disposition", "attachment; filename="+obj.ObjectKey)

	w.WriteHeader(http.StatusOK)

	io.Copy(w, file)
}