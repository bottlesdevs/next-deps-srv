package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

func IssueToken(u models.User, secret string) (string, error) {
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Roles:    u.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func HasRole(claims *Claims, roles ...string) bool {
	for _, want := range roles {
		for _, have := range claims.Roles {
			if strings.EqualFold(have, want) {
				return true
			}
		}
	}
	return false
}
