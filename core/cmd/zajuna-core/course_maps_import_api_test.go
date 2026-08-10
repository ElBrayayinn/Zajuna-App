package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zajuna-app/core/internal/coursemaps"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/zajuna"
)

func TestImportCourseActivitiesAPIStoresSafeTitleAwareMap(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	payload := importCourseActivitiesRequest{
		CourseID:   "41080",
		ProfileURL: "https://zajuna.sena.edu.co/zajuna/user/profile.php",
		PageLinks: []zajuna.ActivityLink{
			{URL: "/zajuna/mod/page/view.php?id=10", Label: "Cronograma General"},
			{URL: "/zajuna/mod/forum/view.php?id=20", Label: "Foro de Dudas e Inquietudes"},
		},
		Jump: []zajuna.ActivityLink{
			{URL: "/zajuna/mod/assign/view.php?id=30", Label: "GA1 Evidencia de aprendizaje"},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newRouterWithServices(dataDir, &memoryCredentialStore{}, nil, store, nil))
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/course-maps/import-activities", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected import status: %d", response.StatusCode)
	}
	var record coursemaps.Record
	if err := json.NewDecoder(response.Body).Decode(&record); err != nil {
		t.Fatal(err)
	}
	if record.Source != "devtools-import" || record.CourseID != "41080" || len(record.Routes) != 3 {
		t.Fatalf("unexpected imported record: %#v", record)
	}
	var cronograma []string
	if err := json.Unmarshal(record.ByItemCode["1.1.1"], &cronograma); err != nil {
		t.Fatal(err)
	}
	if len(cronograma) != 1 || cronograma[0] != "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?forceview=1&id=10" {
		t.Fatalf("title-aware map was not persisted: %#v", cronograma)
	}
	if _, err := store.GetCourseMap(context.Background(), "41080"); err != nil {
		t.Fatalf("imported map was not stored: %v", err)
	}
}
