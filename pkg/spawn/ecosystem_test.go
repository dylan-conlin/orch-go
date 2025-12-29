package spawn

import (
	"strings"
	"testing"
)

func TestIsEcosystemRepo(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		want        bool
	}{
		// Core orchestration repos
		{"orch-go is ecosystem", "orch-go", true},
		{"orch-cli is ecosystem", "orch-cli", true},
		{"kb-cli is ecosystem", "kb-cli", true},
		{"orch-knowledge is ecosystem", "orch-knowledge", true},
		{"beads is ecosystem", "beads", true},
		{"kn is ecosystem", "kn", true},

		// Additional ecosystem repos
		{"beads-ui-svelte is ecosystem", "beads-ui-svelte", true},
		{"glass is ecosystem", "glass", true},
		{"skillc is ecosystem", "skillc", true},
		{"agentlog is ecosystem", "agentlog", true},

		// Non-ecosystem repos
		{"random-project is not ecosystem", "random-project", false},
		{"empty string is not ecosystem", "", false},
		{"price-watch is not ecosystem", "price-watch", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEcosystemRepo(tt.projectName)
			if got != tt.want {
				t.Errorf("IsEcosystemRepo(%q) = %v, want %v", tt.projectName, got, tt.want)
			}
		})
	}
}

func TestExtractQuickReference(t *testing.T) {
	t.Run("extracts quick reference section", func(t *testing.T) {
		content := `# Ecosystem

> Purpose line

---

## Quick Reference

| Repo | Purpose | CLI |
|------|---------|-----|
| orch-go | Orchestration | orch |
| kb-cli | Knowledge base | kb |
| beads | Issue tracking | bd |

---

## Core Repos

### orch-go
More detailed info...
`
		result := ExtractQuickReference(content)

		// Should contain the table header
		if !strings.Contains(result, "## Quick Reference") {
			t.Error("expected result to contain Quick Reference heading")
		}
		// Should contain table rows
		if !strings.Contains(result, "| orch-go |") {
			t.Error("expected result to contain orch-go row")
		}
		if !strings.Contains(result, "| kb-cli |") {
			t.Error("expected result to contain kb-cli row")
		}
		if !strings.Contains(result, "| beads |") {
			t.Error("expected result to contain beads row")
		}

		// Should NOT contain sections after Quick Reference
		if strings.Contains(result, "## Core Repos") {
			t.Error("result should stop before Core Repos section")
		}
		if strings.Contains(result, "More detailed info") {
			t.Error("result should not include content after Quick Reference section")
		}
	})

	t.Run("fallback to first lines if no quick reference", func(t *testing.T) {
		content := `# Ecosystem

This is a simple ecosystem file without Quick Reference section.

Here is some info about projects.
Line 1
Line 2
Line 3
`
		result := ExtractQuickReference(content)

		// Should return some content (fallback behavior)
		if result == "" {
			t.Error("expected non-empty result for fallback case")
		}
		if !strings.Contains(result, "simple ecosystem file") {
			t.Error("expected fallback to include content from file")
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		result := ExtractQuickReference("")
		if result != "" {
			t.Error("expected empty string for empty input")
		}
	})

	t.Run("skips initial frontmatter markers", func(t *testing.T) {
		// Note: The fallback logic only skips lines starting with --- or > at the beginning
		// Once it sees content, it includes everything. This is acceptable for fallback.
		content := `---

# Heading

Content here
`
		result := ExtractQuickReference(content)

		// Should skip initial --- marker
		if strings.HasPrefix(result, "---") {
			t.Error("result should not start with frontmatter marker")
		}
		// Should include content
		if !strings.Contains(result, "# Heading") {
			t.Error("result should include content")
		}
	})
}

func TestGenerateEcosystemContext(t *testing.T) {
	// This test verifies the real ecosystem file can be read and parsed
	// Skip if file doesn't exist (CI environments)

	context := GenerateEcosystemContext()
	if context == "" {
		t.Skip("~/.orch/ECOSYSTEM.md not found - skipping integration test")
	}

	// Should contain Quick Reference section
	if !strings.Contains(context, "Quick Reference") {
		t.Error("expected ecosystem context to contain Quick Reference")
	}

	// Should contain known repos
	expectedRepos := []string{"orch-go", "kb-cli", "beads"}
	for _, repo := range expectedRepos {
		if !strings.Contains(context, repo) {
			t.Errorf("expected ecosystem context to mention %s", repo)
		}
	}

	t.Logf("Ecosystem context length: %d chars", len(context))
}

func TestExpandedOrchEcosystemRepos(t *testing.T) {
	// Verify all expected repos are in the expanded list
	expectedRepos := []string{
		"orch-go", "orch-cli", "kb-cli", "orch-knowledge",
		"beads", "kn", "beads-ui-svelte", "glass", "skillc", "agentlog",
	}

	for _, repo := range expectedRepos {
		if !ExpandedOrchEcosystemRepos[repo] {
			t.Errorf("expected %s to be in ExpandedOrchEcosystemRepos", repo)
		}
	}
}
