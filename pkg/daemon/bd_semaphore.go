// Package daemon provides autonomous overnight processing capabilities.
package daemon

import "os/exec"

// maxConcurrentBD limits concurrent bd CLI subprocess calls.
// All bd subprocesses contend on .beads/jsonl.lock, so unbounded
// concurrency causes process stampede and deadlock.
const maxConcurrentBD = 8

// bdSem gates bd CLI subprocess execution. Acquire a slot before
// running any bd subprocess and release after it completes.
var bdSem = make(chan struct{}, maxConcurrentBD)

// bdOutput runs cmd.Output() gated by the bd semaphore.
func bdOutput(cmd *exec.Cmd) ([]byte, error) {
	bdSem <- struct{}{}
	defer func() { <-bdSem }()
	return cmd.Output()
}

// bdCombinedOutput runs cmd.CombinedOutput() gated by the bd semaphore.
func bdCombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	bdSem <- struct{}{}
	defer func() { <-bdSem }()
	return cmd.CombinedOutput()
}
