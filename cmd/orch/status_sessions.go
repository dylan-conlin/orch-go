package main

import (
	"fmt"
	"os"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

type statusSessionLister interface {
	ListSessionsWithOpts(directory string, opts *opencode.ListSessionsOpts) ([]opencode.Session, error)
}

// listSessionsForStatus fetches OpenCode sessions in best-effort mode.
// If OpenCode is slow or unavailable, status should degrade gracefully instead
// of failing the entire command.
func listSessionsForStatus(client statusSessionLister, listOpts *opencode.ListSessionsOpts, debugTiming bool) ([]opencode.Session, bool) {
	sessions, err := client.ListSessionsWithOpts("", listOpts)
	if err != nil {
		if debugTiming {
			fmt.Fprintf(os.Stderr, "[timing] ListSessions failed (degraded mode): %v\n", err)
		}
		return []opencode.Session{}, false
	}

	return sessions, true
}
