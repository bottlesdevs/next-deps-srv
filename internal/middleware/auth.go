package middleware

import (
	"net/http"
	"strings"

	"github.com/bottlesdevs/next-deps-srv/internal/auth"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
)

const ctxUserKey = "auth_user_claims"

func Auth(s *store.Store, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			claims, err := auth.ValidateToken(secret, token)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			r = r.WithContext(withClaims(r.Context(), claims))
			next.ServeHTTP(w, r)
		})
	}
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFrom(r)
			if claims == nil {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			if !auth.HasRole(claims, roles...) {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	// EventSource can't set headers - allow token via query param
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	return ""
}
