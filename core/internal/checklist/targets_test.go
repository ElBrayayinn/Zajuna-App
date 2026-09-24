package checklist

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zajuna-app/core/internal/coursemaps"
)

func TestBuildCaptureTargetsUsesItemCodesAndEvidenceSlots(t *testing.T) {
	record := coursemaps.Record{
		ByItemCode: map[string]json.RawMessage{
			"2.1.1": json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/user/profile.php"`),
			"1.1.1": json.RawMessage(`[
                "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=10",
                "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=11"
            ]`),
			"1.2.1": json.RawMessage(`[
                "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080&section=1",
                "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080&section=2"
            ]`),
		},
		Routes: []coursemaps.Route{
			{URL: "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=10", Kind: "page"},
			{URL: "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=11", Kind: "page"},
			{URL: "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080&section=1", Kind: "phase"},
			{URL: "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080&section=2", Kind: "phase"},
		},
	}
	targets, summary, err := BuildCaptureTargets(record)
	if err != nil {
		t.Fatal(err)
	}
	if len(CaptureSpecs()) != 62 || summary.ItemCount != 62 {
		t.Fatalf("expected 62 checklist specs, summary=%#v", summary)
	}
	if summary.ResolvedItems != 3 || summary.SlotCount != 4 || summary.UnresolvedItems != 59 {
		t.Fatalf("unexpected target summary: %#v", summary)
	}
	if len(targets) != 4 || targets[0].ItemCode != "1.1.1" || targets[0].SlotNumber != 1 || targets[0].CSSSelector == "" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
	if targets[2].ItemCode != "1.2.1" || targets[2].RouteKind != "phase" {
		t.Fatalf("phase route was not projected: %#v", targets[2])
	}
}

func TestBuildCaptureTargetsSharesOneUnitAcrossCompatibleScheduleItems(t *testing.T) {
	record := coursemaps.Record{
		ByItemCode: map[string]json.RawMessage{
			"1.1.1": json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=10"`),
			"1.1.2": json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=10"`),
		},
		Routes: []coursemaps.Route{{Kind: "page", URL: "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=10"}},
	}
	targets, summary, err := BuildCaptureTargets(record)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || summary.CaptureUnitCount != 1 || summary.CoverageCount != 2 {
		t.Fatalf("expected one physical unit with two logical criteria, targets=%#v summary=%#v", targets, summary)
	}
	if len(targets[0].CoveredItemCodes) != 2 || targets[0].CoveredItemCodes[0] != "1.1.1" || targets[0].CoveredItemCodes[1] != "1.1.2" {
		t.Fatalf("unexpected coverage metadata: %#v", targets[0].CoveredItemCodes)
	}
}

