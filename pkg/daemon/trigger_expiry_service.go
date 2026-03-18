// Package daemon provides autonomous overnight processing capabilities.
// This file contains the default TriggerExpiryService implementation.
package daemon

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// defaultTriggerExpiryService is the production implementation of TriggerExpiryService.
type defaultTriggerExpiryService struct{}

// NewDefaultTriggerExpiryService creates the production trigger expiry service.
func NewDefaultTriggerExpiryService() TriggerExpiryService {
	return &defaultTriggerExpiryService{}
}

func (s *defaultTriggerExpiryService) ListExpiredTriggerIssues(maxAge time.Duration) ([]ExpiredTriggerIssue, error) {
	issues, err := ListIssuesWithLabel(TriggerLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to list daemon:trigger issues: %w", err)
	}

	now := time.Now()
	var expired []ExpiredTriggerIssue
	for _, issue := range issues {
		createdAt, err := parseIssueCreatedAt(issue.ID)
		if err != nil {
			continue // skip issues we can't parse
		}
		age := now.Sub(createdAt)
		if age > maxAge {
			expired = append(expired, ExpiredTriggerIssue{
				ID:     issue.ID,
				Title:  issue.Title,
				Age:    age,
				Labels: issue.Labels,
			})
		}
	}
	return expired, nil
}

func (s *defaultTriggerExpiryService) ExpireTriggerIssue(id, reason string) error {
	// Add daemon:expired label first
	if err := addLabelToIssue(id, "daemon:expired"); err != nil {
		// Non-fatal: proceed with close even if labeling fails
		fmt.Printf("Warning: failed to add daemon:expired label to %s: %v\n", id, err)
	}

	// Close the issue
	return closeIssue(id, reason)
}

// parseIssueCreatedAt retrieves the created_at date for a beads issue.
// Uses RPC client if available, falls back to CLI.
func parseIssueCreatedAt(issueID string) (time.Time, error) {
	// Try RPC first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Show(issueID)
			if err == nil && issue.CreatedAt != "" {
				return time.Parse(time.RFC3339, issue.CreatedAt)
			}
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackShow(issueID, "")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get issue %s: %w", issueID, err)
	}
	if issue.CreatedAt == "" {
		return time.Time{}, fmt.Errorf("issue %s has no created_at", issueID)
	}
	return time.Parse(time.RFC3339, issue.CreatedAt)
}

// addLabelToIssue adds a label to a beads issue.
// Uses RPC client if available, falls back to CLI.
func addLabelToIssue(issueID, label string) error {
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			if err := client.AddLabel(issueID, label); err == nil {
				return nil
			}
		}
	}
	return beads.FallbackAddLabel(issueID, label, "")
}

// closeIssue closes a beads issue with a reason.
// Uses RPC client if available, falls back to CLI.
func closeIssue(issueID, reason string) error {
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			if err := client.CloseIssue(issueID, reason); err == nil {
				return nil
			}
		}
	}
	return beads.FallbackClose(issueID, reason, "")
}
