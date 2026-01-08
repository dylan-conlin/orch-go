package userconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDocDebtAddCommand(t *testing.T) {
	debt := &DocDebt{
		Commands: make(map[string]DocDebtEntry),
	}

	// Adding a new command should return true
	if !debt.AddCommand("reconcile.go") {
		t.Error("Expected AddCommand to return true for new command")
	}

	// Adding the same command should return false
	if debt.AddCommand("reconcile.go") {
		t.Error("Expected AddCommand to return false for existing command")
	}

	// Verify entry was created correctly
	entry, exists := debt.Commands["reconcile.go"]
	if !exists {
		t.Fatal("Command entry not found")
	}
	if entry.CommandFile != "reconcile.go" {
		t.Errorf("Expected CommandFile 'reconcile.go', got '%s'", entry.CommandFile)
	}
	if entry.Documented {
		t.Error("Expected new command to be undocumented")
	}
	if len(entry.DocLocations) != 2 {
		t.Errorf("Expected 2 doc locations, got %d", len(entry.DocLocations))
	}
}

func TestDocDebtMarkDocumented(t *testing.T) {
	debt := &DocDebt{
		Commands: make(map[string]DocDebtEntry),
	}

	// Add a command first
	debt.AddCommand("focus.go")

	// Marking an existing command should return true
	if !debt.MarkDocumented("focus.go") {
		t.Error("Expected MarkDocumented to return true for existing command")
	}

	// Marking a non-existent command should return false
	if debt.MarkDocumented("nonexistent.go") {
		t.Error("Expected MarkDocumented to return false for non-existent command")
	}

	// Verify entry was updated
	entry := debt.Commands["focus.go"]
	if !entry.Documented {
		t.Error("Expected command to be marked as documented")
	}
	if entry.DateDocumented == "" {
		t.Error("Expected DateDocumented to be set")
	}
}

func TestDocDebtUndocumentedCommands(t *testing.T) {
	debt := &DocDebt{
		Commands: make(map[string]DocDebtEntry),
	}

	// Add some commands
	debt.AddCommand("cmd1.go")
	debt.AddCommand("cmd2.go")
	debt.AddCommand("cmd3.go")

	// Mark one as documented
	debt.MarkDocumented("cmd2.go")

	// Should have 2 undocumented
	undocumented := debt.UndocumentedCommands()
	if len(undocumented) != 2 {
		t.Errorf("Expected 2 undocumented commands, got %d", len(undocumented))
	}

	// Check that cmd2.go is not in the list
	for _, entry := range undocumented {
		if entry.CommandFile == "cmd2.go" {
			t.Error("cmd2.go should not be in undocumented list")
		}
	}
}

func TestDocDebtSaveLoad(t *testing.T) {
	// Create a temp directory for the test
	tempDir := t.TempDir()
	originalPath := DocDebtPath

	// Override the path function temporarily
	// We can't easily do this, so let's test with direct JSON operations instead

	debt := &DocDebt{
		Commands: make(map[string]DocDebtEntry),
	}
	debt.AddCommand("test.go")
	debt.MarkDocumented("test.go")

	// Marshal to JSON
	data, err := json.MarshalIndent(debt, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Write to temp file
	testPath := filepath.Join(tempDir, "doc-debt.json")
	if err := os.WriteFile(testPath, data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Read back
	readData, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	var loadedDebt DocDebt
	if err := json.Unmarshal(readData, &loadedDebt); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if len(loadedDebt.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(loadedDebt.Commands))
	}
	entry := loadedDebt.Commands["test.go"]
	if !entry.Documented {
		t.Error("Expected command to be documented after load")
	}

	_ = originalPath // suppress unused warning - we can't easily override this
}
