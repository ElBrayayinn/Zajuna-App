package zajuna

import (
	"net/url"
	"strings"

	"github.com/zajuna-app/core/internal/coursemaps"
)

// The resolver mirrors the proven activity pools from zajuna-sync. Generic
// route groups remain available for diagnostics, while these rules provide
// exact URLs for itemCode/slot whenever the activity title is sufficiently
// descriptive.
var forumResolveModes = map[string]string{
	"9.1.1": "dudas_singleton", "9.1.2": "dudas_singleton", "9.1.5": "dudas_singleton",
	"9.1.3": "tematico_slot", "9.1.4": "tematico_slot", "9.1.6": "tematico_slot", "9.1.7": "tematico_slot",
	"11.1.1": "anuncios_singleton", "11.1.2": "anuncios_singleton", "11.1.3": "anuncios_singleton", "11.1.4": "anuncios_singleton",
	"11.2.1": "anuncios_singleton", "11.2.2": "anuncios_singleton", "11.2.3": "sesion_slot",
	"11.3": "anuncios_singleton", "11.4": "anuncio_slot",
	"14.1.1": "tematico_slot", "14.1.2": "tematico_slot", "15.1": "induccion_singleton",
}

var forumPoolTerms = map[string][]string{
	"dudas_singleton":     {"foro de dudas", "dudas o inquietudes", "dudas e inquietudes"},
	"tematico_slot":       {"foro temático", "foro tematico", "temático", "tematico"},
	"anuncios_singleton":  {"anuncios"},
	"anuncio_slot":        {"anuncio", "comunicativa", "aprendices aprobados"},
	"sesion_slot":         {"sesión en línea", "sesion en linea", "sesión sincrónica", "grabación sesión"},
	"induccion_singleton": {"foro de inducción", "induccion", "inducción"},
}

var pagePoolTerms = map[string][]string{
	"cronograma_general_singleton": {"cronograma general", "cronograma  general"},
	"fase_page_slot":              {"cronograma fase", "cronograma  fase", "fase análisis", "fase analisis", "fase hacer", "fase verificar"},
	"grabacion_slot":              {"grabación", "grabacion", "resumen sesión", "resumen sesion", "sesión en línea", "sesion en linea"},
	"assign_slot":                 {"evidencia", "ga1-", "ga2-", "ga3-"},
}

func buildExactChecklistRouteGroups(routes []coursemaps.Route, courseID, profileURL string) map[string][]string {
	groups := make(map[string][]string)
	origin := "https://zajuna.sena.edu.co"
	if parsed, err := url.Parse(profileURL); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		origin = parsed.Scheme + "://" + parsed.Host
	}
	put := func(codes []string, values []string) {
		if len(values) == 0 {
			return
		}
		for _, code := range codes {
			groups[code] = append([]string(nil), values...)
		}
	}

	cronograma := pickSingletonRoute(routes, []string{"page", "resource"}, pagePoolTerms["cronograma_general_singleton"], 6, nil)
	put([]string{"1.1.1", "1.1.2", "1.1.3", "1.1.4", "1.1.5"}, singleValue(cronograma))

	fases := buildOrderedRoutePool(routes, []string{"page", "resource"}, pagePoolTerms["fase_page_slot"], 6, func(route coursemaps.Route) bool {
		text := resolverText(route)
		return strings.Contains(text, "fase") || strings.Contains(text, "cronograma")
	})
	put([]string{"1.2.1", "1.2.2", "1.2.3", "1.2.4", "1.2.5", "1.2.6", "1.2.7"}, fases)

	forumPools := make(map[string][]string, len(forumPoolTerms))
	for mode, terms := range forumPoolTerms {
		if strings.HasSuffix(mode, "_singleton") {
			forumPools[mode] = singleValue(pickSingletonRoute(routes, []string{"forum"}, terms, 8, nil))
		} else {
			forumPools[mode] = buildOrderedRoutePool(routes, []string{"forum"}, terms, 8, nil)
		}
	}
	for itemCode, mode := range forumResolveModes {
		if strings.HasSuffix(mode, "_singleton") {
			put([]string{itemCode}, forumPools[mode])
		} else {
			put([]string{itemCode}, forumPools[mode])
		}
	}

	assigns := buildOrderedRoutePool(routes, []string{"assign"}, pagePoolTerms["assign_slot"], 4, nil)
	if len(assigns) == 0 {
		assigns = orderedRoutesByKind(routes, []string{"assign"})
	}
	put([]string{"10.1.1", "10.1.2"}, assigns)

	grabaciones := buildOrderedRoutePool(routes, []string{"page", "resource", "url", "route"}, pagePoolTerms["grabacion_slot"], 6, func(route coursemaps.Route) bool {
		return strings.Contains(strings.ToLower(route.URL), "/mod/plugnmeet/") || route.Kind != "route"
	})
	put([]string{"12.1.1", "12.1.2"}, grabaciones)

	if numericCourseID(courseID) {
		groups["5.1"] = []string{origin + "/zajuna/grade/report/grader/index.php?id=" + url.QueryEscape(courseID)}
		coursePage := origin + "/zajuna/course/view.php?id=" + url.QueryEscape(courseID)
		put([]string{"3.1", "4.1", "6.1", "7.1.1", "7.1.2", "7.2", "7.3.1", "7.3.2", "7.3.3", "7.4.1", "7.4.2", "7.4.3", "7.4.4", "8.1", "8.2", "8.3", "13.1.1", "13.1.2", "13.1.3", "13.2.1", "13.2.2"}, []string{coursePage})
	}
	if strings.TrimSpace(profileURL) != "" {
		put([]string{"2.1.1", "2.1.2", "2.1.3", "2.1.4", "2.1.5"}, []string{strings.TrimSpace(profileURL)})
	}
	return groups
}

