package checklist

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/zajuna-app/core/internal/coursemaps"
)

// CaptureSpec describes the deterministic capture contract for one checklist item.
// The URL is resolved from the course map; selectors and label hints are applied
// by the local Chromium capture worker. Group rules are the first local mapping
// pass and remain versionable as Zajuna course structures evolve.
type CaptureSpec struct {
	ItemCode    string
	GroupName   string
	Name        string
	MaxSlots    int
	RouteKinds  []string
	CSSSelector string
	LabelHints  []string
}

type CaptureTarget struct {
	ItemCode             string   `json:"itemCode"`
	GroupName            string   `json:"groupName"`
	Name                 string   `json:"name"`
	URL                  string   `json:"url"`
	RouteKey             string   `json:"routeKey,omitempty"`
	ReviewStatus         string   `json:"reviewStatus,omitempty"`
	SlotNumber           int      `json:"slotNumber"`
	ActivityID           string   `json:"activityId,omitempty"`
	ActivityTitle        string   `json:"activityTitle,omitempty"`
	PhaseSection         int      `json:"phaseSection,omitempty"`
	Technical            bool     `json:"technical,omitempty"`
	CSSSelector          string   `json:"cssSelector"`
	CSSSelectorFallbacks []string `json:"cssSelectorFallbacks,omitempty"`
	RevealSelectors      []string `json:"revealSelectors,omitempty"`
	HideSelectors        []string `json:"hideSelectors,omitempty"`
	ViewportWidth        int      `json:"viewportWidth,omitempty"`
	ViewportHeight       int      `json:"viewportHeight,omitempty"`
	FullPage             bool     `json:"fullPage,omitempty"`
	LabelHint            string   `json:"labelHint,omitempty"`
	RouteKind            string   `json:"routeKind,omitempty"`
	RequireSelector      bool     `json:"requireSelector,omitempty"`
	OwnerOnly            bool     `json:"ownerOnly,omitempty"`
}

type CapturePlanSummary struct {
	ItemCount       int `json:"itemCount"`
	ResolvedItems   int `json:"resolvedItems"`
	UnresolvedItems int `json:"unresolvedItems"`
	SlotCount       int `json:"slotCount"`
	MaxSlotCount    int `json:"maxSlotCount"`
}

func CaptureSpecs() []CaptureSpec {
	items := Items()
	result := make([]CaptureSpec, 0, len(items))
	for _, item := range items {
		plan := captureGroupPlan(item.GroupName)
		labelHints := captureLabelHints(item.ItemCode, plan.labelHints)
		result = append(result, CaptureSpec{
			ItemCode: item.ItemCode, GroupName: item.GroupName,
			Name: item.Description, MaxSlots: item.MaxEvidences,
			RouteKinds: plan.routeKinds, CSSSelector: plan.selector,
			LabelHints: labelHints,
		})
	}
	return result
}

