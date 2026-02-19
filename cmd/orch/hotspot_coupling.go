package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// CouplingCluster represents a group of files that change together across architectural layers.
type CouplingCluster struct {
	Concept       string   // "daemon", "agent-status", "spawn", etc.
	Files         []string // All files in the cluster
	Layers        []string // Distinct layers: "cli", "pkg", "web", "api", "plugins"
	LayerCount    int      // len(Layers)
	FileCount     int      // len(Files)
	CoChangeCount int      // Total cross-surface co-change count
}

// commitInfo holds a parsed git commit with its files.
type commitInfo struct {
	hash  string
	files []string
}

// analyzeCouplingClusters detects cross-layer coupling hotspots from git history.
// Returns hotspots with type="coupling-cluster" and the total number of clusters found.
func analyzeCouplingClusters(projectDir string, daysBack int) ([]Hotspot, int, error) {
	since := fmt.Sprintf("--since=%d days ago", daysBack)
	cmd := exec.Command("git", "log", since, "--pretty=format:%H", "--name-only", "--diff-filter=ACMR")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, 0, fmt.Errorf("git log failed: %w", err)
	}

	commits := parseGitLogCommits(string(output))
	crossSurface := filterCrossSurfaceCommits(commits, 2)

	if len(crossSurface) == 0 {
		return nil, 0, nil
	}

	clusters := buildCouplingClusters(crossSurface)

	var hotspots []Hotspot
	for _, cluster := range clusters {
		score := scoreCouplingCluster(cluster)
		if score >= 15 {
			hotspots = append(hotspots, couplingClusterToHotspot(cluster, score))
		}
	}

	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Score > hotspots[j].Score
	})

	return hotspots, len(clusters), nil
}

// parseGitLogCommits parses git log output (--pretty=format:%H --name-only) into commits.
func parseGitLogCommits(output string) []commitInfo {
	var commits []commitInfo
	var current *commitInfo

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil && len(current.files) > 0 {
				commits = append(commits, *current)
				current = nil
			}
			continue
		}

		// A 40-char hex string is a commit hash
		if len(line) == 40 && isHexString(line) {
			if current != nil && len(current.files) > 0 {
				commits = append(commits, *current)
			}
			current = &commitInfo{hash: line}
			continue
		}

		// Otherwise it's a file path
		if current != nil {
			current.files = append(current.files, line)
		}
	}

	// Don't forget the last commit
	if current != nil && len(current.files) > 0 {
		commits = append(commits, *current)
	}

	return commits
}

// isHexString returns true if s contains only hex characters.
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// filterCrossSurfaceCommits keeps only commits touching minLayers+ architectural layers.
func filterCrossSurfaceCommits(commits []commitInfo, minLayers int) []commitInfo {
	var result []commitInfo
	for _, c := range commits {
		layers := make(map[string]bool)
		for _, f := range c.files {
			if layer := classifyLayer(f); layer != "" {
				layers[layer] = true
			}
		}
		if len(layers) >= minLayers {
			result = append(result, c)
		}
	}
	return result
}

// classifyLayer maps a file path to its architectural layer.
func classifyLayer(path string) string {
	if strings.HasPrefix(path, "cmd/orch/serve") {
		return "api"
	}
	if strings.HasPrefix(path, "cmd/") {
		return "cli"
	}
	if strings.HasPrefix(path, "pkg/") {
		return "pkg"
	}
	if strings.HasPrefix(path, "web/") {
		return "web"
	}
	if strings.HasPrefix(path, "plugins/") {
		return "plugins"
	}
	return ""
}

// extractConcept extracts a concept keyword from a file path.
// Uses directory name first, then file name stem.
func extractConcept(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// For pkg/X/ paths, use the package directory name
	if strings.HasPrefix(path, "pkg/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	// For web/ paths, try to extract from the deepest meaningful directory
	if strings.HasPrefix(path, "web/") {
		// Check stores directory: web/src/lib/stores/daemon.ts -> "daemon"
		if strings.Contains(dir, "/stores") {
			stem := fileNameStem(base)
			return stem
		}
		// Check components directory: web/src/lib/components/agent-card/ -> "agent-card"
		if strings.Contains(dir, "/components/") {
			parts := strings.Split(dir, "/components/")
			if len(parts) == 2 {
				componentDir := strings.Split(parts[1], "/")[0]
				if componentDir != "" {
					return componentDir
				}
			}
		}
		// Fallback: use file name stem
		return fileNameStem(base)
	}

	// For cmd/orch/ paths, extract from file name
	if strings.HasPrefix(path, "cmd/orch/") {
		stem := fileNameStem(base)
		// Handle prefixed files like serve_agents.go -> "agents", spawn_cmd.go -> "spawn"
		if strings.HasPrefix(stem, "serve_") {
			return strings.TrimPrefix(stem, "serve_")
		}
		if strings.HasPrefix(stem, "complete_") {
			return "verify" // complete_verify.go -> verify
		}
		// Remove common suffixes
		stem = strings.TrimSuffix(stem, "_cmd")
		return stem
	}

	// For plugins/ paths, use file name stem
	if strings.HasPrefix(path, "plugins/") {
		return fileNameStem(base)
	}

	return fileNameStem(base)
}

// fileNameStem returns the file name without extension.
func fileNameStem(name string) string {
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext)
}

