package checklist

import (
	"fmt"
	"strings"
	"time"
)

// RouteReview stores the instructor's decision for one grouped capture route.
// ManualURL and ManualSelector are optional overrides used only when the
// automatically discovered route needs correction.
type RouteReview struct {
	FichaID        string    `json:"fichaId"`
	RouteKey       string    `json:"routeKey"`
	Status         string    `json:"status"`
	ManualURL      string    `json:"manualUrl,omitempty"`
	ManualSelector string    `json:"manualSelector,omitempty"`
	Note           string    `json:"note,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

const (
	RouteReviewPending    = "review"
	RouteReviewConfirmed  = "confirmed"
	RouteReviewCorrection = "correction"
)

func ValidRouteReviewStatus(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case RouteReviewPending, RouteReviewConfirmed, RouteReviewCorrection:
		return true
	default:
		return false
	}
}

// RouteKey is deliberately derived from the same fields that the UI uses to
// group targets. It stays stable when several checklist items share a capture.
func RouteKey(target CaptureTarget) string {
	return fmt.Sprintf("%s|%s|%s", strings.TrimSpace(target.GroupName), strings.TrimSpace(target.URL), strings.TrimSpace(target.RouteKind))
}

// ApplyRouteReviews enriches the generated plan with persisted decisions and
// applies only explicit manual overrides. A pending route keeps its automatic
// URL and selector, while a correction can replace either field.
func ApplyRouteReviews(targets []CaptureTarget, reviews []RouteReview) []CaptureTarget {
	byKey := make(map[string]RouteReview, len(reviews))
	for _, review := range reviews {
		if strings.TrimSpace(review.RouteKey) != "" {
			byKey[review.RouteKey] = review
		}
	}
	result := make([]CaptureTarget, len(targets))
	copy(result, targets)
	for index := range result {
		result[index].RouteKey = RouteKey(result[index])
		result[index].ReviewStatus = RouteReviewPending
		if review, ok := byKey[result[index].RouteKey]; ok {
			result[index].ReviewStatus = review.Status
			if strings.TrimSpace(review.ManualURL) != "" {
				result[index].URL = strings.TrimSpace(review.ManualURL)
			}
			if strings.TrimSpace(review.ManualSelector) != "" {
				result[index].CSSSelector = strings.TrimSpace(review.ManualSelector)
				result[index].CSSSelectorFallbacks = []string{result[index].CSSSelector}
			}
		}
	}
	return result
}
