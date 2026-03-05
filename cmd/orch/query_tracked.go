package main

import (
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/discovery"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// AgentStatus is the canonical backend-aware agent status type.
// Defined in pkg/discovery to be importable by any package.
// This alias preserves backward compatibility for all cmd/orch/ callers.
type AgentStatus = discovery.AgentStatus

// checkTmuxWindowAlive delegates to the discovery package's exported variable.
// This alias preserves backward compatibility for tests that mock it.
var checkTmuxWindowAlive = discovery.CheckTmuxWindowAlive

// queryTrackedAgents delegates to the canonical backend-aware query engine
// in pkg/discovery. This is the single entry point for agent discovery
// that dispatches to both OpenCode and Claude CLI backends.
//
// Callers in pkg/ should import pkg/discovery directly instead of going
// through this CLI wrapper.
func queryTrackedAgents(projectDirs []string) ([]AgentStatus, error) {
	return discovery.QueryTrackedAgents(projectDirs)
}

// listTrackedIssues delegates to pkg/discovery.
func listTrackedIssues(projectDirs []string) ([]beads.Issue, error) {
	return discovery.ListTrackedIssues(projectDirs)
}

// listTrackedIssuesCLI queries beads via bd CLI for orch:agent tagged issues.
// Retained in cmd/orch for the test that mocks fallbackListWithLabelFn.
var fallbackListWithLabelFn = beads.FallbackListWithLabel

func listTrackedIssuesCLI() ([]beads.Issue, error) {
	issues, err := fallbackListWithLabelFn("orch:agent")
	if err != nil {
		return nil, err
	}
	return discovery.FilterActiveIssues(issues), nil
}

// filterActiveIssues delegates to pkg/discovery.
func filterActiveIssues(issues []beads.Issue) []beads.Issue {
	return discovery.FilterActiveIssues(issues)
}

// lookupManifestsAcrossProjects delegates to pkg/discovery.
func lookupManifestsAcrossProjects(projectDirs []string, beadsIDs []string) (map[string]*spawn.AgentManifest, error) {
	return discovery.LookupManifestsAcrossProjects(projectDirs, beadsIDs)
}

// extractSessionIDs delegates to pkg/discovery.
func extractSessionIDs(manifests map[string]*spawn.AgentManifest) []string {
	return discovery.ExtractSessionIDs(manifests)
}

// extractLatestPhases delegates to pkg/discovery.
func extractLatestPhases(beadsIDs []string) (map[string]string, map[string]time.Time) {
	return discovery.ExtractLatestPhases(beadsIDs)
}

// latestPhaseFromComments delegates to pkg/discovery.
func latestPhaseFromComments(comments []beads.Comment) string {
	return discovery.LatestPhaseFromComments(comments)
}

// latestPhaseWithTimestamp delegates to pkg/discovery.
func latestPhaseWithTimestamp(comments []beads.Comment) (string, time.Time) {
	return discovery.LatestPhaseWithTimestamp(comments)
}

// unknownLiveness delegates to pkg/discovery.
func unknownLiveness(sessionIDs []string) map[string]opencode.SessionStatusInfo {
	return discovery.UnknownLiveness(sessionIDs)
}

// joinWithReasonCodes delegates to pkg/discovery.
func joinWithReasonCodes(
	issues []beads.Issue,
	manifests map[string]*spawn.AgentManifest,
	liveness map[string]opencode.SessionStatusInfo,
	phases map[string]string,
	phaseTimestamps ...map[string]time.Time,
) []AgentStatus {
	return discovery.JoinWithReasonCodes(issues, manifests, liveness, phases, phaseTimestamps...)
}
