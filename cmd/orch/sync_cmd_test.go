package main

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/artifactsync"
)

func TestPrintDriftReport_Formatting(t *testing.T) {
	report := &artifactsync.DriftReport{
		Entries: []artifactsync.DriftReportEntry{
			{
				ArtifactPath: "CLAUDE.md",
				SectionName:  "Commands",
				Triggers:     []string{"new-command", "new-flag"},
				Events: []artifactsync.DriftEvent{
					{BeadsID: "proj-1", ChangeScopes: []string{"new-command"}},
					{BeadsID: "proj-2", ChangeScopes: []string{"new-flag"}},
				},
			},
			{
				ArtifactPath: ".kb/guides/spawn.md",
				SectionName:  "",
				Triggers:     []string{"new-flag"},
				Events: []artifactsync.DriftEvent{
					{BeadsID: "proj-2", ChangeScopes: []string{"new-flag"}},
				},
			},
		},
	}

	allEvents := []artifactsync.DriftEvent{
		{BeadsID: "proj-1", ChangeScopes: []string{"new-command"}},
		{BeadsID: "proj-2", ChangeScopes: []string{"new-flag"}},
	}

	// Just verify it doesn't panic
	printDriftReport(report, allEvents)
}

func TestSpawnSyncAgent_TaskDescription(t *testing.T) {
	report := &artifactsync.DriftReport{
		Entries: []artifactsync.DriftReportEntry{
			{
				ArtifactPath: "CLAUDE.md",
				SectionName:  "Commands",
				Triggers:     []string{"new-command"},
				Events: []artifactsync.DriftEvent{
					{BeadsID: "proj-1", CommitRange: "abc..def"},
				},
			},
			{
				ArtifactPath: ".kb/guides/spawn.md",
				Triggers:     []string{"new-flag"},
				Events: []artifactsync.DriftEvent{
					{BeadsID: "proj-2", CommitRange: "def..ghi"},
				},
			},
		},
	}

	task := buildSyncTask(report)

	if !strings.Contains(task, "CLAUDE.md:Commands") {
		t.Error("expected task to contain CLAUDE.md:Commands")
	}
	if !strings.Contains(task, ".kb/guides/spawn.md") {
		t.Error("expected task to contain .kb/guides/spawn.md")
	}
	if !strings.Contains(task, "abc..def") {
		t.Error("expected task to contain commit range abc..def")
	}
	if !strings.Contains(task, "new-command") {
		t.Error("expected task to contain trigger new-command")
	}
}
