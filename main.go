package main

import (
	"context"
	"encoding/gob"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/artem-streltsov/url-shortener/internal/database"
	"github.com/artem-streltsov/url-shortener/internal/handlers"
	"github.com/artem-streltsov/url-shortener/internal/safebrowsing"
	"github.com/artem-streltsov/url-shortener/internal/utils"
	"github.com/joho/godotenv"
)

func init() {
	gob.Register(&database.User{})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default values")
	}

	port := utils.GetEnv("PORT", "8080")
	dbPath := utils.GetEnv("DB_PATH", "database/database.sqlite3")
	sessionSecret := utils.GetEnv("SESSION_SECRET_KEY", utils.GenerateRandomString(32))

	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	safeBrowsingAPIKey := os.Getenv("SAFE_BROWSING_API_KEY")
	if safeBrowsingAPIKey != "" {
		if err := safebrowsing.InitSafeBrowsing(); err != nil {
			log.Printf("Error initializing Safe Browsing: %v", err)
		} else {
			defer safebrowsing.Close()
		}
	} else {
		log.Println("Safe Browsing API key not provided, safe browsing feature will be disabled")
	}

	handler := handlers.NewHandler(db, sessionSecret)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler.Routes(),
	}

	go func() {
		log.Printf("Starting server at :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
