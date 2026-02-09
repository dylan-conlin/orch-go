package episodic

import (
	"fmt"
	"strings"
	"time"
)

const (
	BoundarySpawn        = "spawn"
	BoundaryCommand      = "command"
	BoundaryVerification = "verification"
	BoundaryCompletion   = "completion"
)

const (
	OutcomeSuccess  = "success"
	OutcomeFailed   = "failed"
	OutcomeBypassed = "bypassed"
	OutcomeObserved = "observed"
	OutcomeReported = "reported"
)

const (
	EvidenceKindEventsJSONL  = "events_jsonl"
	EvidenceKindActivityJSON = "activity_json"
)

type ValidationState string

const (
	ValidationPending       ValidationState = "pending"
	ValidationStateAccepted ValidationState = "accepted"
	ValidationStateDegraded ValidationState = "degraded"
	ValidationStateRejected ValidationState = "rejected"
)

type ActionMemory struct {
	ID              string          `json:"id,omitempty"`
	Boundary        string          `json:"boundary"`
	Project         string          `json:"project"`
	Workspace       string          `json:"workspace"`
	SessionID       string          `json:"session_id,omitempty"`
	BeadsID         string          `json:"beads_id,omitempty"`
	Action          Action          `json:"action"`
	Outcome         Outcome         `json:"outcome"`
	Evidence        Evidence        `json:"evidence"`
	Confidence      float64         `json:"confidence"`
	ExpiresAt       time.Time       `json:"expires_at"`
	ValidationState ValidationState `json:"validation_state"`
	CreatedAt       time.Time       `json:"created_at"`
	Mutable         bool            `json:"mutable,omitempty"`
}

// Episode is kept as an alias for Phase B naming in higher-level APIs.
type Episode = ActionMemory

type Action struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name"`
	Input string `json:"input,omitempty"`
}

type Outcome struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type Evidence struct {
	Kind      string `json:"kind"`
	Pointer   string `json:"pointer"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Hash      string `json:"hash"`
	Mutable   bool   `json:"mutable,omitempty"`
}

func (m *ActionMemory) SetDefaults(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if strings.TrimSpace(m.Boundary) == "" {
		m.Boundary = BoundaryCommand
	}
	if m.ExpiresAt.IsZero() {
		m.ExpiresAt = ExpiryForBoundary(m.Boundary, now)
	}
	if m.ValidationState == "" {
		m.ValidationState = ValidationPending
	}
}

func (m ActionMemory) Validate() error {
	if strings.TrimSpace(m.Boundary) == "" {
		return fmt.Errorf("boundary is required")
	}
	if strings.TrimSpace(m.Project) == "" {
		return fmt.Errorf("project is required")
	}
	if strings.TrimSpace(m.Workspace) == "" {
		return fmt.Errorf("workspace is required")
	}
	if strings.TrimSpace(m.Action.Name) == "" {
		return fmt.Errorf("action.name is required")
	}
	if strings.TrimSpace(m.Outcome.Status) == "" {
		return fmt.Errorf("outcome.status is required")
	}
	if strings.TrimSpace(m.Outcome.Summary) == "" {
		return fmt.Errorf("outcome.summary is required")
	}
	if strings.TrimSpace(m.Evidence.Kind) == "" {
		return fmt.Errorf("evidence.kind is required")
	}
	if strings.TrimSpace(m.Evidence.Pointer) == "" {
		return fmt.Errorf("evidence.pointer is required")
	}
	if strings.TrimSpace(m.Evidence.Hash) == "" {
		return fmt.Errorf("evidence.hash is required")
	}
	if m.Confidence < 0 || m.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	if m.ExpiresAt.IsZero() {
		return fmt.Errorf("expires_at is required")
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	return nil
}

func ExpiryForBoundary(boundary string, now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	switch strings.TrimSpace(boundary) {
	case BoundarySpawn, BoundaryCommand:
		return now.Add(24 * time.Hour)
	case BoundaryVerification, BoundaryCompletion:
		return now.Add(14 * 24 * time.Hour)
	default:
		return now.Add(24 * time.Hour)
	}
}
