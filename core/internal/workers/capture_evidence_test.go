package workers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/storage/sqlite"
)

func TestCaptureEvidenceWorkerPersistsHTMLWithHash(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><title>Fixture</title></head><body><h1>Contenido local</h1></body></html>`))
	}))
	defer server.Close()
	worker, err := NewCaptureEvidenceWorker(dataDir, store)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(CaptureEvidenceInput{URL: server.URL, Name: "fixture.html"})
	result := worker.Execute(context.Background(), jobs.Job{ID: "job-html", Input: input}, captureReporter{})
	if result.ErrorMessage != "" {
		t.Fatalf("capture failed: %#v", result)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["format"] != "html" || output["sha256"] == "" {
		t.Fatalf("unexpected capture output: %#v", result.Output)
	}
	path, ok := output["path"].(string)
	if !ok {
		t.Fatal("capture output has no path")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListEvidences(context.Background(), 10)
	if err != nil || len(items) != 1 || items[0].Format != "html" || items[0].SHA256 == "" {
		t.Fatalf("evidence was not persisted: %#v (%v)", items, err)
	}
}