func captureLabelHints(itemCode string, fallback []string) []string {
	// The instructor profile is one logical full-page evidence. Per-field hints
	// would crop out the photo, name and the rest of the profile.
	if strings.HasPrefix(itemCode, "2.1.") {
		return nil
	}
	// Cronogramas vary between a native Moodle page and an embedded sheet.
	// The route itself identifies the cronograma; a field label is not stable
	// enough to reject the semantic main region when the HTML uses a different
	// layout.
	if strings.HasPrefix(itemCode, "1.1.") || strings.HasPrefix(itemCode, "1.2.") {
		return nil
	}
	// Forum and announcement pages expose the activity title in the heading,
	// while the actual Moodle discussion rows usually contain only the subject
	// and author. The owner filter is the reliable semantic constraint here;
	// requiring a second title hint would reject valid rows.
	if strings.HasPrefix(itemCode, "9.") || strings.HasPrefix(itemCode, "11.") || strings.HasPrefix(itemCode, "14.") || itemCode == "15.1" {
		return nil
	}
	itemHints := map[string][]string{
		"7.1.1": {"Reporte de Curso"}, "7.1.2": {"Seguimiento a la Formación"}, "7.2": {"Reporte de Curso"},
		"7.3.1": {"Comités evaluativos"}, "7.3.2": {"Documentos de retención"}, "7.3.3": {"Reuniones EEF"},
		"7.4.1": {"Actas de Comité"}, "7.4.2": {"Planes de Mejoramiento"}, "7.4.3": {"Registro de Novedades"}, "7.4.4": {"Llamados de Atención"},
		"8.1": {"Sesiones en Línea"}, "8.2": {"Subsecciones por Fase"}, "8.3": {"Subsecciones por Fase y Mes"},
		"9.1.1": {"Dudas e Inquietudes"}, "9.1.2": {"Dudas e Inquietudes"}, "9.1.3": {"Foro Temático"}, "9.1.4": {"Foro Temático"},
		"9.1.5": {"Dudas e Inquietudes"}, "9.1.6": {"Foro Temático"}, "9.1.7": {"Foro Temático"},
		"10.1.1": {"calificación"}, "10.1.2": {"tres días"},
		"11.1.1": {"inicio de fase"}, "11.1.2": {"Fecha de inicio"}, "11.1.3": {"Instrucciones"}, "11.1.4": {"Pasos a seguir"},
		"11.2.1": {"inicio de actividad"}, "11.2.2": {"cierre de actividad"}, "11.2.3": {"sesión en línea"},
		"11.3": {"aprendices aprobados"}, "11.4": {"Anuncio"},
		"12.1.1": {"Grabación"}, "12.1.2": {"Resumen"},
		"13.1.1": {"Reuniones EEF"}, "13.1.2": {"Comités"}, "13.1.3": {"Documentos de retención"},
		"13.2.1": {"calificaciones"}, "13.2.2": {"Formatos de cierre"},
		"14.1.1": {"Foro Temático"}, "14.1.2": {"Conclusión"}, "15.1": {"netiqueta"},
	}
	if hints, ok := itemHints[itemCode]; ok {
		return hints
	}
	return fallback
}

func BuildCaptureTargets(record coursemaps.Record) ([]CaptureTarget, CapturePlanSummary, error) {
	return BuildCaptureTargetsForActivities(record, nil)
}

