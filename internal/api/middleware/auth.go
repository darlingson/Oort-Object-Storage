package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type contextKey string

const keyContextKey contextKey = "api_key"

func GetAPIKey(ctx context.Context) *models.APIKey {

	val := ctx.Value(keyContextKey)

	if val == nil {
		return nil
	}

	return val.(*models.APIKey)
}

func BearerAuth(keyService *services.KeyService) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				header := r.Header.Get("Authorization")

				if header == "" {
					http.Error(
						w,
						"missing authorization header",
						http.StatusUnauthorized,
					)
					return
				}

				parts := strings.SplitN(header, " ", 2)

				if len(parts) != 2 ||
					!strings.EqualFold(parts[0], "Bearer") {

					http.Error(
						w,
						"invalid authorization format",
						http.StatusUnauthorized,
					)
					return
				}

				rawKey := parts[1]

				key, err := keyService.Validate(
					r.Context(),
					rawKey,
				)

				if err != nil {

					status := http.StatusUnauthorized

					if err == services.ErrKeyExpired {
						status = http.StatusForbidden
					}

					http.Error(
						w,
						http.StatusText(status),
						status,
					)
					return
				}

				ctx := context.WithValue(
					r.Context(),
					keyContextKey,
					key,
				)

				next.ServeHTTP(w, r.WithContext(ctx))
			},
		)
	}
}

func BucketScope(keyService *services.KeyService) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				bucketName := chi.URLParam(r, "bucket")

				if bucketName == "" {
					http.Error(
						w,
						"bucket not specified",
						http.StatusBadRequest,
					)
					return
				}

				key := GetAPIKey(r.Context())

				if key == nil {
					http.Error(
						w,
						http.StatusText(http.StatusUnauthorized),
						http.StatusUnauthorized,
					)
					return
				}

				if !keyService.CanAccessBucket(
					key,
					bucketName,
				) {

					http.Error(
						w,
						http.StatusText(http.StatusForbidden),
						http.StatusForbidden,
					)
					return
				}

				next.ServeHTTP(w, r)
			},
		)
	}
}
