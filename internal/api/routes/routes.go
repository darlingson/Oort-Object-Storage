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

	r.With(
		middleware.Auth(jwtService),
		middleware.RequirePermission("user:create"),
	).Post("/users", userHandler.CreateUser)

	r.With(
		middleware.Auth(jwtService),
		middleware.RequirePermission("user:grant-permission"),
	).Post("/users/{id}/permissions", userHandler.GrantPermission)

	r.With(
		middleware.Auth(jwtService),
		middleware.RequirePermission("apikey:create"),
	).Post("/api-keys", keyHandler.CreateKey)

	r.With(
		middleware.Auth(jwtService),
		middleware.RequirePermission("apikey:list"),
	).Get("/api-keys", keyHandler.ListKeys)

	r.With(
		middleware.Auth(jwtService),
		middleware.RequirePermission("apikey:delete"),
	).Delete("/api-keys/{id}", keyHandler.DeleteKey)

	r.With(
		middleware.Auth(jwtService),
		middleware.RequirePermission("bucket:create"),
	).Post("/buckets", bucketHandler.CreateBucket)

	r.With(
		middleware.Auth(jwtService),
		middleware.RequirePermission("bucket:list"),
	).Get("/buckets", bucketHandler.ListBuckets)

	r.With(
		middleware.Auth(jwtService),
		middleware.RequirePermission("bucket:list"),
	).Get("/buckets/{name}", bucketHandler.GetBucket)

	r.Route("/buckets/{bucket}/objects", func(r chi.Router) {
		r.Use(middleware.ObjectAccess(keyService, jwtService))

		r.Put("/*", objectHandler.UploadObject)
		r.Get("/*", objectHandler.DownloadObject)
		r.Delete("/*", objectHandler.DeleteObject)
		r.Get("/", objectHandler.ListObjects)
	})

	return r
}
