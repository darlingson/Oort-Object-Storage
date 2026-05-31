package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/api/handlers"
)

func SetupRoutes(
	bucketHandler *handlers.BucketHandler,
) http.Handler {

	r := chi.NewRouter()

	r.Get("/", handlers.HealthCheck)

	r.Route("/buckets", func(r chi.Router) {

		r.Post("/", bucketHandler.CreateBucket)

		r.Get("/", bucketHandler.ListBuckets)

		r.Get("/{name}", bucketHandler.GetBucket)

	})

	return r
}