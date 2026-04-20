package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pg-logger/storage"
	"pg-logger/util"
)

type LogHandler struct {
	postgres *storage.Postgres
}

func NewLogHandler(postgres *storage.Postgres) *LogHandler {
	return &LogHandler{postgres: postgres}
}

func (h *LogHandler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createLog(w, r)
	case http.MethodGet:
		h.listLogs(w, r)
	case http.MethodDelete:
		h.deleteLog(w, r)
	default:
		util.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *LogHandler) createLog(w http.ResponseWriter, r *http.Request) {
	var input storage.CreateLogInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("create log decode request failed: %v", err)
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if input.AppCode == "" || input.Level == "" || input.Message == "" {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "app_code, level and message are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	log.Printf("creating log, app_code=%s, level=%s", input.AppCode, input.Level)
	record, err := h.postgres.CreateLog(ctx, input)
	if err != nil {
		log.Printf("create log failed, app_code=%s, err=%v", input.AppCode, err)
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "create log failed"})
		return
	}

	util.WriteJSON(w, http.StatusCreated, record)
}

func (h *LogHandler) listLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	appCode := query.Get("app_code")
	level := strings.ToUpper(query.Get("level"))
	env := query.Get("env")

	limit := int32(100)
	if query.Get("limit") != "" {
		value, err := strconv.ParseInt(query.Get("limit"), 10, 32)
		if err != nil {
			util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a number"})
			return
		}
		limit = int32(value)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	logs, err := h.postgres.ListLogs(ctx, appCode, level, env, limit)
	if err != nil {
		log.Printf("list logs failed: %v", err)
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "list logs failed"})
		return
	}

	util.WriteJSON(w, http.StatusOK, logs)
}

func (h *LogHandler) deleteLog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id, err := strconv.ParseInt(query.Get("id"), 10, 64)
	if err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id must be a number"})
		return
	}

	logTime, err := time.Parse(time.RFC3339, query.Get("log_time"))
	if err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "log_time must be RFC3339 format"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	log.Printf("deleting log, id=%d, log_time=%s", id, logTime.Format(time.RFC3339))
	if err = h.postgres.DeleteLog(ctx, id, logTime); err != nil {
		if err.Error() == "log not found" {
			util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "log not found"})
			return
		}
		log.Printf("delete log failed, id=%d, log_time=%s, err=%v", id, logTime.Format(time.RFC3339), err)
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete log failed"})
		return
	}

	util.WriteJSON(w, http.StatusOK, map[string]string{"message": "log deleted"})
}
