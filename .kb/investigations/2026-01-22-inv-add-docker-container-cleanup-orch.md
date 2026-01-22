## Summary (D.E.K.N.)

**Delta:** Docker containers spawned via `orch spawn --backend docker` are now tracked and cleaned up during `orch complete` and `orch abandon`.

**Evidence:** Build compiles successfully; container name written to `.container_id` file; cleanup functions added to both commands.

**Knowledge:** Using `--name` flag with unique container names enables explicit cleanup while keeping `--rm` for automatic cleanup on normal exit.

**Next:** Close - implementation complete.

**Promote to Decision:** recommend-no (tactical fix from strategic audit finding)

---

# Investigation: Add Docker Container Cleanup Orch

**Question:** How do we track and clean up Docker containers spawned by `orch spawn --backend docker`?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Docker containers were not tracked after spawn

**Evidence:** Current `SpawnDocker` function uses `--rm` flag but doesn't track container name. If tmux window is killed, container may not be gracefully stopped.

**Source:** `pkg/spawn/docker.go:64-86` (original implementation)

**Significance:** Container orphaning can occur when agents are abandoned or completed while container is running.

---

### Finding 2: Container names can be generated from workspace names

**Evidence:** Workspace names are unique per spawn (e.g., `og-feat-add-cleanup-22jan-1234`). Using `orch-{workspace-name}` as container name provides predictable, unique names.

**Source:** `pkg/spawn/config.go` - `WorkspaceName` field in `Config` struct

**Significance:** Enables tracking containers without additional state management.

---

### Finding 3: Docker `--rm` flag handles normal exit cleanup

**Evidence:** Docker's `--rm` flag automatically removes containers when they exit normally. We keep this and add explicit cleanup for forced exits (via `orch complete` or `orch abandon`).

**Source:** Docker documentation for run command

**Significance:** No need to remove `--rm` - we get automatic cleanup on normal exit plus explicit cleanup on forced exit.

---

## Synthesis

**Key Insights:**

1. **Track via file** - Writing container name to `.container_id` in workspace enables stateless cleanup without registry changes.

2. **Graceful cleanup order** - Docker cleanup should happen before tmux window kill to ensure container stops cleanly.

3. **Idempotent cleanup** - Both `docker stop` and `docker rm` handle "container not found" gracefully, so cleanup is safe to run multiple times.

**Answer to Investigation Question:**

Docker containers are now tracked by writing a `.container_id` file containing the container name (`orch-{workspace-name}`) to the workspace during spawn. During `orch complete` or `orch abandon`, this file is read and `docker stop` + `docker rm -f` are executed to clean up the container.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build compiles successfully with new code
- ✅ Container name written to `.container_id` file (code review)
- ✅ Cleanup functions handle missing containers gracefully (code review)

**What's untested:**

- ⚠️ End-to-end test with actual Docker spawn and cleanup (requires Docker running)
- ⚠️ Behavior when container is in various states (running, stopped, paused)

**What would change this:**

- Finding would be invalid if Docker naming rules reject our sanitized names
- Finding would be invalid if `--rm` conflicts with explicit `--name` (doesn't based on Docker docs)

---

## Implementation Summary

**Changes made:**

1. **pkg/spawn/docker.go:**
   - Added `ContainerNamePrefix` constant
   - Added `sanitizeContainerName()` function for Docker naming rules
   - Added `CleanupDockerContainer()` function for stop/rm
   - Added `ReadContainerID()` function to read container name from workspace
   - Modified `SpawnDocker()` to write `.container_id` file and use `--name` flag

2. **cmd/orch/complete_cmd.go:**
   - Added Docker cleanup before tmux window cleanup
   - Added spawn package import

3. **cmd/orch/abandon_cmd.go:**
   - Added Docker cleanup before tmux window cleanup

---

## References

**Files Modified:**
- `pkg/spawn/docker.go` - Container tracking and cleanup functions
- `cmd/orch/complete_cmd.go` - Added cleanup call
- `cmd/orch/abandon_cmd.go` - Added cleanup call

**Commands Run:**
```bash
# Verify build
go build ./...

# Run spawn package tests (pre-existing failures unrelated to changes)
go test ./pkg/spawn/...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-22-inv-strategic-audit-daemon-reliability-multiple.md` - Finding 5 identified this gap

---

## Investigation History

**2026-01-22 16:45:** Investigation started
- Initial question: How to track and clean up Docker containers
- Context: Strategic audit finding identified container orphaning as reliability gap

**2026-01-22 17:00:** Implementation completed
- Status: Complete
- Key outcome: Docker containers now tracked via `.container_id` file and cleaned up during complete/abandon
