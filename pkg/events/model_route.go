package events

import "time"

// ModelRouteData contains the data for a spawn.model_route event.
// Logs the daemon's model routing decision with its resolution source.
type ModelRouteData struct {
	IssueID        string `json:"issue_id"`
	Skill          string `json:"skill"`
	EffectiveModel string `json:"effective_model"`
	BaseModel      string `json:"base_model,omitempty"`
	Source         string `json:"source"`
	ConfigKey      string `json:"config_key,omitempty"`
	Reason         string `json:"reason"`
}

// LogModelRoute logs a daemon model routing decision event.
func (l *Logger) LogModelRoute(data ModelRouteData) error {
	eventData := map[string]interface{}{
		"issue_id":        data.IssueID,
		"skill":           data.Skill,
		"effective_model": data.EffectiveModel,
		"source":          data.Source,
		"reason":          data.Reason,
	}
	if data.BaseModel != "" {
		eventData["base_model"] = data.BaseModel
	}
	if data.ConfigKey != "" {
		eventData["config_key"] = data.ConfigKey
	}

	return l.Log(Event{
		Type:      EventTypeModelRoute,
		SessionID: data.IssueID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}
