package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

func TestEnsureSessionTitleUpdatesSession(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := opencode.NewClient(server.URL)
	ensureSessionTitle(client, "ses_abc123", "og-feat-fix-title [orch-go-21200]")

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH request, got %q", gotMethod)
	}
	if gotPath != "/api/sessions/ses_abc123" {
		t.Fatalf("expected path /api/sessions/ses_abc123, got %q", gotPath)
	}
	if !strings.Contains(gotBody, `"title":"og-feat-fix-title [orch-go-21200]"`) {
		t.Fatalf("expected title payload in request body, got %q", gotBody)
	}
}

func TestEnsureSessionTitleSkipsEmptyInputs(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := opencode.NewClient(server.URL)
	ensureSessionTitle(client, "", "some title")
	ensureSessionTitle(client, "ses_abc123", "")

	if requestCount != 0 {
		t.Fatalf("expected 0 requests for empty inputs, got %d", requestCount)
	}
}

func TestValidateModeModelCombo(t *testing.T) {
	tests := []struct {
		name          string
		backend       string
		modelSpec     model.ModelSpec
		expectWarning bool
		warningText   string
	}{
		{
			name:          "valid: opencode + sonnet",
			backend:       "opencode",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"},
			expectWarning: false,
		},
		{
			name:          "valid: claude + opus",
			backend:       "claude",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-6"},
			expectWarning: false,
		},
		{
			name:          "invalid: opencode + opus",
			backend:       "opencode",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-opus-4-6"},
			expectWarning: true,
			warningText:   "opencode backend with opus model may fail",
		},
		{
			name:          "valid: claude + sonnet (non-optimal but works)",
			backend:       "claude",
			modelSpec:     model.ModelSpec{Provider: "anthropic", ModelID: "claude-sonnet-4-5-20250929"},
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModeModelCombo(tt.backend, tt.modelSpec)

			if tt.expectWarning {
				if err == nil {
					t.Errorf("expected warning but got nil")
				} else if !strings.Contains(err.Error(), tt.warningText) {
					t.Errorf("expected warning containing %q, got %q", tt.warningText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no warning but got: %v", err)
				}
			}
		})
	}
}

func TestFlashModelBlocking(t *testing.T) {
	// Test that flash models are properly identified
	flashModels := []string{
		"flash",
		"flash-2.5",
		"flash3",
		"google/gemini-2.5-flash",
		"google/gemini-3-flash-preview",
	}

	for _, modelStr := range flashModels {
		t.Run(modelStr, func(t *testing.T) {
			resolved := model.Resolve(modelStr)

			// Check that it's a Google/flash model
			if resolved.Provider != "google" {
				t.Errorf("expected provider 'google', got %q", resolved.Provider)
			}

			if !strings.Contains(strings.ToLower(resolved.ModelID), "flash") {
				t.Errorf("expected model ID to contain 'flash', got %q", resolved.ModelID)
			}
		})
	}
}

func TestModelAutoSelection(t *testing.T) {
	tests := []struct {
		name            string
		modelFlag       string
		opusFlag        bool
		expectedBackend string
	}{
		{
			name:            "opus flag forces claude",
			modelFlag:       "",
			opusFlag:        true,
			expectedBackend: "claude",
		},
		{
			name:            "opus model auto-selects claude",
			modelFlag:       "opus",
			opusFlag:        false,
			expectedBackend: "claude",
		},
		{
			name:            "sonnet model uses opencode",
			modelFlag:       "sonnet",
			opusFlag:        false,
			expectedBackend: "opencode",
		},
		{
			name:            "no flags defaults to claude",
			modelFlag:       "",
			opusFlag:        false,
			expectedBackend: "claude",
		},
		{
			name:            "opus alias auto-selects claude",
			modelFlag:       "opus",
			opusFlag:        false,
			expectedBackend: "claude",
		},
		{
			name:            "opus-4.5 legacy alias auto-selects claude",
			modelFlag:       "opus-4.5",
			opusFlag:        false,
			expectedBackend: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the auto-selection logic from runSpawnWithSkillInternal
			backend := "claude"

			if tt.opusFlag {
				backend = "claude"
			} else if tt.modelFlag != "" {
				modelLower := strings.ToLower(tt.modelFlag)
				if modelLower == "opus" || strings.Contains(modelLower, "opus") {
					backend = "claude"
				} else if modelLower == "sonnet" || strings.Contains(modelLower, "sonnet") {
					backend = "opencode"
				}
			}

			if backend != tt.expectedBackend {
				t.Errorf("expected backend %q, got %q", tt.expectedBackend, backend)
			}
		})
	}
}

