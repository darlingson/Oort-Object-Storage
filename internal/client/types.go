package client

import "time"

type LoginResult struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type Bucket struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Object struct {
	ID          string `json:"id"`
	BucketID    string `json:"bucket_id"`
	ObjectKey   string `json:"object_key"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type SignResult struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}
