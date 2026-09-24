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

func TestAnuncioSlotSkipsTechnicalForumsAndSubstringMatches(t *testing.T) {
	routes := []coursemaps.Route{
		{Kind: "forum", URL: "https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=1", Title: "Foro de anuncios del curso"},
		{Kind: "forum", URL: "https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=2", Title: "Foro técnico de GAMIFICACION", Technical: true},
		{Kind: "forum", URL: "https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=3", Title: "Sesión de planeación"},
	}
	groups := buildExactChecklistRouteGroups(routes, "41080", "")
	got := groups["11.4"]
	if len(got) == 0 || got[0] != "https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?forceview=1&id=1" {
		t.Fatalf("11.4 must pick the announcements forum, got %v", got)
	}
	for _, url := range got {
		if url == "https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?forceview=1&id=2" {
			t.Fatal("11.4 must not pick technical forums")
		}
	}
	if containsNormalizedTerm("anuncios", "anuncio") {
		t.Fatal("anuncio must not match inside anuncios via a partial word; use the anuncios term instead")
	}
	if !containsNormalizedTerm("foro de anuncios del curso", "anuncios") {
		t.Fatal("whole-word anuncios must match")
	}
}
