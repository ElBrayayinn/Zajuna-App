package workers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zajuna-app/core/internal/coursemaps"
	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/zajuna"
)

type fakeCourseMapClient struct{}

func (fakeCourseMapClient) Login(context.Context, zajuna.Credentials) (zajuna.Session, error) {
	return zajuna.Session{}, nil
}

func (fakeCourseMapClient) DiscoverCourseMap(_ context.Context, _ zajuna.Session, courseID string, _ zajuna.CrawlOptions) (coursemaps.Record, error) {
	now := time.Now().UTC()
	return coursemaps.Record{
		CourseID:      courseID,
		ByItemCode:    map[string]json.RawMessage{"route.page": json.RawMessage(`["https://fixture.local/zajuna/mod/page/view.php?id=1"]`)},
		Routes:        []coursemaps.Route{{URL: "https://fixture.local/zajuna/mod/page/view.php?id=1", Kind: "page", Depth: 1}},
		LinkCount:     1,
		ItemCodeCount: 1,
		ScrapeStats:   coursemaps.Stats{Total: 1, Pages: 1},
		Source:        "fixture",
		DiscoveredAt:  now,
		UpdatedAt:     now,
	}, nil
}

type fakeCourseMapStore struct{ records []coursemaps.Record }

func (s *fakeCourseMapStore) CreateOrReplaceCourseMap(_ context.Context, record coursemaps.Record) error {
	s.records = append(s.records, record)
	return nil
}

func (s *fakeCourseMapStore) GetCourseMap(context.Context, string) (coursemaps.Record, error) {
	return coursemaps.Record{}, nil
}

func (s *fakeCourseMapStore) ListCourseMaps(context.Context, int) ([]coursemaps.Record, error) {
	return s.records, nil
}

func TestDiscoverCourseMapsWorkerPersistsEachCourse(t *testing.T) {
	store := &fakeCourseMapStore{}
	worker, err := NewDiscoverCourseMapsWorker(fakeCourseMapClient{}, fakeCredentials{}, store)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(discoverCourseMapsInput{Username: "123", DocumentType: "CC", CourseIDs: []string{"41080", "41081"}})
	result := worker.Execute(context.Background(), jobs.Job{Input: input}, fakeReporter{})
	if result.ErrorMessage != "" {
		t.Fatal(result.ErrorMessage)
	}
	if len(store.records) != 2 {
		t.Fatalf("expected two maps, got %d", len(store.records))
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["courses"] != 2 || output["links"] != 2 {
		t.Fatalf("unexpected worker output: %#v", result.Output)
	}
}
