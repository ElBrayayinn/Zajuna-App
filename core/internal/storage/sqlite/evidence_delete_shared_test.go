package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zajuna-app/core/internal/evidence"
	"github.com/zajuna-app/core/internal/zajuna"
)

func TestDeleteEvidenceKeepsFileSharedByAnotherItem(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if _, err := store.UpsertFichas(ctx, []zajuna.Ficha{{ExternalID: "700", Name: "Ficha", CourseID: "9"}}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(ctx, 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("fichas: %#v (%v)", fichas, err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "evidences"), 0o755); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(dir, "evidences", "slot-1.png")
	if err := os.WriteFile(shared, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, item := range []string{"10.1.1", "10.1.2"} {
		if err := store.CreateEvidence(ctx, evidence.Record{
			ID: "e-" + item, FichaID: fichas[0].ID, ItemCode: item, SlotNumber: 1, Name: item, FilePath: shared,
			Format: "png", Source: "capture-checklist", SHA256: "same", CapturedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.DeleteEvidence(ctx, "e-10.1.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(shared); err != nil {
		t.Fatal("the image is still used by 10.1.2 and must not be deleted")
	}
	if _, err := store.DeleteEvidence(ctx, "e-10.1.2"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(shared); !os.IsNotExist(err) {
		t.Fatal("the image must be removed once no evidence references it")
	}
}

func TestReplacingEvidenceKeepsPreviousFileSharedByAnotherItem(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if _, err := store.UpsertFichas(ctx, []zajuna.Ficha{{ExternalID: "701", Name: "Ficha", CourseID: "9"}}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(ctx, 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("fichas: %#v (%v)", fichas, err)
	}
	evidencesDir := filepath.Join(dir, "evidences")
	if err := os.MkdirAll(evidencesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(evidencesDir, "shared.png")
	replacement := filepath.Join(evidencesDir, "replacement.png")
	for _, path := range []string{shared, replacement} {
		if err := os.WriteFile(path, []byte(path), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	create := func(id, item, path string) {
		if err := store.CreateEvidence(ctx, evidence.Record{
			ID: id, FichaID: fichas[0].ID, ItemCode: item, SlotNumber: 1, Name: id, FilePath: path,
			Format: "png", Source: "capture-checklist", SHA256: id, CapturedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	create("first", "10.1.1", shared)
	create("other", "10.1.2", shared)
	create("replacement", "10.1.1", replacement)
	if _, err := os.Stat(shared); err != nil {
		t.Fatalf("shared image removed while another evidence references it: %v", err)
	}
}
