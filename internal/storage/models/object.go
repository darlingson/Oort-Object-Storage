package models

import (
	"time"

	"github.com/google/uuid"
)

type Object struct {
	ID          uuid.UUID `json:"id"`
	BucketID    uuid.UUID `json:"bucket_id"`
	ObjectKey   string    `json:"object_key"`
	StoragePath string    `json:"storage_path"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Checksum    string    `json:"checksum"`
	CreatedAt   time.Time `json:"created_at"`
}