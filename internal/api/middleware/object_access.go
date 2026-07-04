package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

func objectPermission(r *http.Request, bucketName string) string {
	if r.Method == http.MethodPut {
		return "object:upload"
	}
	if r.Method == http.MethodDelete {
		return "object:delete"
	}
	if r.Method == http.MethodGet {
		key := chi.URLParam(r, "*")
		if key == "" || key == "/" {
			return "object:list"
		}
		return "object:download"
	}
	return ""
}

func extractBearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func ObjectAccess(
	keyService *services.KeyService,
	jwtService *services.JWTService,
) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				apiKey := r.Header.Get("X-API-Key")
				authHeader := r.Header.Get("Authorization")

				if apiKey == "" && authHeader == "" {
					http.Error(
						w,
						http.StatusText(http.StatusUnauthorized),
						http.StatusUnauthorized,
					)
					return
				}

				if apiKey != "" {
					key, err := keyService.Validate(
						r.Context(),
						apiKey,
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

					if !keyService.CanAccessBucket(key, bucketName) {
						http.Error(
							w,
							http.StatusText(http.StatusForbidden),
							http.StatusForbidden,
						)
						return
					}

					next.ServeHTTP(w, r)
					return
				}

				tokenStr := extractBearerToken(authHeader)
				if tokenStr == "" {
					http.Error(
						w,
						http.StatusText(http.StatusUnauthorized),
						http.StatusUnauthorized,
					)
					return
				}

				claims, err := jwtService.Validate(tokenStr)
				if err != nil {
					http.Error(
						w,
						http.StatusText(http.StatusUnauthorized),
						http.StatusUnauthorized,
					)
					return
				}

				requiredPerm := objectPermission(r, "")
				if requiredPerm != "" {
					found := false
					for _, p := range claims.Permissions {
						if p == requiredPerm {
							found = true
							break
						}
					}
					if !found {
						http.Error(
							w,
							http.StatusText(http.StatusForbidden),
							http.StatusForbidden,
						)
						return
					}
				}

				ctx := context.WithValue(
					r.Context(),
					UserContextKey,
					claims.UserID,
				)
				ctx = context.WithValue(
					ctx,
					PermissionsContextKey,
					claims.Permissions,
				)

				next.ServeHTTP(w, r.WithContext(ctx))
			},
		)
	}
}
