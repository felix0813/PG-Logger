package handler

import (
	"log"
	"net/http"

	"pg-logger/storage"
)

func RegisterRoutes(mux *http.ServeMux, postgres *storage.Postgres) {
	healthHandler := NewHealthHandler(postgres)
	appHandler := NewAppHandler(postgres)
	logHandler := NewLogHandler(postgres)

	log.Printf("registering routes")
	mux.HandleFunc("/logger/health", healthHandler.Health)
	mux.HandleFunc("/logger/apps", appHandler.HandleApps)
	mux.HandleFunc("/logger/apps/", appHandler.HandleAppByCode)
	mux.HandleFunc("/logger/logs", logHandler.HandleLogs)
}
