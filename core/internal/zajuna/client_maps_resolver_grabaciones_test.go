package zajuna

import (
	"testing"

	"github.com/zajuna-app/core/internal/coursemaps"
)

func TestGrabacionesIgnorePagesThatOnlyMentionSessionsInTheirSection(t *testing.T) {
	routes := []coursemaps.Route{
		{Kind: "page", URL: "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=1", Title: "Actualización de los datos personales", Subsection: "Sesión en línea de inducción"},
		{Kind: "page", URL: "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=2", Title: "Grabación sesión semana 1"},
	}
	groups := buildExactChecklistRouteGroups(routes, "41080", "")
	got := groups["12.1.1"]
	if len(got) != 1 || got[0] != "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?forceview=1&id=2" {
		t.Fatalf("12.1.1 must only use pages titled as recordings, got %v", got)
	}
}

func TestGrabacionesFallBackToCourseSectionsNotToOtherPages(t *testing.T) {
	routes := []coursemaps.Route{
		{Kind: "page", URL: "https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=1", Title: "Actualización de los datos personales"},
	}
	groups := buildExactChecklistRouteGroups(routes, "41080", "")
	got := groups["12.1.1"]
	if len(got) != 1 || got[0] != "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080" {
		t.Fatalf("without recording pages 12.1.1 must use the course page sections, got %v", got)
	}
	if empty := buildExactChecklistRouteGroups(routes, "", "")["12.1.1"]; empty == nil || len(empty) != 0 {
		t.Fatalf("without a course id 12.1.1 must be an explicit empty list, got %v", empty)
	}
}
