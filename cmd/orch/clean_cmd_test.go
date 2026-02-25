package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func TestClassifyTmuxWindows(t *testing.T) {
	tests := []struct {
		name           string
		windows        []tmux.WindowInfo
		activeBeadsIDs map[string]bool
		openIssues     map[string]*verify.Issue
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
			wantStale:     0,
			wantProtected: 1,
		},
		{
			name: "window with closed beads issue and no OpenCode session is stale",
			windows: []tmux.WindowInfo{
				{Name: "🔧 debug [orch-go-1003]", ID: "@3"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
			wantStale:      1,
			wantProtected:  0,
			wantStaleIDs:   []string{"orch-go-1003"},
		},
		{
			name: "mixed: one protected by beads, one stale",
			windows: []tmux.WindowInfo{
				{Name: "🔧 debug [orch-go-1004]", ID: "@4"},
				{Name: "🔍 inv [orch-go-1005]", ID: "@5"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues: map[string]*verify.Issue{
				"orch-go-1004": {ID: "orch-go-1004", Status: "open"},
			},
			wantStale:     1,
			wantProtected: 1,
			wantStaleIDs:  []string{"orch-go-1005"},
		},
		{
			name: "servers and zsh windows are always skipped",
			windows: []tmux.WindowInfo{
				{Name: "servers", ID: "@10"},
				{Name: "zsh", ID: "@11"},
			},
			activeBeadsIDs: map[string]bool{},
			openIssues:     map[string]*verify.Issue{},
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
			wantStale:     0,
			wantProtected: 0, // Not counted as protected because OpenCode check passes first
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
			wantStale:     0,
			wantProtected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stale, protected := classifyTmuxWindows(tt.windows, "workers-orch-go", tt.activeBeadsIDs, tt.openIssues)

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
