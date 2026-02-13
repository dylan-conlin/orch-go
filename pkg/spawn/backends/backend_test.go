package backends

import (
	"testing"
)

func TestSelect(t *testing.T) {
	tests := []struct {
		name           string
		inline         bool
		headless       bool
		tmux           bool
		attach         bool
		isOrchestrator bool
		wantBackend    string
	}{
		{
			name:           "inline flag takes precedence",
			inline:         true,
			headless:       true,
			tmux:           true,
			attach:         true,
			isOrchestrator: true,
			wantBackend:    "inline",
		},
		{
			name:           "headless flag takes precedence over tmux",
			inline:         false,
			headless:       true,
			tmux:           true,
			attach:         false,
			isOrchestrator: false,
			wantBackend:    "headless",
		},
		{
			name:           "tmux flag selects tmux backend",
			inline:         false,
			headless:       false,
			tmux:           true,
			attach:         false,
			isOrchestrator: false,
			wantBackend:    "tmux",
		},
		{
			name:           "attach flag selects tmux backend",
			inline:         false,
			headless:       false,
			tmux:           false,
			attach:         true,
			isOrchestrator: false,
			wantBackend:    "tmux",
		},
		{
			name:           "orchestrator defaults to tmux",
			inline:         false,
			headless:       false,
			tmux:           false,
			attach:         false,
			isOrchestrator: true,
			wantBackend:    "tmux",
		},
		{
			name:           "worker defaults to headless",
			inline:         false,
			headless:       false,
			tmux:           false,
			attach:         false,
			isOrchestrator: false,
			wantBackend:    "headless",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := Select(tt.inline, tt.headless, tt.tmux, tt.attach, tt.isOrchestrator)
			if backend.Name() != tt.wantBackend {
				t.Errorf("Select(%v, %v, %v, %v, %v) = %q, want %q",
					tt.inline, tt.headless, tt.tmux, tt.attach, tt.isOrchestrator,
					backend.Name(), tt.wantBackend)
			}
		})
	}
}

func TestFormatSessionTitle(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		beadsID       string
		want          string
	}{
		{
			name:          "with beads ID",
			workspaceName: "og-feat-test-01jan",
			beadsID:       "orch-go-1234",
			want:          "og-feat-test-01jan [orch-go-1234]",
		},
		{
			name:          "without beads ID",
			workspaceName: "og-feat-test-01jan",
			beadsID:       "",
			want:          "og-feat-test-01jan",
		},
		{
			name:          "empty workspace name with beads ID",
			workspaceName: "",
			beadsID:       "orch-go-1234",
			want:          " [orch-go-1234]",
		},
		{
			name:          "empty both",
			workspaceName: "",
			beadsID:       "",
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSessionTitle(tt.workspaceName, tt.beadsID)
			if got != tt.want {
				t.Errorf("FormatSessionTitle(%q, %q) = %q, want %q",
					tt.workspaceName, tt.beadsID, got, tt.want)
			}
		})
	}
}

func TestBackendNames(t *testing.T) {
	tests := []struct {
		backend Backend
		want    string
	}{
		{&InlineBackend{}, "inline"},
		{&HeadlessBackend{}, "headless"},
		{&TmuxBackend{}, "tmux"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.backend.Name(); got != tt.want {
				t.Errorf("backend.Name() = %q, want %q", got, tt.want)
			}
		})
	}
}
