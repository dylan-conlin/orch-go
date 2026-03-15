package daemon

import (
	"crypto/rand"
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

// DecisionLogEntry contains the inputs needed to log a daemon decision event.
// The tier is computed automatically from class + compliance via ClassifyDecision.
type DecisionLogEntry struct {
	// Class identifies the type of decision being made.
	Class daemonconfig.DecisionClass
	// Compliance is the effective compliance level for this decision.
	Compliance daemonconfig.ComplianceLevel
	// Target is what the decision acts on (e.g., issue ID, agent ID). Optional.
	Target string
	// Reason is a human-readable explanation for the decision. Optional.
	Reason string
}

// LogDecision classifies a daemon decision and logs it as a decision.made event.
// Safe to call with a nil logger (no-op).
func LogDecision(logger *events.Logger, entry DecisionLogEntry) error {
	if logger == nil {
		return nil
	}

	tier := daemonconfig.ClassifyDecision(entry.Class, entry.Compliance)
	baseTier := daemonconfig.ClassifyDecision(entry.Class, daemonconfig.ComplianceStandard)

	return logger.LogDecisionMade(events.DecisionMadeData{
		DecisionID:      generateDecisionID(),
		Class:           entry.Class.String(),
		Category:        entry.Class.Category(),
		Tier:            tier.String(),
		BaseTier:        baseTier.String(),
		ComplianceLevel: entry.Compliance.String(),
		Target:          entry.Target,
		Reason:          entry.Reason,
		Outcome:         "executed",
	})
}

// generateDecisionID returns a short random hex ID for decision correlation.
func generateDecisionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
