package main

import (
	"context"
	"errors"
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
		log.Fatal("missing required env: DATABASE_URL")
	}

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		log.Fatal("missing required env: SERVER_ADDR")
	}

	log.Printf("starting pg-logger server")
	db, err := storage.NewPostgres(databaseURL)
	if err != nil {
		log.Fatalf("init postgres failed: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, db)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", server.Addr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", serveErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
		log.Printf("server shutdown error: %v", shutdownErr)
	}
	log.Printf("server stopped")
}