func TestBuildCaptureTargetsBindsDatesToSelectedActivity(t *testing.T) {
	record := coursemaps.Record{
		CourseURL: "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080",
		Routes: []coursemaps.Route{
			{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=3010294", ActivityID: "3010294", Title: "Informe técnico", PhaseSection: 19, Technical: true},
			{Kind: "grading", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=3010294&action=grading", ActivityID: "3010294", Title: "Calificación: Informe técnico", PhaseSection: 19, Technical: true},
			{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=3010361", ActivityID: "3010361", Title: "Storyboard", PhaseSection: 29, Technical: true},
		},
	}
	targets, _, err := BuildCaptureTargetsForActivities(record, map[string]bool{"3010294": true})
	if err != nil {
		t.Fatal(err)
	}
	var bound, grading []CaptureTarget
	for _, target := range targets {
		if target.ItemCode == "6.1" {
			bound = append(bound, target)
		}
		if target.ItemCode == "10.1.1" || target.ItemCode == "10.1.2" {
			grading = append(grading, target)
		}
	}
	if len(bound) != 1 {
		t.Fatalf("expected one selected activity for 6.1, got %#v", bound)
	}
	// 10.1.x use the activity's grading table (not the 6.1 date card), in
	// 2-row batches, shared by both items: 5 slots, each covering both.
	if len(grading) != 5 {
		t.Fatalf("expected 5 row batches of the grading table, got %#v", grading)
	}
	for index, target := range grading {
		if target.URL != "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=3010294&action=grading" || target.RowBatch != index || target.RowsPerShot != 2 || target.SlotNumber != index+1 {
			t.Fatalf("grading batch %d is wrong: %#v", index, target)
		}
		if len(target.CoveredItemCodes) != 2 || target.CoveredItemCodes[0] != "10.1.1" || target.CoveredItemCodes[1] != "10.1.2" {
			t.Fatalf("grading table must be one shared unit for 10.1.1 and 10.1.2: %#v", target.CoveredItemCodes)
		}
	}
	for _, target := range bound {
		if target.ActivityID != "3010294" || target.URL != record.CourseURL || target.PhaseSection != 19 {
			t.Fatalf("target is not bound to the selected course activity: %#v", target)
		}
		if target.CSSSelector != "#region-main .course-content #module-3010294" || len(target.RevealSelectors) != 1 {
			t.Fatalf("target selector is not activity-specific: %#v", target)
		}
	}
}

func TestBuildCaptureTargetsScopesForumsToTechnicalOwnerContent(t *testing.T) {
	record := coursemaps.Record{
		ByItemCode: map[string]json.RawMessage{
			"9.1.6": json.RawMessage(`["https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=77&forceview=1", "https://zajuna.sena.edu.co/zajuna/mod/forum/discuss.php?d=88"]`),
		},
		Routes: []coursemaps.Route{
			{Kind: "forum", URL: "https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=77", Title: "Foro temático GA2-250201022-AA1-EV01"},
			{Kind: "forum", URL: "https://zajuna.sena.edu.co/zajuna/mod/forum/discuss.php?d=88", Title: "Respuesta de otro usuario"},
			{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=301", ActivityID: "301", Title: "Storyboard GA2-250201022-AA1-EV01", Technical: true},
		},
	}
	targets, _, err := BuildCaptureTargetsForActivities(record, map[string]bool{"301": true})
	if err != nil {
		t.Fatal(err)
	}
	var forums []CaptureTarget
	for _, target := range targets {
		if target.ItemCode == "9.1.6" {
			forums = append(forums, target)
		}
	}
	// Only the real forum route is used; its discussion list is captured in
	// contiguous 2-row batches (rows 1–2, 3–4, …) up to the item limit.
	if len(forums) != 5 {
		t.Fatalf("expected 5 row batches of the only real forum route, got %#v", forums)
	}
	for index, target := range forums {
		if !strings.Contains(target.URL, "forum/view.php?id=77") || target.SlotNumber != index+1 || target.RowBatch != index || target.RowsPerShot != RowsPerShot || target.RowSelector == "" {
			t.Fatalf("forum batch %d is wrong: %#v", index, target)
		}
		if !target.OwnerOnly || !target.RequireSelector {
			t.Fatalf("forum target must require authenticated owner filtering: %#v", target)
		}
	}
}

func TestGradingItemsNeverReuseTheDateCardWithoutGradingRoute(t *testing.T) {
	record := coursemaps.Record{
		CourseURL: "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080",
		Routes: []coursemaps.Route{
			{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=3010294", ActivityID: "3010294", Title: "Informe técnico", PhaseSection: 19, Technical: true},
		},
	}
	targets, summary, err := BuildCaptureTargetsForActivities(record, map[string]bool{"3010294": true})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		for _, code := range target.CoveredItemCodes {
			if code == "10.1.1" || code == "10.1.2" {
				t.Fatalf("10.1.x must not duplicate the 6.1 date card: %#v", target)
			}
		}
	}
	if summary.UnresolvedItems == 0 {
		t.Fatal("10.1.x without a grading route must be reported as unresolved")
	}
}

func TestCourseSectionGroupsNeverTargetTheFirstSection(t *testing.T) {
	for _, group := range []string{"seguimiento_evaluacion", "seguimiento_documentos", "documentos_retencion", "sesiones_linea"} {
		if selector := captureGroupPlan(group).selector; !strings.Contains(selector, ":has-text(") {
			t.Fatalf("%s must scope .section to its named section, got %q", group, selector)
		}
	}
}

func TestForumConfigurationUsesFullForumRegionWithoutOwnerRequirement(t *testing.T) {
	record := coursemaps.Record{ByItemCode: map[string]json.RawMessage{
		"9.1.3": json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=77"`),
	}}
	targets, _, err := BuildCaptureTargets(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.ItemCode == "9.1.3" {
			if target.OwnerOnly || target.CSSSelector != "#page-mod-forum-view #region-main" || !target.RequireSelector || len(target.HideSelectors) != 1 || target.HideSelectors[0] != "#region-main table" {
				t.Fatalf("forum configuration should use a strict full-region capture: %#v", target)
			}
			return
		}
	}
	t.Fatal("forum configuration target was not generated")
}

func TestBuildCaptureTargetsUsesGoogleSheetsAwareCronogramaSelector(t *testing.T) {
	record := coursemaps.Record{ByItemCode: map[string]json.RawMessage{
		"1.1.1": json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=10"`),
	}}
	targets, _, err := BuildCaptureTargets(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.ItemCode == "1.1.1" {
			if !strings.Contains(target.CSSSelector, "docs.google.com/spreadsheets") || target.ViewportWidth != 2560 || target.ViewportHeight != 1200 || !target.FullPage {
				t.Fatalf("cronograma selector does not prioritize embedded Google Sheets: %q", target.CSSSelector)
			}
			return
		}
	}
	t.Fatal("cronograma target was not generated")
}

func TestBuildCaptureTargetsItem31MatchesCourseContentWithoutAFragileHint(t *testing.T) {
	// MDL-124: two independent real courses proved the checklist wording
	// ("material de trabajo", "evidencias") never appears verbatim on the
	// course main page item 3.1 resolves to, so `.section` plus that hint
	// found 284 unrelated nodes and matched none of them, aborting the whole
	// capture batch. See docs/mdl-33-2026-08-26.md.
	record := coursemaps.Record{ByItemCode: map[string]json.RawMessage{
		"3.1": json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/course/view.php?id=27932"`),
	}}
	targets, _, err := BuildCaptureTargets(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.ItemCode == "3.1" {
			if target.CSSSelector != "#region-main .course-content" {
				t.Fatalf("item 3.1 must crop the confirmed course-content wrapper, got %q", target.CSSSelector)
			}
			if target.LabelHint != "" || target.RequireSelector {
				t.Fatalf("item 3.1 must not require a fragile text hint: %#v", target)
			}
			return
		}
	}
	t.Fatal("item 3.1 target was not generated")
}

func TestBuildCaptureTargetsItem41MenuCursoMatchesCourseContentWithoutAFragileHint(t *testing.T) {
	// MDL-124: a live run against a real course
	// (docs/evidence/mdl-124-verify-2026-09-14.json) proved item 4.1's
	// "secciones" hint never appears inside `#region-main .course-content` on
	// its resolved course main page, even though that exact container matched
	// on that exact route for item 3.1 with no hint at all. Requiring the
	// literal checklist wording only pushed 4.1 down its fallback chain to the
	// coarser `#page-content` wrapper instead of the confirmed container.
	record := coursemaps.Record{ByItemCode: map[string]json.RawMessage{
		"4.1": json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/course/view.php?id=27932"`),
	}}
	targets, _, err := BuildCaptureTargets(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.ItemCode == "4.1" {
			if target.CSSSelector != "#region-main .course-content" {
				t.Fatalf("item 4.1 must crop the confirmed course-content wrapper, got %q", target.CSSSelector)
			}
			if target.LabelHint != "" || target.RequireSelector {
				t.Fatalf("item 4.1 must not require a fragile text hint: %#v", target)
			}
			return
		}
	}
	t.Fatal("item 4.1 target was not generated")
}

func TestBuildCaptureTargetsSeguimientoSesionesDocumentosDropFragileHints(t *testing.T) {
	// MDL-124 follow-up: a live run against a real course
	// (docs/mdl-124-seguimiento-2026-09-11.md) proved these ten checklist
	// descriptions never appear as literal page text either — same failure
	// mode as item 3.1, `.course-content .section` matched 284 nodes and 0
	// matched the hint, hard-aborting the item instead of falling back.
	itemCodes := []string{"7.1.1", "7.2", "7.3.2", "7.4.1", "7.4.2", "7.4.3", "7.4.4", "8.2", "8.3", "13.1.3", "13.2.2"}
	byItemCode := make(map[string]json.RawMessage, len(itemCodes))
	for _, itemCode := range itemCodes {
		byItemCode[itemCode] = json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/course/view.php?id=27932"`)
	}
	targets, _, err := BuildCaptureTargets(coursemaps.Record{ByItemCode: byItemCode})
	if err != nil {
		t.Fatal(err)
	}
	found := make(map[string]bool, len(itemCodes))
	for _, target := range targets {
		if target.LabelHint != "" || target.RequireSelector {
			t.Fatalf("%s must not require a fragile text hint: %#v", target.ItemCode, target)
		}
		found[target.ItemCode] = true
	}
	for _, itemCode := range itemCodes {
		if !found[itemCode] {
			t.Fatalf("%s target was not generated", itemCode)
		}
	}
}

func TestBuildCaptureTargetsCapturesInstructorProfileAsFullPage(t *testing.T) {
	record := coursemaps.Record{ByItemCode: map[string]json.RawMessage{
		"2.1.1": json.RawMessage(`"https://zajuna.sena.edu.co/zajuna/user/profile.php"`),
	}}
	targets, _, err := BuildCaptureTargets(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.ItemCode == "2.1.1" {
			if !target.FullPage || !target.RequireSelector || target.CSSSelector != "#page-user-profile" {
				t.Fatalf("profile target must use a strict full-page capture: %#v", target)
			}
			return
		}
	}
	t.Fatal("profile target was not generated")
}

func TestApplyRouteReviewsPersistsDecisionAndManualOverrides(t *testing.T) {
	targets := []CaptureTarget{{GroupName: "perfil_instructor", RouteKind: "page", URL: "https://zajuna.sena.edu.co/zajuna/user/profile.php", CSSSelector: "#page-user-profile", CSSSelectorFallbacks: []string{"#page-user-profile"}}}
	key := RouteKey(targets[0])
	updated := ApplyRouteReviews(targets, []RouteReview{{RouteKey: key, Status: RouteReviewCorrection, ManualURL: "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080", ManualSelector: "#region-main .course-content"}})
	if len(updated) != 1 || updated[0].RouteKey != key || updated[0].ReviewStatus != RouteReviewCorrection {
		t.Fatalf("route review was not projected: %#v", updated)
	}
	if updated[0].URL == targets[0].URL || updated[0].CSSSelector != "#region-main .course-content" || len(updated[0].CSSSelectorFallbacks) != 1 {
		t.Fatalf("manual route override was not applied: %#v", updated[0])
	}
}

func TestDistributeSlotBatchesUsesRemainder(t *testing.T) {
	got := distributeSlotBatches(8, 3)
	if len(got) != 3 || got[0] != 3 || got[1] != 3 || got[2] != 2 {
		t.Fatalf("8 slots among 3 lists: got %v, want 3,3,2", got)
	}
	even := distributeSlotBatches(6, 3)
	if len(even) != 3 || even[0] != 2 || even[1] != 2 || even[2] != 2 {
		t.Fatalf("even split: got %v", even)
	}
}
