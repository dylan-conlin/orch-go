package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// mockProcessChecker is a test double for PaneProcessChecker.
// aliveWindows maps window IDs to whether their process is alive.
type mockProcessChecker struct {
	aliveWindows map[string]bool
}

func (m *mockProcessChecker) HasActiveProcess(windowID string) bool {
	return m.aliveWindows[windowID]
}

// noProcessChecker returns a mock where all processes are dead (idle shell).
func noProcessChecker() *mockProcessChecker {
	return &mockProcessChecker{aliveWindows: map[string]bool{}}
}

func TestClassifyTmuxWindows(t *testing.T) {
	tests := []struct {
		name           string
		windows        []tmux.WindowInfo
		activeBeadsIDs map[string]bool
		openIssues     map[string]*verify.Issue
		processChecker PaneProcessChecker
		wantStale      int
		wantProtected  int
		wantStaleIDs   []string
	}{
		{
			name: "window with active OpenCode session is not stale",
			windows: []tmux.WindowInfo{
				{Name: "🔧 debug [orch-go-1001]", ID: "@1"},
			},
			activeBeadsIDs: map[string]bool{"orch-go-1001": true},
			openIssues:     map[string]*verify.Issue{},
			processChecker: noProcessChecker(),
			wantStale:      0,
			wantProtected:  0,
		},
		{
			name: "window with open beads issue but no OpenCode session is protected",
			windows: []tmux.WindowInfo{
				{Name: "🔧 debug [orch-go-1002]", ID: "@2"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues: map[string]*verify.Issue{
				"orch-go-1002": {ID: "orch-go-1002", Status: "in_progress"},
			},
			processChecker: noProcessChecker(),
			wantStale:      0,
			wantProtected:  1,
		},
		{
			name: "window with closed beads issue and no OpenCode session and dead process is stale",
			windows: []tmux.WindowInfo{
				{Name: "🔧 debug [orch-go-1003]", ID: "@3"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
			processChecker: noProcessChecker(),
			wantStale:      1,
			wantProtected:  0,
			wantStaleIDs:   []string{"orch-go-1003"},
		},
		{
			name: "mixed: one protected by beads, one stale (dead process)",
			windows: []tmux.WindowInfo{
				{Name: "🔧 debug [orch-go-1004]", ID: "@4"},
				{Name: "🔍 inv [orch-go-1005]", ID: "@5"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues: map[string]*verify.Issue{
				"orch-go-1004": {ID: "orch-go-1004", Status: "open"},
			},
			processChecker: noProcessChecker(),
			wantStale:      1,
			wantProtected:  1,
			wantStaleIDs:   []string{"orch-go-1005"},
		},
		{
			name: "servers and zsh windows are always skipped",
			windows: []tmux.WindowInfo{
				{Name: "servers", ID: "@10"},
				{Name: "zsh", ID: "@11"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
			processChecker: noProcessChecker(),
			wantStale:      0,
			wantProtected:  0,
		},
		{
			name: "window without beads ID is skipped",
			windows: []tmux.WindowInfo{
				{Name: "random-window", ID: "@20"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
			processChecker: noProcessChecker(),
			wantStale:      0,
			wantProtected:  0,
		},
		{
			name: "window with both OpenCode session and open beads is not stale",
			windows: []tmux.WindowInfo{
				{Name: "🔧 debug [orch-go-1006]", ID: "@6"},
			},
			activeBeadsIDs: map[string]bool{"orch-go-1006": true},
			openIssues: map[string]*verify.Issue{
				"orch-go-1006": {ID: "orch-go-1006", Status: "in_progress"},
			},
			processChecker: noProcessChecker(),
			wantStale:      0,
			wantProtected:  0, // Not counted as protected because OpenCode check passes first
		},
		{
			name: "daemon-spawned Claude CLI agent scenario: tmux window, no OpenCode session, open beads issue",
			windows: []tmux.WindowInfo{
				{Name: "🐛 systematic-debugging [orch-go-1221]", ID: "@42"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues: map[string]*verify.Issue{
				"orch-go-1221": {ID: "orch-go-1221", Status: "in_progress"},
			},
			processChecker: noProcessChecker(),
			wantStale:      0,
			wantProtected:  1,
		},
		// === NEW: Process liveness tests ===
		{
			name: "active process protects window even with no OpenCode session and no open beads issue",
			windows: []tmux.WindowInfo{
				{Name: "🐛 pw-work-probe [pw-1234]", ID: "@647"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
			processChecker: &mockProcessChecker{aliveWindows: map[string]bool{"@647": true}},
			wantStale:      0,
			wantProtected:  1,
		},
		{
			name: "dead process with no OpenCode session and no open beads issue is stale",
			windows: []tmux.WindowInfo{
				{Name: "🐛 pw-work-probe [pw-1234]", ID: "@647"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
			processChecker: &mockProcessChecker{aliveWindows: map[string]bool{"@647": false}},
			wantStale:      1,
			wantProtected:  0,
			wantStaleIDs:   []string{"pw-1234"},
		},
		{
			name: "nil process checker skips process check (backward compat)",
			windows: []tmux.WindowInfo{
				{Name: "🐛 debug [orch-go-1099]", ID: "@99"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
			processChecker: nil,
			wantStale:      1,
			wantProtected:  0,
			wantStaleIDs:   []string{"orch-go-1099"},
		},
		{
			name: "mixed: one alive process, one dead process, no sessions or beads",
			windows: []tmux.WindowInfo{
				{Name: "🔬 inv-alive [orch-go-2001]", ID: "@100"},
				{Name: "🔬 inv-dead [orch-go-2002]", ID: "@101"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
			processChecker: &mockProcessChecker{aliveWindows: map[string]bool{
				"@100": true,
				"@101": false,
			}},
			wantStale:     1,
			wantProtected: 1,
			wantStaleIDs:  []string{"orch-go-2002"},
		},
		{
			name: "OpenCode server restart scenario: 0 sessions, active process protects window",
			windows: []tmux.WindowInfo{
				{Name: "🐛 og-debug-fix [orch-go-3001]", ID: "@200"},
				{Name: "🏗️ og-feat-impl [orch-go-3002]", ID: "@201"},
				{Name: "🔬 og-inv-done [orch-go-3003]", ID: "@202"},
			},
			activeBeadsIDs: map[string]bool{}, // 0 sessions after restart
			openIssues: map[string]*verify.Issue{
				"orch-go-3001": {ID: "orch-go-3001", Status: "in_progress"},
			},
			processChecker: &mockProcessChecker{aliveWindows: map[string]bool{
				"@200": true,  // also protected by beads
				"@201": true,  // protected ONLY by process check
				"@202": false, // truly stale
			}},
			wantStale:     1,
			wantProtected: 2, // 3001 by beads, 3002 by process
			wantStaleIDs:  []string{"orch-go-3003"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stale, protected := classifyTmuxWindows(tt.windows, "workers-orch-go", tt.activeBeadsIDs, tt.openIssues, tt.processChecker)

			if len(stale) != tt.wantStale {
				t.Errorf("got %d stale windows, want %d", len(stale), tt.wantStale)
			}

			if protected != tt.wantProtected {
				t.Errorf("got %d protected windows, want %d", protected, tt.wantProtected)
			}

			if tt.wantStaleIDs != nil {
				for i, want := range tt.wantStaleIDs {
					if i >= len(stale) {
						t.Errorf("missing stale window at index %d, want beadsID=%s", i, want)
						continue
					}
					if stale[i].beadsID != want {
						t.Errorf("stale[%d].beadsID = %s, want %s", i, stale[i].beadsID, want)
					}
				}
			}
		})
	}
}

func TestIdleShellCommands(t *testing.T) {
	// Verify that common shells are recognized as idle
	shells := []string{"zsh", "bash", "sh", "fish", "-zsh", "-bash", "-sh", "login"}
	for _, s := range shells {
		if !idleShellCommands[s] {
			t.Errorf("expected %q to be an idle shell command", s)
		}
	}

	// Verify that agent processes are NOT idle
	agents := []string{"claude", "opencode", "node", "python", "go"}
	for _, a := range agents {
		if idleShellCommands[a] {
			t.Errorf("expected %q to NOT be an idle shell command", a)
		}
	}
}