// BuildCaptureTargetsForActivities applies the instructor's explicit
// activity selection to activity-bound evidence. General course/profile
// evidence remains available, while dates and assignment evidence are tied to
// the selected activity titles and IDs.
func BuildCaptureTargetsForActivities(record coursemaps.Record, selectedActivityIDs map[string]bool) ([]CaptureTarget, CapturePlanSummary, error) {
	if err := ValidateDefinitions(); err != nil {
		return nil, CapturePlanSummary{}, err
	}
	activitiesByID := make(map[string]coursemaps.Activity)
	for _, activity := range coursemaps.Activities(record) {
		activitiesByID[activity.ID] = activity
	}
	targets := make([]CaptureTarget, 0)
	summary := CapturePlanSummary{ItemCount: len(CaptureSpecs())}
	for _, spec := range CaptureSpecs() {
		if activityBoundItem(spec.ItemCode) && len(selectedActivityIDs) > 0 {
			selected := selectedActivities(activitiesByID, selectedActivityIDs)
			if len(selected) > spec.MaxSlots {
				selected = selected[:spec.MaxSlots]
			}
			for index, activity := range selected {
				name := fmt.Sprintf("%s — %s", spec.Name, activity.Title)
				activitySelector := activityCaptureSelector(activity.ID)
				targets = append(targets, CaptureTarget{
					ItemCode: spec.ItemCode, GroupName: spec.GroupName, Name: name,
					URL: record.CourseURL, SlotNumber: index + 1,
					ActivityID: activity.ID, ActivityTitle: activity.Title, PhaseSection: activity.PhaseSection, Technical: activity.Technical,
					CSSSelector:          activitySelector,
					CSSSelectorFallbacks: activityCaptureSelectorChain(activity.ID),
					RevealSelectors:      activityRevealSelectors(activity),
					RouteKind:            "course", RequireSelector: true,
				})
			}
			if len(selected) > 0 {
				summary.ResolvedItems++
				summary.SlotCount += len(selected)
			}
			summary.MaxSlotCount += spec.MaxSlots
			continue
		}
		urls, err := mappedURLs(record.ByItemCode[spec.ItemCode])
		if err != nil {
			return nil, CapturePlanSummary{}, fmt.Errorf("mapa inválido para %s: %w", spec.ItemCode, err)
		}
		if len(urls) == 0 {
			summary.UnresolvedItems++
			summary.MaxSlotCount += spec.MaxSlots
			continue
		}
		summary.ResolvedItems++
		summary.MaxSlotCount += spec.MaxSlots
		limit := len(urls)
		if limit > spec.MaxSlots {
			limit = spec.MaxSlots
		}
		addedCount := 0
		for index := 0; index < limit; index++ {
			route := routeForURL(record, urls[index])
			if route != nil && !eligibleRouteForGroup(spec.GroupName, *route, selectedActivityIDs, activitiesByID) {
				continue
			}
			activityID := activityIDForURL(record, urls[index])
			if selectionBoundItem(spec.ItemCode) && len(selectedActivityIDs) > 0 {
				if activityID == "" || !selectedActivityIDs[activityID] {
					continue
				}
			}
			hint := ""
			if len(spec.LabelHints) > 0 {
				hint = spec.LabelHints[index%len(spec.LabelHints)]
			}
			activityTitle := ""
			technical := false
			if activity, ok := activitiesByID[activityID]; ok {
				activityTitle = activity.Title
				technical = activity.Technical
			}
			name := spec.Name
			if spec.MaxSlots > 1 {
				name = fmt.Sprintf("%s — Evidencia %d", name, addedCount+1)
			}
			plan := captureGroupPlan(spec.GroupName)
			ownerOnly := ownerOnlyForItem(spec.ItemCode)
			selector := captureSelectorForItem(spec.ItemCode, spec.GroupName, spec.CSSSelector)
			targets = append(targets, CaptureTarget{
				ItemCode: spec.ItemCode, GroupName: spec.GroupName,
				Name: name, URL: urls[index], SlotNumber: index + 1,
				ActivityID: activityID, ActivityTitle: activityTitle, Technical: technical,
				CSSSelector: selector, CSSSelectorFallbacks: captureSelectorChainForItem(spec.ItemCode, spec.GroupName, selector),
				HideSelectors: forumConfigurationHideSelectors(spec.ItemCode, spec.GroupName),
				ViewportWidth: viewportWidthForGroup(spec.GroupName), ViewportHeight: viewportHeightForGroup(spec.GroupName),
				FullPage:  fullPageForGroup(spec.GroupName),
				LabelHint: hint, RouteKind: routeKindForURL(record, spec.ItemCode, urls[index]),
				RequireSelector: plan.ownerOnly || spec.GroupName == "perfil_instructor" || hint != "" || ownerFilteredGroup(spec.GroupName) || spec.GroupName == "cronograma_general" || spec.GroupName == "cronograma_vigente",
				OwnerOnly:       ownerOnly,
			})
			addedCount++
		}
		summary.SlotCount += addedCount
	}
	return targets, summary, nil
}

func activityBoundItem(itemCode string) bool {
	return itemCode == "6.1" || selectionBoundItem(itemCode)
}

func selectionBoundItem(itemCode string) bool {
	switch itemCode {
	case "10.1.1", "10.1.2":
		return true
	default:
		return false
	}
}

func selectedActivities(byID map[string]coursemaps.Activity, selectedIDs map[string]bool) []coursemaps.Activity {
	result := make([]coursemaps.Activity, 0, len(selectedIDs))
	for id := range selectedIDs {
		if activity, ok := byID[id]; ok {
			result = append(result, activity)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].PhaseSection != result[j].PhaseSection {
			return result[i].PhaseSection < result[j].PhaseSection
		}
		return strings.ToLower(result[i].Title) < strings.ToLower(result[j].Title)
	})
	return result
}

