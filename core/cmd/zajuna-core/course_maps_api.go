package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/zajuna-app/core/internal/coursemaps"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/zajuna"
)

type discoverCourseMapsRequest struct {
	Username        string   `json:"username"`
	DocumentType    string   `json:"documentType"`
	CourseIDs       []string `json:"courseIds"`
	MaxDepth        int      `json:"maxDepth"`
	MaxPages        int      `json:"maxPages"`
	MaxLinksPerPage int      `json:"maxLinksPerPage"`
}

type importCourseActivitiesRequest struct {
	CourseID   string                `json:"courseId"`
	ProfileURL string                `json:"profileUrl,omitempty"`
	PageLinks  []zajuna.ActivityLink `json:"pageLinks"`
	Jump       []zajuna.ActivityLink `json:"jump"`
}

func registerCourseMapRoutes(mux *http.ServeMux, store coursemaps.Store, fichas fichaLister, runtime *jobs.Runtime, dataDir string) {
	mux.HandleFunc("GET /api/course-maps", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el almacenamiento de mapas no está disponible"))
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
		items, err := store.ListCourseMaps(r.Context(), limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	})

	mux.HandleFunc("GET /api/course-maps/{courseId}", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el almacenamiento de mapas no está disponible"))
			return
		}
		courseID := strings.TrimSpace(r.PathValue("courseId"))
		if courseID == "" {
			writeError(w, http.StatusBadRequest, errors.New("el curso es obligatorio"))
			return
		}
		item, err := store.GetCourseMap(r.Context(), courseID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, errors.New("no existe un mapa local para ese curso"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	})

	mux.HandleFunc("POST /api/course-maps/import-activities", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el almacenamiento de mapas no está disponible"))
			return
		}
		var request importCourseActivitiesRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("el archivo de actividades es inválido"))
			return
		}
		if len(request.PageLinks)+len(request.Jump) > 5000 {
			writeError(w, http.StatusBadRequest, errors.New("el mapa importado supera el límite de 5000 enlaces"))
			return
		}
		activities := make([]zajuna.ActivityLink, 0, len(request.PageLinks)+len(request.Jump))
		seen := make(map[string]bool, cap(activities))
		for _, activity := range append(request.PageLinks, request.Jump...) {
			activity.URL = strings.TrimSpace(activity.URL)
			activity.Label = strings.TrimSpace(activity.Label)
			if activity.URL == "" || seen[activity.URL] {
				continue
			}
			seen[activity.URL] = true
			activities = append(activities, activity)
		}
		if len(activities) == 0 {
			writeError(w, http.StatusBadRequest, errors.New("el mapa importado no contiene enlaces de actividades"))
			return
		}
		record, err := zajuna.BuildCourseMapFromActivities(request.CourseID, request.ProfileURL, activities)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := store.CreateOrReplaceCourseMap(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, record)
	})

	mux.HandleFunc("POST /api/course-maps/discover", func(w http.ResponseWriter, r *http.Request) {
		if runtime == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("el runtime de jobs no está disponible"))
			return
		}
		var request discoverCourseMapsRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
				writeError(w, http.StatusBadRequest, errors.New("el input de descubrimiento es inválido"))
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
		request.CourseIDs = uniqueNonEmpty(request.CourseIDs)
		if len(request.CourseIDs) == 0 && fichas != nil {
			items, err := fichas.ListFichas(r.Context(), 100)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			for _, item := range items {
				if strings.TrimSpace(item.CourseID) != "" {
					request.CourseIDs = append(request.CourseIDs, item.CourseID)
				}
			}
			request.CourseIDs = uniqueNonEmpty(request.CourseIDs)
		}
		if len(request.CourseIDs) == 0 {
			writeError(w, http.StatusBadRequest, errors.New("sin cursos locales; sincroniza primero Mis cursos o envía courseIds"))
			return
		}
		if request.MaxDepth < 0 || request.MaxDepth > 6 || request.MaxPages < 0 || request.MaxPages > 500 || request.MaxLinksPerPage < 0 || request.MaxLinksPerPage > 1000 {
			writeError(w, http.StatusBadRequest, errors.New("los límites de descubrimiento están fuera de rango"))
			return
		}
		job, err := runtime.Submit(r.Context(), "discover-course-maps", request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, toJobView(job))
	})
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
