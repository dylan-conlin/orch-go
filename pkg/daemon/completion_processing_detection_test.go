package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

func TestSessionHasSuccessfulGitCommit(t *testing.T) {
	tests := []struct {
		name     string
		messages []opencode.Message
		want     bool
	}{
		{
			name: "successful git commit command",
			messages: []opencode.Message{{
				Parts: []opencode.MessagePart{{
					CallID: "call-1",
					Tool:   "bash",
					State: &opencode.ToolState{
						Status: "completed",
						Input: map[string]interface{}{
							"command": "git add . && git commit -m \"feat: test\"",
						},
						Metadata: map[string]interface{}{"exit_code": float64(0)},
					},
				}},
			}},
			want: true,
		},
		{
			name: "failed git commit command",
			messages: []opencode.Message{{
				Parts: []opencode.MessagePart{{
					Tool: "bash",
					State: &opencode.ToolState{
						Status: "failed",
						Input:  map[string]interface{}{"command": "git commit -m \"feat: test\""},
					},
				}},
			}},
			want: false,
		},
		{
			name: "nothing to commit output",
			messages: []opencode.Message{{
				Parts: []opencode.MessagePart{{
					Tool: "bash",
					State: &opencode.ToolState{
						Status: "completed",
						Input:  map[string]interface{}{"command": "git commit -m \"feat: test\""},
						Output: "On branch main\nnothing to commit, working tree clean",
					},
				}},
			}},
			want: false,
		},
		{
			name: "non git command",
			messages: []opencode.Message{{
				Parts: []opencode.MessagePart{{
					Tool: "bash",
					State: &opencode.ToolState{
						Status: "completed",
						Input:  map[string]interface{}{"command": "go test ./..."},
					},
				}},
			}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sessionHasSuccessfulGitCommit(tt.messages)
			if got != tt.want {
				t.Errorf("sessionHasSuccessfulGitCommit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseExitCode(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		wantCode int
		wantOK   bool
	}{
		{name: "missing metadata", metadata: map[string]interface{}{}, wantCode: 0, wantOK: false},
		{name: "int", metadata: map[string]interface{}{"exit_code": 2}, wantCode: 2, wantOK: true},
		{name: "float", metadata: map[string]interface{}{"exit_code": float64(3)}, wantCode: 3, wantOK: true},
		{name: "string", metadata: map[string]interface{}{"exit_code": "4"}, wantCode: 4, wantOK: true},
		{name: "camel case key", metadata: map[string]interface{}{"exitCode": float64(5)}, wantCode: 5, wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotOK := parseExitCode(tt.metadata)
			if gotCode != tt.wantCode || gotOK != tt.wantOK {
				t.Errorf("parseExitCode() = (%d, %v), want (%d, %v)", gotCode, gotOK, tt.wantCode, tt.wantOK)
			}
		})
	}
}

func TestIdleCompletionDetectorSessionSignal(t *testing.T) {
	now := time.Now()
	d := &idleCompletionDetector{
		now: now,
		sessionsByID: map[string]opencode.Session{
			"ses-workspace": {
				ID: "ses-workspace",
				Time: opencode.SessionTime{
					Updated: now.Add(-20 * time.Minute).UnixMilli(),
				},
			},
			"ses-index": {
				ID: "ses-index",
				Time: opencode.SessionTime{
					Updated: now.Add(-12 * time.Minute).UnixMilli(),
				},
			},
		},
		sessionByID: map[string]string{
			"orch-go-idx1": "ses-index",
		},
	}

	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".session_id"), []byte("ses-workspace\n"), 0o644); err != nil {
		t.Fatalf("failed to write session id: %v", err)
	}

	t.Run("resolves from workspace session id", func(t *testing.T) {
		signal, ok := d.sessionSignal("orch-go-abc1", workspace)
		if !ok {
			t.Fatalf("expected signal from workspace session id")
		}
		if signal.SessionID != "ses-workspace" {
			t.Fatalf("SessionID = %q, want ses-workspace", signal.SessionID)
		}
		if signal.IdleDuration < 19*time.Minute {
			t.Fatalf("IdleDuration = %v, expected >= 19m", signal.IdleDuration)
		}
	})

	t.Run("falls back to beads index", func(t *testing.T) {
		signal, ok := d.sessionSignal("orch-go-idx1", "")
		if !ok {
			t.Fatalf("expected signal from beads index")
		}
		if signal.SessionID != "ses-index" {
			t.Fatalf("SessionID = %q, want ses-index", signal.SessionID)
		}
	})

	t.Run("returns false when missing", func(t *testing.T) {
		if _, ok := d.sessionSignal("orch-go-missing", ""); ok {
			t.Fatalf("expected missing session to return false")
		}
	})
}
