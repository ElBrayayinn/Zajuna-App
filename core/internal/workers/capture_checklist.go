package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/checklist"
	"github.com/zajuna-app/core/internal/coursemaps"
	"github.com/zajuna-app/core/internal/evidence"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/secrets"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/zajuna"
)

const (
	CaptureChecklistWorkerID       = "capture-checklist"
	CaptureChecklistTargetWorkerID = "capture-checklist-target"
)

type CaptureChecklistInput struct {
	FichaID      string   `json:"fichaId"`
	Username     string   `json:"username"`
	DocumentType string   `json:"documentType"`
	ItemCodes    []string `json:"itemCodes,omitempty"`
	MaxTargets   int      `json:"maxTargets,omitempty"`
}

type checklistCaptureFichaStore interface {
	GetFicha(context.Context, string) (sqlite.FichaRecord, error)
}

type checklistActivitySelectionStore interface {
	ListSelectedActivityIDs(context.Context, string) (map[string]bool, error)
}

type checklistRouteReviewStore interface {
	ListRouteReviews(context.Context, string) ([]checklist.RouteReview, error)
}

type CaptureChecklistWorker struct {
	runtime     capture.Runtime
	dataDir     string
	client      authenticatedCaptureClient
	credentials secrets.Store
	mapStore    coursemaps.Store
	fichaStore  checklistCaptureFichaStore
	evidence    evidence.Store
	concurrency int
}

func NewCaptureChecklistWorker(runtime capture.Runtime, dataDir string, client authenticatedCaptureClient, credentials secrets.Store, mapStore coursemaps.Store, fichaStore checklistCaptureFichaStore, evidenceStore evidence.Store) (*CaptureChecklistWorker, error) {
	if strings.TrimSpace(dataDir) == "" || client == nil || credentials == nil || mapStore == nil || fichaStore == nil || evidenceStore == nil {
		return nil, errors.New("capture checklist worker requires runtime, data directory, client, credentials and stores")
	}
	return &CaptureChecklistWorker{
		runtime: runtime, dataDir: dataDir, client: client, credentials: credentials,
		mapStore: mapStore, fichaStore: fichaStore, evidence: evidenceStore,
	}, nil
}

func (w *CaptureChecklistWorker) ID() string { return CaptureChecklistWorkerID }

// SetConcurrency limits parallel Chromium sessions used by checklist fan-out.
// Clamped via jobs.ResolveConcurrency (default 2, range 1–4).
func (w *CaptureChecklistWorker) SetConcurrency(n int) {
	w.concurrency = jobs.ResolveConcurrency(n)
}

func (w *CaptureChecklistWorker) fanoutConcurrency() int {
	if w.concurrency < 1 {
		return jobs.DefaultConcurrency
	}
	return w.concurrency
}


