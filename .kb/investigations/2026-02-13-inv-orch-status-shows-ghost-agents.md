## Summary (D.E.K.N.)

**Delta:** Ghost agents inflating Active count caused by two bugs in phantom detection: (1) "api-stalled" SessionID not treated as phantom, (2) Window field persisting from registry even when tmux window is dead.

**Evidence:** Before fix: Active=8 with zero live agents. After fix: Active=3 (actual live agents), Phantom=15 (correctly detected). `orch clean --ghosts --dry-run` identifies all 15 ghosts.

**Knowledge:** Registry is a spawn-time cache that never reconciles with live state. The phantom detection logic at status_cmd.go:482 must check ALL stalled states, and clearing stale Window references is critical for correct phantom detection.

**Next:** Fix implemented and verified. Close issue.

**Authority:** implementation - Bug fix within existing status/clean command patterns, no architectural changes.

---

# Investigation: Orch Status Shows Ghost Agents

**Question:** Why does `orch status` show 8 active agents when zero live tmux windows and zero live OpenCode sessions exist?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker (systematic-debugging)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: "api-stalled" SessionID bypasses phantom detection

**Evidence:** In status_cmd.go:482, the phantom check was:
```go
if agent.SessionID != "" && agent.SessionID != "tmux-stalled" {
    agent.IsPhantom = false
}
```
For opencode/headless mode agents with dead sessions, SessionID is set to "api-stalled" (line 327). Since "api-stalled" != "tmux-stalled", these agents were NOT marked phantom.

**Source:** cmd/orch/status_cmd.go:317-328, 482-488

**Significance:** All cross-project agents (pw-8966, pw-8975, pw-w7z0, pw-ebzn, pw-07mw) and opencode-mode agents (orch-go-1) were incorrectly counted as Active.

---

### Finding 2: Window field persists from registry even when tmux window is dead

**Evidence:** For claude-mode agents, the Window field is set from registry data at line 288:
```go
info.Window = a.TmuxWindow
```
When the tmux window doesn't exist, SessionID is set to "tmux-stalled" (line 301) but Window is never cleared. The phantom check at line 484:
```go
} else if agent.Window != "" {
    agent.IsPhantom = false
}
```
This means dead claude-mode agents always have a non-empty Window, preventing phantom detection.

**Source:** cmd/orch/status_cmd.go:288, 300-302, 484-485

**Significance:** All claude-mode ghost agents (orch-go-knj, orch-go-4, orch-go-6, etc.) were incorrectly counted as Active because their stale Window reference prevented phantom classification.

---

### Finding 3: Registry never reconciles active state with live sources

**Evidence:** Registry stores agents as "active" at spawn time. Neither `orch complete` nor `orch clean` update registry status. The comment at registry.go:86-87 confirms: "clean_cmd.go: Does NOT interact with registry."

**Source:** pkg/registry/registry.go:86-98

**Significance:** Ghost entries accumulate indefinitely in the registry. The `--ghosts` flag on `orch clean` addresses this by cross-referencing registry entries against live tmux windows and OpenCode sessions.

---

## Synthesis

**Key Insights:**

1. **Phantom detection needs to check ALL stalled states** - The original code only excluded "tmux-stalled" from the liveness check, but "api-stalled" is equally indicative of a dead agent.

2. **Window field must reflect reality, not registry cache** - Setting Window from registry data without clearing it when the actual tmux window is dead creates false liveness signals.

3. **Registry needs periodic reconciliation** - The `--ghosts` flag provides on-demand reconciliation by marking dead agents as deleted in the registry.

**Answer to Investigation Question:**

Ghost agents appeared in `orch status` because two bugs in phantom detection caused dead agents to be classified as "active": (1) opencode-mode agents with "api-stalled" sessions bypassed the phantom check, and (2) claude-mode agents with dead tmux windows retained stale Window references that prevented phantom classification. The fix addresses both phantom detection bugs and adds `orch clean --ghosts` for registry purging.

---

## Structured Uncertainty

**What's tested:**

- ✅ Before fix: Active=8 with zero live agents (reproduced)
- ✅ After fix: Active=3, Phantom=15 (verified via `orch status --json`)
- ✅ `orch clean --ghosts --dry-run` correctly identifies all 15 ghost agents
- ✅ All existing tests pass (`go test ./cmd/orch/` - PASS in 2.1s)
- ✅ Build and vet clean (`go build ./cmd/orch/` and `go vet ./cmd/orch/`)

**What's untested:**

- ⚠️ `orch clean --ghosts` without `--dry-run` (registry modification) — not run in production during this session

**What would change this:**

- Finding would be wrong if agents can exist without either tmux window or OpenCode session (but this contradicts the spawn architecture)

---

## References

**Files Examined:**
- cmd/orch/status_cmd.go - Phantom detection logic and agent enrichment
- cmd/orch/clean_cmd.go - Cleanup command structure
- pkg/registry/registry.go - Registry data model and lifecycle methods
- cmd/orch/shared.go - Helper functions for beads ID extraction
- cmd/orch/reconcile.go - Existing zombie reconciliation (for beads issues, not registry)

**Files Modified:**
- cmd/orch/status_cmd.go - Fixed phantom detection (2 lines)
- cmd/orch/clean_cmd.go - Added `--ghosts` flag and `purgeGhostAgents()` function
- cmd/orch/clean_test.go - Updated TestCleanAllFlagLogic to include ghosts flag
