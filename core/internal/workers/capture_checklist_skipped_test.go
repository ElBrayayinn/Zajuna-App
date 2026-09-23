package workers

import (
	"reflect"
	"testing"

	"github.com/zajuna-app/core/internal/checklist"
)

func TestTallyTargetOutcomesDoesNotCountSkippedAsFailure(t *testing.T) {
	tally := tallyTargetOutcomes([]targetOutcome{
		{captured: true, evidenceRecords: 2},
		{skipped: true},
		{skipped: true},
		{failure: "6.1: timeout"},
	})
	if tally.captured != 1 || tally.skipped != 2 || tally.failed != 1 || tally.evidenceRecords != 2 {
		t.Fatalf("unexpected tally %+v", tally)
	}
	if !reflect.DeepEqual(tally.failures, []string{"6.1: timeout"}) {
		t.Fatalf("failures = %v", tally.failures)
	}
	onlySkipped := tallyTargetOutcomes([]targetOutcome{{captured: true}, {skipped: true}})
	if onlySkipped.failed != 0 || len(onlySkipped.failures) != 0 {
		t.Fatalf("skipped outcomes must not produce failures: %+v", onlySkipped)
	}
}

func TestCaptureChecklistPrunePlanKeepsCapturedFailedAndUnexecutedSlots(t *testing.T) {
	planned := []checklist.CaptureTarget{
		{ItemCode: "6.1", SlotNumber: 1},
		{ItemCode: "6.1", SlotNumber: 2},
		{ItemCode: "6.1", SlotNumber: 3},
		{ItemCode: "7.1", CoveredItemCodes: []string{"7.1", "7.2"}, SlotNumber: 1},
		{ItemCode: "7.2", SlotNumber: 2}, // filtered out of this run
	}
	executed := planned[:4]
	outcomes := []targetOutcome{
		{captured: true},          // 6.1#1 fresh
		{failure: "6.1: timeout"}, // 6.1#2 keeps previous evidence
		{skipped: true},           // 6.1#3 empty batch -> prune
		{captured: true},          // 7.1/7.2#1
	}
	itemCodes, keep := captureChecklistPrunePlan(planned, executed, outcomes)
	if !reflect.DeepEqual(itemCodes, []string{"6.1", "7.1", "7.2"}) {
		t.Fatalf("itemCodes = %v", itemCodes)
	}
	want := map[string]map[int]bool{
		"6.1": {1: true, 2: true},
		"7.1": {1: true},
		"7.2": {1: true, 2: true},
	}
	if !reflect.DeepEqual(keep, want) {
		t.Fatalf("keep = %v, want %v", keep, want)
	}
}