func activityIDForURL(record coursemaps.Record, targetURL string) string {
	for _, route := range record.Routes {
		if route.URL != targetURL || route.Kind != "assign" {
			continue
		}
		if strings.TrimSpace(route.ActivityID) != "" {
			return strings.TrimSpace(route.ActivityID)
		}
		parsed, err := url.Parse(route.URL)
		if err == nil {
			return strings.TrimSpace(parsed.Query().Get("id"))
		}
	}
	parsed, err := url.Parse(targetURL)
	if err == nil && strings.Contains(parsed.Path, "/mod/assign/") {
		return strings.TrimSpace(parsed.Query().Get("id"))
	}
	return ""
}

func activityCaptureSelector(activityID string) string {
	id := strings.TrimSpace(activityID)
	return fmt.Sprintf("#region-main .course-content #module-%s", id)
}

func activityCaptureSelectorChain(activityID string) []string {
	id := strings.TrimSpace(activityID)
	if id == "" {
		return nil
	}
	return []string{
		activityCaptureSelector(id),
		fmt.Sprintf("#region-main .course-content li#module-%s .activity-item", id),
		fmt.Sprintf("#region-main .course-content .activity-item:has(a[href*=\"id=%s\"])", id),
		fmt.Sprintf("#region-main .course-content li#module-%s", id),
		fmt.Sprintf("#region-main .course-content a[href*=\"id=%s\"]", id),
	}
}

func activityRevealSelectors(activity coursemaps.Activity) []string {
	if activity.PhaseSection <= 0 {
		return nil
	}
	return []string{fmt.Sprintf("#collapssesection%d", activity.PhaseSection)}
}

func mappedURLs(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var urls []string
	if err := json.Unmarshal(raw, &urls); err != nil {
		var single string
		if singleErr := json.Unmarshal(raw, &single); singleErr != nil {
			return nil, err
		}
		urls = []string{single}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(urls))
	for _, value := range urls {
		value = strings.TrimSpace(value)
		key := canonicalRouteURL(value)
		if value == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result, nil
}

func routeKindForURL(record coursemaps.Record, itemCode, targetURL string) string {
	for _, route := range record.Routes {
		if route.URL == targetURL {
			return route.Kind
		}
	}
	for _, kind := range captureSpecFor(itemCode).RouteKinds {
		return kind
	}
	return "route"
}

func routeForURL(record coursemaps.Record, targetURL string) *coursemaps.Route {
	for index := range record.Routes {
		if record.Routes[index].URL == targetURL || canonicalRouteURL(record.Routes[index].URL) == canonicalRouteURL(targetURL) {
			return &record.Routes[index]
		}
	}
	return nil
}

func canonicalRouteURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}
	query := parsed.Query()
	query.Del("forceview")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// eligibleRouteForGroup prevents the route map's auxiliary forum links from
// becoming evidence. Forum pages are only eligible when they are a real
// view.php activity route, are not a discussion child/order/action URL, and
// are not explicitly transversal. If a route has an activity code, it must
// match one of the instructor-selected activities; generic named forums and
// announcements remain eligible because their author is checked in Chromium.
func eligibleRouteForGroup(groupName string, route coursemaps.Route, selectedActivityIDs map[string]bool, activitiesByID map[string]coursemaps.Activity) bool {
	if !ownerFilteredGroup(groupName) {
		return true
	}
	if route.Kind != "forum" {
		return false
	}
	parsed, err := url.Parse(route.URL)
	if err != nil || !strings.HasSuffix(strings.ToLower(parsed.Path), "/view.php") {
		return false
	}
	query := parsed.Query()
	if query.Get("id") == "" || query.Get("parent") != "" || query.Get("d") != "" || query.Get("o") != "" || query.Get("sesskey") != "" {
		return false
	}
	title := strings.TrimSpace(route.Title)
	if title == "" || genericForumNavigationTitle(title) {
		return false
	}
	if !coursemaps.IsTechnicalActivity(title) {
		return false
	}
	if len(selectedActivityIDs) == 0 {
		return true
	}
	if activityID := strings.TrimSpace(route.ActivityID); activityID != "" {
		return selectedActivityIDs[activityID]
	}
	if routeHasActivityCode(title) {
		for id := range selectedActivityIDs {
			activity, ok := activitiesByID[id]
			if ok && activityCodeOverlap(title, activity.Title) {
				return true
			}
		}
		return false
	}
	return true
}

