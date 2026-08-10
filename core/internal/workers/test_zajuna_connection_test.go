package workers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zajuna-app/core/internal/jobs"
)

func TestTestZajunaConnectionWorkerReturnsFichasCount(t *testing.T) {
	worker, err := NewTestZajunaConnectionWorker(fakeZajunaClient{}, fakeCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(testZajunaConnectionInput{Username: "123", DocumentType: "CC"})
	result := worker.Execute(context.Background(), jobs.Job{Input: input}, fakeReporter{})
	if result.ErrorMessage != "" {
		t.Fatal(result.ErrorMessage)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["authenticated"] != true || output["fichas"] != 1 {
		t.Fatalf("unexpected connection result: %#v", result.Output)
	}
}

func TestTestZajunaConnectionWorkerRequiresUsername(t *testing.T) {
	worker, err := NewTestZajunaConnectionWorker(fakeZajunaClient{}, fakeCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	result := worker.Execute(context.Background(), jobs.Job{Input: []byte(`{}`)}, fakeReporter{})
	if result.ErrorCode != "missing_username" {
		t.Fatalf("unexpected error: %#v", result)
	}
}
