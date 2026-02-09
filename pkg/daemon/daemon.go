// Package daemon provides autonomous overnight processing capabilities.
//
// This package is organized into focused files:
//   - daemon_lifecycle.go: Config, types, constructors, EventLogger
//   - daemon_queue.go: Issue queue, filtering, rejection checks, preview
//   - daemon_spawn.go: Spawn execution, capacity, pool/rate management, Run loop
//   - daemon_periodic.go: Reflection, cleanup, recovery, dead session detection
//   - polish.go: Idle-time polish mode audits and issue creation
//   - daemon_crossproject.go: Cross-project polling, spawning, preview
package daemon
