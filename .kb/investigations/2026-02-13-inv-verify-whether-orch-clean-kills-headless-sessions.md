## Summary (D.E.K.N.)

**Delta:** `orch clean` does NOT kill live headless OpenCode sessions in normal usage; each flag has specific scope and protections, but `--verify-opencode` has an edge-case vulnerability for sessions idle >5min without workspace `.session_id` tracking.

**Evidence:** Code audit of all 7 cleanup code paths in `clean_cmd.go`, `cleanup/sessions.go`, `pkg/opencode/client.go`, and `pkg/daemon/cleanup.go` — traced every path from flag to action.

**Knowledge:** The "only touches registry" claim from kb tried is outdated; clean now has 7 distinct cleanup actions, each with different blast radius. `--ghosts` checks `SessionExists()`, `--verify-opencode` checks `IsSessionProcessing()`. Normal headless spawns write `.session_id` which protects them.

**Next:** Update kb tried entry. Consider adding `IsSessionProcessing()` check to the non-recently-active path in `cleanOrphanedDiskSessions` as defense-in-depth.

**Authority:** implementation - Tactical observation within existing patterns, fix is additive safety check

---

# Investigation: Verify Whether Orch Clean Kills Headless Sessions

**Question:** Does `orch clean` kill live headless OpenCode sessions, either directly (deleting sessions) or indirectly (removing tracking that causes downstream issues)?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| kb tried: "orch clean to remove ghost sessions automatically" | contradicts | Yes - code has evolved significantly | Old claim "Only touches registry" is no longer accurate — clean now has 7 distinct actions |

---

## Findings

### Finding 1: `orch clean` has 7 independent cleanup code paths, each with different blast radius

**Evidence:** The `runClean()` function in `clean_cmd.go:295-511` dispatches to 7 different cleanup actions based on flags:

