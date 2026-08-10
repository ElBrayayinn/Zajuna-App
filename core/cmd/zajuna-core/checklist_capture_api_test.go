package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zajuna-app/core/internal/checklist"
	"github.com/zajuna-app/core/internal/coursemaps"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/zajuna"
)

func TestChecklistTargetsAPIProjectsMapToCaptureTargets(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "3135429", Name: "Ficha de prueba", CourseID: "41080"}}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.CreateOrReplaceCourseMap(context.Background(), coursemaps.Record{
		CourseID: "41080",
		ByItemCode: map[string]json.RawMessage{
			"1.1.1": json.RawMessage(`["https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=10"]`),
		},
		Routes:       []coursemaps.Route{{URL: "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=10", Kind: "page"}},
		LinkCount:    1,
		Source:       "test",
		DiscoveredAt: now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(context.Background(), 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("unexpected fichas: %#v (%v)", fichas, err)
	}
	server := httptest.NewServer(newRouterWithServices(dataDir, &memoryCredentialStore{}, nil, store, nil))
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/api/checklist/targets?fichaId=" + fichas[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected targets status: %d", response.StatusCode)
	}
	var payload struct {
		MapReady bool `json:"mapReady"`
		Summary struct {
			ResolvedItems int `json:"resolvedItems"`
			SlotCount     int `json:"slotCount"`
		} `json:"summary"`
		Targets []struct {
			ItemCode   string `json:"itemCode"`
			SlotNumber int    `json:"slotNumber"`
		} `json:"targets"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !payload.MapReady || payload.Summary.ResolvedItems != 1 || payload.Summary.SlotCount != 1 || len(payload.Targets) != 1 || payload.Targets[0].ItemCode != "1.1.1" {
		t.Fatalf("unexpected capture targets payload: %#v", payload)
	}
}

func TestChecklistActivitySelectionFiltersCaptureTargets(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "3135429", Name: "Ficha de prueba", CourseID: "41080"}}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.CreateOrReplaceCourseMap(context.Background(), coursemaps.Record{
		CourseID:   "41080",
		CourseURL:  "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080",
		ByItemCode: map[string]json.RawMessage{},
		Routes: []coursemaps.Route{
			{Kind: "assign", ActivityID: "301", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=301", Title: "Técnica", PhaseSection: 19, Technical: true},
			{Kind: "assign", ActivityID: "302", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=302", Title: "Otra", PhaseSection: 29, Technical: false},
		},
		Source: "test", DiscoveredAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(context.Background(), 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("unexpected fichas: %#v (%v)", fichas, err)
	}
	server := httptest.NewServer(newRouterWithServices(dataDir, &memoryCredentialStore{}, nil, store, nil))
	defer server.Close()
	body := []byte(`{"fichaId":"` + fichas[0].ID + `","selectedActivityIds":["301"]}`)
	request, err := http.NewRequest(http.MethodPut, server.URL+"/api/checklist/activities", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected selection status: %d", response.StatusCode)
	}
	targetsResponse, err := server.Client().Get(server.URL + "/api/checklist/targets?fichaId=" + fichas[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer targetsResponse.Body.Close()
	var payload struct {
		SelectionConfigured bool `json:"selectionConfigured"`
		Targets             []struct {
			ItemCode   string `json:"itemCode"`
			ActivityID string `json:"activityId"`
			URL        string `json:"url"`
			Require    bool   `json:"requireSelector"`
		} `json:"targets"`
	}
	if err := json.NewDecoder(targetsResponse.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, target := range payload.Targets {
		if target.ItemCode == "6.1" || target.ItemCode == "10.1.1" || target.ItemCode == "10.1.2" {
			seen++
			if target.ActivityID != "301" || target.URL != "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080" || !target.Require {
				t.Fatalf("unselected activity leaked into target plan: %#v", target)
			}
		}
	}
	if !payload.SelectionConfigured || seen != 3 {
		t.Fatalf("selection did not produce three bound targets: %#v", payload)
	}
}

func TestChecklistTargetsWithoutMapExposeDiscoveryAction(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "3135429", Name: "Ficha de prueba", CourseID: "41080"}}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(context.Background(), 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("unexpected fichas: %#v (%v)", fichas, err)
	}
	server := httptest.NewServer(newRouterWithServices(dataDir, &memoryCredentialStore{}, nil, store, nil))
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/api/checklist/targets?fichaId=" + fichas[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected targets status: %d", response.StatusCode)
	}
	var payload struct {
		FichaID   string `json:"fichaId"`
		CourseID  string `json:"courseId"`
		MapReady  bool   `json:"mapReady"`
		Discovery struct {
			Status  string `json:"status"`
			Action  string `json:"action"`
			Message string `json:"message"`
		} `json:"discovery"`
		Targets []checklist.CaptureTarget `json:"targets"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.FichaID != fichas[0].ID || payload.CourseID != "41080" || payload.MapReady || len(payload.Targets) != 0 {
		t.Fatalf("unexpected map readiness payload: %#v", payload)
	}
	if payload.Discovery.Status != "required" || payload.Discovery.Action != "discover-course-maps" || payload.Discovery.Message == "" {
		t.Fatalf("missing discovery action: %#v", payload.Discovery)
	}
}

