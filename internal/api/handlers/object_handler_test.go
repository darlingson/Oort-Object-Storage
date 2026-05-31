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

	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/filesystem"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/tests"
	"github.com/go-chi/chi/v5"
)

func TestUploadAndDownloadObject(t *testing.T) {

	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	bucketRepo := repositories.NewPostgresBucketRepository(db)
	objectRepo := repositories.NewPostgresObjectRepository(db)

	bucketService := services.NewBucketService(bucketRepo)

	dir := t.TempDir()
	fs := filesystem.NewLocalDriver(dir)

	objectService := services.NewObjectService(
		bucketRepo,
		objectRepo,
		fs,
	)

	bucket, _ := bucketService.CreateBucket(context.Background(), "docs")

	handler := NewObjectHandler(objectService)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	_, err = part.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("failed to write to form file: %v", err)
	}

	writer.Close()

	req := httptest.NewRequest(
		http.MethodPut,
		"/buckets/docs/objects/test.txt",
		&body,
	)

	req.Header.Set("Content-Type", writer.FormDataContentType())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucket.Name)
	rctx.URLParams.Add("*", "test.txt") 

	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	handler.UploadObject(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", rr.Code)
	}

	obj, err := objectService.GetObject(
		context.Background(),
		"docs",
		"test.txt",
	)

	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	if obj.ObjectKey != "test.txt" {
		t.Fatalf("wrong object returned")
	}
}

func TestListObjects(t *testing.T) {
	db := tests.NewTestDB(t)
	defer db.Close()

	tests.TruncateTables(t, db)

	bucketRepo := repositories.NewPostgresBucketRepository(db)
	objectRepo := repositories.NewPostgresObjectRepository(db)

	bucketService := services.NewBucketService(bucketRepo)

	dir := t.TempDir()
	fs := filesystem.NewLocalDriver(dir)

	objectService := services.NewObjectService(
		bucketRepo,
		objectRepo,
		fs,
	)

	bucket, _ := bucketService.CreateBucket(context.Background(), "assets")

	file1 := strings.NewReader("image content")
	_, err := objectService.UploadObject(
		context.Background(),
		"assets",
		"logo.png",
		"image/png",
		int64(file1.Len()),
		file1,
	)
	if err != nil {
		t.Fatalf("failed to seed object 1: %v", err)
	}

	file2 := strings.NewReader("text content")
	_, err = objectService.UploadObject(
		context.Background(),
		"assets",
		"readme.txt",
		"text/plain",
		int64(file2.Len()),
		file2,
	)
	if err != nil {
		t.Fatalf("failed to seed object 2: %v", err)
	}

	handler := NewObjectHandler(objectService)

	req := httptest.NewRequest(
		http.MethodGet,
		"/buckets/assets/objects",
		nil,
	)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("bucket", bucket.Name)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	handler.ListObjects(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}

	var objects []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&objects); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if len(objects) != 2 {
		t.Fatalf("expected 2 objects in list, got %d", len(objects))
	}
}