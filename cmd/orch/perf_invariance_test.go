// Performance invariance tests for agent discovery pipeline (orch-go-1097).
//
// Structural gate: orch-go-1096 fixed agent discovery to be O(active) instead of
// O(historical) by removing the orch:agent label on close. These tests ensure
// that invariant holds: query time must not grow with historical agent count.
//
// Regression scenarios caught:
//   - FilterActiveIssues removed or bypassed → closed agents leak into pipeline
//   - orch:agent label not removed on close → bd list returns all historical agents
//   - JoinWithReasonCodes becomes super-linear (e.g., nested loops)
package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/discovery"
	"github.com/dylan-conlin/orch-go/pkg/execution"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// generateMixedIssues creates a mix of active and closed issues for testing.
// Active issues have status "in_progress", closed issues have status "closed".
func generateMixedIssues(activeCount, closedCount int) []beads.Issue {
	issues := make([]beads.Issue, 0, activeCount+closedCount)
	for i := 0; i < activeCount; i++ {
		issues = append(issues, beads.Issue{
			ID:     fmt.Sprintf("orch-go-active-%d", i),
			Title:  fmt.Sprintf("Active agent %d", i),
			Status: "in_progress",
			Labels: []string{"orch:agent"},
		})
	}
	for i := 0; i < closedCount; i++ {
		issues = append(issues, beads.Issue{
			ID:     fmt.Sprintf("orch-go-closed-%d", i),
			Title:  fmt.Sprintf("Closed agent %d", i),
			Status: "closed",
			Labels: []string{"orch:agent"},
		})
	}
	return issues
}

// buildJoinData creates test data for JoinWithReasonCodes at a given scale.
func buildJoinData(n int) ([]beads.Issue, map[string]*spawn.AgentManifest, map[string]execution.SessionStatusInfo, map[string]string) {
	issues := make([]beads.Issue, n)
	manifests := make(map[string]*spawn.AgentManifest, n)
	liveness := make(map[string]execution.SessionStatusInfo, n)
	phases := make(map[string]string, n)

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("orch-go-%d", i)
		sessID := fmt.Sprintf("sess-%d", i)
		issues[i] = beads.Issue{ID: id, Title: fmt.Sprintf("Task %d", i), Status: "in_progress"}
		manifests[id] = &spawn.AgentManifest{
			BeadsID:    id,
			SessionID:  sessID,
			ProjectDir: "/tmp/project",
			Skill:      "feature-impl",
		}
		liveness[sessID] = execution.SessionStatusInfo{Type: "busy"}
		phases[id] = fmt.Sprintf("Implementing - step %d", i)
	}
	return issues, manifests, liveness, phases
}

// TestPerfInvariance_FilterBlocksClosedAgents verifies that FilterActiveIssues
// returns only active issues regardless of how many closed issues exist.
// This is the primary structural gate: if this filter is removed or bypassed,
// downstream processing becomes O(historical).
func TestPerfInvariance_FilterBlocksClosedAgents(t *testing.T) {
	const activeCount = 5
	closedCounts := []int{0, 50, 100, 200}

	for _, closed := range closedCounts {
		t.Run(fmt.Sprintf("closed=%d", closed), func(t *testing.T) {
			issues := generateMixedIssues(activeCount, closed)
			active := discovery.FilterActiveIssues(issues)

			if len(active) != activeCount {
				t.Errorf("with %d closed issues: expected %d active, got %d",
					closed, activeCount, len(active))
			}

			// Verify all returned issues are actually active
			for _, issue := range active {
				if issue.Status != "in_progress" && issue.Status != "open" {
					t.Errorf("non-active issue leaked through filter: %s (status=%s)",
						issue.ID, issue.Status)
				}
			}
		})
	}
}

