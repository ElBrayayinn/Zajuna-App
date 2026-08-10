package evidence

import (
	"encoding/json"
	"testing"
)

func TestEligibleForReportRejectsPublicHomeForAnnouncement(t *testing.T) {
	record := Record{
		Source:   "capture-checklist",
		ItemCode: "11.4",
		Metadata: json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/","groupName":"anuncios_semanales"}`),
	}
	if EligibleForReport(record) {
		t.Fatal("public Zajuna home must not be reportable as an announcement")
	}
}

func TestEligibleForReportAcceptsForumAndProfileDestinations(t *testing.T) {
	cases := []Record{
		{Source: "capture-checklist", ItemCode: "11.4", Metadata: json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=3010173","selector":"#region-main .discussion","groupName":"anuncios_semanales"}`)},
		{Source: "capture-checklist", ItemCode: "9.1.3", Metadata: json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=3010173","selector":"#page-mod-forum-view #region-main","groupName":"foros"}`)},
		{Source: "capture-checklist", ItemCode: "2.1.1", Metadata: json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/user/profile.php?id=7","selector":"#page-user-profile","groupName":"perfil_instructor"}`)},
	}
	for _, record := range cases {
		if !EligibleForReport(record) {
			t.Fatalf("valid destination rejected: %#v", record)
		}
	}
}

func TestEligibleForReportRejectsLegacyPartialProfileAndForumSelectors(t *testing.T) {
	cases := []Record{
		{Source: "capture-checklist", ItemCode: "2.1.2", Metadata: json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/user/profile.php?id=7","selector":"#page-user-profile dl","groupName":"perfil_instructor"}`)},
		{Source: "capture-checklist", ItemCode: "11.4", Metadata: json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=3010173","selector":"#region-main .forum_list .forum","groupName":"anuncios_semanales"}`)},
	}
	for _, record := range cases {
		if EligibleForReport(record) {
			t.Fatalf("legacy selector must not be reportable: %#v", record)
		}
	}
}

func TestEligibleForReportKeepsManualEvidence(t *testing.T) {
	record := Record{Source: "manual-upload", ItemCode: "11.4", Metadata: json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/"}`)}
	if !EligibleForReport(record) {
		t.Fatal("manual evidence must remain available for operator review")
	}
}

func TestEligibleForReportRejectsFixtureCaptureInsideFichaReport(t *testing.T) {
	record := Record{FichaID: "ficha-real", Source: "capture-browser", ItemCode: "QA-COURSE-MENU"}
	if EligibleForReport(record) {
		t.Fatal("fixture capture must not enter a ficha report")
	}
}

func TestEligibleForReportRejectsNarrowLegacyCronograma(t *testing.T) {
	record := Record{
		Source:   "capture-checklist",
		ItemCode: "1.2.1",
		Metadata: json.RawMessage(`{"finalUrl":"https://zajuna.sena.edu.co/zajuna/mod/page/view.php?id=4268124","groupName":"cronograma_vigente","viewportWidth":1920}`),
	}
	if EligibleForReport(record) {
		t.Fatal("narrow legacy cronograma must not be reportable")
	}
}
