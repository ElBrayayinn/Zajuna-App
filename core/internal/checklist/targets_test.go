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

func TestBuildCaptureTargetsBindsDatesToSelectedActivity(t *testing.T) {
	record := coursemaps.Record{
		CourseURL: "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080",
		Routes: []coursemaps.Route{
			{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=3010294", ActivityID: "3010294", Title: "Informe técnico", PhaseSection: 19, Technical: true},
			{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=3010361", ActivityID: "3010361", Title: "Storyboard", PhaseSection: 29, Technical: true},
		},
	}
	targets, _, err := BuildCaptureTargetsForActivities(record, map[string]bool{"3010294": true})
	if err != nil {
		t.Fatal(err)
	}
	var bound []CaptureTarget
	for _, target := range targets {
		if target.ItemCode == "6.1" || target.ItemCode == "10.1.1" || target.ItemCode == "10.1.2" {
			bound = append(bound, target)
		}
	}
	if len(bound) != 3 {
		t.Fatalf("expected one selected activity per bound item, got %#v", bound)
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
	if len(forums) != 1 {
		t.Fatalf("expected only one real forum route, got %#v", forums)
	}
	if !forums[0].OwnerOnly || !forums[0].RequireSelector {
		t.Fatalf("forum target must require authenticated owner filtering: %#v", forums[0])
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
