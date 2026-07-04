package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

type contextKey string

const (
	UserContextKey        contextKey = "user"
	PermissionsContextKey contextKey = "permissions"
)

func GetUserID(ctx context.Context) string {
	val := ctx.Value(UserContextKey)
	if val == nil {
		return ""
	}
	return val.(string)
}

func GetPermissions(ctx context.Context) []string {
	val := ctx.Value(PermissionsContextKey)
	if val == nil {
		return nil
	}
	return val.([]string)
}

func Auth(jwtService *services.JWTService) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				header := r.Header.Get("Authorization")

				if header == "" {
					http.Error(
						w,
						http.StatusText(http.StatusUnauthorized),
						http.StatusUnauthorized,
					)
					return
				}

				parts := strings.SplitN(header, " ", 2)
				if len(parts) != 2 ||
					!strings.EqualFold(parts[0], "Bearer") {

					http.Error(
						w,
						http.StatusText(http.StatusUnauthorized),
						http.StatusUnauthorized,
					)
					return
				}

				claims, err := jwtService.Validate(parts[1])
				if err != nil {
					http.Error(
						w,
						http.StatusText(http.StatusUnauthorized),
						http.StatusUnauthorized,
					)
					return
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
