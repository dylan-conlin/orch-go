package opencode

import (
	"bytes"
	"strings"
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

func TestExtractSessionIDFromReader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{
			name:    "sessionID in first event",
			input:   `{"type":"step_start","sessionID":"ses_abc123"}` + "\n" + `{"type":"text","content":"hello"}` + "\n",
			wantID:  "ses_abc123",
			wantErr: false,
		},
		{
			name:    "sessionID in second event",
			input:   `{"type":"init"}` + "\n" + `{"type":"step_start","sessionID":"ses_xyz789"}` + "\n",
			wantID:  "ses_xyz789",
			wantErr: false,
		},
		{
			name:    "no sessionID in output",
			input:   `{"type":"text","content":"hello"}` + "\n",
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "mixed valid and invalid lines",
			input:   "invalid line\n" + `{"type":"init"}` + "\n" + `{"type":"step_start","sessionID":"ses_mixed"}` + "\n",
			wantID:  "ses_mixed",
			wantErr: false,
		},
		// Tests for npm warning leaking into stdout (baseline-browser-mapping issue)
		{
			name:    "npm warning prepended to JSON without newline",
			input:   `[baseline-browser-mapping] The data in this module is over two months old{"type":"step_start","sessionID":"ses_warn123"}` + "\n",
			wantID:  "ses_warn123",
			wantErr: false,
		},
		{
			name:    "npm warning on separate line then JSON",
			input:   "[baseline-browser-mapping] The data in this module is over two months old\n" + `{"type":"step_start","sessionID":"ses_warn456"}` + "\n",
			wantID:  "ses_warn456",
			wantErr: false,
		},
		{
			name:    "multiple warnings prepended to JSON",
			input:   `[warn] some warning[baseline-browser-mapping] old data{"type":"step_start","sessionID":"ses_multi789"}` + "\n",
			wantID:  "ses_multi789",
			wantErr: false,
		},
		{
			name:    "warning with brackets in JSON",
			input:   `[baseline-browser-mapping] warning{"type":"step_start","sessionID":"ses_bracket","data":{"array":[1,2,3]}}` + "\n",
			wantID:  "ses_bracket",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewBufferString(tt.input)
			id, err := ExtractSessionIDFromReader(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractSessionIDFromReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if id != tt.wantID {
				t.Errorf("ExtractSessionIDFromReader() = %v, want %v", id, tt.wantID)
			}
		})
	}
}

// TestFindSessionIDInLine tests the findSessionIDInLine helper function.
func TestFindSessionIDInLine(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantID string
	}{
		{
			name:   "pure JSON line",
			input:  `{"type":"step_start","sessionID":"ses_pure123"}`,
			wantID: "ses_pure123",
		},
		{
			name:   "npm warning prepended to JSON",
			input:  `[baseline-browser-mapping] The data in this module is over two months old{"type":"step_start","sessionID":"ses_npm456"}`,
			wantID: "ses_npm456",
		},
		{
			name:   "multiple warnings prepended",
			input:  `[warn] msg1[info] msg2{"type":"event","sessionID":"ses_multi"}`,
			wantID: "ses_multi",
		},
		{
			name:   "no sessionID in JSON",
			input:  `{"type":"init","data":"test"}`,
			wantID: "",
		},
		{
			name:   "warning only, no JSON",
			input:  `[baseline-browser-mapping] The data in this module is over two months old`,
			wantID: "",
		},
		{
			name:   "invalid JSON after warning",
			input:  `[warn] message{not valid json}`,
			wantID: "",
		},
		{
			name:   "empty line",
			input:  "",
			wantID: "",
		},
		{
			name:   "JSON with nested braces",
			input:  `[warn] test{"type":"event","sessionID":"ses_nested","config":{"key":"value"}}`,
			wantID: "ses_nested",
		},
		{
			name:   "JSON with array containing braces",
			input:  `[warn]{"type":"test","sessionID":"ses_array","items":[{"a":1},{"b":2}]}`,
			wantID: "ses_array",
		},
		{
			name:   "whitespace before JSON",
			input:  `   {"type":"event","sessionID":"ses_space"}`,
			wantID: "ses_space",
		},
		{
			name:   "warning with timestamp",
			input:  `2026-01-26T10:30:00Z [baseline-browser-mapping] old data{"type":"step_start","sessionID":"ses_ts"}`,
			wantID: "ses_ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSessionIDInLine(tt.input)
			if got != tt.wantID {
				t.Errorf("findSessionIDInLine(%q) = %q, want %q", tt.input, got, tt.wantID)
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

// TestProcessOutputWithStreaming tests ProcessOutputWithStreaming extracts text content.
func TestProcessOutputWithStreaming(t *testing.T) {
	events := []string{
		`{"type":"step_start","timestamp":1766199826875,"sessionID":"ses_xyz","step":{"id":"step_1"}}`,
		`{"type":"text","sessionID":"ses_xyz","content":"Hello, "}`,
		`{"type":"text","sessionID":"ses_xyz","content":"world!"}`,
		`{"type":"step_finish","sessionID":"ses_xyz","step":{"id":"step_1"}}`,
	}

	var output bytes.Buffer
	for _, e := range events {
		output.WriteString(e + "\n")
	}

	var streamedContent bytes.Buffer
	result, err := ProcessOutputWithStreaming(&output, &streamedContent)
	if err != nil {
		t.Fatalf("ProcessOutputWithStreaming() error = %v", err)
	}

	if result.SessionID != "ses_xyz" {
		t.Errorf("SessionID = %v, want ses_xyz", result.SessionID)
	}
	if len(result.Events) != 4 {
		t.Errorf("Events count = %d, want 4", len(result.Events))
	}

	// Verify streamed content contains the text
	streamed := streamedContent.String()
	if !strings.Contains(streamed, "Hello, ") {
		t.Errorf("Streamed content missing 'Hello, ', got: %s", streamed)
	}
	if !strings.Contains(streamed, "world!") {
		t.Errorf("Streamed content missing 'world!', got: %s", streamed)
	}
}

// TestProcessOutputWithStreamingEmpty tests streaming with no text events.
func TestProcessOutputWithStreamingEmpty(t *testing.T) {
	events := []string{
		`{"type":"step_start","sessionID":"ses_xyz","step":{"id":"step_1"}}`,
		`{"type":"step_finish","sessionID":"ses_xyz","step":{"id":"step_1"}}`,
	}

	var output bytes.Buffer
	for _, e := range events {
		output.WriteString(e + "\n")
	}

	var streamedContent bytes.Buffer
	result, err := ProcessOutputWithStreaming(&output, &streamedContent)
	if err != nil {
		t.Fatalf("ProcessOutputWithStreaming() error = %v", err)
	}

	if result.SessionID != "ses_xyz" {
		t.Errorf("SessionID = %v, want ses_xyz", result.SessionID)
	}
	if len(result.Events) != 2 {
		t.Errorf("Events count = %d, want 2", len(result.Events))
	}

	// Streamed content should be empty (no text events)
	if streamedContent.String() != "" {
		t.Errorf("Expected empty streamed content, got: %s", streamedContent.String())
	}
}
