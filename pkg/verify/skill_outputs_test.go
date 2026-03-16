package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSkillNameFromSpawnContext(t *testing.T) {
	tests := []struct {
		name, content, expected string
	}{
		{"SKILL GUIDANCE pattern", "## SKILL GUIDANCE (feature-impl)\n", "feature-impl"},
		{"investigation", "## SKILL GUIDANCE (investigation)\n", "investigation"},
		{"no skill", "TASK: Do something\n", ""},
		{"case insensitive", "## skill guidance (architect)", "architect"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			os.WriteFile(filepath.Join(tmpDir, "SPAWN_CONTEXT.md"), []byte(tt.content), 0644)
			got, err := ExtractSkillNameFromSpawnContext(tmpDir)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
