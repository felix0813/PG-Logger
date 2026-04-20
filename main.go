package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pg-logger/handler"
	"pg-logger/storage"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	db, err := storage.NewPostgres(databaseURL)
	if err != nil {
		log.Fatalf("init postgres failed: %v", err)
	}
	defer db.Close()

	healthHandler := handler.NewHealthHandler(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/logger/health", healthHandler.Health)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", server.Addr)
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Fatalf("server failed: %v", serveErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
		log.Printf("server shutdown error: %v", shutdownErr)
	}
}
