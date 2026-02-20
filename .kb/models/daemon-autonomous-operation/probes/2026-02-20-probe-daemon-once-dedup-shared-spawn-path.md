---
Status: Complete
Date: 2026-02-20
Model: daemon-autonomous-operation
Probe: daemon-once-dedup-shared-spawn-path
---

# Question
Do OnceExcluding and OnceWithSlot share the same dedup and status-update logic after refactor, preserving the model claim that duplicate spawns are prevented by early in_progress updates and session/title checks?

# What I Tested
- `go test ./pkg/daemon -run TestDaemon_SpawnIssue_StatusUpdateFailureReleasesSlot`

# What I Observed
- Test passed: `ok   github.com/dylan-conlin/orch-go/pkg/daemon`

# Model Impact
- Confirms: Dedup/status-update guard logic is centralized in a shared spawn path and remains enforced (status update failure blocks spawn, releases slot, no spawn call).
