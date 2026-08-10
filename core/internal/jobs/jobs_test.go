package jobs

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu     sync.Mutex
	jobs   map[string]Job
	events []Event
}

func newMemoryStore() *memoryStore { return &memoryStore{jobs: map[string]Job{}} }

func (s *memoryStore) CreateJob(_ context.Context, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	return nil
}
func (s *memoryStore) GetJob(_ context.Context, id string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[id], nil
}
func (s *memoryStore) MarkRunning(_ context.Context, id string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	job.Status = StatusRunning
	job.Attempt++
	s.jobs[id] = job
	return job, nil
}
func (s *memoryStore) UpdateProgress(_ context.Context, id, stage string, progress int, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	job.Stage, job.Progress, job.Message = stage, progress, message
	s.jobs[id] = job
	return nil
}
func (s *memoryStore) AppendEvent(_ context.Context, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}
func (s *memoryStore) ListJobEvents(_ context.Context, id string) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Event, 0)
	for _, event := range s.events {
		if event.JobID == id {
			result = append(result, event)
		}
	}
	return result, nil
}
func (s *memoryStore) CompleteJob(_ context.Context, id string, output json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	job.Status, job.Result, job.Progress = StatusCompleted, output, 100
	s.jobs[id] = job
	return nil
}
func (s *memoryStore) RetryJob(_ context.Context, id, code, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	job.Status, job.ErrorCode, job.ErrorMessage = StatusRetrying, code, message
	s.jobs[id] = job
	return nil
}
func (s *memoryStore) FailJob(_ context.Context, id, code, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	job.Status, job.ErrorCode, job.ErrorMessage = StatusFailed, code, message
	s.jobs[id] = job
	return nil
}
func (s *memoryStore) MarkCancelled(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	job.Status = StatusCancelled
	s.jobs[id] = job
	return nil
}

type demoWorker struct{}

func (demoWorker) ID() string { return "demo" }
func (demoWorker) Execute(ctx context.Context, job Job, reporter Reporter) Result {
	if err := reporter.Progress(ctx, "processing", 50, "Procesando fixture"); err != nil {
		return Result{ErrorCode: "progress_failed", ErrorMessage: err.Error()}
	}
	return Result{Output: map[string]any{"job": job.ID, "ok": true}}
}

func TestRuntimeExecutesWorkerAndPersistsResult(t *testing.T) {
	store := newMemoryStore()
	runtime, err := NewRuntime(store, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Register(demoWorker{}); err != nil {
		t.Fatal(err)
	}
	runtime.Start(context.Background())
	defer runtime.Close()

	job, err := runtime.Submit(context.Background(), "demo", map[string]string{"fixture": "ok"})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, _ := store.GetJob(context.Background(), job.ID)
		if stored.Status == StatusCompleted {
			if stored.Progress != 100 {
				t.Fatalf("expected completed progress, got %d", stored.Progress)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("worker did not complete before timeout")
}
