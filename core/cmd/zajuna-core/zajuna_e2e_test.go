package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/coursemaps"
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
		mapCourseID := ""
		if len(fichas) > 0 {
			mapCourseID, _ = fichas[0]["courseId"].(string)
		}
		if mapCourseID == "" {
			t.Fatal("the authenticated sync returned no course id for map E2E")
		}
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
		}
	}
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
