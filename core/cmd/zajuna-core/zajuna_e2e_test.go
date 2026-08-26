package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/coursemaps"
	"github.com/zajuna-app/core/internal/evidence"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/workers"
	"github.com/zajuna-app/core/internal/zajuna"
)

func TestAuthenticatedZajunaE2E(t *testing.T) {
	if os.Getenv("ZAJUNA_E2E") != "1" {
		t.Skip("authenticated Zajuna E2E disabled")
	}
	username := strings.TrimSpace(os.Getenv("ZAJUNA_TEST_USERNAME"))
	password := os.Getenv("ZAJUNA_TEST_PASSWORD")
	if username == "" || password == "" {
		t.Fatal("define ZAJUNA_TEST_USERNAME and ZAJUNA_TEST_PASSWORD; credentials are never printed or committed")
	}
	documentType := strings.TrimSpace(os.Getenv("ZAJUNA_TEST_DOCUMENT_TYPE"))
	if documentType == "" {
		documentType = "CC"
	}
	baseURL := strings.TrimSpace(os.Getenv("ZAJUNA_TEST_BASE_URL"))
	client, err := zajuna.NewClient(baseURL)
	if err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	credentials := &memoryCredentialStore{}
	runtime, err := jobs.NewRuntime(store, 1)
	if err != nil {
		t.Fatal(err)
	}
	syncWorker, err := workers.NewSyncFichasWorker(client, credentials, store)
	if err != nil {
		t.Fatal(err)
	}
	connectionWorker, err := workers.NewTestZajunaConnectionWorker(client, credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Register(connectionWorker); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Register(syncWorker); err != nil {
		t.Fatal(err)
	}
	mapWorker, err := workers.NewDiscoverCourseMapsWorker(client, credentials, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Register(mapWorker); err != nil {
		t.Fatal(err)
	}
	captureWorker, err := workers.NewAuthenticatedCaptureBrowserWorker(capture.Resolve(os.Getenv("ZAJUNA_PLAYWRIGHT_DIR")), dataDir, client, credentials, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Register(captureWorker); err != nil {
		t.Fatal(err)
	}
	checklistCaptureWorker, err := workers.NewCaptureChecklistWorker(capture.Resolve(os.Getenv("ZAJUNA_PLAYWRIGHT_DIR")), dataDir, client, credentials, store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Register(checklistCaptureWorker); err != nil {
		t.Fatal(err)
	}
	runtime.Start(context.Background())
	defer runtime.Close()
	server := httptest.NewServer(newRouterWithServices(dataDir, credentials, runtime, store, nil))
	defer server.Close()

	setupResponse := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/setup", `{"zajunaUsername":"`+escapeJSON(username)+`","zajunaPassword":"`+escapeJSON(password)+`"}`)
	if setupResponse.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, body = %s", setupResponse.StatusCode, setupResponse.Body)
	}
	configBytes, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(configBytes), password) {
		t.Fatal("password was persisted in config.json")
	}

	connectionResponse := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/zajuna/test-connection", `{}`)
	if connectionResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("connection test status = %d, body = %s", connectionResponse.StatusCode, connectionResponse.Body)
	}
	var createdConnection jobView
	decodeJSON(t, connectionResponse.Body, &createdConnection)
	completedConnection := waitForE2EJobTimeout(t, server.Client(), server.URL, createdConnection.ID, 2*time.Minute)
	if completedConnection.Status != jobs.StatusCompleted {
		t.Fatalf("connection test ended with status %s, code = %s, message = %s", completedConnection.Status, completedConnection.ErrorCode, completedConnection.ErrorMessage)
	}
	var connectionOutput map[string]any
	decodeJSON(t, string(completedConnection.Result), &connectionOutput)
	if connectionOutput["authenticated"] != true {
		t.Fatalf("connection test did not report authenticated=true: %#v", connectionOutput)
	}

	syncResponse := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/fichas/sync", `{"documentType":"`+escapeJSON(documentType)+`"}`)
	if syncResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("sync status = %d, body = %s", syncResponse.StatusCode, syncResponse.Body)
	}
	var created jobView
	decodeJSON(t, syncResponse.Body, &created)
	completed := waitForE2EJobTimeout(t, server.Client(), server.URL, created.ID, 2*time.Minute)
	if completed.Status != jobs.StatusCompleted {
		t.Fatalf("authenticated sync ended with status %s, code = %s, message = %s", completed.Status, completed.ErrorCode, completed.ErrorMessage)
	}

	fichas := getE2EList(t, server.Client(), server.URL+"/api/fichas?limit=100")
	if len(fichas) == 0 {
		t.Fatal("authenticated sync completed without local fichas")
	}
	t.Logf("Zajuna authenticated E2E: connection verified and %d fichas persisted locally", len(fichas))
	eventsResponse := doJSON(t, server.Client(), http.MethodGet, server.URL+"/api/jobs/"+created.ID+"/events", "")
	if eventsResponse.StatusCode != http.StatusOK {
		t.Fatalf("events status = %d, body = %s", eventsResponse.StatusCode, eventsResponse.Body)
	}
	var events []jobs.Event
	decodeJSON(t, eventsResponse.Body, &events)
	if !hasJobStage(events, "login") || !hasJobStage(events, "courses") {
		t.Fatalf("authenticated sync did not expose login/courses stages: %#v", events)
	}

	if os.Getenv("ZAJUNA_MAPS_E2E") == "1" || os.Getenv("ZAJUNA_CAPTURE_E2E") == "1" {
		// The account owns several fichas. ZAJUNA_SELECTOR_FICHA_INDEX picks which
		// one to map and register, so the same run can be repeated on a second
		// real course to tell a fragile rule from a course-specific layout.
		fichaIndex := 0
		if configured := strings.TrimSpace(os.Getenv("ZAJUNA_SELECTOR_FICHA_INDEX")); configured != "" {
			parsed, err := strconv.Atoi(configured)
			if err != nil || parsed < 0 || parsed >= len(fichas) {
				t.Fatalf("ZAJUNA_SELECTOR_FICHA_INDEX must be between 0 and %d, got %q", len(fichas)-1, configured)
			}
			fichaIndex = parsed
		}
		selectedFicha := fichas[fichaIndex]
		mapCourseID, _ := selectedFicha["courseId"].(string)
		if mapCourseID == "" {
			t.Fatal("the authenticated sync returned no course id for map E2E")
		}
		t.Logf("Zajuna authenticated E2E: using ficha %d of %d (course %s)", fichaIndex+1, len(fichas), mapCourseID)
		mapsInput := `{"documentType":"` + escapeJSON(documentType) + `","courseIds":["` + escapeJSON(mapCourseID) + `"],"maxDepth":1,"maxPages":20,"maxLinksPerPage":150}`
		mapsResponse := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/course-maps/discover", mapsInput)
		if mapsResponse.StatusCode != http.StatusAccepted {
			t.Fatalf("course map discovery status = %d, body = %s", mapsResponse.StatusCode, mapsResponse.Body)
		}
		var createdMaps jobView
		decodeJSON(t, mapsResponse.Body, &createdMaps)
		completedMaps := waitForE2EJobTimeout(t, server.Client(), server.URL, createdMaps.ID, 2*time.Minute)
		if completedMaps.Status != jobs.StatusCompleted {
			t.Fatalf("course map discovery ended with status %s, code = %s, message = %s", completedMaps.Status, completedMaps.ErrorCode, completedMaps.ErrorMessage)
		}
		mapsListResponse := doJSON(t, server.Client(), http.MethodGet, server.URL+"/api/course-maps?limit=100", "")
		if mapsListResponse.StatusCode != http.StatusOK {
			t.Fatalf("course maps list status = %d, body = %s", mapsListResponse.StatusCode, mapsListResponse.Body)
		}
		var maps []coursemaps.Record
		decodeJSON(t, mapsListResponse.Body, &maps)
		if len(maps) == 0 {
			t.Fatal("course map discovery completed without local maps")
		}
		totalLinks := 0
		for _, item := range maps {
			totalLinks += item.LinkCount
		}
		if totalLinks == 0 {
			t.Fatal("course map discovery returned no links")
		}
		t.Logf("Zajuna authenticated route-map E2E: %d course maps and %d routes persisted locally", len(maps), totalLinks)

		if os.Getenv("ZAJUNA_CAPTURE_E2E") == "1" {
			if os.Getenv("ZAJUNA_PLAYWRIGHT_DIR") == "" {
				t.Fatal("define ZAJUNA_PLAYWRIGHT_DIR for authenticated capture E2E")
			}
			captureURL := maps[0].CourseURL
			for _, route := range maps[0].Routes {
				if route.Kind == "assign" || route.Kind == "phase" {
					captureURL = route.URL
					break
				}
			}
			captureResponse := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/jobs", `{"type":"capture-browser","input":{"url":"`+escapeJSON(captureURL)+`","authenticated":true,"username":"`+escapeJSON(username)+`","documentType":"`+escapeJSON(documentType)+`"}}`)
			if captureResponse.StatusCode != http.StatusAccepted {
				t.Fatalf("authenticated capture status = %d, body = %s", captureResponse.StatusCode, captureResponse.Body)
			}
			var createdCapture jobView
			decodeJSON(t, captureResponse.Body, &createdCapture)
			completedCapture := waitForE2EJobTimeout(t, server.Client(), server.URL, createdCapture.ID, 2*time.Minute)
			if completedCapture.Status != jobs.StatusCompleted {
				t.Fatalf("authenticated capture ended with status %s, code = %s, message = %s", completedCapture.Status, completedCapture.ErrorCode, completedCapture.ErrorMessage)
			}
			var captureOutput map[string]any
			decodeJSON(t, string(completedCapture.Result), &captureOutput)
			if captureOutput["authenticated"] != true {
				t.Fatalf("capture did not report authenticated=true: %#v", captureOutput)
			}
			t.Logf("Zajuna authenticated capture E2E: route captured locally")

			registerZajunaSelectors(t, server, store, selectedFicha, username, documentType)
		}
	}
}

