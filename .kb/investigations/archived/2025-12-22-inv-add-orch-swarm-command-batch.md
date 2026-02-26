<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented `orch swarm` command for batch spawning with concurrency control via WorkerPool.

**Evidence:** All tests pass, command registered successfully, dry-run and help output verified.

**Knowledge:** Reused daemon's WorkerPool pattern; swarm differs from daemon in source (explicit list vs polling) and lifecycle (finite vs continuous).

**Next:** Close - implementation complete and ready for use.

**Confidence:** High (85%) - Unit tests pass; end-to-end testing requires live environment.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Add orch swarm command for batch spawning

**Question:** How should we implement a batch spawning command with concurrency control?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: WorkerPool provides reusable concurrency control

**Evidence:** The daemon package already has a well-tested WorkerPool implementation with Acquire/Release semantics, slot tracking, and status monitoring.

**Source:** `pkg/daemon/pool.go:10-210`

**Significance:** We can reuse this for swarm without reinventing concurrency primitives. The pool handles blocking acquisition and graceful shutdown.

---

### Finding 2: Swarm differs from daemon in source and lifecycle

**Evidence:** Daemon polls `bd ready` continuously; swarm works on a finite list (--issues or --ready snapshot). Daemon runs forever; swarm exits when done.

**Source:** `cmd/orch/daemon.go:139-303` (daemon polling loop)

**Significance:** Swarm needs different progress tracking (total/spawned/completed/failed) and a monitoring phase that daemon doesn't have.

---

### Finding 3: Detach mode enables fire-and-forget workflows

**Evidence:** Users need both modes: attached (wait for all to complete) and detached (spawn and return immediately for CI/scripts).

**Source:** Implementation in `cmd/orch/swarm.go:260-330`

**Significance:** --detach flag provides flexibility for different automation scenarios.

---

## Synthesis

**Key Insights:**

1. **Reuse daemon primitives** - WorkerPool pattern works well for both continuous (daemon) and finite (swarm) workloads.

2. **Progress visualization matters** - Users need real-time feedback: spawned/active/completed/failed counters.

3. **Phase-based monitoring** - Using `bd comment` phase tracking enables swarm to know when agents complete.

**Answer to Investigation Question:**

Implement swarm as a separate command that:
1. Collects issues via --issues (explicit list) or --ready (bd list with triage:ready label)
2. Uses WorkerPool for concurrency control
3. Shows progress counters as agents spawn and complete
4. Supports --detach for fire-and-forget workflows
5. Monitors via Phase: Complete comments for attached mode

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Unit tests pass, command compiles and registers correctly, help output verified. Implementation follows established patterns (WorkerPool, events logging).

**What's certain:**

- ✅ WorkerPool integration works (tested via pool_test.go)
- ✅ Progress tracking logic is correct (tested via swarm_test.go)
- ✅ Command registration works (verified via --help)

**What's uncertain:**

- ⚠️ End-to-end behavior with live OpenCode server (needs manual testing)
- ⚠️ Phase monitoring polling may have edge cases
- ⚠️ Rate limiting interaction with spawn delay

**What would increase confidence to Very High:**

- Live testing with multiple real issues
- Observing completion detection with actual agents
- Testing interrupt handling during swarm

---

## Implementation Recommendations

**Purpose:** Already implemented - this section documents what was done.

### Implemented Approach ⭐

**Separate swarm command reusing WorkerPool** - New `orch swarm` command with --issues, --ready, --concurrency, --detach flags.

**Why this approach:**
- Reuses proven WorkerPool pattern from daemon
- Clear separation of concerns (daemon=continuous, swarm=batch)
- Familiar interface for users (mirrors daemon flags)

**Files created:**
- `cmd/orch/swarm.go` - Main command implementation (626 lines)
- `cmd/orch/swarm_test.go` - Unit tests (231 lines)

### Features Implemented

1. **--issues flag** - Explicit comma-separated list of beads issue IDs
2. **--ready flag** - Spawn from `bd list --label triage:ready` queue
3. **--concurrency flag** - Limit parallel agents (default: 3)
4. **--detach flag** - Fire-and-forget mode (don't wait for completion)
5. **--dry-run flag** - Preview without spawning
6. **--delay flag** - Delay between spawns (default: 5s)
7. **--model flag** - Model for spawned agents
8. **Progress display** - Real-time spawned/active/completed/failed counters

**Success criteria:**
- ✅ Command compiles and registers
- ✅ Unit tests pass
- ✅ Help output displays correctly

---

## References

**Files Examined:**
- `pkg/daemon/pool.go` - WorkerPool implementation to reuse
- `pkg/daemon/daemon.go` - Daemon patterns for issue processing
- `cmd/orch/daemon.go` - CLI patterns for daemon command
- `cmd/orch/main.go` - Spawn command patterns

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test execution
go test ./cmd/orch/... -run TestSwarm -v

# Help output verification
go run ./cmd/orch swarm --help
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-feat-add-orch-swarm-22dec/` - Spawn context for this task

---

## Investigation History

**2025-12-22:** Investigation started
- Initial question: How to implement batch spawning with concurrency control?
- Context: Need to spawn multiple agents in parallel for batch processing

**2025-12-22:** Implementation complete
- Created cmd/orch/swarm.go and cmd/orch/swarm_test.go
- All tests passing, command registered

**2025-12-22:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: `orch swarm` command with WorkerPool concurrency control, progress display, and detach mode
