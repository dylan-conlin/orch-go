package events

import "time"

// ThinIssueDetectedData contains the data for a daemon.thin_issue_detected event.
// Emitted when the daemon detects issues with empty descriptions in the Orient phase.
type ThinIssueDetectedData struct {
	IssueID string `json:"issue_id"`
	Title   string `json:"title"`
}

// LogThinIssueDetected logs an advisory event for an issue with no description.
func (l *Logger) LogThinIssueDetected(data ThinIssueDetectedData) error {
	eventData := map[string]interface{}{
		"issue_id": data.IssueID,
		"title":    data.Title,
	}

	return l.Log(Event{
		Type:      EventTypeThinIssueDetected,
		SessionID: data.IssueID,
		Timestamp: time.Now().Unix(),
		Data:      eventData,
	})
}
