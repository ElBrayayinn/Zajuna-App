package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/secrets"
	"github.com/zajuna-app/core/internal/zajuna"
)

const TestZajunaConnectionWorkerID = "test-zajuna-connection"

type testZajunaConnectionInput struct {
	Username     string `json:"username"`
	DocumentType string `json:"documentType"`
}

type TestZajunaConnectionWorker struct {
	client      zajunaClient
	credentials secrets.Store
}

func NewTestZajunaConnectionWorker(client zajunaClient, credentials secrets.Store) (*TestZajunaConnectionWorker, error) {
	if client == nil || credentials == nil {
		return nil, errors.New("test Zajuna connection worker requires client and credentials")
	}
	return &TestZajunaConnectionWorker{client: client, credentials: credentials}, nil
}

func (w *TestZajunaConnectionWorker) ID() string { return TestZajunaConnectionWorkerID }

func (w *TestZajunaConnectionWorker) Execute(ctx context.Context, job jobs.Job, reporter jobs.Reporter) jobs.Result {
	var input testZajunaConnectionInput
	if err := json.Unmarshal(job.Input, &input); err != nil {
		return jobs.Result{ErrorCode: "invalid_input", ErrorMessage: "la entrada de prueba de conexión no es válida"}
	}
	if input.Username == "" {
		return jobs.Result{ErrorCode: "missing_username", ErrorMessage: "configura el usuario o documento de Zajuna"}
	}
	if err := reporter.Progress(ctx, "credentials", 10, "Preparando sesión segura de Zajuna"); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	password, err := w.credentials.Get(input.Username)
	if err != nil || password == "" {
		return jobs.Result{ErrorCode: "credential_unavailable", ErrorMessage: "no se encontró la contraseña de Zajuna en el almacén seguro"}
	}
	if err := reporter.Progress(ctx, "login", 35, "Iniciando sesión en Zajuna"); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	session, err := w.client.Login(ctx, zajuna.Credentials{DocumentType: input.DocumentType, Document: input.Username, Password: password})
	if err != nil {
		return jobs.Result{Retryable: retryableZajunaError(err), ErrorCode: "zajuna_login_failed", ErrorMessage: fmt.Sprintf("no se pudo iniciar sesión en Zajuna: %v", err)}
	}
	if err := reporter.Progress(ctx, "courses", 75, "Validando acceso a Mis cursos"); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	fichas, err := w.client.ListFichas(ctx, session)
	if err != nil {
		return jobs.Result{Retryable: retryableZajunaError(err), ErrorCode: "zajuna_courses_failed", ErrorMessage: fmt.Sprintf("no se pudo validar Mis cursos: %v", err)}
	}
	if err := reporter.Progress(ctx, "completed", 100, fmt.Sprintf("Conexión válida; %d fichas disponibles", len(fichas))); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	return jobs.Result{Output: map[string]any{"authenticated": true, "fichas": len(fichas)}}
}
