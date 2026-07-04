package middleware

import (
	"net/http"
)

func RequirePermission(permission string) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				perms := GetPermissions(r.Context())

				found := false
				for _, p := range perms {
					if p == permission {
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

				next.ServeHTTP(w, r)
			},
		)
	}
}
