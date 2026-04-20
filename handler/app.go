package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"pg-logger/storage"
	"pg-logger/util"
)

type AppHandler struct {
	postgres *storage.Postgres
}

func NewAppHandler(postgres *storage.Postgres) *AppHandler {
	return &AppHandler{postgres: postgres}
}

func (h *AppHandler) HandleApps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createApp(w, r)
	case http.MethodGet:
		h.listApps(w, r)
	default:
		util.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *AppHandler) HandleAppByCode(w http.ResponseWriter, r *http.Request) {
	appCode := strings.TrimPrefix(r.URL.Path, "/logger/apps/")
	if appCode == "" {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "app_code is required"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateAppByCode(w, r, appCode)
	case http.MethodDelete:
		h.deleteAppByCode(w, r, appCode)
	default:
		util.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *AppHandler) createApp(w http.ResponseWriter, r *http.Request) {
	var input storage.CreateAppInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("create app decode request failed: %v", err)
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if input.AppCode == "" || input.AppName == "" || input.Env == "" {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "app_code, app_name and env are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	log.Printf("creating app, app_code=%s", input.AppCode)
	app, err := h.postgres.CreateApp(ctx, input)
	if err != nil {
		log.Printf("create app failed, app_code=%s, err=%v", input.AppCode, err)
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "create app failed"})
		return
	}

	util.WriteJSON(w, http.StatusCreated, app)
}

func (h *AppHandler) listApps(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	apps, err := h.postgres.ListApps(ctx)
	if err != nil {
		log.Printf("list apps failed: %v", err)
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "list apps failed"})
		return
	}

	util.WriteJSON(w, http.StatusOK, apps)
}

func (h *AppHandler) updateAppByCode(w http.ResponseWriter, r *http.Request, appCode string) {
	var input storage.UpdateAppInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("update app decode request failed, app_code=%s, err=%v", appCode, err)
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if input.AppName == "" || input.Env == "" {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "app_name and env are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	log.Printf("updating app, app_code=%s", appCode)
	app, err := h.postgres.UpdateAppByCode(ctx, appCode, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "app not found"})
			return
		}
		log.Printf("update app failed, app_code=%s, err=%v", appCode, err)
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "update app failed"})
		return
	}

	util.WriteJSON(w, http.StatusOK, app)
}

func (h *AppHandler) deleteAppByCode(w http.ResponseWriter, r *http.Request, appCode string) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	log.Printf("deleting app, app_code=%s", appCode)
	err := h.postgres.DeleteAppByCode(ctx, appCode)
	if err != nil {
		if err.Error() == "app not found" {
			util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "app not found"})
			return
		}
		log.Printf("delete app failed, app_code=%s, err=%v", appCode, err)
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete app failed"})
		return
	}

	util.WriteJSON(w, http.StatusOK, map[string]string{"message": "app deleted"})
}
