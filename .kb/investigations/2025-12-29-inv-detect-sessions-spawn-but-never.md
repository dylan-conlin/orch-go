<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented failed-to-start detection for spawned agents via 3 mechanisms: post-spawn comment monitoring, status display indicator, and doctor health check.

**Evidence:** Build succeeds, all verify package tests pass, new code integrates with existing spawn/status/doctor commands.

**Knowledge:** Sessions that spawn but never report Phase status can now be detected automatically through: (1) 60s background polling after spawn with stderr warning, (2) "⚠️ stalled" status indicator in `orch status` for agents >1 min old with 0 comments, (3) "Session Health" check in `orch doctor` that lists stalled sessions.

**Next:** Close - all three detection mechanisms implemented and tested.

---

# Investigation: Detect Sessions Spawn But Never

**Question:** How can we detect sessions that spawn but never execute (no Phase report)?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Post-spawn comment monitoring in runSpawnHeadless

**Evidence:** Added `monitorFirstComment` goroutine that polls for beads comments after spawn. If no comment within 60s, outputs warning to stderr and logs `session.no_comment_warning` event.

**Source:** `cmd/orch/main.go:1850-1875`

**Significance:** Provides immediate feedback when a spawn fails to start, without blocking the spawn command.

---

### Finding 2: NoComments field in AgentInfo for status display

**Evidence:** Added `NoComments` boolean field to `AgentInfo` struct. Status command now checks if agents >1 min old have 0 beads comments and sets this flag. The `getAgentStatus` function returns "⚠️ stalled" for such agents.

**Source:** `cmd/orch/main.go:2587` (field), `cmd/orch/main.go:2810-2820` (detection), `cmd/orch/main.go:3366-3369` (display)

**Significance:** Makes stalled sessions immediately visible in `orch status` output.

---

### Finding 3: Session Health check in orch doctor

**Evidence:** Added `checkStalledSessions` function that scans active sessions, extracts beads IDs, and checks for sessions >1 min old with no comments. Reports count and IDs of stalled sessions.

**Source:** `cmd/orch/doctor.go:457-535`

**Significance:** Provides a health check for failed-to-start sessions alongside other service checks.

---

## Synthesis

**Key Insights:**

1. **Background monitoring is non-blocking** - The `monitorFirstComment` goroutine runs independently of spawn completion, so it doesn't slow down automation.

2. **Status indicators use existing infrastructure** - The `NoComments` check reuses the existing `commentsMap` that's already fetched for phase detection, so there's no additional overhead.

3. **Doctor check is comprehensive** - The stalled sessions check scans all active sessions across both project-specific and global sessions.

**Answer to Investigation Question:**

Sessions that spawn but never execute can be detected through three complementary mechanisms:
1. Post-spawn 60s timeout warning (proactive)
2. `orch status` "⚠️ stalled" indicator (on-demand visibility)
3. `orch doctor` Session Health check (health monitoring)

---

## Implementation Details

**Files modified:**
- `pkg/verify/check.go` - Added `HasBeadsComment`, `CheckCommentsWithAge`, `WaitForFirstComment` functions
- `pkg/verify/check_test.go` - Added tests for new functions
- `cmd/orch/main.go` - Added `monitorFirstComment` goroutine, `NoComments` field, status display logic
- `cmd/orch/doctor.go` - Added `checkStalledSessions` function

**Success criteria:**
- ✅ Build succeeds
- ✅ All verify package tests pass
- ✅ Post-spawn monitoring integrated
- ✅ Status display shows stalled indicator
- ✅ Doctor check shows stalled sessions
