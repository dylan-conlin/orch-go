package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/focus"
)

func TestBuildDriftAnalysis_GroupsBySkill(t *testing.T) {
	agents := []AgentStatus{
		{BeadsID: "proj-1", Title: "Add auth", Skill: "feature-impl", Status: "active", Phase: "Implementing"},
		{BeadsID: "proj-2", Title: "Fix login bug", Skill: "systematic-debugging", Status: "active", Phase: "Planning"},
		{BeadsID: "proj-3", Title: "Add dashboard", Skill: "feature-impl", Status: "idle", Phase: "Complete"},
		{BeadsID: "proj-4", Title: "Review arch", Skill: "architect", Status: "active"},
	}

	driftResult := focus.DriftResult{
		Goal:    "Ship auth system",
		Verdict: "on-track",
	}

	analysis := buildDriftAnalysis(driftResult, agents, 3)

	if analysis.AgentCount != 4 {
		t.Errorf("expected 4 agents, got %d", analysis.AgentCount)
	}
	if analysis.UntrackedCount != 3 {
		t.Errorf("expected 3 untracked, got %d", analysis.UntrackedCount)
	}
	if analysis.Verdict != "on-track" {
		t.Errorf("expected verdict on-track, got %q", analysis.Verdict)
	}

	// Should have 3 skill groups
	if len(analysis.Groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(analysis.Groups))
	}

	// First group should be feature-impl (2 agents, largest)
	if analysis.Groups[0].Skill != "feature-impl" {
		t.Errorf("expected first group to be feature-impl, got %q", analysis.Groups[0].Skill)
	}
	if len(analysis.Groups[0].Agents) != 2 {
		t.Errorf("expected 2 agents in feature-impl, got %d", len(analysis.Groups[0].Agents))
	}
}

func TestBuildDriftAnalysis_UnknownSkill(t *testing.T) {
	agents := []AgentStatus{
		{BeadsID: "proj-1", Title: "Mystery work", Skill: "", Status: "active"},
	}

	driftResult := focus.DriftResult{
		Goal:    "Some goal",
		Verdict: "unverified",
		Reason:  "Focus has no specific issue",
	}

	analysis := buildDriftAnalysis(driftResult, agents, 0)

	if len(analysis.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(analysis.Groups))
	}
	if analysis.Groups[0].Skill != "(unknown)" {
		t.Errorf("expected skill (unknown), got %q", analysis.Groups[0].Skill)
	}
}

func TestBuildDriftAnalysis_NoAgents(t *testing.T) {
	driftResult := focus.DriftResult{
		Goal:       "Ship MVP",
		Verdict:    "drifting",
		IsDrifting: true,
		Reason:     "No active work on focused issue",
	}

	analysis := buildDriftAnalysis(driftResult, nil, 5)

	if analysis.AgentCount != 0 {
		t.Errorf("expected 0 agents, got %d", analysis.AgentCount)
	}
	if analysis.UntrackedCount != 5 {
		t.Errorf("expected 5 untracked, got %d", analysis.UntrackedCount)
	}
	if len(analysis.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(analysis.Groups))
	}
	if !analysis.IsDrifting {
		t.Error("expected IsDrifting to be true")
	}
	if analysis.Verdict != "drifting" {
		t.Errorf("expected verdict drifting, got %q", analysis.Verdict)
	}
}

func TestBuildDriftAnalysis_NoFocus(t *testing.T) {
	agents := []AgentStatus{
		{BeadsID: "proj-1", Title: "Some work", Skill: "feature-impl", Status: "active"},
	}

	driftResult := focus.DriftResult{
		Verdict: "no-focus",
		Reason:  "No focus set",
	}

	analysis := buildDriftAnalysis(driftResult, agents, 0)

	if analysis.Verdict != "no-focus" {
		t.Errorf("expected verdict no-focus, got %q", analysis.Verdict)
	}
	if analysis.AgentCount != 1 {
		t.Errorf("expected 1 agent, got %d", analysis.AgentCount)
	}
}

func TestBuildDriftAnalysis_GroupsSortedByCount(t *testing.T) {
	agents := []AgentStatus{
		{BeadsID: "a1", Title: "A", Skill: "architect", Status: "active"},
		{BeadsID: "d1", Title: "D1", Skill: "systematic-debugging", Status: "active"},
		{BeadsID: "d2", Title: "D2", Skill: "systematic-debugging", Status: "active"},
		{BeadsID: "d3", Title: "D3", Skill: "systematic-debugging", Status: "active"},
		{BeadsID: "f1", Title: "F1", Skill: "feature-impl", Status: "active"},
		{BeadsID: "f2", Title: "F2", Skill: "feature-impl", Status: "active"},
	}

	driftResult := focus.DriftResult{Verdict: "on-track"}
	analysis := buildDriftAnalysis(driftResult, agents, 0)

	if len(analysis.Groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(analysis.Groups))
	}

	// Sorted by count desc: systematic-debugging (3), feature-impl (2), architect (1)
	if analysis.Groups[0].Skill != "systematic-debugging" {
		t.Errorf("expected first group systematic-debugging, got %q", analysis.Groups[0].Skill)
	}
	if analysis.Groups[1].Skill != "feature-impl" {
		t.Errorf("expected second group feature-impl, got %q", analysis.Groups[1].Skill)
	}
	if analysis.Groups[2].Skill != "architect" {
		t.Errorf("expected third group architect, got %q", analysis.Groups[2].Skill)
	}
}
