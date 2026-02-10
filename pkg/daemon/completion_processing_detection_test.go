package daemon

import (
	"net/http"
	"net/http/httptest"
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

func TestNormalizeIdleCompletionThreshold_Default(t *testing.T) {
	got := normalizeIdleCompletionThreshold(0)
	if got != 15*time.Minute {
		t.Errorf("normalizeIdleCompletionThreshold(0) = %v, want 15m", got)
	}
}

func TestDeleteCompletedAgentSession(t *testing.T) {
	deleted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/session/ses_abc" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := deleteCompletedAgentSession(CompletedAgent{SessionID: "ses_abc"}, srv.URL)
	if err != nil {
		t.Fatalf("deleteCompletedAgentSession() error = %v", err)
	}
	if !deleted {
		t.Fatal("expected session to be deleted")
	}
}

func TestDeleteCompletedAgentSession_IgnoreNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := deleteCompletedAgentSession(CompletedAgent{SessionID: "ses_missing"}, srv.URL)
	if err != nil {
		t.Fatalf("deleteCompletedAgentSession() should ignore not found, got: %v", err)
	}
}
