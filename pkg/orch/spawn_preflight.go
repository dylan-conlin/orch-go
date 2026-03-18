package orch

import (
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
)

// RunPreFlightChecks performs all pre-spawn validation checks.
// Gates removed (zero fires in 30d): verification, concurrency, ratelimit, drain.
// Hotspot gate is advisory-only (never blocks) — daemon extraction cascades handle structural health.
func RunPreFlightChecks(input *SpawnInput, preCheckDir string, bypassTriage bool, overrideReason string, hotspotCheckFunc func(string, string) (*gates.HotspotResult, error), agreementsCheckFunc func(string) (*gates.AgreementsResult, error), openQuestionCheckFunc gates.OpenQuestionChecker) (*gates.HotspotResult, *gates.AgreementsResult, *gates.OpenQuestionResult, error) {
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
		var err error
		hotspotResult, err = gates.CheckHotspot(preCheckDir, input.Task, input.SkillName, input.DaemonDriven, hotspotCheckFunc)
		if err != nil {
			// Advisory gate should never error, but handle gracefully
			logGateDecision("hotspot", "error", input.SkillName, input.IssueID, err.Error(), nil)
		}
		if hotspotResult != nil && hotspotResult.HasCriticalHotspot {
			logGateDecision("hotspot", "advisory", input.SkillName, input.IssueID, "critical hotspot — daemon will schedule extraction", hotspotResult.CriticalFiles)
		} else {
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
