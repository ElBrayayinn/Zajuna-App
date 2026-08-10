package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/storage/sqlite"
)

type fichaLister interface {
	ListFichas(ctx context.Context, limit int) ([]sqlite.FichaRecord, error)
}

type fichaView struct {
	ID         string  `json:"id"`
	ExternalID string  `json:"externalId"`
	Name       string  `json:"name"`
	CourseID   string  `json:"courseId"`
	Status     string  `json:"status"`
	SyncedAt   *string `json:"syncedAt,omitempty"`
	UpdatedAt  string  `json:"updatedAt"`
}

type syncFichasRequest struct {
	Username     string `json:"username"`
	DocumentType string `json:"documentType"`
}

func registerFichaRoutes(mux *http.ServeMux, store fichaLister, runtime *jobs.Runtime, dataDir string) {
	mux.HandleFunc("GET /api/fichas", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el almacenamiento de fichas no está disponible"))
			return
		}
		limit := 50
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed < 1 || parsed > 100 {
				writeError(w, http.StatusBadRequest, errors.New("limit debe ser un número entre 1 y 100"))
				return
			}
			limit = parsed
		}
		items, err := store.ListFichas(r.Context(), limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		views := make([]fichaView, 0, len(items))
		for _, item := range items {
			views = append(views, toFichaView(item))
		}
		writeJSON(w, http.StatusOK, views)
	})

	mux.HandleFunc("POST /api/fichas/sync", func(w http.ResponseWriter, r *http.Request) {
		if runtime == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el runtime de jobs no está disponible"))
			return
		}
		var request syncFichasRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
				writeError(w, http.StatusBadRequest, errors.New("el input de sincronización es inválido"))
				return
			}
		}
		request.Username = strings.TrimSpace(request.Username)
		if request.Username == "" {
			config, err := readConfig(dataDir)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			request.Username = config.ZajunaUsername
		}
		if request.Username == "" {
			writeError(w, http.StatusBadRequest, errors.New("configura primero el usuario de Zajuna"))
			return
		}
		if request.DocumentType == "" {
			request.DocumentType = "CC"
		}
		job, err := runtime.Submit(r.Context(), "sync-fichas", request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, toJobView(job))
	})
}

func toFichaView(item sqlite.FichaRecord) fichaView {
	view := fichaView{
		ID: item.ID, ExternalID: item.ExternalID, Name: item.Name,
		CourseID: item.CourseID, Status: item.Status,
		UpdatedAt: item.UpdatedAt.Format("2006-01-02T15:04:05.999Z07:00"),
	}
	if item.SyncedAt != nil {
		value := item.SyncedAt.Format("2006-01-02T15:04:05.999Z07:00")
		view.SyncedAt = &value
	}
	return view
}