func TestChecklistActivitiesWithoutMapExposeDiscoveryAction(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "3135429", Name: "Ficha de prueba", CourseID: "41080"}}); err != nil {
		t.Fatal(err)
	}
	fichas, err := store.ListFichas(context.Background(), 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("unexpected fichas: %#v (%v)", fichas, err)
	}
	server := httptest.NewServer(newRouterWithServices(dataDir, &memoryCredentialStore{}, nil, store, nil))
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/api/checklist/activities?fichaId=" + fichas[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected activities status: %d", response.StatusCode)
	}
	var payload struct {
		FichaID   string `json:"fichaId"`
		CourseID  string `json:"courseId"`
		MapReady  bool   `json:"mapReady"`
		Activities []struct {
			ID string `json:"id"`
		} `json:"activities"`
		Discovery struct {
			Status string `json:"status"`
			Action string `json:"action"`
		} `json:"discovery"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.FichaID != fichas[0].ID || payload.CourseID != "41080" || payload.MapReady || len(payload.Activities) != 0 {
		t.Fatalf("unexpected activities readiness payload: %#v", payload)
	}
	if payload.Discovery.Status != "required" || payload.Discovery.Action != "discover-course-maps" {
		t.Fatalf("missing discovery action: %#v", payload.Discovery)
	}
}

func TestChecklistCaptureMissingMapDoesNotCreateFailedJob(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "3135429", Name: "Ficha de prueba", CourseID: "41080"}}); err != nil {
		t.Fatal(err)
	}
	runtime, err := jobs.NewRuntime(store, 1)
	if err != nil {
		t.Fatal(err)
	}
	runtime.Start(context.Background())
	defer runtime.Close()
	server := httptest.NewServer(newRouterWithServices(dataDir, &memoryCredentialStore{}, runtime, store, nil))
	defer server.Close()
	fichas, err := store.ListFichas(context.Background(), 10)
	if err != nil || len(fichas) != 1 {
		t.Fatalf("unexpected fichas: %#v (%v)", fichas, err)
	}
	response := doJSON(t, server.Client(), http.MethodPost, server.URL+"/api/checklist/capture", `{"fichaId":"`+fichas[0].ID+`","username":"qa-user","documentType":"CC"}`)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected route preflight conflict, got %d body=%s", response.StatusCode, response.Body)
	}
	var payload struct {
		Code     string `json:"code"`
		Action   string `json:"action"`
		CourseID string `json:"courseId"`
	}
	decodeJSON(t, response.Body, &payload)
	if payload.Code != "course_map_required" || payload.Action != "discover-course-maps" || payload.CourseID != "41080" {
		t.Fatalf("unexpected route preflight payload: %#v", payload)
	}
	jobsList, err := store.ListJobs(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobsList) != 0 {
		t.Fatalf("capture preflight should not create a failed job: %#v", jobsList)
	}
}
