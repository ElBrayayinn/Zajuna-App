package evidence

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// realCaptureMetadata is the exact metadata shape the capture worker writes,
// taken from a live run against a real Zajuna course. It exists so a field that
// changes type is caught here instead of silently emptying the register: a
// string/int mismatch on phaseSection once dropped every observation.
const realCaptureMetadata = `{
  "url": "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080&section=3",
  "finalUrl": "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080&section=3",
  "title": "FASE 2 EJECUTAR",
  "routeKey": "course:41080:section:3",
  "reviewStatus": "pending",
  "selector": "#region-main .course-content",
  "selectorFallbacks": ["#region-main .course-content", "#region-main", "#page-content"],
  "selectorMatched": true,
  "labelHint": "FASE 2",
  "routeKind": "course",
  "groupName": "cronograma_general",
  "revealSelectors": ["#region-main .course-content [aria-expanded='false']"],
  "hideSelectors": [],
  "viewportWidth": 1440,
  "viewportHeight": 900,
  "fullPage": false,
  "phaseSection": 3,
  "jobId": "job-1",
  "activityId": "31",
  "activityTitle": "Actividad de proyecto",
  "technical": true,
  "ownerOnly": false,
  "coveredItemCodes": ["3.1", "3.2"],
  "captureUnitKey": "course:41080:section:3"
}`

func TestBuildSelectorReportGroupsRealMetadata(t *testing.T) {
	capturedAt := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{ID: "e1", FichaID: "ficha-1", ItemCode: "3.2", SlotNumber: 1, Metadata: json.RawMessage(realCaptureMetadata), CapturedAt: capturedAt},
		{ID: "e2", FichaID: "ficha-1", ItemCode: "3.1", SlotNumber: 1, Metadata: json.RawMessage(realCaptureMetadata), CapturedAt: capturedAt},
	}
	report := BuildSelectorReport("ficha-1", capturedAt, records)

	if report.SkippedRecords != 0 {
		t.Fatalf("real capture metadata was not decoded; %d records were skipped", report.SkippedRecords)
	}
	if report.CaptureUnits != 1 {
		t.Fatalf("two items on one route must collapse into one capture unit, got %d", report.CaptureUnits)
	}
	if report.ItemsCovered != 2 {
		t.Fatalf("itemsCovered = %d, want 2", report.ItemsCovered)
	}
	if report.MatchedUnits != 1 || report.FullPageUnits != 0 {
		t.Fatalf("matched/fullPage = %d/%d, want 1/0", report.MatchedUnits, report.FullPageUnits)
	}
	observation := report.Observations[0]
	if got := strings.Join(observation.ItemCodes, ","); got != "3.1,3.2" {
		t.Fatalf("item codes were not merged and sorted: %s", got)
	}
	if observation.PhaseSection != 3 {
		t.Fatalf("phaseSection = %d, want 3", observation.PhaseSection)
	}
	if observation.Viewport != "1440x900" {
		t.Fatalf("viewport = %q, want 1440x900", observation.Viewport)
	}
	if observation.Redirected {
		t.Fatal("an unchanged final URL must not be reported as a redirect")
	}
	if len(report.Usage) != 1 || report.Usage[0].Matched != 1 {
		t.Fatalf("selector usage was not tallied: %+v", report.Usage)
	}
}

// TestBuildSelectorReportKeepsRoutesAndDropsIdentity locks the redaction rules
// for the committed artefact.
func TestBuildSelectorReportKeepsRoutesAndDropsIdentity(t *testing.T) {
	records := []Record{{
		ID:       "e1",
		FichaID:  "ficha-1",
		ItemCode: "1.1",
		Name:     "Perfil del instructor",
		FilePath: `C:\Users\operador\AppData\Zajuna\evidence\e1.png`,
		Metadata: json.RawMessage(`{
			"url": "https://zajuna.sena.edu.co/zajuna/user/profile.php?id=99",
			"finalUrl": "https://zajuna.sena.edu.co/zajuna/login/index.php",
			"title": "NOMBRE APELLIDO FIXTURE",
			"selector": ".userprofile .fullname",
			"selectorMatched": false,
			"groupName": "perfil_instructor",
			"ownerOnly": true
		}`),
	}}
	report := BuildSelectorReport("ficha-1", time.Now(), records)
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(encoded)

	for _, leak := range []string{"NOMBRE APELLIDO", "operador", "AppData", "zajuna.sena.edu.co"} {
		if strings.Contains(rendered, leak) {
			t.Fatalf("the register leaked %q: %s", leak, rendered)
		}
	}
	observation := report.Observations[0]
	if observation.RoutePath != "/zajuna/user/profile.php?id=99" {
		t.Fatalf("the Moodle route must survive redaction, got %q", observation.RoutePath)
	}
	if !observation.Redirected {
		t.Fatal("a capture that ended on the login page must be flagged as redirected")
	}
	if report.FullPageUnits != 1 {
		t.Fatalf("an unmatched selector must count as a full-page capture, got %d", report.FullPageUnits)
	}
}

