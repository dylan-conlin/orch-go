<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.

**See guide:** `.kb/guides/completion.md` - Consolidated completion workflow reference
-->

## Summary (D.E.K.N.)

**Delta:** Workspace accumulation (410 active) is primarily from stale workspaces (132 older than 7 days) that `orch clean --stale` would archive, but isn't being run regularly.

**Evidence:** Of 409 non-archived workspaces: 141 have SYNTHESIS.md (completed), 9 are meta-orchestrator (expected no synthesis), 259 are regular agents without synthesis; `orch clean --stale --dry-run` shows 132 workspaces ready to archive.

**Knowledge:** The cleanup mechanism exists but isn't automated. Workspace archival requires explicit `orch clean --stale` invocation. No auto-archive on completion or scheduled cleanup.

**Next:** Add automatic stale workspace archival to daemon poll cycle OR add to `orch complete` workflow.

**Promote to Decision:** recommend-yes - This should establish a pattern for automatic workspace lifecycle management.

---

# Investigation: Address 340 Active Workspaces Completion

**Question:** Why are 340+ workspaces accumulating without being archived, and what strategy should be used to address this?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** agent (spawned)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Workspace Distribution Analysis

**Evidence:** 
- Total non-archived workspaces: 409
- With SYNTHESIS.md (completed): 141
- Meta-orchestrator (no synthesis expected): 9
- Regular agents without synthesis: 259

Age distribution:
- Last 7 days: 256 workspaces
- 8-14 days: 132 workspaces (stale, archivable)
- Older than 14 days: 0 (suggests archival was run ~2 weeks ago)

**Source:** 
```bash
ls .orch/workspace/ | wc -l  # 409
# Counted SYNTHESIS.md presence, .tier files, spawn_time calculations
```

**Significance:** The bulk of "active" workspaces are actually stale (132 are >7 days old and meet archival criteria). Recent workspaces (256 in last 7 days) represent normal high-volume work, not a completion gap.

---

### Finding 2: Existing Cleanup Mechanism is Manual

**Evidence:**
`orch clean --stale` exists and works correctly:
- Checks `.spawn_time` file age
- Checks completion indicators (SYNTHESIS.md, light tier, .beads_id)
- Archives to `.orch/workspace/archived/`
- Dry-run shows 132 stale workspaces ready to archive

However:
- No automatic invocation in daemon or completion workflow
- `orch complete` only cleans tmux windows, not workspace directories
- Daemon only reconciles worker pool capacity, not workspace cleanup

**Source:** 
- `cmd/orch/clean_cmd.go` (lines 823-996) - `archiveStaleWorkspaces()`
- `pkg/daemon/daemon.go` (lines 370-393) - `ReconcileWithOpenCode()` only handles pool capacity

**Significance:** The cleanup infrastructure exists but is entirely manual. The 132 stale workspaces would be archived if someone ran `orch clean --stale`.

---

### Finding 3: Completion Workflow Gaps

**Evidence:**
`orch complete` performs:
- Phase verification (Phase: Complete)
- Beads issue closure
- Tmux window cleanup
- Auto-rebuild if Go changes
- Cache invalidation

But does NOT:
- Archive completed workspace
- Clean up OpenCode disk sessions
- Run any `orch clean` operations

**Source:** `cmd/orch/complete_cmd.go` (entire file reviewed)

**Significance:** Completion closes the tracking issue but leaves workspace artifacts in place. This is by design (workspaces preserved for investigation reference), but without periodic cleanup they accumulate.

---

### Finding 4: Dashboard Visibility Already Exists

**Evidence:**
The dashboard at `http://localhost:5188` shows completed agents in the "Archive" section. This includes:
- Workspaces with SYNTHESIS.md
- Agents whose beads issue is closed
- Last updated timestamps

**Source:** `cmd/orch/serve_agents.go` (lines 541-645) - completed workspace discovery

**Significance:** Dashboard already surfaces stale/completed workspaces for review. The gap is in automated cleanup, not visibility.

---

## Synthesis

**Key Insights:**

1. **Not a completion gap, but an archival gap** - Agents ARE completing (141 have SYNTHESIS.md), but completed workspaces aren't being archived. The issue is workspace lifecycle management, not agent completion behavior.

2. **Manual cleanup exists but isn't being used** - `orch clean --stale` would immediately resolve 132 stale workspaces. The tool exists but requires explicit invocation.

3. **High spawn volume is normal** - 256 workspaces in the last 7 days represents active development. This is expected behavior, not a problem.

**Answer to Investigation Question:**