func ownerFilteredGroup(groupName string) bool {
	switch groupName {
	case "foros", "anuncios_fase", "anuncios_semanales", "conclusion_foros", "netiqueta":
		return true
	default:
		return false
	}
}

func ownerOnlyForItem(itemCode string) bool {
	switch itemCode {
	case "9.1.5", "9.1.6", "9.1.7",
		"11.1.1", "11.1.2", "11.1.3", "11.1.4", "11.2.1", "11.2.2", "11.2.3", "11.3", "11.4",
		"14.1.1", "14.1.2", "15.1":
		return true
	default:
		return false
	}
}

// RequiresInstructorIdentity tells the worker whether a batch contains a
// target that must be scoped to the authenticated instructor before capture.
func RequiresInstructorIdentity(targets []CaptureTarget) bool {
	for _, target := range targets {
		if target.OwnerOnly {
			return true
		}
	}
	return false
}

func genericForumNavigationTitle(title string) bool {
	value := strings.ToLower(strings.TrimSpace(title))
	for _, term := range []string{"debate", "comenzado por", "último mensaje", "ultimo mensaje", "réplicas", "replicas", "fijar esta discusión", "fijar esta discusion", "mostrar comentarios", "excepciones"} {
		if value == term {
			return true
		}
	}
	return false
}

func routeHasActivityCode(title string) bool {
	value := strings.ToUpper(title)
	return strings.Contains(value, "GA") && strings.Contains(value, "AA")
}

func activityCodeOverlap(left, right string) bool {
	left = strings.ToUpper(strings.TrimSpace(left))
	right = strings.ToUpper(strings.TrimSpace(right))
	if left == "" || right == "" {
		return false
	}
	for _, leftToken := range activityReferencePattern.FindAllString(left, -1) {
		for _, rightToken := range activityReferencePattern.FindAllString(right, -1) {
			if leftToken == rightToken {
				return true
			}
		}
	}
	return false
}

var activityReferencePattern = regexp.MustCompile(`(?i)AA\d+(?:-EV\d+)?`)

type groupPlan struct {
	routeKinds []string
	selector   string
	labelHints []string
	ownerOnly  bool
}

func captureGroupPlan(groupName string) groupPlan {
	switch groupName {
	case "cronograma_general":
		return groupPlan{[]string{"page"}, `#region-main .course-content:has(iframe[src*="docs.google.com/spreadsheets"]), #region-main .course-content`, []string{"Fases", "Actividades de proyecto", "Actividades de aprendizaje", "duración", "Fecha de inicio"}, false}
	case "cronograma_vigente":
		return groupPlan{[]string{"phase"}, "#region-main .course-content .section", []string{"Nombre de la Fase", "actividades de proyecto", "Actividades de aprendizaje", "Resultados de Aprendizaje", "Fecha de inicio", "Evidencias a presentar", "Instructor"}, false}
	case "perfil_instructor":
		return groupPlan{[]string{"profile"}, "#page-user-profile", nil, false}
	case "disponibilidad":
		return groupPlan{[]string{"page", "resource", "url"}, "#region-main .course-content .section", []string{"material de trabajo", "evidencias"}, false}
	case "menu_curso":
		return groupPlan{[]string{"course"}, "#region-main .course-content", []string{"secciones"}, false}
	case "calificaciones":
		return groupPlan{[]string{"grading"}, "#region-main .gradereport-grader-table", []string{"calificaciones"}, false}
	case "configuracion", "seguimiento_evaluacion", "seguimiento_documentos", "sesiones_linea", "documentos_retencion":
		return groupPlan{[]string{"page", "course", "phase"}, "#region-main .course-content .section", nil, false}
	case "foros", "anuncios_fase", "anuncios_semanales", "conclusion_foros", "netiqueta":
		return groupPlan{[]string{"forum"}, "#region-main .forum_list .forum", []string{"Foro", "Anuncio", "sesión en línea"}, true}
	case "evidencias_aprendizaje":
		return groupPlan{[]string{"assign", "grading"}, "#region-main .assign", []string{"calificación", "retroalimentación"}, false}
	case "sesiones_semanales":
		return groupPlan{[]string{"page", "resource", "url"}, "#region-main .course-content .section", []string{"Grabación", "Resumen"}, false}
	default:
		return groupPlan{[]string{"page", "course"}, "#region-main", nil, false}
	}
}

