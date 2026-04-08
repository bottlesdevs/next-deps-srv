package middleware

import (
	"context"

	"github.com/bottlesdevs/next-deps-srv/internal/auth"
)

type contextKey string

const claimsKey contextKey = ctxUserKey

func withClaims(ctx context.Context, c *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFrom(r interface{ Context() context.Context }) *auth.Claims {
	v := r.Context().Value(claimsKey)
	if v == nil {
		return nil
	}
	c, _ := v.(*auth.Claims)
	return c
}
