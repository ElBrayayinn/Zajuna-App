package workers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/zajuna-app/core/internal/jobs"
	"github.com/zajuna-app/core/internal/zajuna"
)

type fakeCredentials struct{}

func (fakeCredentials) Set(string, string) error   { return nil }
func (fakeCredentials) Get(string) (string, error) { return "secret", nil }

type fakeZajunaClient struct{}

func (fakeZajunaClient) Login(context.Context, zajuna.Credentials) (zajuna.Session, error) {
	return zajuna.Session{}, nil
}
func (fakeZajunaClient) ListFichas(context.Context, zajuna.Session) ([]zajuna.Ficha, error) {
	return []zajuna.Ficha{{ExternalID: "3135429", Name: "Programa demo", CourseID: "41080"}}, nil
}

type authFailureZajunaClient struct{}

func (authFailureZajunaClient) Login(context.Context, zajuna.Credentials) (zajuna.Session, error) {
	return zajuna.Session{}, errors.Join(zajuna.ErrAuthentication, errors.New("fixture"))
}

func (authFailureZajunaClient) ListFichas(context.Context, zajuna.Session) ([]zajuna.Ficha, error) {
	return nil, nil
}

type fakeFichaStore struct{ count int }

func (s *fakeFichaStore) UpsertFichas(_ context.Context, fichas []zajuna.Ficha) (int, error) {
	s.count += len(fichas)
	return len(fichas), nil
}

type fakeReporter struct{}

func (fakeReporter) Progress(context.Context, string, int, string) error { return nil }
func (fakeReporter) Event(context.Context, string, string, any) error    { return nil }

func TestSyncFichasWorkerPersistsResult(t *testing.T) {
	store := &fakeFichaStore{}
	worker, err := NewSyncFichasWorker(fakeZajunaClient{}, fakeCredentials{}, store)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(syncFichasInput{Username: "123", DocumentType: "CC"})
	result := worker.Execute(context.Background(), jobs.Job{Input: input}, fakeReporter{})
	if result.ErrorMessage != "" {
		t.Fatal(result.ErrorMessage)
	}
	if store.count != 1 {
		t.Fatalf("expected one ficha, got %d", store.count)
	}
}

func TestSyncFichasWorkerDoesNotRetryRejectedCredentials(t *testing.T) {
	worker, err := NewSyncFichasWorker(authFailureZajunaClient{}, fakeCredentials{}, &fakeFichaStore{})
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(syncFichasInput{Username: "123", DocumentType: "CC"})
	result := worker.Execute(context.Background(), jobs.Job{Input: input}, fakeReporter{})
	if result.Retryable {
		t.Fatal("authentication rejection must not be retried automatically")
	}
	if result.ErrorCode != "zajuna_login_failed" {
		t.Fatalf("unexpected error: %#v", result)
	}
}
