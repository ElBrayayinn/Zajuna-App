package workers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/zajuna"
)

type captureReporter struct{}

func (captureReporter) Progress(context.Context, string, int, string) error { return nil }
func (captureReporter) Event(context.Context, string, string, any) error    { return nil }

func TestCaptureBrowserWorkerValidatesAndConstrainsOutput(t *testing.T) {
	worker, err := NewCaptureBrowserWorker(capture.Resolve(filepath.Join("C:", "zajuna-core.exe")), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if worker.runtime.Installed() {
		t.Skip("runtime Chromium disponible; este test cubre la validación cuando falta")
	}
	input, _ := json.Marshal(CaptureBrowserInput{URL: "http://127.0.0.1:1234", OutputPath: "evidences/browser/example.png"})
	result := worker.Execute(context.Background(), jobs.Job{ID: "job-capture", Input: input}, captureReporter{})
	if result.ErrorCode != "browser_not_installed" {
		t.Fatalf("expected capture to reach runtime validation, got %#v", result)
	}
	if path, err := worker.outputPath("job-capture", "evidences/browser/example.png"); err != nil || filepath.Base(path) != "example.png" {
		t.Fatalf("unexpected safe output path: %s (%v)", path, err)
	}
	if _, err := worker.outputPath("job-capture", "..\\outside.png"); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}

func TestCaptureBrowserWorkerPersistsPNGSmoke(t *testing.T) {
	if os.Getenv("ZAJUNA_RUN_BROWSER_SMOKE") != "1" {
		t.Skip("browser smoke disabled")
	}
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>PNG fixture</title></head><body><h1>Captura de prueba</h1></body></html>`))
	}))
	defer server.Close()
	worker, err := NewCaptureBrowserWorker(capture.Resolve(""), dataDir, store)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(CaptureBrowserInput{URL: server.URL, Name: "fixture.png"})
	result := worker.Execute(context.Background(), jobs.Job{ID: "job-png", Input: input}, captureReporter{})
	if result.ErrorMessage != "" {
		t.Fatal(result.ErrorMessage)
	}
	path := result.Output.(map[string]any)["path"].(string)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListEvidences(context.Background(), 10)
	if err != nil || len(items) != 1 || items[0].Format != "png" {
		t.Fatalf("PNG evidence was not persisted: %#v (%v)", items, err)
	}
}

type fixtureAuthenticatedCaptureClient struct {
	session zajuna.Session
}

func (c fixtureAuthenticatedCaptureClient) Login(context.Context, zajuna.Credentials) (zajuna.Session, error) {
	return c.session, nil
}

func TestCaptureBrowserWorkerTransfersEphemeralSessionToChromium(t *testing.T) {
	if os.Getenv("ZAJUNA_RUN_BROWSER_SMOKE") != "1" {
		t.Skip("browser smoke disabled")
	}
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, cookieErr := r.Cookie("session")
		if cookieErr != nil || cookie.Value != "authenticated" {
			http.Redirect(w, r, "/login/index.php", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(`<html><head><title>Ruta autenticada</title></head><body><h1>Contenido protegido</h1></body></html>`))
	}))
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(server.URL)
	jar.SetCookies(parsed, []*http.Cookie{{Name: "session", Value: "authenticated", Path: "/"}})
	client := fixtureAuthenticatedCaptureClient{session: zajuna.Session{Client: &http.Client{Jar: jar}, BaseURL: server.URL}}
	worker, err := NewAuthenticatedCaptureBrowserWorker(capture.Resolve(""), dataDir, client, fakeCredentials{}, store)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(CaptureBrowserInput{URL: server.URL + "/protected", Name: "authenticated.png", Authenticated: true, Username: "fixture-user", DocumentType: "CC"})
	result := worker.Execute(context.Background(), jobs.Job{ID: "job-authenticated-png", Input: input}, captureReporter{})
	if result.ErrorMessage != "" {
		t.Fatal(result.ErrorMessage)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["authenticated"] != true {
		t.Fatalf("unexpected authenticated capture output: %#v", result.Output)
	}
	path := output["path"].(string)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
