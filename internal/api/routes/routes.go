package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/api/handlers"
	"github.com/darlingson/Oort-Object-Storage/internal/api/middleware"
	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

func SetupRoutes(
	bucketHandler *handlers.BucketHandler,
	objectHandler *handlers.ObjectHandler,
	keyService *services.KeyService,
) http.Handler {

	r := chi.NewRouter()

	r.Get("/health", handlers.HealthCheck)

	r.Group(func(r chi.Router) {
		r.Use(middleware.BearerAuth(keyService))

		r.Post("/buckets", bucketHandler.CreateBucket)

		r.Group(func(r chi.Router) {
			r.Use(middleware.BucketScope(keyService))

			r.Route("/buckets/{bucket}/objects", func(r chi.Router) {
				r.Put("/*", objectHandler.UploadObject)
				r.Get("/*", objectHandler.DownloadObject)
				r.Delete("/*", objectHandler.DeleteObject)
				r.Get("/", objectHandler.ListObjects)
			})
		})
	})

	r.Get("/buckets", bucketHandler.ListBuckets)
	r.Get("/buckets/{name}", bucketHandler.GetBucket)

	return r
}