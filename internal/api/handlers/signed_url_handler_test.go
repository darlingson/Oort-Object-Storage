package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/filesystem"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
)

func setupSignedURLTest(t *testing.T) (*services.SignedURLService, *services.ObjectService, string, string) {
	t.Helper()

	db := tests.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	tests.TruncateTables(t, db)

	bucketRepo := repositories.NewPostgresBucketRepository(db)
	objectRepo := repositories.NewPostgresObjectRepository(db)
	bucketService := services.NewBucketService(bucketRepo)

	dir := t.TempDir()
	fs := filesystem.NewLocalDriver(dir)
	objectService := services.NewObjectService(bucketRepo, objectRepo, fs)

	bucket, err := bucketService.CreateBucket(context.Background(), "docs")
	if err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}

	signedURLRepo := repositories.NewPostgresSignedURLRepository(db)
	signedURLSvc := services.NewSignedURLService(signedURLRepo)

	return signedURLSvc, objectService, bucket.Name, dir
}

func TestSignUploadURLHandler(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	body := `{"operation":"upload","expires_in":"1h"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/buckets/"+bucketName+"/objects/test.txt/sign",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("key", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Sign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp SignResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.URL == "" {
		t.Fatal("expected non-empty URL")
	}
	if !strings.Contains(resp.URL, "token=") {
		t.Fatal("expected URL to contain token")
	}
}

func TestSignDownloadURLHandler(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	body := `{"operation":"download","expires_in":"2h"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/buckets/"+bucketName+"/objects/test.txt/sign",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("key", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Sign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp SignResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.URL == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestSignInvalidOperation(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	body := `{"operation":"delete","expires_in":"1h"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/buckets/"+bucketName+"/objects/test.txt/sign",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("key", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Sign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSignInvalidBody(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/buckets/"+bucketName+"/objects/test.txt/sign",
		strings.NewReader("not json"),
	)
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("key", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.Sign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUploadViaSignedURL(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	signed, err := signedURLSvc.Sign(context.Background(), services.SignInput{
		BucketName: bucketName,
		ObjectKey:  "uploaded.txt",
		Operation:  "upload",
		ExpiresIn:  1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "uploaded.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, err = part.Write([]byte("signed upload content"))
	if err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(
		http.MethodPut,
		"/buckets/"+bucketName+"/objects/uploaded.txt?token="+signed.Token,
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("*", "uploaded.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	_, err = signedURLSvc.Validate(
		context.Background(),
		signed.Token,
		bucketName,
		"uploaded.txt",
		"upload",
	)
	if err != services.ErrTokenNotFound {
		t.Fatal("expected token to be consumed after upload")
	}
}

func TestHandleUploadMissingToken(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	req := httptest.NewRequest(
		http.MethodPut,
		"/buckets/"+bucketName+"/objects/test.txt",
		nil,
	)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("*", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUploadExpiredToken(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	signed, err := signedURLSvc.Sign(context.Background(), services.SignInput{
		BucketName: bucketName,
		ObjectKey:  "expired.txt",
		Operation:  "upload",
		ExpiresIn:  -1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPut,
		"/buckets/"+bucketName+"/objects/expired.txt?token="+signed.Token,
		nil,
	)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("*", "expired.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.HandleUpload(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDownloadViaSignedURL(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)

	file := strings.NewReader("download content")
	obj, err := objectSvc.UploadObject(
		context.Background(),
		bucketName,
		"download.txt",
		"text/plain",
		int64(file.Len()),
		file,
	)
	if err != nil {
		t.Fatalf("failed to upload object: %v", err)
	}

	signed, err := signedURLSvc.Sign(context.Background(), services.SignInput{
		BucketName: bucketName,
		ObjectKey:  obj.ObjectKey,
		Operation:  "download",
		ExpiresIn:  1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	req := httptest.NewRequest(
		http.MethodGet,
		"/buckets/"+bucketName+"/objects/download.txt?token="+signed.Token,
		nil,
	)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("*", "download.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.HandleDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDownloadMissingToken(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	req := httptest.NewRequest(
		http.MethodGet,
		"/buckets/"+bucketName+"/objects/test.txt",
		nil,
	)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("*", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.HandleDownload(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDownloadExpiredToken(t *testing.T) {
	signedURLSvc, objectSvc, bucketName, _ := setupSignedURLTest(t)
	handler := NewSignedURLHandler(signedURLSvc, objectSvc)

	signed, err := signedURLSvc.Sign(context.Background(), services.SignInput{
		BucketName: bucketName,
		ObjectKey:  "expired-dl.txt",
		Operation:  "download",
		ExpiresIn:  -1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/buckets/"+bucketName+"/objects/expired-dl.txt?token="+signed.Token,
		nil,
	)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucketName)
	rctx.URLParams.Add("*", "expired-dl.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.HandleDownload(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}
