package zajuna

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/zajuna-app/core/internal/coursemaps"
)

// ActivityLink is the safe, non-secret payload emitted by the DevTools
// diagnostic script and accepted by the local map importer.
type ActivityLink struct {
	URL   string `json:"path"`
	Label string `json:"label"`
}

// BuildCourseMapFromActivities turns exported browser links/titles into the
// same local map produced by the authenticated crawler. It never accepts or
// stores cookies, passwords or browser tokens.
func BuildCourseMapFromActivities(courseID, profileURL string, activities []ActivityLink) (coursemaps.Record, error) {
	courseID = strings.TrimSpace(courseID)
	if !numericCourseID(courseID) {
		return coursemaps.Record{}, errors.New("el ID del curso debe ser numérico")
	}
	origin := "https://zajuna.sena.edu.co"
	if parsed, err := url.Parse(profileURL); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		origin = parsed.Scheme + "://" + parsed.Host
	}
	baseURL := origin + "/zajuna/course/view.php?id=" + url.QueryEscape(courseID)
	profileURL = strings.TrimSpace(profileURL)
	if profileURL == "" {
		profileURL = origin + "/zajuna/user/profile.php"
	}
	routeIndex := make(map[string]coursemaps.Route)
	routeOrder := make([]string, 0, len(activities))
	for _, activity := range activities {
		target, ok := normalizeInternalURL(baseURL, activity.URL, origin)
		if !ok {
			continue
		}
		if _, exists := routeIndex[target]; !exists {
			routeOrder = append(routeOrder, target)
		}
		mergeRoute(routeIndex, coursemaps.Route{URL: target, Kind: classifyRoute(target), Title: cleanText(activity.Label), Depth: 1, SourceURL: baseURL})
	}
	routes := make([]coursemaps.Route, 0, len(routeOrder))
	for _, target := range routeOrder {
		routes = append(routes, routeIndex[target])
	}
	byItemCode, stats := groupRoutesForCourse(routes, courseID, profileURL)
	now := time.Now().UTC()
	return coursemaps.Record{
		CourseID: courseID, CourseURL: baseURL, ProfileURL: profileURL,
		ByItemCode: byItemCode, Routes: routes, LinkCount: len(routes),
		ItemCodeCount: len(byItemCode), ScrapeStats: stats, Source: "devtools-import",
		DiscoveredAt: now, UpdatedAt: now,
	}, nil
}
