// Package daemon provides autonomous overnight processing capabilities.
// This file contains the default TriggerScanService implementation.
package daemon

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// defaultTriggerScanService is the production implementation of TriggerScanService.
type defaultTriggerScanService struct{}

// NewDefaultTriggerScanService creates the production trigger scan service.
func NewDefaultTriggerScanService() TriggerScanService {
	return &defaultTriggerScanService{}
}

func (s *defaultTriggerScanService) CountOpenTriggerIssues() (int, error) {
	issues, err := ListIssuesWithLabel(TriggerLabel)
	if err != nil {
		return 0, err
	}
	return len(issues), nil
}

func (s *defaultTriggerScanService) HasOpenTriggerIssue(detectorName, key string) (bool, error) {
	// Use detector-specific label for more precise dedup
	label := fmt.Sprintf("daemon:trigger:%s", detectorName)
	issues, err := ListIssuesWithLabel(label)
	if err != nil {
		// Fallback: check the general trigger label with key in title
		issues, err = ListIssuesWithLabel(TriggerLabel)
		if err != nil {
			return false, err
		}
	}
	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue.Title), strings.ToLower(key)) {
			return true, nil
		}
	}
	return false, nil
}

func (s *defaultTriggerScanService) CreateTriggerIssue(suggestion TriggerSuggestion) (string, error) {
	detectorLabel := fmt.Sprintf("daemon:trigger:%s", suggestion.Detector)
	labels := []string{TriggerLabel, detectorLabel, "triage:ready"}
	labels = append(labels, suggestion.Labels...)

	// Try RPC first, fallback to CLI
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{
				Title:       suggestion.Title,
				Description: suggestion.Description,
				IssueType:   suggestion.IssueType,
				Priority:    suggestion.Priority,
				Labels:      labels,
			})
			if err == nil {
				return issue.ID, nil
			}
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackCreate(
		suggestion.Title,
		suggestion.Description,
		suggestion.IssueType,
		suggestion.Priority,
		labels,
		"",
	)
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}