type routeMatch struct {
	route coursemaps.Route
	index int
	score int
}

func pickSingletonRoute(routes []coursemaps.Route, kinds, terms []string, minimum int, filter func(coursemaps.Route) bool) *routeMatch {
	matches := matchingRoutes(routes, kinds, terms, minimum, filter)
	if len(matches) == 0 {
		return nil
	}
	best := matches[0]
	for _, match := range matches[1:] {
		if match.score > best.score || (match.score == best.score && match.index < best.index) {
			best = match
		}
	}
	return &best
}

func buildOrderedRoutePool(routes []coursemaps.Route, kinds, terms []string, minimum int, filter func(coursemaps.Route) bool) []string {
	matches := matchingRoutes(routes, kinds, terms, minimum, filter)
	values := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		value := forceViewURL(match.route.URL)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	return values
}

func orderedRoutesByKind(routes []coursemaps.Route, kinds []string) []string {
	values := make([]string, 0)
	seen := map[string]bool{}
	for _, route := range routes {
		if !routeHasKind(route, kinds) {
			continue
		}
		value := forceViewURL(route.URL)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	return values
}

func matchingRoutes(routes []coursemaps.Route, kinds, terms []string, minimum int, filter func(coursemaps.Route) bool) []routeMatch {
	matches := make([]routeMatch, 0)
	for index, route := range routes {
		if !routeHasKind(route, kinds) || (filter != nil && !filter(route)) {
			continue
		}
		score := scoreRoute(route, terms)
		if score >= minimum {
			matches = append(matches, routeMatch{route: route, index: index, score: score})
		}
	}
	return matches
}

func routeHasKind(route coursemaps.Route, kinds []string) bool {
	for _, kind := range kinds {
		if route.Kind == kind {
			return true
		}
	}
	return false
}

func scoreRoute(route coursemaps.Route, terms []string) int {
	label := normalizeResolverText(route.Title)
	searchable := resolverText(route)
	score := 0
	for _, term := range terms {
		term = normalizeResolverText(term)
		if term == "" {
			continue
		}
		if label == term {
			score += 50
		} else if strings.Contains(label, term) {
			score += len([]rune(term)) + 10
		} else if strings.Contains(searchable, term) {
			score += 4
		}
	}
	return score
}

func resolverText(route coursemaps.Route) string {
	return normalizeResolverText(strings.Join([]string{route.Title, route.PhaseName, route.Subsection, route.URL}, " "))
}

func normalizeResolverText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	).Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func singleValue(match *routeMatch) []string {
	if match == nil {
		return nil
	}
	value := forceViewURL(match.route.URL)
	if value == "" {
		return nil
	}
	return []string{value}
}

func forceViewURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.Contains(strings.ToLower(rawURL), "/user/profile.php") || strings.Contains(strings.ToLower(rawURL), "forceview=1") {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return rawURL
	}
	if !strings.Contains(strings.ToLower(parsed.Path), "/mod/") || !strings.HasSuffix(strings.ToLower(parsed.Path), "/view.php") {
		return rawURL
	}
	query := parsed.Query()
	query.Set("forceview", "1")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func numericCourseID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
