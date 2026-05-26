package handlers

import (
	"io"
	"net/http"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)

	io.WriteString(w, "Oort Object Storage API is running")
}