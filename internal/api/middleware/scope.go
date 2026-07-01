package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

func BucketScope(keyService *services.KeyService) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				rawKey := r.Header.Get("X-API-Key")

				if rawKey == "" {
					http.Error(
						w,
						http.StatusText(http.StatusForbidden),
						http.StatusForbidden,
					)
					return
				}

				key, err := keyService.Validate(
					r.Context(),
					rawKey,
				)

				if err != nil {
					http.Error(
						w,
						http.StatusText(http.StatusForbidden),
						http.StatusForbidden,
					)
					return
				}

				bucketName := chi.URLParam(r, "bucket")

				if bucketName == "" {
					http.Error(
						w,
						http.StatusText(http.StatusBadRequest),
						http.StatusBadRequest,
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
