package daemon

import (
	"testing"
)

func TestCleanupResultFields(t *testing.T) {
	result := CleanupResult{
		SessionsDeleted:        1,
		WorkspacesArchived:     2,
		InvestigationsArchived: 3,
		Message:                "test",
	}

	if result.SessionsDeleted != 1 {
		t.Errorf("Expected SessionsDeleted=1, got %d", result.SessionsDeleted)
	}
	if result.WorkspacesArchived != 2 {
		t.Errorf("Expected WorkspacesArchived=2, got %d", result.WorkspacesArchived)
	}
	if result.InvestigationsArchived != 3 {
		t.Errorf("Expected InvestigationsArchived=3, got %d", result.InvestigationsArchived)
	}
}
