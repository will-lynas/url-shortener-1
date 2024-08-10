package auth

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey []byte

var (
	ErrTokenExpired = errors.New("token has expired")
)

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func InitJWTKey() {
	key := os.Getenv("JWT_SECRET_KEY")
	if key == "" {
		log.Fatalf("JWT_SECRET_KEY not provided")
	}
	jwtKey = []byte(key)
}

func GenerateToken(userID int64) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return jwtKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return claims, ErrTokenExpired
		}
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func RefreshToken(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return jwtKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			if time.Since(claims.ExpiresAt.Time) > 24*time.Hour {
				return "", errors.New("token has been expired for too long")
			}
		} else {
			return "", err
		}
	}

	if !token.Valid && !errors.Is(err, jwt.ErrTokenExpired) {
		return "", errors.New("invalid token")
	}

	return GenerateToken(claims.UserID)
}
