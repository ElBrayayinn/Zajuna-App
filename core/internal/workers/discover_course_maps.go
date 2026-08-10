package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zajuna-app/core/internal/coursemaps"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/secrets"
	"github.com/zajuna-app/core/internal/zajuna"
)

const DiscoverCourseMapsWorkerID = "discover-course-maps"

type discoverCourseMapsInput struct {
	Username        string   `json:"username"`
	DocumentType    string   `json:"documentType"`
	CourseIDs       []string `json:"courseIds"`
	MaxDepth        int      `json:"maxDepth"`
	MaxPages        int      `json:"maxPages"`
	MaxLinksPerPage int      `json:"maxLinksPerPage"`
}

type courseMapClient interface {
	Login(ctx context.Context, credentials zajuna.Credentials) (zajuna.Session, error)
	DiscoverCourseMap(ctx context.Context, session zajuna.Session, courseID string, options zajuna.CrawlOptions) (coursemaps.Record, error)
}

type DiscoverCourseMapsWorker struct {
	client      courseMapClient
	credentials secrets.Store
	store       coursemaps.Store
}

func NewDiscoverCourseMapsWorker(client courseMapClient, credentials secrets.Store, store coursemaps.Store) (*DiscoverCourseMapsWorker, error) {
	if client == nil || credentials == nil || store == nil {
		return nil, errors.New("discover course maps worker requires client, credentials and store")
	}
	return &DiscoverCourseMapsWorker{client: client, credentials: credentials, store: store}, nil
}

func (w *DiscoverCourseMapsWorker) ID() string { return DiscoverCourseMapsWorkerID }

func (w *DiscoverCourseMapsWorker) Execute(ctx context.Context, job jobs.Job, reporter jobs.Reporter) jobs.Result {
	var input discoverCourseMapsInput
	if err := json.Unmarshal(job.Input, &input); err != nil {
		return jobs.Result{ErrorCode: "invalid_input", ErrorMessage: "la entrada del descubrimiento no es válida"}
	}
	if input.Username == "" {
		return jobs.Result{ErrorCode: "missing_username", ErrorMessage: "configura el usuario o documento de Zajuna"}
	}
	if len(input.CourseIDs) == 0 {
		return jobs.Result{ErrorCode: "missing_courses", ErrorMessage: "sin cursos para descubrir"}
	}
	if err := reporter.Progress(ctx, "credentials", 5, "Preparando sesión segura de Zajuna"); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	password, err := w.credentials.Get(input.Username)
	if err != nil || password == "" {
		return jobs.Result{ErrorCode: "credential_unavailable", ErrorMessage: "no se encontró la contraseña de Zajuna en el almacén seguro"}
	}
	if err := reporter.Progress(ctx, "login", 15, "Validando sesión para descubrir rutas"); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	session, err := w.client.Login(ctx, zajuna.Credentials{DocumentType: input.DocumentType, Document: input.Username, Password: password})
	if err != nil {
		return jobs.Result{Retryable: retryableZajunaError(err), ErrorCode: "zajuna_login_failed", ErrorMessage: fmt.Sprintf("no se pudo iniciar sesión en Zajuna: %v", err)}
	}

	created := 0
	links := 0
	warnings := 0
	for index, courseID := range input.CourseIDs {
		progress := 20 + ((index * 70) / len(input.CourseIDs))
		if err := reporter.Progress(ctx, "discovering", progress, fmt.Sprintf("Descubriendo rutas del curso %s (%d de %d)", courseID, index+1, len(input.CourseIDs))); err != nil {
			return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
		}
		record, err := w.client.DiscoverCourseMap(ctx, session, courseID, zajuna.CrawlOptions{
			MaxDepth:        input.MaxDepth,
			MaxPages:        input.MaxPages,
			MaxLinksPerPage: input.MaxLinksPerPage,
		})
		if errors.Is(err, zajuna.ErrSessionExpired) {
			if progressErr := reporter.Progress(ctx, "login", progress, "Renovando sesión de Zajuna"); progressErr != nil {
				return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: progressErr.Error()}
			}
			session, err = w.client.Login(ctx, zajuna.Credentials{DocumentType: input.DocumentType, Document: input.Username, Password: password})
			if err == nil {
				record, err = w.client.DiscoverCourseMap(ctx, session, courseID, zajuna.CrawlOptions{
					MaxDepth:        input.MaxDepth,
					MaxPages:        input.MaxPages,
					MaxLinksPerPage: input.MaxLinksPerPage,
				})
			}
		}
		if err != nil {
			return jobs.Result{Retryable: retryableZajunaError(err), ErrorCode: "course_map_failed", ErrorMessage: fmt.Sprintf("no se pudo descubrir el curso %s: %v", courseID, err)}
		}
		if err := w.store.CreateOrReplaceCourseMap(ctx, record); err != nil {
			return jobs.Result{ErrorCode: "course_map_persist_failed", ErrorMessage: fmt.Sprintf("no se pudo guardar el mapa del curso %s: %v", courseID, err)}
		}
		created++
		links += record.LinkCount
		if record.Warning != "" {
			warnings++
		}
	}
	if err := reporter.Progress(ctx, "completed", 100, fmt.Sprintf("Se guardaron %d mapas locales", created)); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	return jobs.Result{Output: map[string]any{
		"courses":  created,
		"links":    links,
		"warnings": warnings,
	}}
}
