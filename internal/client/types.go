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
