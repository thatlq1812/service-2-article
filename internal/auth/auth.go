package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"
)

var (
	ErrMissingToken = errors.New("missing authorization")
	ErrInvalidToken = errors.New("invalid token")
)

type Claims struct {
	UserID uint64 `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func ValidateToken(tokenString, jwtSecret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func ExtractTokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrMissingToken
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return "", ErrMissingToken
	}

	token := strings.TrimPrefix(authHeader[0], "Bearer ")
	if token == authHeader[0] {
		return "", ErrMissingToken
	}

	return token, nil
}

func GetUserIDFromContext(ctx context.Context, jwtSecret string) (uint64, error) {
	token, err := ExtractTokenFromContext(ctx)
	if err != nil {
		return 0, err
	}

	claims, err := ValidateToken(token, jwtSecret)
	if err != nil {
		return 0, err
	}

	return claims.UserID, nil
}
