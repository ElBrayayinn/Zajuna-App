package evidence

import (
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MDL-33 asks for the Zajuna selectors and capture rules to be registered from
// the authenticated E2E against a real course. BuildSelectorReport turns the
// evidence rows a real capture produced into that register.
//
// The report is built only from evidence metadata, which the capture worker has
// already passed through security.RedactURL and security.RedactText. It also
// drops the page title and the absolute file path on purpose: a title can carry
// the instructor's name, and a path leaks the operator's home directory. What
// remains is the route and the selector rules, which is exactly what has to be
// reviewed and committed.

// SelectorObservation is one capture unit: the route that was opened, the
// selector that produced the screenshot, and the rules applied around it.
type SelectorObservation struct {
	CaptureUnitKey string   `json:"captureUnitKey,omitempty"`
	RoutePath      string   `json:"routePath,omitempty"`
	RouteKind      string   `json:"routeKind,omitempty"`
	GroupName      string   `json:"groupName,omitempty"`
	PhaseSection   int      `json:"phaseSection,omitempty"`
	ActivityID     string   `json:"activityId,omitempty"`
	ItemCodes      []string `json:"itemCodes"`
	// PrimarySelector is the rule the checklist intended to use; Selector is the
	// one that actually produced the screenshot. When they differ the course
	// needed a coarser rule, which is the finding worth reviewing.
	PrimarySelector string   `json:"primarySelector,omitempty"`
	Selector        string   `json:"selector,omitempty"`
	SelectorMatched bool     `json:"selectorMatched"`
	UsedFallback    bool     `json:"usedFallback,omitempty"`
	Fallbacks       []string `json:"fallbacks,omitempty"`
	LabelHint       string   `json:"labelHint,omitempty"`
	RevealSelectors []string `json:"revealSelectors,omitempty"`
	HideSelectors   []string `json:"hideSelectors,omitempty"`
	Viewport        string   `json:"viewport,omitempty"`
	FullPage        bool     `json:"fullPage,omitempty"`
	OwnerOnly       bool     `json:"ownerOnly,omitempty"`
	Technical       bool     `json:"technical,omitempty"`
	Redirected      bool     `json:"redirected,omitempty"`
}

// SelectorUsage counts how often one selector produced evidence, so a reviewer
// can tell a load-bearing rule from a rule that never fires.
type SelectorUsage struct {
	Selector string `json:"selector"`
	Matched  int    `json:"matched"`
	Fallback int    `json:"fallback"`
}

// SelectorReport is the committed artefact for MDL-33.
type SelectorReport struct {
	GeneratedAt time.Time `json:"generatedAt"`
	FichaID     string    `json:"fichaId,omitempty"`
	// CaptureOutcome records how the capture job ended. A partial failure is a
	// real result on a live course and its diagnostic names the rule that did
	// not match, so it belongs in the register instead of being hidden.
	CaptureOutcome string `json:"captureOutcome,omitempty"`
	CaptureUnits   int    `json:"captureUnits"`
	ItemsCovered   int    `json:"itemsCovered"`
	// MatchedUnits counted a selector that produced a screenshot.
	MatchedUnits int `json:"matchedUnits"`
	// FallbackChainUnits matched, but with a fallback instead of the intended
	// rule. FullPageUnits matched nothing and captured the whole page.
	FallbackChainUnits int `json:"fallbackChainUnits"`
	FullPageUnits      int `json:"fullPageUnits"`
	// SkippedRecords counts evidence rows whose metadata could not be decoded.
	// It must stay at zero: a silent skip once emptied the whole register when a
	// metadata field changed type.
	SkippedRecords int                   `json:"skippedRecords,omitempty"`
	Usage          []SelectorUsage       `json:"selectorUsage"`
	Observations   []SelectorObservation `json:"observations"`
}

// captureMetadata mirrors the keys the capture worker writes. Unknown keys are
// ignored so the report survives new metadata without changes here.
type captureMetadata struct {
	URL               string   `json:"url"`
	FinalURL          string   `json:"finalUrl"`
	RouteKey          string   `json:"routeKey"`
	Selector          string   `json:"selector"`
	SelectorFallbacks []string `json:"selectorFallbacks"`
	SelectorMatched   bool     `json:"selectorMatched"`
	LabelHint         string   `json:"labelHint"`
	RouteKind         string   `json:"routeKind"`
	GroupName         string   `json:"groupName"`
	RevealSelectors   []string `json:"revealSelectors"`
	HideSelectors     []string `json:"hideSelectors"`
	ViewportWidth     int      `json:"viewportWidth"`
	ViewportHeight    int      `json:"viewportHeight"`
	FullPage          bool     `json:"fullPage"`
	PhaseSection      int      `json:"phaseSection"`
	ActivityID        string   `json:"activityId"`
	OwnerOnly         bool     `json:"ownerOnly"`
	Technical         bool     `json:"technical"`
	CaptureUnitKey    string   `json:"captureUnitKey"`
}

// BuildSelectorReport groups the evidence rows by capture unit. One capture unit
// can cover several checklist items, so the item codes are merged instead of
// repeating the same selector once per item.
func BuildSelectorReport(fichaID string, generatedAt time.Time, records []Record) SelectorReport {
	byUnit := map[string]*SelectorObservation{}
	order := make([]string, 0, len(records))
	items := map[string]bool{}
	skipped := 0

	for _, record := range records {
		if len(record.Metadata) == 0 {
			skipped++
			continue
		}
		var metadata captureMetadata
		if err := json.Unmarshal(record.Metadata, &metadata); err != nil {
			skipped++
			continue
		}
		key := firstNonBlank(metadata.CaptureUnitKey, metadata.RouteKey, routePath(metadata.URL), record.ItemCode)
		observation, seen := byUnit[key]
		if !seen {
			observation = &SelectorObservation{
				CaptureUnitKey:  RouteOnly(metadata.CaptureUnitKey),
				RoutePath:       routePath(metadata.URL),
				RouteKind:       metadata.RouteKind,
				GroupName:       metadata.GroupName,
				PhaseSection:    metadata.PhaseSection,
				ActivityID:      metadata.ActivityID,
				PrimarySelector: primarySelector(metadata),
				Selector:        metadata.Selector,
				SelectorMatched: metadata.SelectorMatched,
				UsedFallback:    metadata.SelectorMatched && metadata.Selector != primarySelector(metadata),
				Fallbacks:       dedupe(metadata.SelectorFallbacks),
				LabelHint:       metadata.LabelHint,
				RevealSelectors: dedupe(metadata.RevealSelectors),
				HideSelectors:   dedupe(metadata.HideSelectors),
				Viewport:        viewport(metadata.ViewportWidth, metadata.ViewportHeight),
				FullPage:        metadata.FullPage,
				OwnerOnly:       metadata.OwnerOnly,
				Technical:       metadata.Technical,
				Redirected:      routePath(metadata.FinalURL) != routePath(metadata.URL),
			}
			byUnit[key] = observation
			order = append(order, key)
		}
		if record.ItemCode != "" && !containsString(observation.ItemCodes, record.ItemCode) {
			observation.ItemCodes = append(observation.ItemCodes, record.ItemCode)
			items[record.ItemCode] = true
		}
	}

	report := SelectorReport{
		GeneratedAt:    generatedAt.UTC(),
		FichaID:        fichaID,
		ItemsCovered:   len(items),
		SkippedRecords: skipped,
		Observations:   make([]SelectorObservation, 0, len(order)),
	}
	usage := map[string]*SelectorUsage{}
	for _, key := range order {
		observation := byUnit[key]
		sort.Strings(observation.ItemCodes)
		report.Observations = append(report.Observations, *observation)
		if observation.SelectorMatched {
			report.MatchedUnits++
			if observation.UsedFallback {
				report.FallbackChainUnits++
			}
		} else {
			report.FullPageUnits++
		}
		if observation.Selector == "" {
			continue
		}
		entry, seen := usage[observation.Selector]
		if !seen {
			entry = &SelectorUsage{Selector: observation.Selector}
			usage[observation.Selector] = entry
		}
		if observation.SelectorMatched {
			entry.Matched++
		} else {
			entry.Fallback++
		}
	}
	report.CaptureUnits = len(report.Observations)
	report.Usage = make([]SelectorUsage, 0, len(usage))
	for _, entry := range usage {
		report.Usage = append(report.Usage, *entry)
	}
	sort.Slice(report.Usage, func(first, second int) bool {
		left, right := report.Usage[first], report.Usage[second]
		if left.Matched != right.Matched {
			return left.Matched > right.Matched
		}
		return left.Selector < right.Selector
	})
	sort.Slice(report.Observations, func(first, second int) bool {
		left, right := report.Observations[first], report.Observations[second]
		if left.GroupName != right.GroupName {
			return left.GroupName < right.GroupName
		}
		if len(left.ItemCodes) > 0 && len(right.ItemCodes) > 0 && left.ItemCodes[0] != right.ItemCodes[0] {
			return left.ItemCodes[0] < right.ItemCodes[0]
		}
		return left.RoutePath < right.RoutePath
	})
	return report
}

// primarySelector recovers the rule the checklist intended. The capture worker
// builds the fallback chain with the primary rule first, so its head is the
// intended selector.
func primarySelector(metadata captureMetadata) string {
	for _, candidate := range metadata.SelectorFallbacks {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return metadata.Selector
}

var absoluteURLPattern = regexp.MustCompile(`https?://[^\s"'|)]+`)

// RouteOnly rewrites every absolute URL inside a diagnostic string to its
// Moodle route. The register is a rule book, so it must not depend on the
// deployment host, and the capture worker embeds full URLs in its keys.
func RouteOnly(value string) string {
	return absoluteURLPattern.ReplaceAllStringFunc(value, func(match string) string {
		if path := routePath(match); path != "" {
			return path
		}
		return match
	})
}

// routePath keeps only the Moodle route, so the report reads as a rule book and
// carries no host or credential material.
func routePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.RawQuery == "" {
		return parsed.Path
	}
	return parsed.Path + "?" + parsed.RawQuery
}

func viewport(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	return strconv.Itoa(width) + "x" + strconv.Itoa(height)
}

func dedupe(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || containsString(result, value) {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
