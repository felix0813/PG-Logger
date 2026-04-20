package handler

import (
	"context"
	"net/http"
	"time"

	"pg-logger/storage"
	"pg-logger/util"
)

type HealthHandler struct {
	postgres *storage.Postgres
}

func NewHealthHandler(postgres *storage.Postgres) *HealthHandler {
	return &HealthHandler{postgres: postgres}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	response := map[string]string{
		"service": "up",
	}

	if err := h.postgres.HealthCheck(ctx); err != nil {
		response["database"] = "down"
		util.WriteJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	response["database"] = "up"
	util.WriteJSON(w, http.StatusOK, response)
}