func (w *CaptureChecklistWorker) Execute(ctx context.Context, job jobs.Job, reporter jobs.Reporter) jobs.Result {
	var input CaptureChecklistInput
	if err := json.Unmarshal(job.Input, &input); err != nil {
		return jobs.Result{ErrorCode: "invalid_input", ErrorMessage: "la entrada de captura del checklist no es válida"}
	}
	input.FichaID = strings.TrimSpace(input.FichaID)
	input.Username = strings.TrimSpace(input.Username)
	input.DocumentType = strings.TrimSpace(input.DocumentType)
	if input.FichaID == "" || input.Username == "" {
		return jobs.Result{ErrorCode: "invalid_input", ErrorMessage: "fichaId y usuario de Zajuna son obligatorios"}
	}
	if input.DocumentType == "" {
		input.DocumentType = "CC"
	}
	if !w.runtime.Installed() {
		return jobs.Result{ErrorCode: "browser_not_installed", ErrorMessage: fmt.Sprintf("runtime Chromium no instalado en %s; ejecuta npm run browser:install", w.runtime.Root)}
	}

	ficha, err := w.fichaStore.GetFicha(ctx, input.FichaID)
	if err != nil {
		return jobs.Result{ErrorCode: "ficha_not_found", ErrorMessage: "no se encontró la ficha local seleccionada"}
	}
	record, err := w.mapStore.GetCourseMap(ctx, ficha.CourseID)
	if err != nil {
		return jobs.Result{ErrorCode: "course_map_not_found", ErrorMessage: "la ficha todavía no tiene un mapa de rutas; ejecuta Buscar rutas primero"}
	}
	selectedActivityIDs := map[string]bool(nil)
	selectionStore, hasSelectionStore := w.fichaStore.(checklistActivitySelectionStore)
	if hasSelectionStore {
		selectedActivityIDs, err = selectionStore.ListSelectedActivityIDs(ctx, input.FichaID)
		if err != nil {
			return jobs.Result{ErrorCode: "activity_selection_read_failed", ErrorMessage: fmt.Sprintf("no se pudieron leer las actividades seleccionadas: %v", err), Retryable: true}
		}
	}
	if hasSelectionStore && len(selectedActivityIDs) == 0 && captureRequiresActivitySelection(input.ItemCodes) {
		return jobs.Result{ErrorCode: "activities_not_selected", ErrorMessage: "selecciona primero las actividades que pertenecen al instructor para filtrar fechas y evidencias"}
	}
	targets, summary, err := checklist.BuildCaptureTargetsForActivities(record, selectedActivityIDs)
	if err != nil {
		return jobs.Result{ErrorCode: "checklist_map_invalid", ErrorMessage: err.Error()}
	}
	if reviewStore, ok := w.fichaStore.(checklistRouteReviewStore); ok {
		reviews, reviewErr := reviewStore.ListRouteReviews(ctx, input.FichaID)
		if reviewErr != nil {
			return jobs.Result{ErrorCode: "route_review_read_failed", ErrorMessage: "no se pudieron leer las revisiones de rutas", Retryable: true}
		}
		targets = checklist.ApplyRouteReviews(targets, reviews)
	} else {
		targets = checklist.ApplyRouteReviews(targets, nil)
	}
	targets = filterCaptureTargets(targets, input.ItemCodes)
	if input.MaxTargets > 0 && len(targets) > input.MaxTargets {
		targets = targets[:input.MaxTargets]
	}
	summary.CaptureUnitCount = len(targets)
	summary.CoverageCount = captureTargetCoverageCount(targets)
	if len(targets) == 0 {
		return jobs.Result{ErrorCode: "checklist_map_empty", ErrorMessage: "el mapa no tiene rutas asociadas a los items seleccionados del checklist"}
	}
	if err := reporter.Progress(ctx, "credentials", 5, "Preparando captura dirigida por checklist"); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	password, err := w.credentials.Get(input.Username)
	if err != nil || password == "" {
		return jobs.Result{ErrorCode: "credential_unavailable", ErrorMessage: "no se encontró la contraseña de Zajuna en el almacén seguro"}
	}
	if err := reporter.Progress(ctx, "login", 12, "Validando sesión de Zajuna para las evidencias"); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	session, err := w.client.Login(ctx, zajuna.Credentials{DocumentType: input.DocumentType, Document: input.Username, Password: password})
	if err != nil {
		return jobs.Result{Retryable: retryableZajunaError(err), ErrorCode: "zajuna_login_failed", ErrorMessage: fmt.Sprintf("no se pudo iniciar sesión para el checklist: %v", err)}
	}
	baseURL, err := url.Parse(session.BaseURL)
	if err != nil || baseURL.Host == "" {
		return jobs.Result{ErrorCode: "invalid_zajuna_session", ErrorMessage: "la sesión de Zajuna no tiene un origen válido"}
	}
	useBrowser := strings.EqualFold(baseURL.Hostname(), "zajuna.sena.edu.co")
	ownerName := ""
	if useBrowser && checklist.RequiresInstructorIdentity(targets) {
		if err := reporter.Progress(ctx, "identity", 16, "Identificando al instructor autenticado para filtrar foros y anuncios"); err != nil {
			return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
		}
		identitySession, identityErr := w.openChecklistBrowserSession(ctx, baseURL, input, password)
		if identityErr != nil {
			return jobs.Result{Retryable: retryableZajunaError(identityErr) && !errors.Is(identityErr, capture.ErrBlockedPage), ErrorCode: "zajuna_browser_login_failed", ErrorMessage: fmt.Sprintf("no se pudo iniciar sesión en Chromium para el checklist: %v", identityErr)}
		}
		ownerName, err = identitySession.AuthenticatedOwnerName(ctx, record.ProfileURL)
		identitySession.Close()
		if err != nil {
			return jobs.Result{ErrorCode: "instructor_identity_unavailable", ErrorMessage: fmt.Sprintf("no se pudo verificar el instructor autenticado: %v", err)}
		}
	}

	targetItemCodes := make(map[string]bool)
	for _, target := range targets {
		for _, itemCode := range coveredItemCodes(target) {
			targetItemCodes[itemCode] = true
		}
	}

	outcomes := make([]targetOutcome, len(targets))
	var completed int64
	concurrency := w.fanoutConcurrency()
	if err := reporter.Progress(ctx, "capture", 18, fmt.Sprintf("Capturando %d evidencias con hasta %d sesiones en paralelo", len(targets), concurrency)); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}

	// Contiguous FIFO fan-out of target indices. Each in-flight target uses its
	// own Chromium session when browser auth is required: BrowserSession is not
	// safe to share across goroutines. Trade-off: up to C parallel logins.
	var cookieMu sync.Mutex
	fanoutErr := orderedFanout(ctx, len(targets), concurrency, func(taskCtx context.Context, index int) error {
		target := targets[index]
		if err := taskCtx.Err(); err != nil {
			outcomes[index] = targetOutcome{failure: target.ItemCode + ": captura cancelada"}
			return err
		}
		outcome := w.captureChecklistTarget(taskCtx, checklistTargetParams{
			JobID:      job.ID,
			Input:      input,
			Target:     target,
			BaseURL:    baseURL,
			Session:    session,
			Password:   password,
			OwnerName:  ownerName,
			UseBrowser: useBrowser,
			CookieMu:   &cookieMu,
		})
		outcomes[index] = outcome
		done := int(atomic.AddInt64(&completed, 1))
		percent := 18 + ((done * 76) / len(targets))
		_ = reporter.Progress(taskCtx, "capture", percent, fmt.Sprintf("Captura checklist %d de %d", done, len(targets)))
		if outcome.captured {
			_ = reporter.Event(taskCtx, "evidence_captured", "Evidencia guardada", map[string]any{
				"itemCode": target.ItemCode, "coveredItemCodes": coveredItemCodes(target),
				"slotNumber": target.SlotNumber, "index": index,
			})
		}
		return nil
	})
	if fanoutErr != nil && ctx.Err() == nil && !errors.Is(fanoutErr, context.Canceled) {
		return jobs.Result{ErrorCode: "capture_fanout_failed", ErrorMessage: fanoutErr.Error()}
	}
	if err := ctx.Err(); err != nil {
		return jobs.Result{ErrorCode: "capture_cancelled", ErrorMessage: err.Error()}
	}

	captured := 0
	evidenceRecords := 0
	failed := 0
	failures := make([]string, 0)
	for _, outcome := range outcomes {
		if outcome.captured {
			captured++
			evidenceRecords += outcome.evidenceRecords
			continue
		}
		failed++
		if outcome.failure != "" {
			failures = append(failures, outcome.failure)
		}
	}

	groupCount := 0
	if groupStore, ok := w.evidence.(evidence.GroupStore); ok {
		groups, groupErr := groupStore.RebuildEvidenceGroups(ctx, input.FichaID)
		if groupErr != nil {
			return jobs.Result{ErrorCode: "evidence_group_failed", ErrorMessage: fmt.Sprintf("no se pudieron construir los grupos de evidencia: %v", groupErr), Retryable: true}
		}
		groupCount = len(groups)
		_ = reporter.Event(ctx, "evidence_groups_rebuilt", "Evidencias agrupadas para evitar duplicados", map[string]any{"fichaId": input.FichaID, "groupCount": groupCount})
	}
	if err := reporter.Progress(ctx, "completed", 100, fmt.Sprintf("Captura dirigida terminada: %d guardadas, %d con error", captured, failed)); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	if failed > 0 {
		message := fmt.Sprintf("captura incompleta: %d guardadas, %d con error", captured, failed)
		if len(failures) > 0 {
			message += ". Primer error: " + failures[0]
		}
		return jobs.Result{ErrorCode: "capture_partial_failure", ErrorMessage: message}
	}
	return jobs.Result{Output: map[string]any{
		"fichaId": input.FichaID, "courseId": ficha.CourseID, "targets": len(targets), "captured": captured,
		"failed": failed, "unresolved": summary.UnresolvedItems, "slotCount": len(targets), "captureUnitCount": len(targets), "coverageCount": evidenceRecords,
		"targetItems": len(targetItemCodes), "itemCount": summary.ItemCount, "groupCount": groupCount, "failures": failures,
	}}
}