Workspaces accumulate because:
1. `orch complete` closes issues but intentionally preserves workspaces for reference
2. `orch clean --stale` must be run manually to archive old workspaces
3. No automated cleanup is triggered by completion or on a schedule

The 340+ workspace count is misleading - 132 are stale (>7 days old) and ready to archive, 256 are recent active work. Running `orch clean --stale` once would reduce the count to ~277.

---

## Structured Uncertainty

**What's tested:**

- ✅ Workspace count and age distribution (verified: ran ls, spawn_time parsing)
- ✅ `orch clean --stale --dry-run` identifies 132 archivable workspaces (verified: ran command)
- ✅ SYNTHESIS.md presence as completion indicator (verified: counted files)
- ✅ Current cleanup mechanisms don't auto-archive (verified: code review)

**What's untested:**

- ⚠️ Whether auto-archival on completion would cause issues (not implemented)
- ⚠️ Whether daemon-based cleanup would impact performance (not tested)
- ⚠️ Whether 7-day default is the right threshold (not validated with user)

**What would change this:**

- Finding would be wrong if `orch clean --stale` has a bug that prevents actual archival
- Finding would be incomplete if there are untracked workspaces without .spawn_time files
- Recommendation would change if workspace preservation is needed for longer than 7 days

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Add Auto-Archive to `orch complete`** - When completing an agent, also archive the workspace if it has SYNTHESIS.md

**Why this approach:**
- Tight coupling: archival happens at natural completion point
- No additional processes or schedules needed
- User already expects completion to "clean up" after the agent

**Trade-offs accepted:**
- Workspaces archived immediately on completion (no grace period for review)
- User can use `--preserve-workspace` flag if they want to keep it

**Implementation sequence:**
1. Add `--no-archive` flag to `orch complete` (opt-out)
2. After successful completion, move workspace to archived/
3. Log archival in events

### Alternative Approaches Considered

**Option B: Daemon-based periodic cleanup**
- **Pros:** Runs automatically in background, no user action needed
- **Cons:** Adds complexity to daemon, may archive during active work sessions
- **When to use instead:** If immediate archival on complete causes issues

**Option C: Schedule-based cleanup (cron/launchd)**
- **Pros:** Completely decoupled from completion workflow
- **Cons:** Requires separate scheduling, may miss cleanups if not running
- **When to use instead:** If daemon and complete modifications are undesirable

**Rationale for recommendation:** Option A is simplest and aligns with user mental model ("complete = done = cleaned up").

---

### Implementation Details

**What to implement first:**
- Run `orch clean --stale` immediately to clear 132 stale workspaces (manual, instant fix)
- Then implement auto-archive in `orch complete` for ongoing maintenance

**Things to watch out for:**
- ⚠️ Orchestrator workspaces should NOT be auto-archived (check `isOrchestratorWorkspace`)
- ⚠️ Cross-project workspaces need careful path handling
- ⚠️ Archive operation should be after all other completion steps (fail-safe order)

**Areas needing further investigation:**
- Should there be a "recently archived" view in dashboard?
- Should archived workspaces be automatically deleted after N days?

**Success criteria:**
- ✅ Workspace count stays below 300 during normal operation
- ✅ `orch complete` archives workspace unless `--no-archive` specified
- ✅ No manual `orch clean --stale` needed during regular workflow

---

## References

**Files Examined:**
- `cmd/orch/clean_cmd.go` - Cleanup command implementation
- `cmd/orch/complete_cmd.go` - Completion workflow
- `pkg/daemon/daemon.go` - Daemon reconciliation
- `.orch/workspace/` - Actual workspace contents

**Commands Run:**
```bash
# Count workspaces
ls .orch/workspace/ | wc -l  # 409

# Count by completion status
for dir in .orch/workspace/*/; do [counting SYNTHESIS.md presence]; done

# Preview stale archival
orch clean --stale --dry-run  # 132 would be archived
```

**Related Artifacts:**
- **Decision:** 'orch review done' processes completions - Related to completion workflow

---

## Investigation History

**2026-01-07 16:XX:** Investigation started
- Initial question: Why 340+ active workspaces accumulating?
- Context: Orchestrator noticed workspace count unusually high

**2026-01-07 16:XX:** Found workspace distribution
- 141 completed, 9 meta-orch, 259 without synthesis
- 132 are stale (>7 days old), 256 are recent (<7 days)

**2026-01-07 16:XX:** Investigation completed
- Status: Complete
- Key outcome: Not a completion gap but an archival gap - `orch clean --stale` needs to run more regularly, recommend adding auto-archive to `orch complete`
