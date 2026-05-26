package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/api/handlers"
)

func SetupRoutes() http.Handler {

	r := chi.NewRouter()

	r.Get("/", handlers.HealthCheck)

	return r
}