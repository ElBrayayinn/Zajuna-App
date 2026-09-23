package sqlite

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zajuna-app/core/internal/evidence"
	"github.com/zajuna-app/core/internal/zajuna"
)

func openGroupTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "3135429", Name: "Programa demo", CourseID: "41080"}}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(context.Background(), 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("unexpected fichas: %#v (%v)", fichas, err)
	}
	return store, fichas[0].ID
}

func TestEvidenceGroupsCollapseIdenticalContentAcrossPagesAndItems(t *testing.T) {
	store, fichaID := openGroupTestStore(t)
	course := json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080","selector":"#region-main","groupName":"menu_curso"}`)
	forum := json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=9","selector":"#region-main .discussion","groupName":"foros"}`)
	for _, record := range []evidence.Record{
		// Same physical file shared via coveredItemCodes.
		{ID: "shared-a", FichaID: fichaID, ItemCode: "1.2.1", SlotNumber: 1, Name: "Menú", FilePath: "menu.png", Format: "png", Source: "capture-checklist", SHA256: "AAA", Metadata: course},
		{ID: "shared-b", FichaID: fichaID, ItemCode: "1.2.2", SlotNumber: 1, Name: "Menú", FilePath: "menu.png", Format: "png", Source: "capture-checklist", SHA256: "aaa", Metadata: course},
		// Separate capture with identical bytes under a different page/group.
		{ID: "shared-c", FichaID: fichaID, ItemCode: "10.1.1", SlotNumber: 1, Name: "Foro", FilePath: "forum.png", Format: "png", Source: "capture-checklist", SHA256: "aaa", Metadata: forum},
		// Different bytes on the same page: must not be merged.
		{ID: "other", FichaID: fichaID, ItemCode: "6.1", SlotNumber: 1, Name: "Otra", FilePath: "other.png", Format: "png", Source: "capture-checklist", SHA256: "bbb", Metadata: course},
	} {
		if err := store.CreateEvidence(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	groups, err := store.RebuildEvidenceGroups(context.Background(), fichaID)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected two content groups, got %#v", groups)
	}
	var shared *evidence.Group
	for index := range groups {
		if groups[index].GroupKey == "hash:aaa" {
			shared = &groups[index]
		}
	}
	if shared == nil || shared.Confidence != "exact" || len(shared.Evidences) != 3 {
		t.Fatalf("identical content was not collapsed: %#v", groups)
	}
	if strings.Join(shared.ItemCodes, ",") != "1.2.1,1.2.2,10.1.1" {
		t.Fatalf("unexpected covered items: %v", shared.ItemCodes)
	}
	if !strings.Contains(shared.Reason, "1.2.1, 1.2.2, 10.1.1") {
		t.Fatalf("reason should list covered items: %q", shared.Reason)
	}
}

func TestEvidenceGroupSignatureFallsBackToPageWithoutHash(t *testing.T) {
	metadata := json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/user/profile.php?id=7","selector":"#page-user-profile","groupName":"perfil_instructor"}`)
	keyA, confidence, _ := evidenceGroupSignature(evidence.Record{ID: "a", ItemCode: "2.1.1", Metadata: metadata})
	keyB, _, _ := evidenceGroupSignature(evidence.Record{ID: "b", ItemCode: "2.1.2", Metadata: metadata})
	if keyA != keyB || !strings.HasPrefix(keyA, "page:") || confidence != "suggested" {
		t.Fatalf("records without hash should group by page signature: %q %q %q", keyA, keyB, confidence)
	}
	hashed, confidence, _ := evidenceGroupSignature(evidence.Record{ID: "c", ItemCode: "2.1.1", SHA256: "ABC", Metadata: metadata})
	if hashed != "hash:abc" || confidence != "exact" {
		t.Fatalf("hash must take precedence over page signature: %q %q", hashed, confidence)
	}
}
