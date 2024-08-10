package safebrowsing

import (
	"fmt"
	"log"
	"os"

	safebrowsing "github.com/google/safebrowsing"
)

var sb *safebrowsing.SafeBrowser

func InitSafeBrowsing() error {
	safeBrowsingAPIKey := os.Getenv("SAFE_BROWSING_API_KEY")
	if safeBrowsingAPIKey == "" {
		log.Fatalf("SAFE_BROWSING_API_KEY not provided")
	}

	safeBrowsingDBPath := os.Getenv("SAFE_BROWSING_DB_PATH")
	if safeBrowsingDBPath == "" {
		log.Fatalf("SAFE_BROWSING_DB_PATH not provided")
	}

	config := &safebrowsing.Config{
		APIKey: safeBrowsingAPIKey,
		ID:     "url-shortener",
		DBPath: safeBrowsingDBPath,
	}

	var err error
	sb, err = safebrowsing.NewSafeBrowser(*config)
	if err != nil {
		return fmt.Errorf("failed to create SafeBrowser: %v", err)
	}

	return nil
}

func IsSafeURL(url string) (bool, error) {
	if sb == nil {
		return false, fmt.Errorf("SafeBrowser is not initialized")
	}

	threats, err := sb.LookupURLs([]string{url})
	if err != nil {
		return false, fmt.Errorf("URL not safe: %v", err)
	}

	isSafe := len(threats[0]) == 0
	return isSafe, nil
}

func Close() {
	if sb != nil {
		sb.Close()
	}
}
