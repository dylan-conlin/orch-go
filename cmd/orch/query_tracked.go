package main

import (
	"github.com/dylan-conlin/orch-go/pkg/discovery"
)

// queryTrackedAgents delegates to the canonical discovery.QueryTrackedAgents.
// This is the single entry point for agent status derivation in the codebase.
//
// All status derivation logic lives in pkg/discovery to prevent Class 5
// (Contradictory Authority Signals) defects from duplicate implementations.
//
// See: .kb/decisions/2026-02-18-two-lane-agent-discovery.md
func queryTrackedAgents(projectDirs []string) ([]discovery.AgentStatus, error) {
	return discovery.QueryTrackedAgents(projectDirs)
}