// TestPerfInvariance_FilterPipelineTiming verifies that the filter + join pipeline
// maintains constant downstream time regardless of closed agent count.
func TestPerfInvariance_FilterPipelineTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}

	const activeCount = 5
	const iterations = 500

	// Helper to time the pipeline with a given closed count.
	timePipeline := func(closedCount int) time.Duration {
		allIssues := generateMixedIssues(activeCount, closedCount)

		start := time.Now()
		for i := 0; i < iterations; i++ {
			issues := discovery.FilterActiveIssues(allIssues)
			// Simulate downstream: build join data for active issues only
			manifests := make(map[string]*spawn.AgentManifest, len(issues))
			liveness := make(map[string]execution.SessionStatusInfo, len(issues))
			phases := make(map[string]string, len(issues))
			for _, issue := range issues {
				sessID := "sess-" + issue.ID
				manifests[issue.ID] = &spawn.AgentManifest{
					BeadsID:   issue.ID,
					SessionID: sessID,
				}
				liveness[sessID] = execution.SessionStatusInfo{Type: "busy"}
				phases[issue.ID] = "Implementing"
			}
			discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)
		}
		return time.Since(start)
	}

	// Warm up
	timePipeline(0)

	baseline := timePipeline(0)
	t200 := timePipeline(200)

	ratio := float64(t200) / float64(baseline)
	t.Logf("Pipeline baseline (0 closed, %d iters): %v", iterations, baseline)
	t.Logf("Pipeline with 200 closed (%d iters): %v", iterations, t200)
	t.Logf("Ratio: %.2fx", ratio)

	// With proper filtering, adding 200 closed issues should only add
	// FilterActiveIssues overhead (scanning 200 more status strings).
	// The downstream join work is identical (5 active agents both times).
	// Threshold of 3x is generous to avoid flakiness — a real regression
	// (closed agents leaking into join) would show 40x+ ratio.
	if ratio > 3.0 {
		t.Errorf("Pipeline performance degraded with historical agents: ratio=%.2fx (threshold: 3.0x). "+
			"This suggests closed agents are leaking into downstream processing (O(historical) regression).", ratio)
	}
}

// TestPerfInvariance_JoinLinearScaling verifies that JoinWithReasonCodes
// scales linearly with input size, not super-linearly.
// Catches regressions like nested loops in the join logic.
func TestPerfInvariance_JoinLinearScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}

	const iterations = 1000

	// Warm up
	issues, manifests, liveness, phases := buildJoinData(5)
	discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)

	// Time with 5 agents (baseline)
	issues5, m5, l5, p5 := buildJoinData(5)
	start := time.Now()
	for i := 0; i < iterations; i++ {
		discovery.JoinWithReasonCodes(issues5, m5, l5, p5)
	}
	baseline := time.Since(start)

	// Time with 200 agents (should be ~40x baseline for linear scaling)
	issues200, m200, l200, p200 := buildJoinData(200)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		discovery.JoinWithReasonCodes(issues200, m200, l200, p200)
	}
	scaled := time.Since(start)

	ratio := float64(scaled) / float64(baseline)
	t.Logf("Join baseline (5 agents, %d iters): %v", iterations, baseline)
	t.Logf("Join scaled (200 agents, %d iters): %v", iterations, scaled)
	t.Logf("Ratio: %.1fx (expected ~40x for linear scaling from 5→200)", ratio)

	// Linear scaling: 200/5 = 40x. Allow up to 80x for GC/cache effects.
	// Super-linear (O(N^2)) would show ~1600x ratio.
	if ratio > 80 {
		t.Errorf("JoinWithReasonCodes appears super-linear: ratio=%.1fx (threshold: 80x). "+
			"Expected ~40x for O(N) scaling from 5→200 agents.", ratio)
	}
}

// BenchmarkJoinWithReasonCodes benchmarks the join function at varying scales
// for continuous performance monitoring.
func BenchmarkJoinWithReasonCodes(b *testing.B) {
	for _, n := range []int{5, 50, 100, 200} {
		b.Run(fmt.Sprintf("agents=%d", n), func(b *testing.B) {
			issues, manifests, liveness, phases := buildJoinData(n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				discovery.JoinWithReasonCodes(issues, manifests, liveness, phases)
			}
		})
	}
}

// BenchmarkFilterActiveIssues benchmarks filter performance with varying
// historical agent counts. After orch-go-1096, the orch:agent label is
// removed on close, so this function should only see active issues in
// production. But if someone reintroduces O(historical) scanning, this
// benchmark surfaces the scaling cost.
func BenchmarkFilterActiveIssues(b *testing.B) {
	for _, closed := range []int{0, 50, 100, 200, 500, 1000} {
		b.Run(fmt.Sprintf("active=5_closed=%d", closed), func(b *testing.B) {
			issues := generateMixedIssues(5, closed)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				discovery.FilterActiveIssues(issues)
			}
		})
	}
}
