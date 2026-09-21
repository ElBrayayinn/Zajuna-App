package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/checklist"
	"github.com/zajuna-app/core/internal/evidence"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/security"
	"github.com/zajuna-app/core/internal/zajuna"
)

type checklistTargetParams struct {
	JobID      string
	Input      CaptureChecklistInput
	Target     checklist.CaptureTarget
	BaseURL    *url.URL
	Session    zajuna.Session
	Password   string
	OwnerName  string
	UseBrowser bool
	CookieMu   *sync.Mutex
}

type targetOutcome struct {
	captured        bool
	evidenceRecords int
	failure         string
}

func (w *CaptureChecklistWorker) openChecklistBrowserSession(ctx context.Context, baseURL *url.URL, input CaptureChecklistInput, password string) (*capture.BrowserSession, error) {
	loginURL := *baseURL
	loginURL.Path = "/zajuna/login/index.php"
	loginURL.RawQuery = ""
	return w.runtime.OpenBrowserSession(ctx, capture.BrowserCredentials{
		LoginURL:     loginURL.String(),
		DocumentType: input.DocumentType,
		Document:     input.Username,
		Password:     password,
	})
}

func (w *CaptureChecklistWorker) captureChecklistTarget(ctx context.Context, params checklistTargetParams) targetOutcome {
	target := params.Target
	parsedTarget, parseErr := security.ValidateHTTPURL(target.URL, []string{params.BaseURL.String()}, false)
	if parseErr != nil || parsedTarget.Host == "" || parsedTarget.Scheme != params.BaseURL.Scheme || parsedTarget.Host != params.BaseURL.Host {
		return targetOutcome{failure: target.ItemCode + ": origen de URL no permitido"}
	}
	outputPath := filepath.Join(w.dataDir, "evidences", "checklist", safePathPart(params.Input.FichaID), safePathPart(target.ItemCode), fmt.Sprintf("slot-%d.png", target.SlotNumber))
	options := capture.CaptureOptions{
		Selector: target.CSSSelector, Selectors: target.CSSSelectorFallbacks,
		RevealSelectors: target.RevealSelectors, HideSelectors: target.HideSelectors,
		ViewportWidth: target.ViewportWidth, ViewportHeight: target.ViewportHeight,
		FullPage: target.FullPage, LabelHint: target.LabelHint, OwnerName: params.OwnerName,
		RequireSelector: target.RequireSelector, OwnerOnly: target.OwnerOnly,
	}

	var captureResult capture.CaptureResult
	var captureErr error
	if params.UseBrowser {
		browserSession, err := w.openChecklistBrowserSession(ctx, params.BaseURL, params.Input, params.Password)
		if err != nil {
			return targetOutcome{failure: target.ItemCode + ": " + err.Error()}
		}
		defer browserSession.Close()
		captureResult, captureErr = browserSession.CaptureURLWithMetadataAndOptions(ctx, target.URL, outputPath, options)
	} else {
		if params.CookieMu != nil {
			params.CookieMu.Lock()
		}
		cookies := make([]capture.BrowserCookie, 0)
		for _, cookie := range params.Session.CookiesForURL(target.URL) {
			converted, convErr := capture.BrowserCookieForTarget(cookie, parsedTarget)
			if convErr != nil {
				continue
			}
			cookies = append(cookies, converted)
		}
		if params.CookieMu != nil {
			params.CookieMu.Unlock()
		}
		if len(cookies) == 0 {
			return targetOutcome{failure: target.ItemCode + ": sesión sin cookies para la ruta"}
		}
		captureResult, captureErr = w.runtime.CaptureURLWithMetadataAndCookiesAndOptions(ctx, target.URL, outputPath, cookies, options)
	}
	if captureErr != nil {
		if errors.Is(captureErr, capture.ErrLoginPage) {
			return targetOutcome{failure: target.ItemCode + ": sesión de Zajuna expirada o página de login"}
		}
		if errors.Is(captureErr, capture.ErrChallengePage) {
			return targetOutcome{failure: target.ItemCode + ": Zajuna pidió CAPTCHA o MFA"}
		}
		return targetOutcome{failure: target.ItemCode + ": " + captureErr.Error()}
	}
	if isZajunaLoginURL(captureResult.FinalURL) {
		return targetOutcome{failure: target.ItemCode + ": Zajuna redirigió a login"}
	}
	if _, finalErr := security.ValidateHTTPURL(captureResult.FinalURL, []string{params.BaseURL.String()}, false); finalErr != nil {
		return targetOutcome{failure: target.ItemCode + ": redirección fuera del origen permitido"}
	}
	hash, hashErr := fileSHA256(outputPath)
	if hashErr != nil {
		return targetOutcome{failure: target.ItemCode + ": no se pudo calcular el hash"}
	}
	metadata, _ := json.Marshal(map[string]any{
		"url": security.RedactURL(target.URL), "finalUrl": security.RedactURL(captureResult.FinalURL),
		"title": security.RedactText(captureResult.Title), "routeKey": target.RouteKey, "reviewStatus": target.ReviewStatus,
		"selector": captureResult.Selector, "selectorFallbacks": target.CSSSelectorFallbacks, "selectorMatched": captureResult.SelectorMatched,
		"labelHint": target.LabelHint, "routeKind": target.RouteKind, "groupName": target.GroupName,
		"revealSelectors": target.RevealSelectors, "hideSelectors": target.HideSelectors,
		"viewportWidth": target.ViewportWidth, "viewportHeight": target.ViewportHeight, "fullPage": target.FullPage,
		"phaseSection": target.PhaseSection, "jobId": params.JobID,
		"activityId": target.ActivityID, "activityTitle": target.ActivityTitle, "technical": target.Technical, "ownerOnly": target.OwnerOnly,
		"coveredItemCodes": coveredItemCodes(target), "captureUnitKey": target.RouteKey,
	})
	capturedAt := time.Now().UTC()
	evidenceRecords := 0
	for _, itemCode := range coveredItemCodes(target) {
		evidenceID := artifactID("evidence", params.Input.FichaID, itemCode+"#"+strconv.Itoa(target.SlotNumber), "")
		if err := w.evidence.CreateEvidence(ctx, evidence.Record{
			ID: evidenceID, FichaID: params.Input.FichaID, ItemCode: itemCode, SlotNumber: target.SlotNumber,
			Name: target.Name, FilePath: outputPath, Format: "png", Source: "capture-checklist", SHA256: hash,
			Metadata: metadata, CapturedAt: capturedAt,
		}); err != nil {
			return targetOutcome{failure: itemCode + ": no se pudo registrar la evidencia"}
		}
		evidenceRecords++
	}
	return targetOutcome{captured: true, evidenceRecords: evidenceRecords}
}

