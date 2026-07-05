package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

type SignedURLHandler struct {
	signedURLSvc *services.SignedURLService
	objectSvc    *services.ObjectService
}

func NewSignedURLHandler(
	signedURLSvc *services.SignedURLService,
	objectSvc *services.ObjectService,
) *SignedURLHandler {

	return &SignedURLHandler{
		signedURLSvc: signedURLSvc,
		objectSvc:    objectSvc,
	}
}

type SignRequest struct {
	Operation string `json:"operation"`
	ExpiresIn string `json:"expires_in"`
}

type SignResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

func (h *SignedURLHandler) Sign(
	w http.ResponseWriter,
	r *http.Request,
) {

	bucketName := chi.URLParam(r, "bucket")
	objectKey := chi.URLParam(r, "*")
	if objectKey == "" {
		objectKey = chi.URLParam(r, "key")
	}

	var req SignRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Operation != "upload" && req.Operation != "download" {
		http.Error(
			w,
			"operation must be upload or download",
			http.StatusBadRequest,
		)
		return
	}

	expiresIn, err := time.ParseDuration(req.ExpiresIn)
	if err != nil {
		expiresIn = 1 * time.Hour
	}

	result, err := h.signedURLSvc.Sign(r.Context(), services.SignInput{
		BucketName: bucketName,
		ObjectKey:  objectKey,
		Operation:  req.Operation,
		ExpiresIn:  expiresIn,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	url := scheme + "://" + r.Host +
		"/buckets/" + bucketName +
		"/objects/" + objectKey +
		"?token=" + result.Token

	resp := SignResponse{
		URL:       url,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func tokenFromQuery(r *http.Request) string {
	return r.URL.Query().Get("token")
}

func (h *SignedURLHandler) HandleUpload(
	w http.ResponseWriter,
	r *http.Request,
) {

	bucketName := chi.URLParam(r, "bucket")
	objectKey := chi.URLParam(r, "*")
	token := tokenFromQuery(r)

	if token == "" {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	result, err := h.signedURLSvc.Validate(
		r.Context(), token, bucketName, objectKey, "upload",
	)
	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, services.ErrTokenNotFound) ||
			errors.Is(err, services.ErrTokenExpired) {
			status = http.StatusUnauthorized
		}
		http.Error(w, err.Error(), status)
		return
	}

	var body io.Reader
	var size int64
	contentType := r.Header.Get("Content-Type")

	if ct := r.Header.Get("Content-Type"); len(ct) >= 9 && ct[:9] == "multipart" {
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()
		body = file
		size = header.Size
		contentType = header.Header.Get("Content-Type")
	} else {
		body = r.Body
		size = r.ContentLength
	}

	obj, err := h.objectSvc.UploadObject(
		r.Context(),
		result.BucketName,
		result.ObjectKey,
		contentType,
		size,
		body,
	)
	if err != nil {
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	_ = h.signedURLSvc.Consume(r.Context(), result.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(obj)
}

func (h *SignedURLHandler) HandleDownload(
	w http.ResponseWriter,
	r *http.Request,
) {

	bucketName := chi.URLParam(r, "bucket")
	objectKey := chi.URLParam(r, "*")
	token := tokenFromQuery(r)

	if token == "" {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	result, err := h.signedURLSvc.Validate(
		r.Context(), token, bucketName, objectKey, "download",
	)
	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, services.ErrTokenNotFound) ||
			errors.Is(err, services.ErrTokenExpired) {
			status = http.StatusUnauthorized
		}
		http.Error(w, err.Error(), status)
		return
	}

	obj, err := h.objectSvc.GetObject(
		r.Context(),
		result.BucketName,
		result.ObjectKey,
	)
	if err != nil {
		http.Error(w, "object not found", http.StatusNotFound)
		return
	}

	file, err := h.objectSvc.OpenObjectFile(obj.StoragePath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(obj.SizeBytes, 10))
	io.Copy(w, file)
}
