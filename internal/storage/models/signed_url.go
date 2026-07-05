package models

import (
	"time"

	"github.com/google/uuid"
)

type SignedURL struct {
	ID         uuid.UUID
	BucketName string
	ObjectKey  string
	Token      string
	Operation  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
}
