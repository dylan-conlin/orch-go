package main

import (
	"testing"
)

func TestClassifyLayer(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"cmd/orch/main.go", "cli"},
		{"cmd/orch/serve.go", "api"},
		{"cmd/orch/serve_agents.go", "api"},
		{"cmd/orch/serve_beads.go", "api"},
		{"pkg/daemon/daemon.go", "pkg"},
		{"pkg/spawn/config.go", "pkg"},
		{"web/src/routes/+page.svelte", "web"},
		{"web/src/lib/stores/agents.ts", "web"},
		{"plugins/coaching.ts", "plugins"},
		{"go.mod", ""},
		{"README.md", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := classifyLayer(tt.path)
			if result != tt.expected {
				t.Errorf("classifyLayer(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractConcept(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"pkg/daemon/daemon.go", "daemon"},
		{"cmd/orch/daemon.go", "daemon"},
		{"web/src/lib/stores/daemon.ts", "daemon"},
		{"cmd/orch/spawn_cmd.go", "spawn"},
		{"pkg/spawn/config.go", "spawn"},
		{"cmd/orch/serve_agents.go", "agents"},
		{"web/src/lib/stores/agents.ts", "agents"},
		{"web/src/lib/components/agent-card/AgentCard.svelte", "agent-card"},
		{"pkg/verify/check.go", "verify"},
		{"cmd/orch/complete_verify.go", "verify"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractConcept(tt.path)
			if result != tt.expected {
				t.Errorf("extractConcept(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsHealthyCoupling(t *testing.T) {
	tests := []struct {
		name     string
		fileA    string
		fileB    string
		expected bool
	}{
		{
			name:     "test pair",
			fileA:    "pkg/daemon/daemon.go",
			fileB:    "pkg/daemon/daemon_test.go",
			expected: true,
		},
		{
			name:     "same directory",
			fileA:    "pkg/daemon/daemon.go",
			fileB:    "pkg/daemon/config.go",
			expected: true,
		},
		{
			name:     "cross layer - not healthy",
			fileA:    "cmd/orch/daemon.go",
			fileB:    "pkg/daemon/daemon.go",
			expected: false,
		},
		{
			name:     "web internal coupling",
			fileA:    "web/src/lib/stores/agents.ts",
			fileB:    "web/src/lib/components/agent-card/AgentCard.svelte",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHealthyCoupling(tt.fileA, tt.fileB)
			if result != tt.expected {
				t.Errorf("isHealthyCoupling(%q, %q) = %v, want %v",
					tt.fileA, tt.fileB, result, tt.expected)
			}
		})
	}
}

func TestGenerateCouplingRecommendation(t *testing.T) {
	tests := []struct {
		concept  string
		score    float64
		layers   int
		contains string
	}{
		{"daemon", 180.0, 3, "CRITICAL"},
		{"agent-status", 67.0, 3, "HIGH"},
		{"session", 20.0, 2, "MODERATE"},
	}

	for _, tt := range tests {
		t.Run(tt.concept, func(t *testing.T) {
			rec := generateCouplingRecommendation(tt.concept, tt.score, tt.layers)
			if !hotspotContains(rec, tt.contains) {
				t.Errorf("generateCouplingRecommendation(%q, %.1f, %d) = %q, should contain %q",
					tt.concept, tt.score, tt.layers, rec, tt.contains)
			}
		})
	}
}

func TestParseGitLogForCoupling(t *testing.T) {
	// Simulate git log output with --name-only format (exactly 40-char hex hashes)
	gitOutput := `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

cmd/orch/daemon.go
pkg/daemon/daemon.go
web/src/lib/stores/daemon.ts

bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb

cmd/orch/spawn_cmd.go
pkg/spawn/config.go

cccccccccccccccccccccccccccccccccccccccc

pkg/daemon/daemon.go
pkg/daemon/daemon_test.go

dddddddddddddddddddddddddddddddddddddddd

cmd/orch/main.go
`

	commits := parseGitLogCommits(gitOutput)

	if len(commits) != 4 {
		t.Fatalf("Expected 4 commits, got %d", len(commits))
	}

	// First commit: 3 files across 3 layers (cli, pkg, web)
	if len(commits[0].files) != 3 {
		t.Errorf("First commit: expected 3 files, got %d", len(commits[0].files))
	}

	// Second commit: 2 files across 2 layers (cli, pkg)
	if len(commits[1].files) != 2 {
		t.Errorf("Second commit: expected 2 files, got %d", len(commits[1].files))
	}

	// Third commit: 2 files, 1 layer (pkg only) - should be filtered as same-layer
	if len(commits[2].files) != 2 {
		t.Errorf("Third commit: expected 2 files, got %d", len(commits[2].files))
	}
}

func TestFilterCrossSurfaceCommits(t *testing.T) {
	commits := []commitInfo{
		{hash: "abc", files: []string{"cmd/orch/daemon.go", "pkg/daemon/daemon.go", "web/src/lib/stores/daemon.ts"}},
		{hash: "def", files: []string{"cmd/orch/spawn_cmd.go", "pkg/spawn/config.go"}},
		{hash: "ghi", files: []string{"pkg/daemon/daemon.go", "pkg/daemon/daemon_test.go"}},
		{hash: "jkl", files: []string{"cmd/orch/main.go"}},
	}

	crossSurface := filterCrossSurfaceCommits(commits, 2)

	// Only first two commits touch 2+ layers
	if len(crossSurface) != 2 {
		t.Errorf("Expected 2 cross-surface commits, got %d", len(crossSurface))
	}
}

func TestBuildCouplingClusters(t *testing.T) {
	crossSurfaceCommits := []commitInfo{
		{hash: "abc", files: []string{"cmd/orch/daemon.go", "pkg/daemon/daemon.go", "web/src/lib/stores/daemon.ts"}},
		{hash: "def", files: []string{"cmd/orch/daemon.go", "pkg/daemon/daemon.go"}},
		{hash: "ghi", files: []string{"cmd/orch/spawn_cmd.go", "pkg/spawn/config.go"}},
	}

	clusters := buildCouplingClusters(crossSurfaceCommits)

	// Should find at least "daemon" and "spawn" clusters
	daemonFound := false
	spawnFound := false
	for _, c := range clusters {
		if c.Concept == "daemon" {
			daemonFound = true
			if c.LayerCount < 2 {
				t.Errorf("daemon cluster: expected 2+ layers, got %d", c.LayerCount)
			}
		}
		if c.Concept == "spawn" {
			spawnFound = true
		}
	}

	if !daemonFound {
		t.Error("Expected to find daemon cluster")
	}
	if !spawnFound {
		t.Error("Expected to find spawn cluster")
	}
}

func TestCouplingClusterScoring(t *testing.T) {
	cluster := CouplingCluster{
		Concept:       "daemon",
		Files:         []string{"cmd/orch/daemon.go", "pkg/daemon/daemon.go", "web/src/lib/stores/daemon.ts"},
		Layers:        []string{"cli", "pkg", "web"},
		LayerCount:    3,
		FileCount:     3,
		CoChangeCount: 6,
	}

	score := scoreCouplingCluster(cluster)

	// score = layers(3) * files(3) * avg_frequency(6/3=2.0) = 18.0
	expectedScore := 18.0
	if score != expectedScore {
		t.Errorf("scoreCouplingCluster() = %.1f, want %.1f", score, expectedScore)
	}
}

func TestCouplingClusterToHotspot(t *testing.T) {
	cluster := CouplingCluster{
		Concept:       "daemon",
		Files:         []string{"cmd/orch/daemon.go", "pkg/daemon/daemon.go", "web/src/stores/daemon.ts"},
		Layers:        []string{"cli", "pkg", "web"},
		LayerCount:    3,
		FileCount:     3,
		CoChangeCount: 24,
	}

	score := scoreCouplingCluster(cluster)
	hotspot := couplingClusterToHotspot(cluster, score)

	if hotspot.Type != "coupling-cluster" {
		t.Errorf("Expected type 'coupling-cluster', got %q", hotspot.Type)
	}
	if hotspot.Path != "daemon" {
		t.Errorf("Expected path 'daemon', got %q", hotspot.Path)
	}
	if hotspot.Score != int(score) {
		t.Errorf("Expected score %d, got %d", int(score), hotspot.Score)
	}
	if len(hotspot.RelatedFiles) != 3 {
		t.Errorf("Expected 3 related files, got %d", len(hotspot.RelatedFiles))
	}
}

func TestMatchPathToHotspots_CouplingCluster(t *testing.T) {
	hotspots := []Hotspot{
		{
			Path:         "daemon",
			Type:         "coupling-cluster",
			Score:        180,
			RelatedFiles: []string{"cmd/orch/daemon.go", "pkg/daemon/daemon.go"},
		},
	}

	tests := []struct {
		name          string
		path          string
		expectedMatch bool
		expectedScore int
	}{
		{
			name:          "concept name in path",
			path:          "pkg/daemon/config.go",
			expectedMatch: true,
			expectedScore: 180,
		},
		{
			name:          "concept in cmd path",
			path:          "cmd/orch/daemon.go",
			expectedMatch: true,
			expectedScore: 180,
		},
		{
			name:          "no match",
			path:          "pkg/spawn/config.go",
			expectedMatch: false,
			expectedScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := matchPathToHotspots(tt.path, hotspots)
			if matched != tt.expectedMatch {
				t.Errorf("matchPathToHotspots(%q) matched = %v, want %v", tt.path, matched, tt.expectedMatch)
			}
			if score != tt.expectedScore {
				t.Errorf("matchPathToHotspots(%q) score = %d, want %d", tt.path, score, tt.expectedScore)
			}
		})
	}
}
