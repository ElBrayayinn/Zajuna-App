package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/zajuna-app/core/internal/storage/sqlite"
)

type notificationStore interface {
	ListNotifications(ctx context.Context, limit int) ([]sqlite.NotificationRecord, error)
	MarkNotificationRead(ctx context.Context, id string) error
	MarkAllNotificationsRead(ctx context.Context) error
}

type notificationView struct {
	ID        string  `json:"id"`
	Kind      string  `json:"kind"`
	Title     string  `json:"title"`
	Message   string  `json:"message"`
	JobID     string  `json:"jobId,omitempty"`
	ReadAt    *string `json:"readAt,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

func registerNotificationRoutes(mux *http.ServeMux, store notificationStore) {
	mux.HandleFunc("GET /api/notifications", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("las notificaciones locales no están disponibles"))
			return
		}
		items, err := store.ListNotifications(r.Context(), 50)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		views := make([]notificationView, 0, len(items))
		for _, item := range items {
			views = append(views, toNotificationView(item))
		}
		writeJSON(w, http.StatusOK, views)
	})

	mux.HandleFunc("POST /api/notifications/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("las notificaciones locales no están disponibles"))
			return
		}
		if err := store.MarkNotificationRead(r.Context(), r.PathValue("id")); err != nil {
			writeError(w, http.StatusNotFound, errors.New("notificación no encontrada"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"read": true})
	})

	mux.HandleFunc("POST /api/notifications/read-all", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("las notificaciones locales no están disponibles"))
			return
		}
		if err := store.MarkAllNotificationsRead(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"read": true})
	})
}

func toNotificationView(item sqlite.NotificationRecord) notificationView {
	view := notificationView{ID: item.ID, Kind: item.Kind, Title: item.Title, Message: item.Message, JobID: item.JobID, CreatedAt: item.CreatedAt.Format(time.RFC3339Nano)}
	if item.ReadAt != nil {
		value := item.ReadAt.Format(time.RFC3339Nano)
		view.ReadAt = &value
	}
	return view
}