// registerZajunaSelectors closes the MDL-33 P1 item: it drives the real
// checklist capture against a live course and writes down which Zajuna selector
// actually produced each piece of evidence. Set ZAJUNA_SELECTOR_REPORT to the
// artefact path to commit the register; without it the report is written beside
// the temporary data directory and only summarised in the log.
func registerZajunaSelectors(t *testing.T, server *httptest.Server, store *sqlite.Store, ficha map[string]any, username, documentType string) {
	t.Helper()
	fichaID, _ := ficha["id"].(string)
	if fichaID == "" {
		t.Fatal("the authenticated sync returned no ficha id for the selector register")
	}
	// The checklist capture refuses to run until the operator has said which
	// activities belong to the instructor, so the register reproduces that step
	// against the real course map instead of bypassing it.
	selectRealCourseActivities(t, server, fichaID)

	maxTargets := 12
	if configured := strings.TrimSpace(os.Getenv("ZAJUNA_SELECTOR_MAX_TARGETS")); configured != "" {
		parsed, err := strconv.Atoi(configured)
		if err != nil || parsed <= 0 || parsed > 200 {
			t.Fatalf("ZAJUNA_SELECTOR_MAX_TARGETS must be between 1 and 200, got %q", configured)
		}
		maxTargets = parsed
	}
	input := `{"fichaId":"` + escapeJSON(fichaID) + `","username":"` + escapeJSON(username) +
		`","documentType":"` + escapeJSON(documentType) + `","maxTargets":` + strconv.Itoa(maxTargets) + `}`
	response := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/checklist/capture", input)
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("checklist capture status = %d, body = %s", response.StatusCode, response.Body)
	}
	var created jobView
	decodeJSON(t, response.Body, &created)
	// A real course exposes hundreds of routes, so allow more time than the
	// single-route capture above.
	completed := waitForE2EJobTimeout(t, server.Client(), server.URL, created.ID, 20*time.Minute)
	// A live course can legitimately leave one rule unmatched, and the register
	// exists to record exactly that. Only a run that saved nothing is a failure.
	outcome := string(completed.Status)
	switch {
	case completed.Status == jobs.StatusCompleted:
	case completed.ErrorCode == "capture_partial_failure":
		outcome = completed.ErrorCode + ": " + completed.ErrorMessage
		t.Logf("Zajuna selector register: partial capture recorded as a finding: %s", completed.ErrorMessage)
	default:
		t.Fatalf("checklist capture ended with status %s, code = %s, message = %s", completed.Status, completed.ErrorCode, completed.ErrorMessage)
	}

	records, err := store.ListEvidencesByFicha(context.Background(), fichaID, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatal("the checklist capture produced no evidence to register selectors from")
	}
	report := evidence.BuildSelectorReport(fichaID, time.Now(), records)
	report.CaptureOutcome = evidence.RouteOnly(outcome)
	if report.CaptureUnits == 0 {
		t.Fatal("the selector register came out empty")
	}
	if report.MatchedUnits == 0 {
		t.Fatalf("no selector matched on the real course; the register would document only fallbacks: %+v", report.Usage)
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// The ficha id identifies a real SENA group, so it never reaches the
	// committed artefact.
	report.FichaID = ""
	if target := strings.TrimSpace(os.Getenv("ZAJUNA_SELECTOR_REPORT")); target != "" {
		anonymised, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, append(anonymised, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("Zajuna selector register written to %s", target)
	} else {
		t.Logf("Zajuna selector register (set ZAJUNA_SELECTOR_REPORT to persist it):\n%s", encoded)
	}
	t.Logf("Zajuna selector register: %d capture units, %d checklist items, %d matched (%d via a coarser fallback), %d fell back to full page",
		report.CaptureUnits, report.ItemsCovered, report.MatchedUnits, report.FallbackChainUnits, report.FullPageUnits)
}

func waitForE2EJobTimeout(t *testing.T, client *http.Client, baseURL, id string, timeout time.Duration) jobView {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response := doJSON(t, client, http.MethodGet, baseURL+"/api/jobs/"+id, "")
		if response.StatusCode != http.StatusOK {
			t.Fatalf("job %s status = %d, body = %s", id, response.StatusCode, response.Body)
		}
		var view jobView
		decodeJSON(t, response.Body, &view)
		if view.Status == jobs.StatusCompleted || view.Status == jobs.StatusFailed || view.Status == jobs.StatusCancelled {
			return view
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("job %s did not finish before timeout", id)
	return jobView{}
}

func hasJobStage(events []jobs.Event, wanted string) bool {
	for _, event := range events {
		if event.Stage == wanted {
			return true
		}
	}
	return false
}

func escapeJSON(value string) string {
	encoded, _ := json.Marshal(value)
	return strings.Trim(string(encoded), `"`)
}

// selectRealCourseActivities mirrors the operator step that scopes the
// checklist to the instructor's own activities. ZAJUNA_SELECTOR_ACTIVITIES caps
// how many are selected so one run stays bounded on a course with hundreds of
// routes.
func selectRealCourseActivities(t *testing.T, server *httptest.Server, fichaID string) {
	t.Helper()
	listResponse := doJSON(t, server.Client(), http.MethodGet, server.URL+"/api/checklist/activities?fichaId="+url.QueryEscape(fichaID), "")
	if listResponse.StatusCode != http.StatusOK {
		t.Fatalf("checklist activities status = %d, body = %s", listResponse.StatusCode, listResponse.Body)
	}
	var view struct {
		MapReady   bool `json:"mapReady"`
		Activities []struct {
			ID        string `json:"id"`
			Technical bool   `json:"technical"`
		} `json:"activities"`
	}
	decodeJSON(t, listResponse.Body, &view)
	if !view.MapReady || len(view.Activities) == 0 {
		t.Fatalf("the real course map exposed no activities to select: mapReady=%v count=%d", view.MapReady, len(view.Activities))
	}
	limit := 6
	if configured := strings.TrimSpace(os.Getenv("ZAJUNA_SELECTOR_ACTIVITIES")); configured != "" {
		parsed, err := strconv.Atoi(configured)
		if err != nil || parsed <= 0 {
			t.Fatalf("ZAJUNA_SELECTOR_ACTIVITIES must be a positive number, got %q", configured)
		}
		limit = parsed
	}
	ids := make([]string, 0, limit)
	for _, activity := range view.Activities {
		if len(ids) >= limit {
			break
		}
		if activity.ID != "" {
			ids = append(ids, activity.ID)
		}
	}
	encodedIDs, err := json.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	putResponse := doJSON(t, server.Client(), http.MethodPut, server.URL+"/api/checklist/activities",
		`{"fichaId":"`+escapeJSON(fichaID)+`","selectedActivityIds":`+string(encodedIDs)+`}`)
	if putResponse.StatusCode != http.StatusOK {
		t.Fatalf("checklist activity selection status = %d, body = %s", putResponse.StatusCode, putResponse.Body)
	}
	t.Logf("Zajuna selector register: %d of %d real activities selected", len(ids), len(view.Activities))
}
