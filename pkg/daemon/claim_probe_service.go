package daemon

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

const claimProbeLabel = "daemon:claim-probe"

// defaultClaimProbeService is the production implementation of ClaimProbeService.
type defaultClaimProbeService struct{}

// NewDefaultClaimProbeService creates the production claim probe service.
func NewDefaultClaimProbeService() ClaimProbeService {
	return &defaultClaimProbeService{}
}

func (s *defaultClaimProbeService) HasOpenProbeForClaim(claimID, modelName string) (bool, error) {
	issues, err := ListIssuesWithLabel(claimProbeLabel)
	if err != nil {
		return false, err
	}
	// Match by claim ID in issue title (format: "Probe: ... [CLAIM-ID]")
	needle := fmt.Sprintf("[%s]", claimID)
	for _, issue := range issues {
		if strings.Contains(issue.Title, needle) {
			return true, nil
		}
	}
	return false, nil
}

func (s *defaultClaimProbeService) CreateProbeIssue(claimID, claimText, falsifiesIf, modelName string) (string, error) {
	title := fmt.Sprintf("Probe: %s — %s [%s]", truncateStr(claimText, 60), truncateStr(falsifiesIf, 40), claimID)
	description := fmt.Sprintf("Claim: %s\nModel: %s\nFalsifies if: %s\n\nInvestigate whether this claim holds. Look for evidence that confirms or contradicts it.", claimText, modelName, falsifiesIf)

	labels := []string{claimProbeLabel, fmt.Sprintf("claim:%s", claimID), "triage:ready"}

	// Try RPC first, fallback to CLI
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{
				Title:       title,
				Description: description,
				IssueType:   "task",
				Priority:    3,
				Labels:      labels,
			})
			if err == nil {
				return issue.ID, nil
			}
		}
	}

	// Fallback to CLI
	issue, err := beads.FallbackCreate(title, description, "task", 3, labels, "")
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
