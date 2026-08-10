package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/zajuna-app/core/internal/checklist"
	"github.com/zajuna-app/core/internal/evidence"
)

func (s *Store) ListEvidencesByFicha(ctx context.Context, fichaID string, limit int) ([]evidence.Record, error) {
	if strings.TrimSpace(fichaID) == "" {
		return nil, errors.New("fichaId es obligatorio")
	}
	if limit <= 0 {
		limit = 1000
	}
	if limit > 10000 {
		limit = 10000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(ficha_id, ''), item_code, slot_number, name, file_path, format, source, sha256, metadata_json, captured_at
		FROM evidences WHERE ficha_id = ? ORDER BY captured_at, id LIMIT ?
	`, fichaID, limit)
	if err != nil {
		return nil, fmt.Errorf("list evidences by ficha: %w", err)
	}
	defer rows.Close()
	result := make([]evidence.Record, 0)
	for rows.Next() {
		item, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read evidences by ficha: %w", err)
	}
	return result, nil
}

func (s *Store) RebuildEvidenceGroups(ctx context.Context, fichaID string) ([]evidence.Group, error) {
	records, err := s.ListEvidencesByFicha(ctx, fichaID, 10000)
	if err != nil {
		return nil, err
	}

	groupsByKey := make(map[string]*evidence.Group)
	for _, record := range records {
		if !evidence.EligibleForReport(record) {
			continue
		}
		key, confidence, reason := evidenceGroupSignature(record)
		group := groupsByKey[key]
		if group == nil {
			group = &evidence.Group{
				FichaID: fichaID, GroupKey: key, Title: record.Name,
				Confidence: confidence, Reason: reason,
			}
			groupsByKey[key] = group
		}
		group.EvidenceIDs = append(group.EvidenceIDs, record.ID)
		group.ItemCodes = appendUnique(group.ItemCodes, record.ItemCode)
		group.Evidences = append(group.Evidences, record)
	}

	groups := make([]evidence.Group, 0, len(groupsByKey))
	for _, group := range groupsByKey {
		group.ID = evidenceGroupID(fichaID, group.GroupKey)
		sort.Strings(group.EvidenceIDs)
		sort.Strings(group.ItemCodes)
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].GroupKey < groups[j].GroupKey })

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin evidence groups rebuild: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM evidence_groups WHERE ficha_id = ?`, fichaID); err != nil {
		return nil, fmt.Errorf("clear evidence groups: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, group := range groups {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO evidence_groups(id, ficha_id, group_key, title, confidence, reason, created_at, updated_at)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		`, group.ID, fichaID, group.GroupKey, group.Title, group.Confidence, group.Reason, now, now); err != nil {
			return nil, fmt.Errorf("insert evidence group %s: %w", group.ID, err)
		}
		for _, record := range group.Evidences {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO evidence_group_members(group_id, evidence_id, item_code, slot_number)
				VALUES(?, ?, ?, ?)
			`, group.ID, record.ID, record.ItemCode, record.SlotNumber); err != nil {
				return nil, fmt.Errorf("link evidence group %s: %w", group.ID, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit evidence groups: %w", err)
	}
	return groups, nil
}

func (s *Store) ListEvidenceGroups(ctx context.Context, fichaID string) ([]evidence.Group, error) {
	if strings.TrimSpace(fichaID) == "" {
		return nil, errors.New("fichaId es obligatorio")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, g.ficha_id, g.group_key, g.title, g.confidence, g.reason,
		       e.id, COALESCE(e.ficha_id, ''), e.item_code, e.slot_number, e.name,
		       e.file_path, e.format, e.source, e.sha256, e.metadata_json, e.captured_at
		FROM evidence_groups g
		JOIN evidence_group_members m ON m.group_id = g.id
		JOIN evidences e ON e.id = m.evidence_id
		WHERE g.ficha_id = ?
		ORDER BY g.group_key, e.captured_at, e.id
	`, fichaID)
	if err != nil {
		return nil, fmt.Errorf("list evidence groups: %w", err)
	}
	defer rows.Close()
	byID := make(map[string]*evidence.Group)
	ordered := make([]*evidence.Group, 0)
	for rows.Next() {
		var group evidence.Group
		var item evidence.Record
		var metadata, capturedAt string
		if err := rows.Scan(&group.ID, &group.FichaID, &group.GroupKey, &group.Title, &group.Confidence, &group.Reason,
			&item.ID, &item.FichaID, &item.ItemCode, &item.SlotNumber, &item.Name, &item.FilePath, &item.Format,
			&item.Source, &item.SHA256, &metadata, &capturedAt); err != nil {
			return nil, fmt.Errorf("scan evidence group: %w", err)
		}
		item.Metadata = json.RawMessage(metadata)
		item.CapturedAt, _ = time.Parse(time.RFC3339Nano, capturedAt)
		existing := byID[group.ID]
		if existing == nil {
			existing = &evidence.Group{ID: group.ID, FichaID: group.FichaID, GroupKey: group.GroupKey, Title: group.Title, Confidence: group.Confidence, Reason: group.Reason}
			byID[group.ID] = existing
			ordered = append(ordered, existing)
		}
		existing.EvidenceIDs = append(existing.EvidenceIDs, item.ID)
		existing.ItemCodes = appendUnique(existing.ItemCodes, item.ItemCode)
		existing.Evidences = append(existing.Evidences, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read evidence groups: %w", err)
	}
	result := make([]evidence.Group, 0, len(ordered))
	for _, group := range ordered {
		result = append(result, *group)
	}
	return result, nil
}

func scanEvidence(scanner interface{ Scan(dest ...any) error }) (evidence.Record, error) {
	var item evidence.Record
	var metadata, capturedAt string
	if err := scanner.Scan(&item.ID, &item.FichaID, &item.ItemCode, &item.SlotNumber, &item.Name, &item.FilePath, &item.Format, &item.Source, &item.SHA256, &metadata, &capturedAt); err != nil {
		return evidence.Record{}, fmt.Errorf("scan evidence: %w", err)
	}
	item.Metadata = json.RawMessage(metadata)
	item.CapturedAt, _ = time.Parse(time.RFC3339Nano, capturedAt)
	return item, nil
}

func evidenceGroupSignature(record evidence.Record) (string, string, string) {
	var metadata struct {
		URL             string `json:"url"`
		FinalURL        string `json:"finalUrl"`
		Selector        string `json:"selector"`
		SelectorMatched bool   `json:"selectorMatched"`
		GroupName       string `json:"groupName"`
	}
	_ = json.Unmarshal(record.Metadata, &metadata)
	pageURL := metadata.FinalURL
	if strings.TrimSpace(pageURL) == "" {
		pageURL = metadata.URL
	}
	pageURL = normalizeEvidenceURL(pageURL)
	selector := strings.TrimSpace(metadata.Selector)
	if pageURL != "" {
		groupName := strings.TrimSpace(metadata.GroupName)
		if groupName == "" {
			groupName = checklistGroupName(record.ItemCode)
		}
		return "page:" + pageURL + "|selector:" + selector + "|group:" + groupName, "suggested", "misma URL, selector y grupo funcional"
	}
	if hash := strings.TrimSpace(record.SHA256); hash != "" {
		return "hash:" + hash, "exact", "mismo SHA-256"
	}
	return "record:" + record.ID, "unique", "sin firma compartida"
}

func checklistGroupName(itemCode string) string {
	for _, item := range checklist.Items() {
		if item.ItemCode == itemCode {
			return item.GroupName
		}
	}
	return ""
}

func normalizeEvidenceURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	parsed.RawQuery = parsed.Query().Encode()
	return parsed.String()
}

func evidenceGroupID(fichaID, groupKey string) string {
	sum := sha256.Sum256([]byte(fichaID + "\x00" + groupKey))
	return "evidence-group-" + hex.EncodeToString(sum[:8])
}

func appendUnique(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