| Flag | Function | What it touches | Kills sessions? |
|------|----------|-----------------|-----------------|
| (none) | `findCleanableWorkspaces()` | Read-only scan | No |
| `--windows` | tmux `KillWindow()` | Tmux windows of completed workspaces | No (tmux only) |
| `--phantoms` | `cleanPhantomWindows()` | Tmux windows without active OpenCode sessions | No (tmux only) |
| `--ghosts` | `purgeGhostAgents()` | Registry entries | No (registry only) |
| `--verify-opencode` | `cleanOrphanedDiskSessions()` | OpenCode sessions via `DeleteSession()` | **YES** |
| `--investigations` | `archiveEmptyInvestigations()` | Investigation markdown files | No |
| `--stale` | `archiveStaleWorkspaces()` | Workspace directories | No (moves, doesn't delete sessions) |
| `--sessions` | `CleanStaleSessions()` | OpenCode sessions via `DeleteSession()` | **YES** (7+ day threshold) |

**Source:** `cmd/orch/clean_cmd.go:86-98` (flag dispatch), `cmd/orch/clean_cmd.go:295-511` (runClean)

**Significance:** Only 2 of 7 paths can actually kill OpenCode sessions: `--verify-opencode` and `--sessions`. The default `orch clean` with no flags is completely safe — it's read-only.

---

### Finding 2: `--verify-opencode` protects live sessions via 3-layer defense, but has a gap for idle sessions

**Evidence:** `cleanOrphanedDiskSessions()` at `clean_cmd.go:516-644` uses this logic:

```
Layer 1: Is session tracked via workspace .session_id file?
  → YES: Skip (safe)
  → NO: Continue to Layer 2

Layer 2: Was session updated within last 5 minutes?
  → YES: Continue to Layer 3
  → NO: Mark as orphaned → DELETE

Layer 3: Is session currently processing (IsSessionProcessing)?
  → YES: Skip (safe)
  → NO: Mark as orphaned → DELETE
```

The **gap** is in Layer 2: A session updated 6+ minutes ago that is technically still live (e.g., between LLM turns, waiting for tool execution, idle but not abandoned) bypasses the `IsSessionProcessing()` check entirely and gets deleted.

For normal headless spawns via `orch spawn`, Layer 1 protects them because `headless.go:60` writes `.session_id` to the workspace. The gap only affects sessions without workspace tracking.

**Source:** `cmd/orch/clean_cmd.go:569-588` (3-layer logic), `pkg/spawn/backends/headless.go:60` (`.session_id` write), `pkg/opencode/client.go:382-408` (`IsSessionProcessing`)

**Significance:** Normal headless spawns are protected by Layer 1. The gap only affects sessions missing `.session_id` (interactive sessions, sessions where workspace was manually deleted, or sessions where write failed).

---

### Finding 3: `--ghosts` removes registry entries but does NOT kill OpenCode sessions

**Evidence:** `purgeGhostAgents()` at `clean_cmd.go:1059-1157` defines a "ghost" as an agent in the registry with:
- No live tmux window (`tmux.WindowExistsByID()` returns false)
- AND no live OpenCode session (`SessionExists()` returns false)

For headless sessions: `SessionExists()` calls `GET /session/{id}` (line `client.go:349`) which returns 200 for any persisted session — including idle ones. A live headless session WILL return true here. So it won't be classified as a ghost.

Even if classified as a ghost, the action is `agentReg.Remove(ghost.ID)` (line 1141) which only sets `status: deleted` in the registry JSON. It does NOT call `DeleteSession()`. The OpenCode session continues running.

**Source:** `cmd/orch/clean_cmd.go:1096-1117` (ghost detection), `cmd/orch/clean_cmd.go:1140-1143` (removal action), `pkg/opencode/client.go:344-361` (`SessionExists`)

**Significance:** The `--ghosts` flag cannot kill headless sessions. It only cleans stale registry metadata.

---

### Finding 4: `--phantoms` is entirely tmux-scoped and cannot affect headless sessions

**Evidence:** `cleanPhantomWindows()` at `clean_cmd.go:650-750`:
1. Lists all OpenCode sessions updated within 30 minutes
2. Scans tmux windows in worker sessions
3. If a tmux window's beads ID isn't in the active session map → phantom → `tmux.KillWindow()`

Headless sessions have no tmux windows. This code path literally cannot reach them.

**Source:** `cmd/orch/clean_cmd.go:684-718` (phantom detection loop)

**Significance:** `--phantoms` has zero risk to headless sessions.

---

### Finding 5: `--sessions` (daemon periodic cleanup) has strong protection via age threshold

**Evidence:** `CleanStaleSessions()` in `cleanup/sessions.go:31-131`:
- Only targets sessions with `Updated` timestamp older than N days (default 7)
- Checks `IsSessionProcessing()` for ALL candidates (not just recent ones — unlike `--verify-opencode`)
- Default daemon config: 7-day threshold, runs every 6 hours, preserves orchestrators

A live headless session would have been updated within minutes/hours, far below the 7-day threshold.

**Source:** `pkg/cleanup/sessions.go:52-69` (filtering logic), `pkg/daemon/daemon.go:99-103` (default config)

**Significance:** `--sessions` is the safest session-deletion path. 7-day-old sessions are definitionally not "live."

---

### Finding 6: `--stale` can create orphaned sessions for future `--verify-opencode` runs

**Evidence:** `archiveStaleWorkspaces()` moves workspace directories from `.orch/workspace/X/` to `.orch/workspace/archived/X/`. The `cleanOrphanedDiskSessions()` function only scans one level deep in `.orch/workspace/` — it does NOT recurse into `archived/`. So after archival, the `.session_id` file moves into `archived/` and the session becomes "orphaned" from `--verify-opencode`'s perspective.

However, `--stale` only archives workspaces that are:
1. Older than N days (default 7) AND
2. Completed (SYNTHESIS.md exists, or light tier, or has .beads_id)

A live agent's workspace wouldn't meet these criteria (no SYNTHESIS.md, beads still open).

In `--all` mode, `--verify-opencode` runs BEFORE `--stale` (lines 349 vs 385), so there's no cascade within a single run.

**Source:** `cmd/orch/clean_cmd.go:884-1057` (archiveStaleWorkspaces), `cmd/orch/clean_cmd.go:540-554` (workspace scanning)

**Significance:** `--stale` can theoretically create orphaned sessions, but only for completed workspaces. Cross-run cascade requires `--stale` followed by `--verify-opencode`, but the affected sessions would be from completed agents anyway.

---

## Synthesis

**Key Insights:**

1. **`orch clean` without flags is completely safe** — It's read-only, just listing completed workspaces. The "kills live sessions" concern only applies to specific flags (`--verify-opencode`, `--sessions`, and `--all`).

2. **Normal headless spawns are protected by `.session_id` tracking** — The headless backend always writes `.session_id` to the workspace (`headless.go:60`), and `--verify-opencode` uses this to identify tracked sessions. The vulnerability only exists for sessions without this file.

3. **The `--verify-opencode` gap is in the idle detection logic** — Sessions idle > 5 minutes that lack workspace tracking skip the `IsSessionProcessing()` check entirely. This is the only path that could theoretically kill a live session.

4. **The prior kb tried claim is outdated** — "Only touches registry" was true for an earlier version. Current clean has 7 distinct actions, 2 of which can delete actual OpenCode sessions.

**Answer to Investigation Question:**

`orch clean` does NOT kill live headless OpenCode sessions in normal operation. Headless sessions spawned via `orch spawn` are protected by workspace `.session_id` tracking (Layer 1 defense). The `--ghosts` flag only removes registry entries. The `--phantoms` flag only affects tmux windows. The `--sessions` flag has a 7-day threshold.

The only vulnerability exists in `--verify-opencode` for sessions that (a) lack workspace `.session_id` tracking AND (b) have been idle for > 5 minutes. This affects interactive sessions or sessions where the workspace was deleted, not normally-spawned agents.

**Bug status: NOT CONFIRMED for the stated scenario.** Live headless sessions spawned via `orch spawn` are safe. However, a minor defense-in-depth improvement is recommended (see below).

---

## Structured Uncertainty

**What's tested:**

- ✅ All 7 cleanup code paths traced from flag to action (code audit of clean_cmd.go, cleanup/sessions.go)
- ✅ `.session_id` is written by headless backend (`pkg/spawn/backends/headless.go:60`)
- ✅ `SessionExists()` returns true for persisted sessions (`pkg/opencode/client.go:344-361`)
- ✅ `--ghosts` only calls `agentReg.Remove()`, not `DeleteSession()` (`clean_cmd.go:1141`)
- ✅ `--phantoms` only calls `tmux.KillWindow()` (`clean_cmd.go:740`)
- ✅ `--verify-opencode` runs before `--stale` in `--all` mode (lines 349 vs 385)
- ✅ `IsSessionProcessing()` checks last message finish state (`client.go:382-408`)

**What's untested:**

- ⚠️ What happens if `WriteSessionID` fails silently (network error, disk full, race condition) — headless.go:60 logs warning but continues
- ⚠️ Whether OpenCode `DELETE /session/{id}` actually terminates an in-flight agent or just removes metadata
- ⚠️ Behavior under concurrent `orch clean --all` + active spawn (race conditions)

**What would change this:**

- If `WriteSessionID` fails more often than expected, more sessions would be untracked → vulnerable to `--verify-opencode`
- If OpenCode `DeleteSession` terminates in-flight work (not just metadata), the impact of `--verify-opencode` gap is more severe
- If daemon cleanup interval is reduced from 6h to minutes, more frequent `--sessions` runs could hit younger sessions

---

## Implementation Recommendations

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `IsSessionProcessing()` check for non-recent sessions in `--verify-opencode` | implementation | Additive safety check, no architectural change |
| Update kb tried entry to reflect current 7-action architecture | implementation | Documentation accuracy, within scope |

### Recommended Approach ⭐

**Defense-in-depth: Check `IsSessionProcessing()` for ALL orphaned sessions** - In `cleanOrphanedDiskSessions()`, check `IsSessionProcessing()` for all candidate sessions, not just recently active ones.

**Why this approach:**
- Closes the idle-session gap without adding complexity
- Currently `IsSessionProcessing` is only checked when `isRecentlyActive == true` (5min threshold)
- Adding it to the non-recent path prevents deletion of sessions that are between turns but still live
- Cost is one `GetMessages` API call per orphaned session — typically <10 sessions

**Trade-offs accepted:**
- Slightly slower `--verify-opencode` (extra API call per orphaned session)
- Still can't protect sessions that appear idle but have work pending (e.g., waiting for external tool)

**Implementation sequence:**
1. Move `IsSessionProcessing()` check outside the `isRecentlyActive` conditional
2. Update kb tried entry with current architecture
3. Consider adding `IsSessionActive()` (30min idle check) as additional filter

### Alternative Approaches Considered

**Option B: Require confirmation for session deletion**
- **Pros:** Zero risk of accidental deletion
- **Cons:** Breaks automation, daemon can't self-clean
- **When to use instead:** If false-positive deletion rate is high

**Option C: Never delete sessions via clean, rely on OpenCode's own cleanup**
- **Pros:** Eliminates the entire risk category
- **Cons:** Session accumulation causes performance issues
- **When to use instead:** If OpenCode adds native session TTL

---

## References

**Files Examined:**
- `cmd/orch/clean_cmd.go` - All 7 cleanup code paths, flag dispatch, helper functions
- `pkg/cleanup/sessions.go` - `CleanStaleSessions()` used by `--sessions` and daemon
- `pkg/opencode/client.go:344-408` - `SessionExists`, `IsSessionActive`, `IsSessionProcessing`
- `pkg/opencode/client.go:836-887` - `ListDiskSessions`, `DeleteSession`
- `pkg/registry/registry.go` - Registry data model, `Remove()` is tombstone-only
- `pkg/spawn/backends/headless.go` - Headless spawn writes `.session_id`
- `pkg/spawn/session.go` - `WriteSessionID`, `ReadSessionID`
- `pkg/daemon/daemon.go:999-1061` - Daemon periodic cleanup config and logic
- `pkg/daemon/cleanup.go` - Daemon cleanup wrapper
- `cmd/orch/shared.go:30-76` - `extractBeadsIDFromTitle`, `extractBeadsIDFromWindowName`

**Related Artifacts:**
- **Model:** Agent Lifecycle State Model - Confirms dashboard reconciles multiple state sources
- **Model:** Spawn Architecture - `.session_id` tracking is part of workspace lifecycle
- **Model:** Daemon Autonomous Operation - Daemon periodic cleanup uses 7-day threshold
- **Constraint:** orch status can show phantom agents - Confirmed, `--phantoms` addresses this
