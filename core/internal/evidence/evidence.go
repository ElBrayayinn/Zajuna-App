package evidence

import (
	"context"
	"encoding/json"
	"time"
)

type Record struct {
	ID         string
	FichaID    string
	ItemCode   string
	SlotNumber int
	Name       string
	FilePath   string
	Format     string
	Source     string
	SHA256     string
	Metadata   json.RawMessage
	CapturedAt time.Time
}

// Group is a persisted equivalence group for evidence records. A group keeps
// one logical evidence visible in reports while retaining every checklist
// item/slot that it covers.
type Group struct {
	ID          string
	FichaID     string
	GroupKey    string
	Title       string
	Confidence  string
	Reason      string
	EvidenceIDs []string
	ItemCodes   []string
	Evidences   []Record
}

type Store interface {
	CreateEvidence(ctx context.Context, record Record) error
	GetEvidence(ctx context.Context, id string) (Record, error)
	ListEvidences(ctx context.Context, limit int) ([]Record, error)
}

type DeleteStore interface {
	Store
	DeleteEvidence(ctx context.Context, id string) (Record, error)
}

// GroupStore is optional so existing capture workers and test doubles can
// continue to use the basic evidence contract.
type GroupStore interface {
	Store
	ListEvidencesByFicha(ctx context.Context, fichaID string, limit int) ([]Record, error)
	RebuildEvidenceGroups(ctx context.Context, fichaID string) ([]Group, error)
	ListEvidenceGroups(ctx context.Context, fichaID string) ([]Group, error)
}
