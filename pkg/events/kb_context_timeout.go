package events

import "time"

// KBContextTimeoutData contains the data for a kb.context.timeout event.
type KBContextTimeoutData struct {
	Query       string
	ProjectDir  string
	Skill       string
	BeadsID     string
	WorkspaceID string
}

// LogKBContextTimeout logs when pre-spawn kb context lookup times out.
func (l *Logger) LogKBContextTimeout(data KBContextTimeoutData) error {
	eventData := map[string]interface{}{
		"query":       data.Query,
		"project_dir": data.ProjectDir,
		"skill":       data.Skill,
	}
	if data.BeadsID != "" {
		eventData["beads_id"] = data.BeadsID
	}
	if data.WorkspaceID != "" {
		eventData["workspace_id"] = data.WorkspaceID
	}

	sessionID := data.WorkspaceID
	if sessionID == "" {
		sessionID = data.BeadsID
	}

	return l.Log(Event{
		Type:      EventTypeKBContextTimeout,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}