// TestIsCriticalInfrastructureWork tests the narrowed infrastructure detection.
// Only CRITICAL infrastructure (server lifecycle) should trigger, not general orch work.
func TestIsCriticalInfrastructureWork(t *testing.T) {
	tests := []struct {
		name    string
		task    string
		beadsID string
		want    bool
	}{
		// CRITICAL infrastructure - should trigger
		{
			name:    "opencode server keyword",
			task:    "fix opencode server crash",
			beadsID: "",
			want:    true, // matches "opencode server"
		},
		{
			name:    "serve.go in task",
			task:    "update cmd/orch/serve.go logging",
			beadsID: "",
			want:    true, // matches "serve.go"
		},
		{
			name:    "pkg/opencode path",
			task:    "refactor pkg/opencode/client.go",
			beadsID: "",
			want:    true, // matches "pkg/opencode"
		},
		{
			name:    "case insensitive opencode server",
			task:    "Fix OpenCode Server Bug",
			beadsID: "",
			want:    true, // matches "opencode server"
		},
		{
			name:    "server restart",
			task:    "implement server restart handling",
			beadsID: "",
			want:    true, // matches "server restart"
		},
		{
			name:    "opencode api work",
			task:    "update opencode api endpoints",
			beadsID: "",
			want:    true, // matches "opencode api"
		},

		// NON-CRITICAL - should NOT trigger (narrowed scope)
		{
			name:    "spawn logic (not critical)",
			task:    "update spawn logic to handle errors",
			beadsID: "",
			want:    false, // spawn logic doesn't restart server
		},
		{
			name:    "dashboard (not critical)",
			task:    "fix dashboard agent count",
			beadsID: "",
			want:    false, // dashboard is frontend, not server
		},
		{
			name:    "pkg/spawn (not critical)",
			task:    "refactor pkg/spawn/context.go",
			beadsID: "",
			want:    false, // spawn context doesn't restart server
		},
		{
			name:    "skillc (not critical)",
			task:    "fix skillc compilation issue",
			beadsID: "",
			want:    false, // skill compiler is separate tool
		},
		{
			name:    "orchestration infrastructure phrase (not critical)",
			task:    "improve orchestration infrastructure",
			beadsID: "",
			want:    false, // too generic to be critical
		},
		{
			name:    "agents.ts (not critical)",
			task:    "update agents.ts store logic",
			beadsID: "",
			want:    false, // frontend component
		},
		{
			name:    "non-infrastructure task",
			task:    "add user authentication feature",
			beadsID: "",
			want:    false,
		},
		{
			name:    "regular feature work",
			task:    "implement user profile page",
			beadsID: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCriticalInfrastructureWork(tt.task, tt.beadsID)
			if got != tt.want {
				t.Errorf("isCriticalInfrastructureWork(%q, %q) = %v, want %v", tt.task, tt.beadsID, got, tt.want)
			}
		})
	}
}

func TestRequiresResourceLifecycleAudit(t *testing.T) {
	tests := []struct {
		name    string
		task    string
		beadsID string
		want    bool
	}{
		{
			name:    "pkg/daemon path triggers audit",
			task:    "fix leak in pkg/daemon/scheduler.go",
			beadsID: "",
			want:    true,
		},
		{
			name:    "pkg/spawn path triggers audit",
			task:    "update pkg/spawn/context.go template",
			beadsID: "",
			want:    true,
		},
		{
			name:    "cmd/orch/serve wildcard path triggers audit",
			task:    "refactor cmd/orch/serve_status.go startup flow",
			beadsID: "",
			want:    true,
		},
		{
			name:    "exec.Command trigger",
			task:    "audit exec.Command usage in spawn pipeline",
			beadsID: "",
			want:    true,
		},
		{
			name:    "goroutine trigger",
			task:    "bound goroutine lifecycle in cleanup loop",
			beadsID: "",
			want:    true,
		},
		{
			name:    "go func trigger",
			task:    "replace go func( with context-aware worker pool",
			beadsID: "",
			want:    true,
		},
		{
			name:    "non-infrastructure task does not trigger",
			task:    "add user profile endpoint",
			beadsID: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiresResourceLifecycleAudit(tt.task, tt.beadsID)
			if got != tt.want {
				t.Errorf("requiresResourceLifecycleAudit(%q, %q) = %v, want %v", tt.task, tt.beadsID, got, tt.want)
			}
		})
	}
}

func TestNoTrackWaitHints(t *testing.T) {
	tests := []struct {
		name         string
		beadsID      string
		noTrack      bool
		wantHandle   string
		wantWaitCmd  string
		wantResolved bool
	}{
		{
			name:         "tracked spawn returns no hints",
			beadsID:      "orch-go-1234",
			noTrack:      false,
			wantHandle:   "",
			wantWaitCmd:  "",
			wantResolved: false,
		},
		{
			name:         "untracked id returns wait command and display alias",
			beadsID:      "orch-go-untracked-1768090360",
			noTrack:      true,
			wantHandle:   "orch-go-untracked-1768090360 (status: untracked-Jan10-1612)",
			wantWaitCmd:  "orch wait orch-go-untracked-1768090360",
			wantResolved: true,
		},
		{
			name:         "empty beads id returns no hints",
			beadsID:      "",
			noTrack:      true,
			wantHandle:   "",
			wantWaitCmd:  "",
			wantResolved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, waitCmd, ok := noTrackWaitHints(tt.beadsID, tt.noTrack)
			if ok != tt.wantResolved {
				t.Fatalf("noTrackWaitHints() ok = %v, want %v", ok, tt.wantResolved)
			}
			if handle != tt.wantHandle {
				t.Fatalf("noTrackWaitHints() handle = %q, want %q", handle, tt.wantHandle)
			}
			if waitCmd != tt.wantWaitCmd {
				t.Fatalf("noTrackWaitHints() waitCmd = %q, want %q", waitCmd, tt.wantWaitCmd)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no ANSI codes",
			input: "Error: Session not found",
			want:  "Error: Session not found",
		},
		{
			name:  "red bold error from opencode",
			input: "\x1b[91m\x1b[1mError: \x1b[0mSession not found",
			want:  "Error: Session not found",
		},
		{
			name:  "various colors",
			input: "\x1b[32mGreen\x1b[0m \x1b[33mYellow\x1b[0m",
			want:  "Green Yellow",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only ANSI codes",
			input: "\x1b[0m\x1b[1m\x1b[91m",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