func captureSelectorChain(groupName, primary string) []string {
	selectors := make([]string, 0, 8)
	add := func(selector string) {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			return
		}
		for _, existing := range selectors {
			if existing == selector {
				return
			}
		}
		selectors = append(selectors, selector)
	}
	add(primary)
	groupSelectors := map[string][]string{
		"cronograma_general":     {"#region-main .course-content", "#region-main"},
		"cronograma_vigente":     {"#region-main .course-content .section", "#region-main .course-content", "#region-main"},
		"disponibilidad":         {"#region-main .course-content .section", "#region-main .course-content"},
		"perfil_instructor":      {"#page-user-profile", "#region-main"},
		"menu_curso":             {"#region-main .course-content", ".course-content"},
		"calificaciones":         {"#region-main .gradereport-grader-table", "#region-main table", "#region-main"},
		"foros":                  {"#region-main .forum_list .forum", "#region-main .forumpost", "#region-main [data-region='post']"},
		"anuncios_fase":          {"#region-main .forum_list .forum", "#region-main .forumpost", "#region-main [data-region='post']"},
		"anuncios_semanales":     {"#region-main .forum_list .forum", "#region-main .forumpost", "#region-main [data-region='post']"},
		"conclusion_foros":       {"#region-main .forum_list .forum", "#region-main .forumpost", "#region-main [data-region='post']"},
		"netiqueta":              {"#region-main .forum_list .forum", "#region-main .forumpost", "#region-main [data-region='post']"},
		"evidencias_aprendizaje": {"#region-main .assign", "#region-main table", "#region-main"},
		"sesiones_semanales":     {"#region-main .course-content .section", "#region-main"},
	}
	for _, selector := range groupSelectors[groupName] {
		add(selector)
	}
	if ownerFilteredGroup(groupName) {
		selectors = append(selectors,
			"#region-main table.forumheaderlist tr",
			"#region-main .discussion",
			"#region-main .forum-post",
			"#region-main article",
		)
		return selectors
	}
	for _, selector := range []string{"#region-main .course-content", "#region-main", "#page-user-profile", ".course-content", "#page-content"} {
		add(selector)
	}
	return selectors
}

func captureSelectorForItem(itemCode, groupName, fallback string) string {
	if groupName == "foros" && (itemCode == "9.1.1" || itemCode == "9.1.2" || itemCode == "9.1.3" || itemCode == "9.1.4") {
		return "#page-mod-forum-view #region-main"
	}
	return fallback
}

func captureSelectorChainForItem(itemCode, groupName, primary string) []string {
	return captureSelectorChain(groupName, primary)
}

func forumConfigurationHideSelectors(itemCode, groupName string) []string {
	if groupName == "foros" && !ownerOnlyForItem(itemCode) {
		return []string{"#region-main table"}
	}
	return nil
}

func viewportWidthForGroup(groupName string) int {
	if groupName == "cronograma_general" || groupName == "cronograma_vigente" {
		return 2560
	}
	return 0
}

func viewportHeightForGroup(groupName string) int {
	if groupName == "cronograma_general" || groupName == "cronograma_vigente" {
		return 1200
	}
	return 0
}

func fullPageForGroup(groupName string) bool {
	// The instructor profile is a single logical evidence. It must include the
	// photo, identity, description, contact details and availability instead of
	// cropping only the first visible profile card. Cronogramas use the same
	// contract so native HTML phases include the complete table below the fold.
	return groupName == "perfil_instructor" || groupName == "cronograma_general" || groupName == "cronograma_vigente"
}

func captureSpecFor(itemCode string) CaptureSpec {
	for _, spec := range CaptureSpecs() {
		if spec.ItemCode == itemCode {
			return spec
		}
	}
	return CaptureSpec{ItemCode: itemCode, RouteKinds: []string{"route"}}
}
