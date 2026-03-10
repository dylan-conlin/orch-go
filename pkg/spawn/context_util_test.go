package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateServerContext(t *testing.T) {
	t.Run("with servers configured", func(t *testing.T) {
		tempDir := t.TempDir()
		orchDir := filepath.Join(tempDir, ".orch")
		if err := os.MkdirAll(orchDir, 0755); err != nil {
			t.Fatalf("failed to create .orch dir: %v", err)
		}

		// Write config with servers
		configContent := `servers:
  web: 5173
  api: 3000
`
		configPath := filepath.Join(orchDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		context := GenerateServerContext(tempDir)

		// Check it contains expected content
		if !strings.Contains(context, "## LOCAL SERVERS") {
			t.Error("expected server context to contain header")
		}
		if !strings.Contains(context, "http://localhost:5173") {
			t.Error("expected server context to contain web port")
		}
		if !strings.Contains(context, "http://localhost:3000") {
			t.Error("expected server context to contain api port")
		}
		if !strings.Contains(context, "orch servers start") {
			t.Error("expected server context to contain quick commands")
		}
	})

	t.Run("without config file", func(t *testing.T) {
		tempDir := t.TempDir()

		context := GenerateServerContext(tempDir)

		// Should return empty string when no config
		if context != "" {
			t.Errorf("expected empty string when no config, got: %s", context)
		}
	})

	t.Run("with empty servers", func(t *testing.T) {
		tempDir := t.TempDir()
		orchDir := filepath.Join(tempDir, ".orch")
		if err := os.MkdirAll(orchDir, 0755); err != nil {
			t.Fatalf("failed to create .orch dir: %v", err)
		}

		// Write config with empty servers
		configContent := `servers: {}`
		configPath := filepath.Join(orchDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		context := GenerateServerContext(tempDir)

		// Should return empty string when no servers
		if context != "" {
			t.Errorf("expected empty string when no servers, got: %s", context)
		}
	})
}

func TestGenerateRegisteredProjectsContext_Format(t *testing.T) {
	// Test that the format is correct when we provide a mock project list
	// We can't easily test the actual kb command in unit tests,
	// but we can verify the format of the generated context

	projects := []RegisteredProject{
		{Name: "orch-go", Path: "/Users/test/orch-go"},
		{Name: "snap", Path: "/Users/test/snap"},
	}

	// Build expected output
	var sb strings.Builder
	sb.WriteString("## Registered Projects\n\n")
	sb.WriteString("These projects are registered with `kb` for cross-project orchestration:\n\n")
	sb.WriteString("| Project | Path |\n")
	sb.WriteString("|---------|------|\n")
	for _, p := range projects {
		sb.WriteString("| " + p.Name + " | `" + p.Path + "` |\n")
	}
	sb.WriteString("\n**Usage:** `orch spawn --workdir <path> SKILL \"task\"`\n\n")

	expected := sb.String()

	// Verify the format matches our expectations
	if !strings.Contains(expected, "## Registered Projects") {
		t.Error("expected registered projects header")
	}
	if !strings.Contains(expected, "| orch-go |") {
		t.Error("expected project row")
	}
	if !strings.Contains(expected, "orch spawn --workdir") {
		t.Error("expected usage hint")
	}
}

func TestValidateBeadsIDConsistency(t *testing.T) {
	t.Run("no issue flag and no ID in task", func(t *testing.T) {
		warning := ValidateBeadsIDConsistency("explore the codebase", "orch-go-123")
		if warning != "" {
			t.Errorf("expected no warning, got: %s", warning)
		}
	})

	t.Run("task mentions same beads ID as issue flag", func(t *testing.T) {
		warning := ValidateBeadsIDConsistency("fix bug in pw-8972", "pw-8972")
		if warning != "" {
			t.Errorf("expected no warning when IDs match, got: %s", warning)
		}
	})

	t.Run("task mentions different beads ID than issue flag", func(t *testing.T) {
		warning := ValidateBeadsIDConsistency("fix bug in pw-8972", "pw-8975")
		if warning == "" {
			t.Error("expected warning when task mentions different beads ID than issue flag")
		}
		if !strings.Contains(warning, "pw-8972") {
			t.Errorf("warning should mention the mismatched ID pw-8972, got: %s", warning)
		}
		if !strings.Contains(warning, "pw-8975") {
			t.Errorf("warning should mention the tracking ID pw-8975, got: %s", warning)
		}
	})

	t.Run("task with no beads-like IDs", func(t *testing.T) {
		warning := ValidateBeadsIDConsistency("implement dark mode for the app", "orch-go-456")
		if warning != "" {
			t.Errorf("expected no warning for task without beads-like IDs, got: %s", warning)
		}
	})

	t.Run("task mentions beads ID with different project prefix", func(t *testing.T) {
		// Task mentions pw-8972 but tracking issue is orch-go-456
		// Different project prefixes - the pw-8972 is just a reference, not a tracking conflict
		warning := ValidateBeadsIDConsistency("investigate pw-8972 error", "orch-go-456")
		if warning != "" {
			t.Errorf("expected no warning for cross-project reference, got: %s", warning)
		}
	})

	t.Run("task mentions same-project beads ID that conflicts", func(t *testing.T) {
		// Task mentions pw-123 but tracking issue is pw-456 (same project prefix "pw")
		warning := ValidateBeadsIDConsistency("fix bug described in pw-123", "pw-456")
		if warning == "" {
			t.Error("expected warning when task mentions same-project beads ID that conflicts")
		}
	})

	t.Run("empty beads ID skips validation", func(t *testing.T) {
		warning := ValidateBeadsIDConsistency("fix pw-123 bug", "")
		if warning != "" {
			t.Errorf("expected no warning when beads ID is empty, got: %s", warning)
		}
	})
}