func captureRequiresActivitySelection(itemCodes []string) bool {
	if len(itemCodes) == 0 {
		return true
	}
	for _, itemCode := range itemCodes {
		switch strings.TrimSpace(itemCode) {
		case "6.1", "10.1.1", "10.1.2":
			return true
		}
	}
	return false
}

func filterCaptureTargets(targets []checklist.CaptureTarget, itemCodes []string) []checklist.CaptureTarget {
	if len(itemCodes) == 0 {
		return targets
	}
	allowed := make(map[string]bool, len(itemCodes))
	for _, code := range itemCodes {
		if code = strings.TrimSpace(code); code != "" {
			allowed[code] = true
		}
	}
	filtered := make([]checklist.CaptureTarget, 0, len(targets))
	for _, target := range targets {
		if allowed[target.ItemCode] || anyAllowedCoverage(target, allowed) {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

func coveredItemCodes(target checklist.CaptureTarget) []string {
	if len(target.CoveredItemCodes) == 0 {
		return []string{target.ItemCode}
	}
	return target.CoveredItemCodes
}

func anyAllowedCoverage(target checklist.CaptureTarget, allowed map[string]bool) bool {
	for _, itemCode := range coveredItemCodes(target) {
		if allowed[itemCode] {
			return true
		}
	}
	return false
}

func captureTargetCoverageCount(targets []checklist.CaptureTarget) int {
	count := 0
	for _, target := range targets {
		count += len(coveredItemCodes(target))
	}
	return count
}

func safePathPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	value = strings.NewReplacer("\\", "_", "/", "_", ":", "_", "..", "_").Replace(value)
	return value
}
