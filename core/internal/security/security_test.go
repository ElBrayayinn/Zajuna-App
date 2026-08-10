package security

import "testing"

func TestRedactURLRemovesCredentialQueryParameters(t *testing.T) {
	value := RedactURL("https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080&sesskey=secret&section=2#fragment")
	if value != "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080&section=2" {
		t.Fatalf("unexpected redacted URL: %s", value)
	}
}

func TestValidateHTTPURLRejectsPrivateAndUnapprovedOrigins(t *testing.T) {
	if _, err := ValidateHTTPURL("http://127.0.0.1:8080/debug", []string{"https://zajuna.sena.edu.co"}, false); err == nil {
		t.Fatal("expected private URL rejection")
	}
	if _, err := ValidateHTTPURL("https://example.com/", []string{"https://zajuna.sena.edu.co"}, false); err == nil {
		t.Fatal("expected unapproved origin rejection")
	}
}

func TestValidateHTTPURLAllowsFixtureWhenExplicitlyEnabled(t *testing.T) {
	if _, err := ValidateHTTPURL("http://127.0.0.1:8080/fixture", nil, true); err != nil {
		t.Fatalf("expected fixture URL to be allowed: %v", err)
	}
}
