package capture

import (
	"testing"
)

func TestBlockedPageMarkers(t *testing.T) {
	blocked := "Web Page Blocked!\nPlease contact the administrator.\nAttack ID: 20000051\nMessage ID: 123"
	content := blocked
	if !containsBlockedPageMarkers(content) {
		t.Fatal("expected WAF markers to be detected")
	}
	if containsBlockedPageMarkers("Zajuna course content") {
		t.Fatal("did not expect normal page content to be detected as blocked")
	}
}
