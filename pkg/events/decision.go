package events

import "time"

// DecisionMadeData contains the data for a decision.made event.
// Logs every daemon decision with its classification tier for observability.
type DecisionMadeData struct {
	DecisionID      string `json:"decision_id"`                // Unique ID (UUID) for this decision
	Class           string `json:"class"`                      // Decision class (e.g., "select_issue", "auto_complete_light")
	Category        string `json:"category"`                   // Category (e.g., "spawn", "completion", "knowledge")
	Tier            string `json:"tier"`                       // Effective tier after compliance modulation
	BaseTier        string `json:"base_tier"`                  // Base tier before compliance modulation
	ComplianceLevel string `json:"compliance_level"`           // Compliance level that modulated the tier
	Target          string `json:"target,omitempty"`           // What the decision acts on (e.g., issue ID, agent ID)
	Reason          string `json:"reason,omitempty"`           // Human-readable reason for the decision
	Outcome         string `json:"outcome,omitempty"`          // "executed", "vetoed", "pending", "expired"
}

// LogDecisionMade logs a daemon decision event with classification metadata.
func (l *Logger) LogDecisionMade(data DecisionMadeData) error {
	eventData := map[string]interface{}{
		"decision_id":      data.DecisionID,
		"class":            data.Class,
		"category":         data.Category,
		"tier":             data.Tier,
		"base_tier":        data.BaseTier,
		"compliance_level": data.ComplianceLevel,
	}
	if data.Target != "" {
		eventData["target"] = data.Target
	}
	if data.Reason != "" {
		eventData["reason"] = data.Reason
	}
	if data.Outcome != "" {
		eventData["outcome"] = data.Outcome
	}

	return l.Log(Event{
		Type:      EventTypeDecisionMade,
		SessionID: data.Target, // Use target as session ID for grouping
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}
