package workers

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/zajuna-app/core/internal/capture"
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

func TestAbsentZajunaContentIsNotACaptureFailure(t *testing.T) {
	tally := tallyTargetOutcomes([]targetOutcome{
		{captured: true, evidenceRecords: 1},
		{failure: "9.1.6: el selector requerido no apareció en la página destino: la lista no tiene publicaciones del instructor autenticado"},
		{failure: "10.1.1: el selector requerido no apareció en la página destino: #region-main table.generaltable (candidatos=0)"},
		{failure: "7.2: navegar para captura: timeout"},
	})
	if tally.absent != 2 || tally.failed != 1 || len(tally.absences) != 2 {
		t.Fatalf("unexpected tally: %#v", tally)
	}
}

func TestCaptureChecklistPrunePlanDropsEvidenceOfAbsentSlots(t *testing.T) {
	// 9.1.6 slot 3 was captured by an older rule (an instructor discussion
	// without replies). The new rule finds no answered discussion: that
	// absence must retire the old evidence instead of keeping it approved.
	planned := []checklist.CaptureTarget{
		{ItemCode: "9.1.6", SlotNumber: 1},
		{ItemCode: "9.1.6", SlotNumber: 3},
	}
	outcomes := []targetOutcome{
		{captured: true},
		{failure: "9.1.6: el selector requerido no apareció en la página destino: la lista no tiene respuestas del instructor autenticado"},
	}
	_, keep := captureChecklistPrunePlan(planned, planned, outcomes)
	if !reflect.DeepEqual(keep, map[string]map[int]bool{"9.1.6": {1: true}}) {
		t.Fatalf("keep = %v", keep)
	}
}

func TestSemanticAbsenceOnlyForForumsWithoutDates(t *testing.T) {
	notFound := fmt.Errorf("%w: selector (candidatos=0)", capture.ErrSelectorNotFound)
	dates := checklist.CaptureTarget{ItemCode: "9.1.3", SemanticCheck: checklist.SemanticForumDates}
	message := semanticAbsence(dates, notFound)
	if message == "" || !absentContent("9.1.3: "+message) {
		t.Fatalf("a forum without dates is an absence, got %q", message)
	}
	if semanticAbsence(checklist.CaptureTarget{ItemCode: "7.2"}, notFound) != "" {
		t.Fatal("items without a semantic rule keep their failure")
	}
	if semanticAbsence(dates, errors.New("navegar para captura: timeout")) != "" {
		t.Fatal("a navigation error is a failure, not an absence")
	}
	if !absentContent("9.1.5: la lista no tiene respuestas del instructor autenticado") {
		t.Fatal("no instructor replies is an absence")
	}
}
