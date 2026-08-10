package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zajuna-app/core/internal/coursemaps"
)

// ListSelectedActivityIDs returns the activity IDs explicitly selected by the
// instructor for a ficha. An empty result means the selection step has not
// been completed yet.
func (s *Store) ListSelectedActivityIDs(ctx context.Context, fichaID string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT activity_id FROM ficha_activity_selections WHERE ficha_id = ?`, strings.TrimSpace(fichaID))
	if err != nil {
		return nil, fmt.Errorf("list selected activities: %w", err)
	}
	defer rows.Close()
	result := make(map[string]bool)
	for rows.Next() {
		var activityID string
		if err := rows.Scan(&activityID); err != nil {
			return nil, fmt.Errorf("scan selected activity: %w", err)
		}
		result[activityID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read selected activities: %w", err)
	}
	return result, nil
}

// ReplaceSelectedActivities persists the user's explicit activity choices as
// a ficha-scoped snapshot. The caller validates that every activity belongs
// to the current course map before calling this method.
func (s *Store) ReplaceSelectedActivities(ctx context.Context, fichaID string, activities []coursemaps.Activity) error {
	fichaID = strings.TrimSpace(fichaID)
	if fichaID == "" {
		return fmt.Errorf("fichaId es obligatorio")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin activity selection: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM ficha_activity_selections WHERE ficha_id = ?`, fichaID); err != nil {
		return fmt.Errorf("clear activity selection: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, activity := range activities {
		if strings.TrimSpace(activity.ID) == "" || strings.TrimSpace(activity.URL) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ficha_activity_selections(ficha_id, activity_id, title, url, phase_name, phase_section, subsection, technical, updated_at)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fichaID, activity.ID, activity.Title, activity.URL, activity.PhaseName, activity.PhaseSection, activity.Subsection, boolToInt(activity.Technical), now); err != nil {
			return fmt.Errorf("save selected activity %s: %w", activity.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit activity selection: %w", err)
	}
	return nil
}

// GetSelectedActivities is useful for diagnostics and report metadata.
func (s *Store) GetSelectedActivities(ctx context.Context, fichaID string) ([]coursemaps.Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT activity_id, title, url, phase_name, phase_section, subsection, technical
		FROM ficha_activity_selections WHERE ficha_id = ? ORDER BY phase_section, lower(title), activity_id
	`, strings.TrimSpace(fichaID))
	if err != nil {
		return nil, fmt.Errorf("list selected activity records: %w", err)
	}
	defer rows.Close()
	result := make([]coursemaps.Activity, 0)
	for rows.Next() {
		var activity coursemaps.Activity
		var technical int
		if err := rows.Scan(&activity.ID, &activity.Title, &activity.URL, &activity.PhaseName, &activity.PhaseSection, &activity.Subsection, &technical); err != nil {
			return nil, fmt.Errorf("scan selected activity record: %w", err)
		}
		activity.Technical = technical == 1
		result = append(result, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read selected activity records: %w", err)
	}
	return result, nil
}

var _ interface {
	ListSelectedActivityIDs(context.Context, string) (map[string]bool, error)
	ReplaceSelectedActivities(context.Context, string, []coursemaps.Activity) error
} = (*Store)(nil)
