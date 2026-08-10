package workers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/evidence"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/zajuna"
)

func TestExportReportWorkerCreatesHTMLReport(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.CreateEvidence(context.Background(), evidence.Record{ID: "evidence-report", Name: "fixture.html", FilePath: "evidences/html/fixture.html", Format: "html", Source: "test", SHA256: "abc123"}); err != nil {
		t.Fatal(err)
	}
	worker, err := NewExportReportWorker(dataDir, store, store, capture.Resolve(""))
	if err != nil {
		t.Fatal(err)
	}
	result := worker.Execute(context.Background(), jobs.Job{ID: "job-report", Input: []byte(`{"title":"Reporte de prueba","format":"html"}`)}, captureReporter{})
	if result.ErrorMessage != "" {
		t.Fatalf("report failed: %#v", result)
	}
	output := result.Output.(map[string]any)
	path := output["path"].(string)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "Reporte de prueba") || !strings.Contains(string(contents), "fixture.html") {
		t.Fatal("report does not contain expected evidence")
	}
	items, err := store.ListReports(context.Background(), 10)
	if err != nil || len(items) != 1 || items[0].Format != "html" {
		t.Fatalf("report was not persisted: %#v (%v)", items, err)
	}
}

func TestExportReportWorkerCreatesPDFSmoke(t *testing.T) {
	if os.Getenv("ZAJUNA_RUN_BROWSER_SMOKE") != "1" {
		t.Skip("browser smoke disabled")
	}
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.CreateEvidence(context.Background(), evidence.Record{ID: "evidence-pdf", Name: "fixture.html", FilePath: "evidences/html/fixture.html", Format: "html", Source: "test", SHA256: "abc123"}); err != nil {
		t.Fatal(err)
	}
	worker, err := NewExportReportWorker(dataDir, store, store, capture.Resolve(""))
	if err != nil {
		t.Fatal(err)
	}
	result := worker.Execute(context.Background(), jobs.Job{ID: "job-pdf", Input: []byte(`{"title":"PDF de prueba","format":"pdf"}`)}, captureReporter{})
	if result.ErrorMessage != "" {
		t.Fatal(result.ErrorMessage)
	}
	path := result.Output.(map[string]any)["path"].(string)
	contents, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) < 4 || string(contents[:4]) != "%PDF" {
		t.Fatalf("generated file is not a PDF: %q", contents[:min(12, len(contents))])
	}
}

func TestExportReportWorkerUsesEvidenceGroupsForFicha(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "3135429", Name: "Programa demo", CourseID: "41080"}}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(context.Background(), 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("unexpected fichas: %#v (%v)", fichas, err)
	}
	metadata := []byte(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/user/profile.php?id=7","selector":"#page-user-profile","selectorMatched":true}`)
	for _, record := range []evidence.Record{
		{ID: "group-report-1", FichaID: fichas[0].ID, ItemCode: "2.1.1", SlotNumber: 1, Name: "Perfil académico", FilePath: "profile-1.png", Format: "png", Source: "capture-checklist", SHA256: "group-hash-one", Metadata: metadata},
		{ID: "group-report-2", FichaID: fichas[0].ID, ItemCode: "2.1.2", SlotNumber: 1, Name: "Correo institucional", FilePath: "profile-2.png", Format: "png", Source: "capture-checklist", SHA256: "group-hash-two", Metadata: metadata},
	} {
		if err := store.CreateEvidence(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	worker, err := NewExportReportWorker(dataDir, store, store, capture.Resolve(""))
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(ExportReportInput{Title: "Reporte agrupado", Format: "html", FichaID: fichas[0].ID})
	result := worker.Execute(context.Background(), jobs.Job{ID: "job-group-report", Input: input}, captureReporter{})
	if result.ErrorMessage != "" {
		t.Fatalf("grouped report failed: %#v", result)
	}
	path := result.Output.(map[string]any)["path"].(string)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(contents)
	if !strings.Contains(text, "Reporte agrupado") || !strings.Contains(text, "Tareas cubiertas") || !strings.Contains(text, "2.1.1, 2.1.2") {
		t.Fatalf("grouped report does not expose shared task coverage: %s", text)
	}
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
