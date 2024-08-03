package utils

import (
	"crypto/sha256"
	"math/rand"
	"net/url"
	"os"
	"time"
)

const base62Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func encodeBytesToBase62(input []byte) string {
	result := make([]byte, 0, 10)
	for _, b := range input {
		result = append(result, base62Chars[b%62])
	}
	return string(result)
}

func GenerateKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return encodeBytesToBase62(hash[:])[:10]
}

func IsValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GenerateRandomString(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = base62Chars[seededRand.Intn(len(base62Chars))]
	}
	return string(b)
}
