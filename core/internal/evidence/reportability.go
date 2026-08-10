package evidence

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

// EligibleForReport keeps a captured checklist artifact in the grouped/report
// view only when its navigation destination is compatible with the checklist
// item. The artifact itself remains persisted, so an operator can inspect or
// remove it later from the evidence lifecycle UI.
//
// This is deliberately conservative for automated captures: a public Zajuna
// home/login page must never be presented as evidence for a course section,
// profile, forum, or announcement.
func EligibleForReport(record Record) bool {
	if strings.TrimSpace(record.FichaID) != "" && strings.TrimSpace(record.Source) == "capture-browser" && !checklistItemCodePattern.MatchString(strings.TrimSpace(record.ItemCode)) {
		return false
	}
	if strings.TrimSpace(record.Source) != "capture-checklist" {
		return true
	}
	var metadata struct {
		URL       string `json:"url"`
		FinalURL  string `json:"finalUrl"`
		GroupName string `json:"groupName"`
		Selector  string `json:"selector"`
		Viewport  int    `json:"viewportWidth"`
	}
	if err := json.Unmarshal(record.Metadata, &metadata); err != nil {
		return true
	}
	rawURL := strings.TrimSpace(metadata.FinalURL)
	if rawURL == "" {
		rawURL = strings.TrimSpace(metadata.URL)
	}
	if rawURL == "" {
		return true
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return false
	}
	path := strings.ToLower(strings.TrimRight(parsed.Path, "/"))
	if isZajunaHost(parsed.Host) && isPublicOrLoginPath(path) {
		return false
	}

	groupName := strings.TrimSpace(metadata.GroupName)
	if groupName == "" {
		groupName = inferredGroupName(record.ItemCode)
	}
	if strings.HasPrefix(groupName, "cronograma_") && metadata.Viewport < 2560 {
		return false
	}
	if !isZajunaHost(parsed.Host) {
		return true
	}
	switch groupName {
	case "perfil_instructor":
		return strings.Contains(path, "/user/profile.php") && strings.TrimSpace(metadata.Selector) == "#page-user-profile"
	case "menu_curso":
		return strings.Contains(path, "/course/view.php")
	case "foros", "anuncios_fase", "anuncios_semanales", "conclusion_foros", "netiqueta":
		if !strings.Contains(path, "/mod/forum/view.php") {
			return false
		}
		if groupName == "foros" && forumConfigurationItem(record.ItemCode) {
			return strings.TrimSpace(metadata.Selector) == "#page-mod-forum-view #region-main"
		}
		return ownerForumSelector(metadata.Selector)
	default:
		return true
	}
}

var checklistItemCodePattern = regexp.MustCompile(`^\d+(?:\.\d+)+$`)

func forumConfigurationItem(itemCode string) bool {
	switch itemCode {
	case "9.1.1", "9.1.2", "9.1.3", "9.1.4":
		return true
	default:
		return false
	}
}

func ownerForumSelector(selector string) bool {
	selector = strings.ToLower(strings.TrimSpace(selector))
	for _, term := range []string{".discussion", ".forumpost", "[data-region='post']", ".forum-post", "forumheaderlist tr", " article"} {
		if strings.Contains(selector, term) {
			return true
		}
	}
	return false
}

func isZajunaHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "zajuna.sena.edu.co" || strings.HasSuffix(host, ".zajuna.sena.edu.co")
}

func isPublicOrLoginPath(path string) bool {
	return path == "/zajuna" || path == "/zajuna/index.php" ||
		strings.Contains(path, "/login/") || strings.HasSuffix(path, "/login")
}

func inferredGroupName(itemCode string) string {
	switch {
	case strings.HasPrefix(itemCode, "1."):
		return "cronograma"
	case strings.HasPrefix(itemCode, "2."):
		return "perfil_instructor"
	case itemCode == "4.1":
		return "menu_curso"
	case strings.HasPrefix(itemCode, "9."):
		return "foros"
	case strings.HasPrefix(itemCode, "11."):
		return "anuncios_semanales"
	case strings.HasPrefix(itemCode, "14."):
		return "conclusion_foros"
	case strings.HasPrefix(itemCode, "15."):
		return "netiqueta"
	default:
		return ""
	}
}
