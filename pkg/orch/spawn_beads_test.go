package orch

import (
	"testing"
)

func TestResolveCrossRepoBeadsDir(t *testing.T) {
	tests := []struct {
		name               string
		beadsID            string
		cwd                string
		projectDir         string
		issueExistsInTarget bool
		want               string
	}{
		{
			name:               "same project - no override needed",
			beadsID:            "orch-go-7zg08",
			cwd:                "/Users/test/Documents/orch-go",
			projectDir:         "/Users/test/Documents/orch-go",
			issueExistsInTarget: true,
			want:               "",
		},
		{
			name:               "issue in CWD project, agent works in target - inject BEADS_DIR",
			beadsID:            "orch-go-7zg08",
			cwd:                "/Users/test/Documents/orch-go",
			projectDir:         "/Users/test/Documents/toolshed",
			issueExistsInTarget: false,
			want:               "/Users/test/Documents/orch-go/.beads",
		},
		{
			name:               "issue in target project, spawned from different CWD - no override",
			beadsID:            "tw-jpnq",
			cwd:                "/Users/test/Documents/orch-go",
			projectDir:         "/Users/test/Documents/toolshed",
			issueExistsInTarget: true,
			want:               "",
		},
		{
			name:               "daemon spawns target-project issue - no override needed",
			beadsID:            "pw-abc1",
			cwd:                "/Users/test/Documents/orch-go",
			projectDir:         "/Users/test/Documents/price-watch",
			issueExistsInTarget: true,
			want:               "",
		},
		{
			name:               "daemon spawns CWD issue in foreign repo - inject BEADS_DIR",
			beadsID:            "orch-go-def2",
			cwd:                "/Users/test/Documents/orch-go",
			projectDir:         "/Users/test/Documents/price-watch",
			issueExistsInTarget: false,
			want:               "/Users/test/Documents/orch-go/.beads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issueExists := func(beadsID, projectDir string) bool {
				return tt.issueExistsInTarget
			}
			got := ResolveCrossRepoBeadsDir(tt.beadsID, tt.cwd, tt.projectDir, issueExists)
			if got != tt.want {
				t.Errorf("ResolveCrossRepoBeadsDir(%q, %q, %q) = %q, want %q",
					tt.beadsID, tt.cwd, tt.projectDir, got, tt.want)
			}
		})
	}
}

func TestDetermineBeadsID_PassesProjectDir(t *testing.T) {
	// Verify that determineBeadsID passes projectDir to createBeadsFn.
	// This is critical for cross-project spawns: issues must be created
	// in the target project's .beads/, not the source (CWD) project's.
	tests := []struct {
		name       string
		projectDir string
		noTrack    bool
	}{
		{
			name:       "normal spawn passes projectDir",
			projectDir: "/Users/test/Documents/price-watch",
			noTrack:    false,
		},
		{
			name:       "no-track spawn passes projectDir",
			projectDir: "/Users/test/Documents/price-watch",
			noTrack:    true,
		},
		{
			name:       "empty dir falls back to CWD in beads",
			projectDir: "",
			noTrack:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedDir string
			createFn := func(projectName, skillName, task, dir string) (string, error) {
				capturedDir = dir
				return "test-abc123", nil
			}
			_, err := determineBeadsID("test-project", "test-skill", "test task", "", tt.noTrack, createFn, tt.projectDir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if capturedDir != tt.projectDir {
				t.Errorf("createBeadsFn received dir=%q, want %q", capturedDir, tt.projectDir)
			}
		})
	}
}

func TestDetectCrossRepo(t *testing.T) {
	tests := []struct {
		name       string
		cwd        string
		projectDir string
		want       string
	}{
		{
			name:       "same project - not cross-repo",
			cwd:        "/Users/test/Documents/orch-go",
			projectDir: "/Users/test/Documents/orch-go",
			want:       "",
		},
		{
			name:       "different project - cross-repo",
			cwd:        "/Users/test/Documents/orch-go",
			projectDir: "/Users/test/Documents/price-watch",
			want:       "orch-go",
		},
		{
			name:       "different path same basename - not cross-repo",
			cwd:        "/Users/test/Documents/orch-go",
			projectDir: "/Users/other/projects/orch-go",
			want:       "",
		},
		{
			name:       "empty cwd - not cross-repo",
			cwd:        "",
			projectDir: "/Users/test/Documents/price-watch",
			want:       "",
		},
		{
			name:       "empty projectDir - not cross-repo",
			cwd:        "/Users/test/Documents/orch-go",
			projectDir: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectCrossRepo(tt.cwd, tt.projectDir)
			if got != tt.want {
				t.Errorf("DetectCrossRepo(%q, %q) = %q, want %q", tt.cwd, tt.projectDir, got, tt.want)
			}
		})
	}
}
