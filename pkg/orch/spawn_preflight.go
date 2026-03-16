package orch

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// RunPreFlightChecks performs all pre-spawn validation checks.
// Gates removed (zero fires in 30d): verification, concurrency, ratelimit, drain.
func RunPreFlightChecks(input *SpawnInput, preCheckDir string, bypassTriage, forceHotspot bool, architectRef, overrideReason string, hotspotCheckFunc func(string, string) (*gates.HotspotResult, error), agreementsCheckFunc func(string) (*gates.AgreementsResult, error), openQuestionCheckFunc gates.OpenQuestionChecker) (*gates.HotspotResult, *gates.AgreementsResult, *gates.OpenQuestionResult, error) {
	if err := gates.CheckTriageBypass(input.DaemonDriven, bypassTriage, input.SkillName, input.Task); err != nil {
		logGateDecision("triage", "block", input.SkillName, input.IssueID, "manual spawn without --bypass-triage", nil)
		return nil, nil, nil, err
	}
	if !input.DaemonDriven && bypassTriage {
		gates.LogTriageBypass(input.SkillName, input.Task, overrideReason)
		logGateDecision("triage", "bypass", input.SkillName, input.IssueID, overrideReason, nil)
	} else if input.DaemonDriven {
		logGateDecision("triage", "allow", input.SkillName, input.IssueID, "daemon-driven spawn", nil)
	}

	CheckGovernance(input.Task, input.SkillName, input.DaemonDriven)

	var hotspotResult *gates.HotspotResult
	if hotspotCheckFunc != nil {
		architectVerifier := buildArchitectVerifier()
		architectFinder := buildArchitectFinder()
		var err error
		hotspotResult, err = gates.CheckHotspot(preCheckDir, input.Task, input.SkillName, input.DaemonDriven, forceHotspot, architectRef, overrideReason, hotspotCheckFunc, architectVerifier, architectFinder)
		if err != nil {
			var targetFiles []string
			if hotspotResult != nil {
				targetFiles = hotspotResult.CriticalFiles
			}
			logGateDecision("hotspot", "block", input.SkillName, input.IssueID, err.Error(), targetFiles)
			return nil, nil, nil, err
		}
		if forceHotspot && hotspotResult != nil && hotspotResult.HasCriticalHotspot {
			logGateDecision("hotspot", "bypass", input.SkillName, input.IssueID, overrideReason, hotspotResult.CriticalFiles)
		} else if hotspotResult != nil && hotspotResult.HasCriticalHotspot {
			logGateDecision("hotspot", "bypass", input.SkillName, input.IssueID, "auto-detected prior architect review", hotspotResult.CriticalFiles)
		} else if hotspotResult == nil || !hotspotResult.HasCriticalHotspot {
			logGateDecision("hotspot", "allow", input.SkillName, input.IssueID, "no critical hotspot files", nil)
		}
	}

	var agreementsResult *gates.AgreementsResult
	if agreementsCheckFunc != nil {
		agreementsResult, _ = gates.CheckAgreements(preCheckDir, input.DaemonDriven, agreementsCheckFunc)
	}

	var openQuestionResult *gates.OpenQuestionResult
	if openQuestionCheckFunc != nil && input.IssueID != "" {
		openQuestionResult, _ = gates.CheckOpenQuestions(input.IssueID, input.DaemonDriven, openQuestionCheckFunc)
	}

	return hotspotResult, agreementsResult, openQuestionResult, nil
}

func buildArchitectVerifier() gates.ArchitectVerifier {
	return func(issueID string) error {
		issue, err := verify.GetIssue(issueID, "")
		if err != nil {
			return fmt.Errorf("--architect-ref %s: issue not found", issueID)
		}
		if !isArchitectIssue(issue) {
			return fmt.Errorf("--architect-ref %s: not an architect issue (type=%s)", issueID, issue.IssueType)
		}
		if issue.Status != "closed" {
			return fmt.Errorf("--architect-ref %s: architect review not complete (status=%s)", issueID, issue.Status)
		}
		return nil
	}
}

func buildArchitectFinder() gates.ArchitectFinder {
	return func(criticalFiles []string) (string, error) {
		return FindPriorArchitectReview(criticalFiles)
	}
}

func FindPriorArchitectReview(criticalFiles []string) (string, error) {
	if len(criticalFiles) == 0 {
		return "", nil
	}
	searchTerms := extractSearchTerms(criticalFiles)
	if len(searchTerms) == 0 {
		return "", nil
	}
	socketPath, err := beads.FindSocketPath("")
	if err != nil {
		return "", nil
	}
	client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
	defer client.Close()
	issues, err := client.List(&beads.ListArgs{Status: "closed", Labels: []string{"skill:architect"}})
	if err != nil {
		return "", nil
	}
	titleIssues, err := client.List(&beads.ListArgs{Status: "closed", Title: "architect:"})
	if err == nil {
		seen := make(map[string]bool)
		for _, i := range issues {
			seen[i.ID] = true
		}
		for _, i := range titleIssues {
			if !seen[i.ID] {
				issues = append(issues, i)
			}
		}
	}
	for _, issue := range issues {
		titleLower := strings.ToLower(issue.Title)
		for _, term := range searchTerms {
			if strings.Contains(titleLower, term) {
				return issue.ID, nil
			}
		}
	}
	return "", nil
}

func extractSearchTerms(criticalFiles []string) []string {
	seen := make(map[string]bool)
	var terms []string
	for _, file := range criticalFiles {
		normalized := strings.ToLower(strings.TrimSpace(file))
		if normalized == "" {
			continue
		}
		if !seen[normalized] {
			terms = append(terms, normalized)
			seen[normalized] = true
		}
		parts := strings.Split(normalized, "/")
		basename := parts[len(parts)-1]
		nameOnly := strings.TrimSuffix(basename, ".go")
		if nameOnly != "" && !seen[nameOnly] {
			terms = append(terms, nameOnly)
			seen[nameOnly] = true
		}
	}
	return terms
}

func isArchitectIssue(issue *verify.Issue) bool {
	for _, label := range issue.Labels {
		if label == "skill:architect" {
			return true
		}
	}
	if strings.Contains(strings.ToLower(issue.Title), "architect:") {
		return true
	}
	return false
}

func logGateDecision(gateName, decision, skill, beadsID, reason string, targetFiles []string) {
	logger := events.NewLogger(events.DefaultLogPath())
	_ = logger.LogGateDecision(events.GateDecisionData{
		GateName:    gateName,
		Decision:    decision,
		Skill:       skill,
		BeadsID:     beadsID,
		TargetFiles: targetFiles,
		Reason:      reason,
	})
}
