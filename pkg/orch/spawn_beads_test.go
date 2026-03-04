package orch

import (
	"testing"
)

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
