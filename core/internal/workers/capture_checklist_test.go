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
	"time"

	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/coursemaps"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/zajuna"
)

func TestCaptureChecklistWorkerCapturesDirectedTargetSmoke(t *testing.T) {
	if os.Getenv("ZAJUNA_RUN_BROWSER_SMOKE") != "1" {
		t.Skip("browser smoke disabled")
	}
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "3135429", Name: "Ficha fixture", CourseID: "41080"}}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(context.Background(), 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("unexpected ficha: %#v (%v)", fichas, err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, cookieErr := r.Cookie("session")
		if cookieErr != nil || cookie.Value != "authenticated" {
			http.Redirect(w, r, "/login/index.php", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(`<html><body><main id="region-main"><section class="course-content"><h2>Fases</h2><p>Contenido fixture</p></section></main></body></html>`))
	}))
	defer server.Close()
	pageURL := server.URL + "/protected"
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(server.URL)
	jar.SetCookies(parsed, []*http.Cookie{{Name: "session", Value: "authenticated", Path: "/"}})
	client := fixtureAuthenticatedCaptureClient{session: zajuna.Session{Client: &http.Client{Jar: jar}, BaseURL: server.URL}}
	now := time.Now().UTC()
	mapURL, _ := json.Marshal([]string{pageURL})
	if err := store.CreateOrReplaceCourseMap(context.Background(), coursemaps.Record{
		CourseID:   "41080",
		ByItemCode: map[string]json.RawMessage{"1.1.1": mapURL},
		Routes:     []coursemaps.Route{{URL: pageURL, Kind: "page"}},
		Source:     "fixture", DiscoveredAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	worker, err := NewCaptureChecklistWorker(capture.Resolve(""), dataDir, client, fakeCredentials{}, store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(CaptureChecklistInput{FichaID: fichas[0].ID, Username: "fixture-user", DocumentType: "CC", ItemCodes: []string{"1.1.1"}, MaxTargets: 1})
	result := worker.Execute(context.Background(), jobs.Job{ID: "job-checklist-capture", Input: input}, captureReporter{})
	if result.ErrorMessage != "" {
		t.Fatal(result.ErrorMessage)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["captured"] != 1 || output["failed"] != 0 {
		t.Fatalf("unexpected directed capture output: %#v", result.Output)
	}
	evidences, err := store.ListEvidences(context.Background(), 10)
	if err != nil || len(evidences) != 1 || evidences[0].Source != "capture-checklist" || evidences[0].SlotNumber != 1 {
		t.Fatalf("directed evidence was not persisted: %#v (%v)", evidences, err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "evidences", "checklist", fichas[0].ID, "1.1.1", "slot-1.png")); err != nil {
		t.Fatalf("directed screenshot missing: %v", err)
	}
}

