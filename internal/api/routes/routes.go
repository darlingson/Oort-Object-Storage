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
	keyHandler *handlers.KeyHandler,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	keyService *services.KeyService,
	jwtService *services.JWTService,
) http.Handler {

	r := chi.NewRouter()

	r.Get("/health", handlers.HealthCheck)

	r.Post("/auth/login", authHandler.Login)

	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))
		r.Use(middleware.RequirePermission("user:create"))

		r.Post("/", userHandler.CreateUser)
	})

	r.Route("/api-keys", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))
		r.Use(middleware.RequirePermission("apikey:create"))

		r.Post("/", keyHandler.CreateKey)
		r.Get("/", keyHandler.ListKeys)
		r.Delete("/{id}", keyHandler.DeleteKey)
	})

	r.Route("/buckets", func(r chi.Router) {
		r.Use(middleware.Auth(jwtService))
		r.Use(middleware.RequirePermission("bucket:create"))

		r.Post("/", bucketHandler.CreateBucket)
		r.Get("/", bucketHandler.ListBuckets)
		r.Get("/{name}", bucketHandler.GetBucket)
	})

	r.Route("/buckets/{bucket}/objects", func(r chi.Router) {
		r.Use(middleware.ObjectAccess(keyService, jwtService))

		r.Put("/*", objectHandler.UploadObject)
		r.Get("/*", objectHandler.DownloadObject)
		r.Delete("/*", objectHandler.DeleteObject)
		r.Get("/", objectHandler.ListObjects)
	})

	return r
}