func TestBuildSelectorReportReportsUndecodableMetadata(t *testing.T) {
	records := []Record{
		{ID: "e1", ItemCode: "1.1", Metadata: json.RawMessage(`{"phaseSection":"three"}`)},
		{ID: "e2", ItemCode: "1.2"},
	}
	report := BuildSelectorReport("ficha-1", time.Now(), records)
	if report.SkippedRecords != 2 {
		t.Fatalf("skippedRecords = %d, want 2 so a metadata change cannot empty the register silently", report.SkippedRecords)
	}
	if report.CaptureUnits != 0 {
		t.Fatalf("captureUnits = %d, want 0", report.CaptureUnits)
	}
}

// TestRouteOnlyStripsDeploymentHost covers the capture worker's keys, which
// embed absolute URLs, so the register stays host-independent.
func TestRouteOnlyStripsDeploymentHost(t *testing.T) {
	key := `configuracion|https://zajuna.sena.edu.co/zajuna/course/view.php?id=27932|course|#region-main .course-content #module-2228189|2228189|9|3`
	got := RouteOnly(key)
	if strings.Contains(got, "zajuna.sena.edu.co") {
		t.Fatalf("the deployment host survived: %s", got)
	}
	if !strings.Contains(got, "/zajuna/course/view.php?id=27932") {
		t.Fatalf("the Moodle route was lost: %s", got)
	}
	if !strings.Contains(got, "#region-main .course-content #module-2228189") {
		t.Fatalf("the selector was damaged: %s", got)
	}
	if RouteOnly("no urls here #region-main") != "no urls here #region-main" {
		t.Fatal("a diagnostic without URLs must be left untouched")
	}
}

// TestBuildSelectorReportFlagsCoarserRules records the finding a live course
// produced: the intended rule missed and a broad fallback took the screenshot.
// That must be visible rather than counted as a clean match.
func TestBuildSelectorReportFlagsCoarserRules(t *testing.T) {
	records := []Record{{
		ID:       "e1",
		ItemCode: "1.2.1",
		Metadata: json.RawMessage(`{
			"url": "https://zajuna.sena.edu.co/zajuna/mod/resource/view.php?id=4288499",
			"finalUrl": "https://zajuna.sena.edu.co/zajuna/mod/resource/view.php?id=4288499",
			"selector": "#region-main",
			"selectorMatched": true,
			"selectorFallbacks": ["#region-main .course-content .section", "#region-main .course-content", "#region-main"],
			"groupName": "cronograma_vigente"
		}`),
	}}
	report := BuildSelectorReport("ficha-1", time.Now(), records)
	observation := report.Observations[0]
	if observation.PrimarySelector != "#region-main .course-content .section" {
		t.Fatalf("primarySelector = %q", observation.PrimarySelector)
	}
	if !observation.UsedFallback {
		t.Fatal("a screenshot taken by a fallback rule must be flagged")
	}
	if report.MatchedUnits != 1 || report.FallbackChainUnits != 1 || report.FullPageUnits != 0 {
		t.Fatalf("matched/fallbackChain/fullPage = %d/%d/%d, want 1/1/0",
			report.MatchedUnits, report.FallbackChainUnits, report.FullPageUnits)
	}
}

// TestBuildSelectorReportDoesNotFlagThePrimaryRule keeps the flag honest when
// the intended selector is the one that matched.
func TestBuildSelectorReportDoesNotFlagThePrimaryRule(t *testing.T) {
	records := []Record{{
		ID:       "e1",
		ItemCode: "6.1",
		Metadata: json.RawMessage(`{
			"selector": "#region-main .course-content #module-2228189",
			"selectorMatched": true,
			"selectorFallbacks": ["#region-main .course-content #module-2228189", "#region-main"],
			"groupName": "configuracion"
		}`),
	}}
	report := BuildSelectorReport("ficha-1", time.Now(), records)
	if report.Observations[0].UsedFallback || report.FallbackChainUnits != 0 {
		t.Fatal("the intended rule must not be reported as a fallback")
	}
}