// buildCouplingClusters groups cross-surface commit files by concept.
func buildCouplingClusters(commits []commitInfo) []CouplingCluster {
	// Track files per concept and co-change counts
	conceptFiles := make(map[string]map[string]bool)   // concept -> set of files
	conceptLayers := make(map[string]map[string]bool)   // concept -> set of layers
	conceptCoChanges := make(map[string]int)            // concept -> number of cross-surface commits

	for _, c := range commits {
		// Extract concepts from this commit's files
		commitConcepts := make(map[string]bool)
		for _, f := range c.files {
			concept := extractConcept(f)
			if concept == "" {
				continue
			}
			layer := classifyLayer(f)
			if layer == "" {
				continue
			}

			if conceptFiles[concept] == nil {
				conceptFiles[concept] = make(map[string]bool)
			}
			if conceptLayers[concept] == nil {
				conceptLayers[concept] = make(map[string]bool)
			}

			conceptFiles[concept][f] = true
			conceptLayers[concept][layer] = true
			commitConcepts[concept] = true
		}

		// Count this commit for each concept it touches
		for concept := range commitConcepts {
			conceptCoChanges[concept]++
		}
	}

	// Build clusters, filtering out healthy coupling
	var clusters []CouplingCluster
	for concept, fileSet := range conceptFiles {
		layerSet := conceptLayers[concept]

		// Skip single-layer clusters (healthy coupling)
		if len(layerSet) < 2 {
			continue
		}

		// Collect files, filtering out healthy pairs
		var files []string
		for f := range fileSet {
			files = append(files, f)
		}
		sort.Strings(files)

		// Filter healthy coupling: remove files where all their co-changes
		// are with files in the same directory or test pairs
		var unhealthyFiles []string
		for _, f := range files {
			hasUnhealthyPair := false
			for _, other := range files {
				if f == other {
					continue
				}
				if !isHealthyCoupling(f, other) {
					hasUnhealthyPair = true
					break
				}
			}
			if hasUnhealthyPair {
				unhealthyFiles = append(unhealthyFiles, f)
			}
		}

		if len(unhealthyFiles) < 2 {
			continue
		}

		// Recalculate layers from unhealthy files
		finalLayers := make(map[string]bool)
		for _, f := range unhealthyFiles {
			if layer := classifyLayer(f); layer != "" {
				finalLayers[layer] = true
			}
		}

		if len(finalLayers) < 2 {
			continue
		}

		var layerList []string
		for l := range finalLayers {
			layerList = append(layerList, l)
		}
		sort.Strings(layerList)

		clusters = append(clusters, CouplingCluster{
			Concept:       concept,
			Files:         unhealthyFiles,
			Layers:        layerList,
			LayerCount:    len(layerList),
			FileCount:     len(unhealthyFiles),
			CoChangeCount: conceptCoChanges[concept],
		})
	}

	return clusters
}

// isHealthyCoupling returns true if the coupling between two files is expected/healthy.
func isHealthyCoupling(fileA, fileB string) bool {
	dirA := filepath.Dir(fileA)
	dirB := filepath.Dir(fileB)

	// Same directory = healthy
	if dirA == dirB {
		return true
	}

	// Test pair = healthy
	if isTestPair(fileA, fileB) {
		return true
	}

	// Both in web/ = healthy (frontend internal coupling is expected)
	if strings.HasPrefix(fileA, "web/") && strings.HasPrefix(fileB, "web/") {
		return true
	}

	return false
}

// isTestPair returns true if one file is a test file for the other.
func isTestPair(fileA, fileB string) bool {
	testSuffixes := []string{"_test.go", ".test.ts", ".test.js", ".spec.ts", ".spec.js"}
	for _, suffix := range testSuffixes {
		if strings.HasSuffix(fileA, suffix) || strings.HasSuffix(fileB, suffix) {
			return true
		}
	}
	return false
}

// scoreCouplingCluster calculates the coupling score for a cluster.
// Formula: layer_count * file_count * avg_co_change_frequency
func scoreCouplingCluster(cluster CouplingCluster) float64 {
	if cluster.FileCount == 0 {
		return 0
	}
	avgFrequency := float64(cluster.CoChangeCount) / float64(cluster.FileCount)
	return float64(cluster.LayerCount) * float64(cluster.FileCount) * avgFrequency
}

// couplingClusterToHotspot converts a CouplingCluster to a Hotspot entry.
func couplingClusterToHotspot(cluster CouplingCluster, score float64) Hotspot {
	layerStr := strings.Join(cluster.Layers, ", ")
	details := fmt.Sprintf("%d files across %d layers (%s), %d cross-surface commits",
		cluster.FileCount, cluster.LayerCount, layerStr, cluster.CoChangeCount)

	return Hotspot{
		Path:           cluster.Concept,
		Type:           "coupling-cluster",
		Score:          int(score),
		Details:        details,
		RelatedFiles:   cluster.Files,
		Recommendation: generateCouplingRecommendation(cluster.Concept, score, cluster.LayerCount),
	}
}

// generateCouplingRecommendation creates a recommendation based on coupling score.
func generateCouplingRecommendation(concept string, score float64, layers int) string {
	if score >= 100 {
		return fmt.Sprintf("CRITICAL: '%s' spans %d layers with high coupling - spawn architect to design extraction before modifying", concept, layers)
	}
	if score >= 40 {
		return fmt.Sprintf("HIGH: '%s' has significant cross-layer coupling - review full touch surface before making changes", concept)
	}
	return fmt.Sprintf("MODERATE: '%s' shows cross-layer coupling - be aware of downstream impacts when modifying", concept)
}
