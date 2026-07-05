package middleware

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijacking not supported")
}

func extractObjectKey(r *http.Request) string {
	prefix := "/buckets/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	objPrefix := "/objects/"
	objRest := strings.TrimPrefix(rest[len(parts[0]):], objPrefix)
	return objRest
}

func objectPermission(r *http.Request) string {
	if r.Method == http.MethodPut {
		return "object:upload"
	}
	if r.Method == http.MethodDelete {
		return "object:delete"
	}
	if r.Method == http.MethodPost {
		return "object:sign"
	}
	if r.Method == http.MethodGet {
		key := extractObjectKey(r)
		if key == "" {
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
	signedURLSvc *services.SignedURLService,
) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				token := r.URL.Query().Get("token")
				if token != "" && signedURLSvc != nil {
					bucketName := chi.URLParam(r, "bucket")
					objectKey := extractObjectKey(r)

					perm := objectPermission(r)
					op := "download"
					if perm == "object:upload" {
						op = "upload"
					}
					result, err := signedURLSvc.Validate(
						r.Context(), token, bucketName, objectKey, op,
					)
					if err != nil {
						status := http.StatusForbidden
						if errors.Is(err, services.ErrTokenNotFound) ||
							errors.Is(err, services.ErrTokenExpired) {
							status = http.StatusUnauthorized
						}
						http.Error(w, err.Error(), status)
						return
					}

					ctx := context.WithValue(
						r.Context(),
						UserContextKey,
						"",
					)
					ctx = context.WithValue(
						ctx,
						PermissionsContextKey,
						[]string{},
					)

					sw := &statusRecorder{ResponseWriter: w}
					next.ServeHTTP(sw, r.WithContext(ctx))

					if op == "upload" && result != nil && sw.status < 300 {
						_ = signedURLSvc.Consume(
							r.Context(), result.ID,
						)
					}
					return
				}

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

				requiredPerm := objectPermission(r)
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