// CaptureChecklistTargetInput is the payload for a single-target checklist job.
type CaptureChecklistTargetInput struct {
	ParentJobID  string                  `json:"parentJobId,omitempty"`
	Index        int                     `json:"index"`
	FichaID      string                  `json:"fichaId"`
	Username     string                  `json:"username"`
	DocumentType string                  `json:"documentType"`
	OwnerName    string                  `json:"ownerName,omitempty"`
	UseBrowser   bool                    `json:"useBrowser,omitempty"`
	Target       checklist.CaptureTarget `json:"target"`
}

// CaptureChecklistTargetWorker captures one checklist target as its own job.
// Parent capture-checklist fans out in-process for UI compatibility; this
// worker stays registered so the same unit can also ride the jobs FIFO.
type CaptureChecklistTargetWorker struct {
	parent *CaptureChecklistWorker
}

func NewCaptureChecklistTargetWorker(parent *CaptureChecklistWorker) (*CaptureChecklistTargetWorker, error) {
	if parent == nil {
		return nil, errors.New("capture checklist target worker requires parent worker")
	}
	return &CaptureChecklistTargetWorker{parent: parent}, nil
}

func (w *CaptureChecklistTargetWorker) ID() string { return CaptureChecklistTargetWorkerID }

func (w *CaptureChecklistTargetWorker) Execute(ctx context.Context, job jobs.Job, reporter jobs.Reporter) jobs.Result {
	var input CaptureChecklistTargetInput
	if err := json.Unmarshal(job.Input, &input); err != nil {
		return jobs.Result{ErrorCode: "invalid_input", ErrorMessage: "la entrada de captura de objetivo del checklist no es válida"}
	}
	input.FichaID = strings.TrimSpace(input.FichaID)
	input.Username = strings.TrimSpace(input.Username)
	input.DocumentType = strings.TrimSpace(input.DocumentType)
	if input.DocumentType == "" {
		input.DocumentType = "CC"
	}
	if input.FichaID == "" || input.Username == "" || strings.TrimSpace(input.Target.URL) == "" {
		return jobs.Result{ErrorCode: "invalid_input", ErrorMessage: "fichaId, usuario y objetivo son obligatorios"}
	}
	password, err := w.parent.credentials.Get(input.Username)
	if err != nil || password == "" {
		return jobs.Result{ErrorCode: "credential_unavailable", ErrorMessage: "no se encontró la contraseña de Zajuna en el almacén seguro"}
	}
	session, err := w.parent.client.Login(ctx, zajuna.Credentials{DocumentType: input.DocumentType, Document: input.Username, Password: password})
	if err != nil {
		return jobs.Result{Retryable: retryableZajunaError(err), ErrorCode: "zajuna_login_failed", ErrorMessage: fmt.Sprintf("no se pudo iniciar sesión para el objetivo: %v", err)}
	}
	baseURL, err := url.Parse(session.BaseURL)
	if err != nil || baseURL.Host == "" {
		return jobs.Result{ErrorCode: "invalid_zajuna_session", ErrorMessage: "la sesión de Zajuna no tiene un origen válido"}
	}
	useBrowser := input.UseBrowser || strings.EqualFold(baseURL.Hostname(), "zajuna.sena.edu.co")
	if err := reporter.Progress(ctx, "capture", 20, fmt.Sprintf("Capturando objetivo %s", input.Target.ItemCode)); err != nil {
		return jobs.Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	outcome := w.parent.captureChecklistTarget(ctx, checklistTargetParams{
		JobID: job.ID,
		Input: CaptureChecklistInput{FichaID: input.FichaID, Username: input.Username, DocumentType: input.DocumentType},
		Target: input.Target, BaseURL: baseURL, Session: session, Password: password,
		OwnerName: input.OwnerName, UseBrowser: useBrowser,
	})
	if !outcome.captured {
		return jobs.Result{ErrorCode: "capture_target_failed", ErrorMessage: outcome.failure}
	}
	_ = reporter.Progress(ctx, "completed", 100, "Objetivo capturado")
	return jobs.Result{Output: map[string]any{
		"fichaId": input.FichaID, "itemCode": input.Target.ItemCode, "slotNumber": input.Target.SlotNumber,
		"index": input.Index, "parentJobId": input.ParentJobID, "evidenceRecords": outcome.evidenceRecords,
	}}
}
