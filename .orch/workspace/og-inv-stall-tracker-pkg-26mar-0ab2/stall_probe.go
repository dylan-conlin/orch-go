package main

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/execution"
)

func main() {
	tracker := daemon.NewStallTracker(3 * time.Second)
	tokens := &execution.TokenStats{InputTokens: 100, OutputTokens: 50}
	sessionID := "probe-session"

	fmt.Printf("t=0 update => stalled=%v\n", tracker.Update(sessionID, tokens))
	time.Sleep(1 * time.Second)
	fmt.Printf("t=1s update => stalled=%v duration=%v\n", tracker.Update(sessionID, tokens), tracker.GetStallDuration(sessionID, tokens))
	time.Sleep(1 * time.Second)
	fmt.Printf("t=2s update => stalled=%v duration=%v\n", tracker.Update(sessionID, tokens), tracker.GetStallDuration(sessionID, tokens))
	time.Sleep(1 * time.Second)
	fmt.Printf("t=3s update => stalled=%v duration=%v\n", tracker.Update(sessionID, tokens), tracker.GetStallDuration(sessionID, tokens))
	time.Sleep(4 * time.Second)
	fmt.Printf("t=7s update => stalled=%v duration=%v\n", tracker.Update(sessionID, tokens), tracker.GetStallDuration(sessionID, tokens))

	progress := &execution.TokenStats{InputTokens: 100, OutputTokens: 60}
	fmt.Printf("progress update => stalled=%v duration=%v\n", tracker.Update(sessionID, progress), tracker.GetStallDuration(sessionID, progress))
}
