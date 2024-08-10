package utils

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"net/url"
	"time"

	"github.com/artem-streltsov/url-shortener/internal/database"
)

const base62Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

type contextKey string

const userContextKey contextKey = "user"

func encodeBytesToBase62(input []byte) string {
	result := make([]byte, 0, 10)
	for _, b := range input {
		result = append(result, base62Chars[b%62])
	}
	return string(result)
}

func GenerateKey(url string) string {
	timestamp := time.Now().UnixNano()
	timestampBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBytes, uint64(timestamp))
	combinedBytes := append([]byte(url), timestampBytes...)
	hash := sha256.Sum256(combinedBytes)

	return encodeBytesToBase62(hash[:])[:10]
}

func IsValidURL(urlStr string) (string, bool) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr, false
	}

	if u.Scheme == "" {
		urlStr = "https://" + urlStr
		u, err = url.Parse(urlStr)
		if err != nil {
			return urlStr, false
		}
	}

	return urlStr, u.Scheme != "" && u.Host != ""
}

func ContextWithUser(ctx context.Context, user *database.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) *database.User {
	user, ok := ctx.Value(userContextKey).(*database.User)
	if !ok {
		return nil
	}
	return user
}
