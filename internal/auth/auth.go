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
	ErrMissingToken     = errors.New("missing authorization")
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenBlacklisted = errors.New("token has been revoked")
)

// TokenBlacklistChecker interface for checking if token is blacklisted
type TokenBlacklistChecker interface {
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

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

// GetUserIDFromContextWithBlacklist validates token and checks Redis blacklist
func GetUserIDFromContextWithBlacklist(ctx context.Context, jwtSecret string, blacklistChecker TokenBlacklistChecker) (uint64, error) {
	// Extract token
	token, err := ExtractTokenFromContext(ctx)
	if err != nil {
		return 0, err
	}

	// Validate JWT signature and expiry
	claims, err := ValidateToken(token, jwtSecret)
	if err != nil {
		return 0, err
	}

	// Check if token is blacklisted (logged out)
	isBlacklisted, err := blacklistChecker.IsTokenBlacklisted(ctx, token)
	if err != nil {
		// Log error but don't fail - fail open for Redis issues
		// In production, you might want to fail closed (reject if Redis unavailable)
		return 0, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	if isBlacklisted {
		return 0, ErrTokenBlacklisted
	}

	return claims.UserID, nil
}
