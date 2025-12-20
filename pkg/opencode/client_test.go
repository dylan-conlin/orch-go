package opencode

import (
	"bytes"
	"testing"
)

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantErr  bool
	}{
		{
			name:     "step_start event",
			input:    `{"type":"step_start","step":{"id":"step_123"}}`,
			wantType: "step_start",
			wantErr:  false,
		},
		{
			name:     "text event",
			input:    `{"type":"text","content":"hello"}`,
			wantType: "text",
			wantErr:  false,
		},
		{
			name:     "step_finish event",
			input:    `{"type":"step_finish","step":{"id":"step_123"}}`,
			wantType: "step_finish",
			wantErr:  false,
		},
		{
			name:     "invalid json",
			input:    `not json`,
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseEvent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && event.Type != tt.wantType {
				t.Errorf("ParseEvent() type = %v, want %v", event.Type, tt.wantType)
			}
		})
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name    string
		events  []string
		wantID  string
		wantErr bool
	}{
		{
			name: "sessionID at top level (actual opencode format)",
			events: []string{
				`{"type":"step_start","timestamp":1766199826875,"sessionID":"ses_abc123"}`,
			},
			wantID:  "ses_abc123",
			wantErr: false,
		},
		{
			name: "no sessionID in output",
			events: []string{
				`{"type":"text","content":"hello"}`,
			},
			wantID:  "",
			wantErr: true,
		},
		{
			name: "sessionID in second event",
			events: []string{
				`{"type":"init"}`,
				`{"type":"step_start","sessionID":"ses_xyz789"}`,
			},
			wantID:  "ses_xyz789",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ExtractSessionID(tt.events)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractSessionID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if id != tt.wantID {
				t.Errorf("ExtractSessionID() = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestProcessOutput(t *testing.T) {
	// Use actual opencode format with sessionID at top level
	events := []string{
		`{"type":"step_start","timestamp":1766199826875,"sessionID":"ses_xyz","step":{"id":"step_1"}}`,
		`{"type":"text","sessionID":"ses_xyz","content":"Hello!"}`,
		`{"type":"step_finish","sessionID":"ses_xyz","step":{"id":"step_1"}}`,
	}

	var output bytes.Buffer
	for _, e := range events {
		output.WriteString(e + "\n")
	}

	result, err := ProcessOutput(&output)
	if err != nil {
		t.Fatalf("ProcessOutput() error = %v", err)
	}

	if result.SessionID != "ses_xyz" {
		t.Errorf("SessionID = %v, want ses_xyz", result.SessionID)
	}
	if len(result.Events) != 3 {
		t.Errorf("Events count = %d, want 3", len(result.Events))
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("http://127.0.0.1:4096")
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.ServerURL != "http://127.0.0.1:4096" {
		t.Errorf("ServerURL = %v, want http://127.0.0.1:4096", client.ServerURL)
	}
}

func TestBuildSpawnCommand(t *testing.T) {
	client := NewClient("http://127.0.0.1:4096")
	cmd := client.BuildSpawnCommand("say hello", "test-title")

	expectedArgs := []string{
		"run",
		"--attach", "http://127.0.0.1:4096",
		"--format", "json",
		"--title", "test-title",
		"say hello",
	}

	if len(cmd.Args) < len(expectedArgs)+1 { // +1 for command name
		t.Errorf("BuildSpawnCommand() args length = %v, want at least %v", len(cmd.Args), len(expectedArgs)+1)
	}
}

func TestBuildAskCommand(t *testing.T) {
	client := NewClient("http://127.0.0.1:4096")
	cmd := client.BuildAskCommand("ses_123", "what did you do?")

	expectedArgs := []string{
		"run",
		"--attach", "http://127.0.0.1:4096",
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
