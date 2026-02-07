package opencode

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:4096")
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.ServerURL != "http://localhost:4096" {
		t.Errorf("ServerURL = %v, want http://localhost:4096", client.ServerURL)
	}
}

func TestBuildSpawnCommand(t *testing.T) {
	client := NewClient("http://localhost:4096")
	cmd := client.BuildSpawnCommand("say hello", "test-title", "", "")

	expectedArgs := []string{
		"run",
		"--attach", "http://localhost:4096",
		"--format", "json",
		"--title", "test-title",
		"say hello",
	}

	if len(cmd.Args) < len(expectedArgs)+1 { // +1 for command name
		t.Errorf("BuildSpawnCommand() args length = %v, want at least %v", len(cmd.Args), len(expectedArgs)+1)
	}
}

func TestBuildSpawnCommandWithModel(t *testing.T) {
	client := NewClient("http://localhost:4096")
	cmd := client.BuildSpawnCommand("say hello", "test-title", "anthropic/claude-opus-4", "")

	expectedArgs := []string{
		"run",
		"--attach", "http://localhost:4096",
		"--format", "json",
		"--model", "anthropic/claude-opus-4",
		"--title", "test-title",
		"say hello",
	}

	// Check that all expected args are present
	found := 0
	for _, expected := range expectedArgs {
		for _, arg := range cmd.Args {
			if arg == expected {
				found++
				break
			}
		}
	}

	if found < len(expectedArgs) {
		t.Errorf("BuildSpawnCommand() missing expected args, found %v of %v. Args: %v", found, len(expectedArgs), cmd.Args)
	}

	// Verify --model flag is included
	hasModel := false
	for i, arg := range cmd.Args {
		if arg == "--model" && i+1 < len(cmd.Args) && cmd.Args[i+1] == "anthropic/claude-opus-4" {
			hasModel = true
			break
		}
	}
	if !hasModel {
		t.Errorf("BuildSpawnCommand() should include --model flag when model is provided. Args: %v", cmd.Args)
	}
}

func TestBuildSpawnCommandWithoutModel(t *testing.T) {
	client := NewClient("http://localhost:4096")
	cmd := client.BuildSpawnCommand("say hello", "test-title", "", "")

	// Verify --model flag is NOT included when model is empty
	for i, arg := range cmd.Args {
		if arg == "--model" {
			t.Errorf("BuildSpawnCommand() should not include --model flag when model is empty. Found at index %d. Args: %v", i, cmd.Args)
		}
	}
}

func TestBuildAskCommand(t *testing.T) {
	client := NewClient("http://localhost:4096")
	cmd := client.BuildAskCommand("ses_123", "what did you do?")

	expectedArgs := []string{
		"run",
		"--attach", "http://localhost:4096",
		"--session", "ses_123",
		"--format", "json",
		"what did you do?",
	}

	found := 0
	for _, expected := range expectedArgs {
		for _, arg := range cmd.Args {
			if arg == expected {
				found++
				break
			}
		}
	}

	if found < len(expectedArgs) {
		t.Errorf("BuildAskCommand() missing expected args, found %v of %v", found, len(expectedArgs))
	}
}

// TestParseModelSpec tests the parseModelSpec helper function.
func TestParseModelSpec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		provider string
		modelID  string
	}{
		{
			name:     "valid provider/modelID format",
			input:    "google/gemini-2.5-flash",
			wantNil:  false,
			provider: "google",
			modelID:  "gemini-2.5-flash",
		},
		{
			name:     "valid anthropic model",
			input:    "anthropic/claude-opus-4-5-20251101",
			wantNil:  false,
			provider: "anthropic",
			modelID:  "claude-opus-4-5-20251101",
		},
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "no slash",
			input:   "claude-opus-4",
			wantNil: true,
		},
		{
			name:    "empty provider",
			input:   "/modelID",
			wantNil: true,
		},
		{
			name:    "empty modelID",
			input:   "provider/",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseModelSpec(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("parseModelSpec(%q) = %v, want nil", tt.input, result)
				}
				return
			}
			if result == nil {
				t.Fatalf("parseModelSpec(%q) = nil, want non-nil", tt.input)
			}
			if result["providerID"] != tt.provider {
				t.Errorf("providerID = %v, want %v", result["providerID"], tt.provider)
			}
			if result["modelID"] != tt.modelID {
				t.Errorf("modelID = %v, want %v", result["modelID"], tt.modelID)
			}
		})
	}
}
