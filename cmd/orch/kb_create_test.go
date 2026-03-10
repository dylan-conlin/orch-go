package main

import (
	"os"
	"strings"
	"testing"
)

func TestRunKBCreateModel(t *testing.T) {
	// Create a temp directory to simulate project root
	tmpDir := t.TempDir()

	t.Run("creates model directory structure and model.md from template", func(t *testing.T) {
		// Set up .kb/models/ with TEMPLATE.md
		modelsDir := tmpDir + "/proj1/.kb/models"
		if err := os.MkdirAll(modelsDir, 0755); err != nil {
			t.Fatal(err)
		}
		templateContent := "# Model: {Title}\n\n**Domain:** {System area}\n"
		if err := os.WriteFile(modelsDir+"/TEMPLATE.md", []byte(templateContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := runKBCreateModel("test-model", tmpDir+"/proj1")
		if err != nil {
			t.Fatalf("runKBCreateModel failed: %v", err)
		}

		// Verify directory was created
		modelDir := modelsDir + "/test-model"
		info, err := os.Stat(modelDir)
		if err != nil {
			t.Fatalf("model directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected directory")
		}

		// Verify probes/ subdirectory was created
		probesDir := modelDir + "/probes"
		info, err = os.Stat(probesDir)
		if err != nil {
			t.Fatalf("probes directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected probes directory")
		}

		// Verify model.md was created from template
		modelFile := modelDir + "/model.md"
		content, err := os.ReadFile(modelFile)
		if err != nil {
			t.Fatalf("model.md not created: %v", err)
		}
		if string(content) != templateContent {
			t.Errorf("model.md content doesn't match template:\ngot: %q\nwant: %q", string(content), templateContent)
		}
	})

	t.Run("errors if model already exists", func(t *testing.T) {
		projDir := tmpDir + "/proj2"
		modelsDir := projDir + "/.kb/models"
		if err := os.MkdirAll(modelsDir+"/existing-model/probes", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(modelsDir+"/TEMPLATE.md", []byte("template"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(modelsDir+"/existing-model/model.md", []byte("existing"), 0644); err != nil {
			t.Fatal(err)
		}

		err := runKBCreateModel("existing-model", projDir)
		if err == nil {
			t.Error("expected error for existing model")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("errors if template is missing", func(t *testing.T) {
		projDir := tmpDir + "/proj3"
		modelsDir := projDir + "/.kb/models"
		if err := os.MkdirAll(modelsDir, 0755); err != nil {
			t.Fatal(err)
		}

		err := runKBCreateModel("new-model", projDir)
		if err == nil {
			t.Error("expected error for missing template")
		}
		if !strings.Contains(err.Error(), "TEMPLATE.md") {
			t.Errorf("expected TEMPLATE.md error, got: %v", err)
		}
	})

	t.Run("validates model name format", func(t *testing.T) {
		projDir := tmpDir + "/proj4"
		modelsDir := projDir + "/.kb/models"
		if err := os.MkdirAll(modelsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(modelsDir+"/TEMPLATE.md", []byte("template"), 0644); err != nil {
			t.Fatal(err)
		}

		// Names with spaces should fail
		err := runKBCreateModel("bad name", projDir)
		if err == nil {
			t.Error("expected error for name with spaces")
		}

		// Names with uppercase should fail
		err = runKBCreateModel("BadName", projDir)
		if err == nil {
			t.Error("expected error for uppercase name")
		}

		// Empty name should fail
		err = runKBCreateModel("", projDir)
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
}
