package tmux

import (
	"testing"
)

func TestGetIncludedProjects(t *testing.T) {
	configs := DefaultMultiProjectConfigs()

	tests := []struct {
		name     string
		project  string
		expected []string
	}{
		{
			name:    "price-watch includes pw alias",
			project: "price-watch",
			// Should include price-watch (self) + pw (beads ID prefix alias)
			expected: []string{"price-watch", "pw"},
		},
		{
			name:     "specs-platform includes sp alias",
			project:  "specs-platform",
			expected: []string{"specs-platform", "sp"},
		},
		{
			name:     "toolshed includes to alias",
			project:  "toolshed",
			expected: []string{"toolshed", "to"},
		},
		{
			name:    "orch-go includes ecosystem repos and work projects",
			project: "orch-go",
			// Should include all ecosystem repos plus work projects with aliases
			expected: []string{"orch-go", "orch-cli", "beads", "kb-cli", "orch-knowledge", "opencode", "price-watch", "pw", "specs-platform", "sp", "toolshed", "to"},
		},
		{
			name:     "unknown project returns just itself",
			project:  "unknown-project",
			expected: []string{"unknown-project"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetIncludedProjects(tt.project, configs)
			if len(got) != len(tt.expected) {
				t.Errorf("GetIncludedProjects(%q) = %v, want %v", tt.project, got, tt.expected)
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("GetIncludedProjects(%q)[%d] = %q, want %q", tt.project, i, v, tt.expected[i])
				}
			}
		})
	}
}
