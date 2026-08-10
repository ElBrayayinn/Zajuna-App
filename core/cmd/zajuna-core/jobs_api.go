package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/zajuna-app/core/internal/jobs"
)

type createJobRequest struct {
	Type  string          `json:"type"`
	Input json.RawMessage `json:"input"`
}

type jobView struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Status       jobs.Status     `json:"status"`
	Result       json.RawMessage `json:"result,omitempty"`
	Progress     int             `json:"progress"`
	Stage        string          `json:"stage"`
	Message      string          `json:"message"`
	Attempt      int             `json:"attempt"`
	MaxAttempts  int             `json:"maxAttempts"`
	ErrorCode    string          `json:"errorCode,omitempty"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	CreatedAt    string          `json:"createdAt"`
	StartedAt    *string         `json:"startedAt,omitempty"`
	FinishedAt   *string         `json:"finishedAt,omitempty"`
	UpdatedAt    string          `json:"updatedAt"`
}

type jobLister interface {
	ListJobs(ctx context.Context, limit int) ([]jobs.Job, error)
}

func registerJobRoutes(mux *http.ServeMux, runtime *jobs.Runtime, listers ...jobLister) {
	var lister jobLister
	if len(listers) > 0 {
		lister = listers[0]
	}

	mux.HandleFunc("GET /api/jobs", func(w http.ResponseWriter, r *http.Request) {
		if lister == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el almacenamiento de jobs no está disponible"))
			return
		}
		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed < 1 || parsed > 100 {
				writeError(w, http.StatusBadRequest, errors.New("limit debe ser un número entre 1 y 100"))
				return
			}
			limit = parsed
		}
		items, err := lister.ListJobs(r.Context(), limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		views := make([]jobView, 0, len(items))
		for _, item := range items {
			views = append(views, toJobView(item))
		}
		writeJSON(w, http.StatusOK, views)
	})

	mux.HandleFunc("POST /api/jobs", func(w http.ResponseWriter, r *http.Request) {
		if runtime == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el runtime de jobs no está disponible"))
			return
		}
		var request createJobRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Type == "" {
			writeError(w, http.StatusBadRequest, errors.New("el tipo de job es obligatorio y el JSON debe ser válido"))
			return
		}
		if len(request.Input) == 0 {
			request.Input = json.RawMessage(`{}`)
		}
		var input any
		if err := json.Unmarshal(request.Input, &input); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("input de job inválido"))
			return
		}
		job, err := runtime.Submit(r.Context(), request.Type, input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, toJobView(job))
	})

	mux.HandleFunc("GET /api/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		if runtime == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el runtime de jobs no está disponible"))
			return
		}
		job, err := runtime.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusNotFound, errors.New("job no encontrado"))
			return
		}
		writeJSON(w, http.StatusOK, toJobView(job))
	})

	mux.HandleFunc("GET /api/jobs/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		if runtime == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el runtime de jobs no está disponible"))
			return
		}
		events, err := runtime.Events(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusNotFound, errors.New("no se pudieron leer los eventos del job"))
			return
		}
		writeJSON(w, http.StatusOK, events)
	})

	mux.HandleFunc("POST /api/jobs/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		if runtime == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el runtime de jobs no está disponible"))
			return
		}
		if err := runtime.Cancel(r.Context(), r.PathValue("id")); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
	})
}

func toJobView(job jobs.Job) jobView {
	view := jobView{
		ID:           job.ID,
		Type:         job.Type,
		Status:       job.Status,
		Result:       job.Result,
		Progress:     job.Progress,
		Stage:        job.Stage,
		Message:      job.Message,
		Attempt:      job.Attempt,
		MaxAttempts:  job.MaxAttempts,
		ErrorCode:    job.ErrorCode,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt.Format("2006-01-02T15:04:05.999Z07:00"),
		UpdatedAt:    job.UpdatedAt.Format("2006-01-02T15:04:05.999Z07:00"),
	}
	if job.StartedAt != nil {
		value := job.StartedAt.Format("2006-01-02T15:04:05.999Z07:00")
		view.StartedAt = &value
	}
	if job.FinishedAt != nil {
		value := job.FinishedAt.Format("2006-01-02T15:04:05.999Z07:00")
		view.FinishedAt = &value
	}
	return view
}
