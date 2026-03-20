package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/claims"
	"github.com/dylan-conlin/orch-go/pkg/kbmetrics"
	"github.com/dylan-conlin/orch-go/pkg/orient"
	"github.com/dylan-conlin/orch-go/pkg/thread"
)

// collectExploreCandidates aggregates 6 signals into explore-ready recommendations.
// Signals: tension clusters, stale decisions, thread accumulation, untested claim
// density, investigation orphan clusters, and code hotspots without models.
func collectExploreCandidates(projectDir, modelsDir string, now time.Time) []orient.ExploreCandidate {
	var candidates []orient.ExploreCandidate

	// 1. Tension clusters — cross-model claim conflicts
	candidates = append(candidates, exploreCandidatesFromTensionClusters(modelsDir)...)

	// 2. Stale decisions — unanchored decisions needing review
	kbDir := filepath.Join(projectDir, ".kb")
	candidates = append(candidates, exploreCandidatesFromStaleDecisions(kbDir, projectDir)...)

	// 3. Thread accumulation — threads with many entries but no resolution
	threadsDir := filepath.Join(projectDir, ".kb", "threads")
	candidates = append(candidates, exploreCandidatesFromThreads(threadsDir)...)

	// 4. Untested claim density — models with high untested claim counts
	candidates = append(candidates, exploreCandidatesFromUntestedClaims(modelsDir, now)...)

	// 5. Investigation orphan clusters — positive-unlinked findings
	candidates = append(candidates, exploreCandidatesFromOrphans(kbDir)...)

	// 6. Code hotspots without models — hotspot areas lacking KB coverage
	candidates = append(candidates, exploreCandidatesFromHotspots(projectDir, modelsDir)...)

	// Sort by score descending, limit to top 5
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}

	return candidates
}

// exploreCandidatesFromTensionClusters finds cross-model claim conflicts.
func exploreCandidatesFromTensionClusters(modelsDir string) []orient.ExploreCandidate {
	files, err := claims.ScanAll(modelsDir)
	if err != nil || len(files) == 0 {
		return nil
	}

	clusters := claims.FindClusters(files, 2)
	var candidates []orient.ExploreCandidate
	for _, tc := range clusters {
		if len(candidates) >= 2 {
			break
		}
		candidates = append(candidates, orient.ExploreCandidate{
			Question: fmt.Sprintf("Resolve tension cluster %s: %d claims across %s converge on %s in %s",
				tc.ID, len(tc.Claims), strings.Join(tc.Models, ", "), tc.TargetClaim, tc.TargetModel),
			Signal: "tension-cluster",
			Score:  tc.Score,
			Reason: fmt.Sprintf("%d models, domains: %s", len(tc.Models), strings.Join(tc.DomainTags, ", ")),
		})
	}
	return candidates
}

// exploreCandidatesFromStaleDecisions finds unanchored decisions worth revisiting.
func exploreCandidatesFromStaleDecisions(kbDir, projectDir string) []orient.ExploreCandidate {
	reports, err := kbmetrics.AuditDecisions(kbDir, projectDir)
	if err != nil {
		return nil
	}

	var candidates []orient.ExploreCandidate
	for _, r := range reports {
		if r.Score != "unanchored" {
			continue
		}
		if len(candidates) >= 2 {
			break
		}
		candidates = append(candidates, orient.ExploreCandidate{
			Question: fmt.Sprintf("Investigate whether decision '%s' is still valid and needs enforcement",
				r.Decision.Title),
			Signal: "stale-decision",
			Score:  3.0, // unanchored decisions are moderately urgent
			Reason: fmt.Sprintf("Decision from %s has no tests, gates, or hooks enforcing it", r.Decision.Date),
		})
	}
	return candidates
}

// exploreCandidatesFromThreads finds threads with high entry count but no resolution.
func exploreCandidatesFromThreads(threadsDir string) []orient.ExploreCandidate {
	summaries, err := thread.ActiveThreads(threadsDir, 30) // wider window for accumulation
	if err != nil || len(summaries) == 0 {
		return nil
	}

	var candidates []orient.ExploreCandidate
	for _, s := range summaries {
		if s.EntryCount < 5 {
			continue
		}
		if len(candidates) >= 2 {
			break
		}
		candidates = append(candidates, orient.ExploreCandidate{
			Question: fmt.Sprintf("Synthesize thread '%s' (%d entries) into a decision or model",
				s.Title, s.EntryCount),
			Signal: "thread-accumulation",
			Score:  float64(s.EntryCount) * 0.5,
			Reason: fmt.Sprintf("%d entries accumulated without resolution", s.EntryCount),
		})
	}
	return candidates
}

