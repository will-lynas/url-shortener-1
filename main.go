package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/artem-streltsov/url-shortener/internal/auth"
	"github.com/artem-streltsov/url-shortener/internal/database"
	"github.com/artem-streltsov/url-shortener/internal/handlers"
	"github.com/artem-streltsov/url-shortener/internal/safebrowsing"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("No .env file found")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalf("PORT not provided")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		log.Fatalf("DB_PATH not provided")
	}

	auth.InitJWTKey()

	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
		log.Fatalf("Error creating database directory %v: %v", dbDir, err)
	}

	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	if err := safebrowsing.InitSafeBrowsing(); err != nil {
		log.Fatalf("Error initializing Safe Browsing: %v", err)
	}
	defer safebrowsing.Close()

	certFile := os.Getenv("TLS_CERT_PATH")
	if certFile == "" {
		log.Fatalf("TLS_CERT_PATH not provided")
	}

	keyFile := os.Getenv("TLS_KEY_PATH")
	if keyFile == "" {
		log.Fatalf("TLS_KEY_PATH not provided")
	}

	handler := handlers.NewHandler(db)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler.Routes(),
	}

	go func() {
		log.Printf("Starting HTTPS server at :%s", port)
		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting HTTPS server: %v", err)
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
