package sqlite

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zajuna-app/core/internal/checklist"
)

func (s *Store) ListRouteReviews(ctx context.Context, fichaID string) ([]checklist.RouteReview, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ficha_id, route_key, status, manual_url, manual_selector, note, updated_at
		FROM checklist_route_reviews WHERE ficha_id = ? ORDER BY updated_at DESC, route_key
	`, strings.TrimSpace(fichaID))
	if err != nil {
		return nil, fmt.Errorf("list route reviews: %w", err)
	}
	defer rows.Close()
	result := make([]checklist.RouteReview, 0)
	for rows.Next() {
		var review checklist.RouteReview
		var updatedAt string
		if err := rows.Scan(&review.FichaID, &review.RouteKey, &review.Status, &review.ManualURL, &review.ManualSelector, &review.Note, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan route review: %w", err)
		}
		review.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		result = append(result, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read route reviews: %w", err)
	}
	return result, nil
}

func (s *Store) UpsertRouteReview(ctx context.Context, review checklist.RouteReview) error {
	review.FichaID = strings.TrimSpace(review.FichaID)
	review.RouteKey = strings.TrimSpace(review.RouteKey)
	review.Status = strings.ToLower(strings.TrimSpace(review.Status))
	if review.FichaID == "" || review.RouteKey == "" {
		return errors.New("la revisión requiere ficha y ruta")
	}
	if !checklist.ValidRouteReviewStatus(review.Status) {
		return fmt.Errorf("estado de revisión inválido: %s", review.Status)
	}
	if len(review.ManualURL) > 2048 || len(review.ManualSelector) > 2048 || len(review.Note) > 500 {
		return errors.New("la revisión de ruta excede el tamaño permitido")
	}
	if review.UpdatedAt.IsZero() {
		review.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO checklist_route_reviews(ficha_id, route_key, status, manual_url, manual_selector, note, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ficha_id, route_key) DO UPDATE SET
			status = excluded.status,
			manual_url = excluded.manual_url,
			manual_selector = excluded.manual_selector,
			note = excluded.note,
			updated_at = excluded.updated_at
	`, review.FichaID, review.RouteKey, review.Status, strings.TrimSpace(review.ManualURL), strings.TrimSpace(review.ManualSelector), strings.TrimSpace(review.Note), review.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save route review: %w", err)
	}
	return nil
}