// exploreCandidatesFromUntestedClaims finds models with high untested claim density.
func exploreCandidatesFromUntestedClaims(modelsDir string, now time.Time) []orient.ExploreCandidate {
	files, err := claims.ScanAll(modelsDir)
	if err != nil || len(files) == 0 {
		return nil
	}

	type modelUntested struct {
		name     string
		untested int
		total    int
	}

	var models []modelUntested
	for name, f := range files {
		untested := 0
		for _, c := range f.Claims {
			if c.Confidence == claims.Unconfirmed || c.IsStale(now) {
				untested++
			}
		}
		if untested >= 3 {
			models = append(models, modelUntested{name: name, untested: untested, total: len(f.Claims)})
		}
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].untested > models[j].untested
	})

	var candidates []orient.ExploreCandidate
	for _, m := range models {
		if len(candidates) >= 2 {
			break
		}
		candidates = append(candidates, orient.ExploreCandidate{
			Question: fmt.Sprintf("Probe untested claims in model '%s' (%d of %d unconfirmed/stale)",
				m.name, m.untested, m.total),
			Signal: "untested-claims",
			Score:  float64(m.untested) * 1.5,
			Reason: fmt.Sprintf("%d claims need validation in %s", m.untested, m.name),
		})
	}
	return candidates
}

// exploreCandidatesFromOrphans finds positive-unlinked investigation clusters.
func exploreCandidatesFromOrphans(kbDir string) []orient.ExploreCandidate {
	report, err := kbmetrics.ComputeStratifiedOrphanRate(kbDir)
	if err != nil {
		return nil
	}

	positiveCount := report.Categories[kbmetrics.CategoryPositiveUnlinked]
	if positiveCount < 3 {
		return nil
	}

	return []orient.ExploreCandidate{{
		Question: fmt.Sprintf("Link %d positive-unlinked orphan investigations to models or decisions",
			positiveCount),
		Signal: "orphan-cluster",
		Score:  float64(positiveCount) * 0.8,
		Reason: fmt.Sprintf("%d investigations have findings but no model/decision connection", positiveCount),
	}}
}

// exploreCandidatesFromHotspots finds code hotspot areas without KB model coverage.
func exploreCandidatesFromHotspots(projectDir, modelsDir string) []orient.ExploreCandidate {
	// Get bloat hotspots (files > 800 lines)
	hotspots, _, err := analyzeBloatFiles(projectDir, 800)
	if err != nil || len(hotspots) == 0 {
		return nil
	}

	// Get existing model names for coverage check
	modelNames := collectModelNames(modelsDir)

	var candidates []orient.ExploreCandidate
	for _, h := range hotspots {
		if len(candidates) >= 2 {
			break
		}
		// Extract domain keywords from hotspot path
		base := strings.TrimSuffix(filepath.Base(h.Path), filepath.Ext(h.Path))
		keywords := strings.Split(strings.ReplaceAll(base, "_", "-"), "-")

		// Check if any model covers this hotspot area
		covered := false
		for _, modelName := range modelNames {
			for _, kw := range keywords {
				if len(kw) > 3 && strings.Contains(modelName, kw) {
					covered = true
					break
				}
			}
			if covered {
				break
			}
		}

		if !covered {
			candidates = append(candidates, orient.ExploreCandidate{
				Question: fmt.Sprintf("Create knowledge model for hotspot area '%s' (%d lines, no model coverage)",
					h.Path, h.Score),
				Signal: "hotspot-no-model",
				Score:  float64(h.Score) / 500.0, // normalize: 1500-line file = 3.0
				Reason: fmt.Sprintf("%d-line file has no corresponding KB model", h.Score),
			})
		}
	}
	return candidates
}

// collectModelNames returns the list of model directory names from .kb/models/.
func collectModelNames(modelsDir string) []string {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}
