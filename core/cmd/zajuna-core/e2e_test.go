package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/storage/backup"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/workers"
)

func TestLocalE2ESetupCaptureReportBackup(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	runtime, err := jobs.NewRuntime(store, 1)
	if err != nil {
		t.Fatal(err)
	}
	captureWorker, err := workers.NewCaptureEvidenceWorker(dataDir, store)
	if err != nil {
		t.Fatal(err)
	}
	reportWorker, err := workers.NewExportReportWorker(dataDir, store, store, capture.Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Register(captureWorker); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Register(reportWorker); err != nil {
		t.Fatal(err)
	}
	runtime.Start(context.Background())
	defer runtime.Close()

	credentials := &memoryCredentialStore{}
	manager, err := backup.NewManager(dataDir, store)
	if err != nil {
		t.Fatal(err)
	}
	router := newRouterWithServices(dataDir, credentials, runtime, store, manager)
	app := httptest.NewServer(router)
	defer app.Close()

	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<!doctype html><html><head><title>Ficha demo</title></head><body><h1>Contenido local</h1></body></html>`)
	}))
	defer fixture.Close()

	assertStatus(t, app.Client(), http.MethodGet, app.URL+"/api/health", nil, http.StatusOK)

	setupBody := `{"zajunaUsername":"qa-user","zajunaPassword":"qa-password"}`
	setupResponse := doJSON(t, app.Client(), http.MethodPost, app.URL+"/api/setup", setupBody)
	if setupResponse.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, body = %s", setupResponse.StatusCode, setupResponse.Body)
	}
	var setup map[string]bool
	decodeJSON(t, setupResponse.Body, &setup)
	if !setup["saved"] {
		t.Fatalf("setup response did not confirm local configuration: %+v", setup)
	}
	configBytes, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(configBytes), "qa-password") {
		t.Fatal("setup persisted the password in config.json")
	}
	config, err := readConfig(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if !config.SetupComplete || !config.CredentialsStored || config.ZajunaUsername != "qa-user" {
		t.Fatalf("local setup was not persisted: %+v", config)
	}
	if got, ok := credentials.passwords["qa-user"]; !ok || got != "qa-password" {
		t.Fatalf("password was not stored in the local credential store: %+v", credentials.passwords)
	}

	jobResponse := doJSON(t, app.Client(), http.MethodPost, app.URL+"/api/jobs", `{"type":"capture-evidence","input":{"url":"`+fixture.URL+`","itemCode":"QA-001","slotNumber":1,"name":"Ficha demo"}}`)
	if jobResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("capture job status = %d, body = %s", jobResponse.StatusCode, jobResponse.Body)
	}
	var createdJob jobView
	decodeJSON(t, jobResponse.Body, &createdJob)
	completedCapture := waitForE2EJob(t, app.Client(), app.URL, createdJob.ID)
	if completedCapture.Status != jobs.StatusCompleted {
		t.Fatalf("capture job ended with status %s, error = %s", completedCapture.Status, completedCapture.ErrorMessage)
	}

	evidences := getE2EList(t, app.Client(), app.URL+"/api/evidences?limit=10")
	if len(evidences) != 1 {
		t.Fatalf("evidence count = %d, want 1", len(evidences))
	}
	evidenceID, _ := evidences[0]["id"].(string)
	evidenceDownload := doJSON(t, app.Client(), http.MethodGet, app.URL+"/api/evidences/"+evidenceID+"/download", "")
	if evidenceDownload.StatusCode != http.StatusOK || !strings.Contains(evidenceDownload.Body, "Contenido local") {
		t.Fatalf("evidence download failed: status = %d, body = %s", evidenceDownload.StatusCode, evidenceDownload.Body)
	}

	reportResponse := doJSON(t, app.Client(), http.MethodPost, app.URL+"/api/reports", `{"name":"QA report","format":"html","evidenceLimit":10}`)
	if reportResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("report job status = %d, body = %s", reportResponse.StatusCode, reportResponse.Body)
	}
	var createdReportJob jobView
	decodeJSON(t, reportResponse.Body, &createdReportJob)
	completedReport := waitForE2EJob(t, app.Client(), app.URL, createdReportJob.ID)
	if completedReport.Status != jobs.StatusCompleted {
		t.Fatalf("report job ended with status %s, error = %s", completedReport.Status, completedReport.ErrorMessage)
	}

	reports := getE2EList(t, app.Client(), app.URL+"/api/reports?limit=10")
	if len(reports) != 1 {
		t.Fatalf("report count = %d, want 1", len(reports))
	}
	reportID, _ := reports[0]["id"].(string)
	reportDownload := doJSON(t, app.Client(), http.MethodGet, app.URL+"/api/reports/"+reportID+"/download", "")
	if reportDownload.StatusCode != http.StatusOK || !strings.Contains(reportDownload.Body, "Ficha demo") {
		t.Fatalf("report download failed: status = %d, body = %s", reportDownload.StatusCode, reportDownload.Body)
	}

	backupResponse := doJSON(t, app.Client(), http.MethodPost, app.URL+"/api/backups", `{}`)
	if backupResponse.StatusCode != http.StatusCreated {
		t.Fatalf("backup status = %d, body = %s", backupResponse.StatusCode, backupResponse.Body)
	}
	var backupResult struct {
		Name string `json:"name"`
	}
	decodeJSON(t, backupResponse.Body, &backupResult)
	downloadResponse, err := app.Client().Get(app.URL + "/api/backups/" + backupResult.Name + "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer downloadResponse.Body.Close()
	if downloadResponse.StatusCode != http.StatusOK {
		t.Fatalf("backup download status = %d", downloadResponse.StatusCode)
	}
	archiveBytes, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		t.Fatalf("backup archive cannot be opened: %v", err)
	}
	archiveNames := make(map[string]bool)
	for _, file := range archive.File {
		archiveNames[file.Name] = true
	}
	if !archiveNames["manifest.json"] || !archiveNames["database.sqlite"] {
		t.Fatalf("backup archive is missing expected files: %v", archiveNames)
	}

	scheduleResponse := doJSON(t, app.Client(), http.MethodPost, app.URL+"/api/schedules", `{"workerType":"capture-evidence","intervalSeconds":3600,"input":{"url":"`+fixture.URL+`"}}`)
	if scheduleResponse.StatusCode != http.StatusCreated {
		t.Fatalf("schedule status = %d, body = %s", scheduleResponse.StatusCode, scheduleResponse.Body)
	}
	schedules := getE2EList(t, app.Client(), app.URL+"/api/schedules")
	if len(schedules) != 1 {
		t.Fatalf("schedule count = %d, want 1", len(schedules))
	}
}

func waitForE2EJob(t *testing.T, client *http.Client, baseURL, id string) jobView {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
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
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s did not finish before timeout", id)
	return jobView{}
}

func getE2EList(t *testing.T, client *http.Client, url string) []map[string]any {
	t.Helper()
	response := doJSON(t, client, http.MethodGet, url, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", url, response.StatusCode, response.Body)
	}
	var values []map[string]any
	decodeJSON(t, response.Body, &values)
	return values
}

func doJSON(t *testing.T, client *http.Client, method, url, body string) responseBody {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return responseBody{StatusCode: response.StatusCode, Body: string(data)}
}

type responseBody struct {
	StatusCode int
	Body       string
}

func decodeJSON(t *testing.T, body string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), target); err != nil {
		t.Fatalf("invalid JSON %q: %v", body, err)
	}
}

func assertStatus(t *testing.T, client *http.Client, method, url string, body io.Reader, want int) {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != want {
		data, _ := io.ReadAll(response.Body)
		t.Fatalf("%s %s status = %d, want %d, body = %s", method, url, response.StatusCode, want, data)
	}
}
