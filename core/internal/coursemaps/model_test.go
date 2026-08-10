package coursemaps

import "testing"

func TestActivitiesDeduplicatesAssignmentViews(t *testing.T) {
	record := Record{Routes: []Route{
		{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=301", Title: "Actividad", PhaseSection: 19},
		{Kind: "grading", URL: "https://zajuna.sena.edu.co/zajuna/grade/grader/index.php?id=301", Title: "Calificación", ActivityID: "301"},
		{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=301", Title: "Actividad técnica", ActivityID: "301", Technical: true, PhaseSection: 19},
		{Kind: "assign", URL: "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=302", Title: "Otra actividad", ActivityID: "302", PhaseSection: 29},
	}}
	activities := Activities(record)
	if len(activities) != 2 {
		t.Fatalf("expected two unique assignments, got %#v", activities)
	}
	if activities[0].ID != "301" || !activities[0].Technical || activities[0].Title != "Actividad técnica" {
		t.Fatalf("assignment metadata was not merged deterministically: %#v", activities)
	}
}

func TestIsTechnicalActivityExcludesKnownTransversalCompetencies(t *testing.T) {
	if IsTechnicalActivity("Video. GA1-220501046-AA1-EV01") {
		t.Fatal("TIC transversal activity was classified as technical")
	}
	if !IsTechnicalActivity("Storyboard GA2-250201022-AA1-EV01") {
		t.Fatal("technical activity was not classified as technical")
	}
	if !IsTechnicalActivity("Anuncio de apertura de fase") {
		t.Fatal("short generic title should remain eligible until author filtering")
	}
}
