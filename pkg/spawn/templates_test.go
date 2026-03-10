package spawn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSynthesisTemplate(t *testing.T) {
	t.Run("creates template when missing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Ensure template doesn't exist initially
		templatePath := filepath.Join(tempDir, ".orch", "templates", "SYNTHESIS.md")
		if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
			t.Fatal("template should not exist initially")
		}

		// Call EnsureSynthesisTemplate
		if err := EnsureSynthesisTemplate(tempDir); err != nil {
			t.Fatalf("EnsureSynthesisTemplate failed: %v", err)
		}

		// Check template was created
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			t.Error("template should exist after EnsureSynthesisTemplate")
		}

		// Check content
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if !strings.Contains(string(content), "# Session Synthesis") {
			t.Error("template should contain synthesis header")
		}
		if !strings.Contains(string(content), "## TLDR") {
			t.Error("template should contain TLDR section")
		}
		if !strings.Contains(string(content), "## Delta") {
			t.Error("template should contain Delta section")
		}
	})

	t.Run("does not overwrite existing template", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create templates directory and custom template
		templatesDir := filepath.Join(tempDir, ".orch", "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		customContent := "# Custom Synthesis Template\n\nThis is a custom template."
		templatePath := filepath.Join(templatesDir, "SYNTHESIS.md")
		if err := os.WriteFile(templatePath, []byte(customContent), 0644); err != nil {
			t.Fatalf("failed to write custom template: %v", err)
		}

		// Call EnsureSynthesisTemplate
		if err := EnsureSynthesisTemplate(tempDir); err != nil {
			t.Fatalf("EnsureSynthesisTemplate failed: %v", err)
		}

		// Check content was NOT overwritten
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if string(content) != customContent {
			t.Error("existing template should not be overwritten")
		}
	})
}

func TestEnsureFailureReportTemplate(t *testing.T) {
	t.Run("creates template when missing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Ensure template doesn't exist initially
		templatePath := filepath.Join(tempDir, ".orch", "templates", "FAILURE_REPORT.md")
		if _, err := os.Stat(templatePath); !os.IsNotExist(err) {
			t.Fatal("template should not exist initially")
		}

		// Call EnsureFailureReportTemplate
		if err := EnsureFailureReportTemplate(tempDir); err != nil {
			t.Fatalf("EnsureFailureReportTemplate failed: %v", err)
		}

		// Check template was created
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			t.Error("template should exist after EnsureFailureReportTemplate")
		}

		// Check content
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if !strings.Contains(string(content), "# Failure Report") {
			t.Error("template should contain failure report header")
		}
		if !strings.Contains(string(content), "## Failure Summary") {
			t.Error("template should contain Failure Summary section")
		}
		if !strings.Contains(string(content), "## Recovery Recommendations") {
			t.Error("template should contain Recovery Recommendations section")
		}
	})

	t.Run("does not overwrite existing template", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create templates directory and custom template
		templatesDir := filepath.Join(tempDir, ".orch", "templates")
		if err := os.MkdirAll(templatesDir, 0755); err != nil {
			t.Fatalf("failed to create templates dir: %v", err)
		}

		customContent := "# Custom Failure Report Template\n\nThis is a custom template."
		templatePath := filepath.Join(templatesDir, "FAILURE_REPORT.md")
		if err := os.WriteFile(templatePath, []byte(customContent), 0644); err != nil {
			t.Fatalf("failed to write custom template: %v", err)
		}

		// Call EnsureFailureReportTemplate
		if err := EnsureFailureReportTemplate(tempDir); err != nil {
			t.Fatalf("EnsureFailureReportTemplate failed: %v", err)
		}

		// Check content was NOT overwritten
		content, err := os.ReadFile(templatePath)
		if err != nil {
			t.Fatalf("failed to read template: %v", err)
		}

		if string(content) != customContent {
			t.Error("existing template should not be overwritten")
		}
	})
}

func TestWriteFailureReport(t *testing.T) {
	tempDir := t.TempDir()
	workspacePath := filepath.Join(tempDir, "og-test-workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	reportPath, err := WriteFailureReport(
		workspacePath,
		"og-test-workspace",
		"test-123",
		"Out of context",
		"implement test feature",
	)
	if err != nil {
		t.Fatalf("WriteFailureReport failed: %v", err)
	}

	// Check file was created
	expectedPath := filepath.Join(workspacePath, "FAILURE_REPORT.md")
	if reportPath != expectedPath {
		t.Errorf("expected report path %q, got %q", expectedPath, reportPath)
	}

	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("failure report should exist after WriteFailureReport")
	}

	// Check content
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read failure report: %v", err)
	}

	checks := []string{
		"# Failure Report",
		"**Agent:** og-test-workspace",
		"**Issue:** test-123",
		"**Reason:** Out of context",
		"**Task:** implement test feature",
		"## Failure Summary",
		"## Recovery Recommendations",
		"orch spawn {skill}",
		"--issue test-123",
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("failure report should contain %q", check)
		}
	}
}

func TestGenerateFailureReport(t *testing.T) {
	report := generateFailureReport(
		"og-debug-test-21dec",
		"orch-go-abc",
		"Stuck in loop",
		"debug the authentication issue",
	)

	checks := []string{
		"**Agent:** og-debug-test-21dec",
		"**Issue:** orch-go-abc",
		"**Reason:** Stuck in loop",
		"**Task:** debug the authentication issue",
		"**Primary Cause:** Stuck in loop",
		"--issue orch-go-abc",
		"bd show orch-go-abc",
	}

	for _, check := range checks {
		if !strings.Contains(report, check) {
			t.Errorf("generated report should contain %q", check)
		}
	}
}
