<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `DefaultActiveCount()` in daemon counts ALL active OpenCode sessions including untracked agents (spawned with `--no-track`), causing capacity to be inflated by sessions that shouldn't count against daemon limits.

**Evidence:** 9 sessions active in last 30 min, but 1 was untracked (`orch-go-untracked-1766786808`); daemon reported 9 instead of 8.

**Knowledge:** Untracked agents have beads IDs containing `-untracked-` pattern; daemon capacity should only count tracked (daemon-spawned) agents.

**Next:** Fix implemented - `DefaultActiveCount()` now extracts beads ID from session title and filters out untracked sessions.

**Confidence:** High (90%) - Fix is minimal, tested, and follows existing patterns in the codebase.

---

# Investigation: Daemon Capacity Count Stale After Initial Spawn Cycle

**Question:** Why does daemon capacity count untracked agents toward its limit when they shouldn't affect daemon capacity?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: DefaultActiveCount() counts ALL active sessions without filtering

**Evidence:** 
- `pkg/daemon/daemon.go:416-460` (original): Simply increments `activeCount++` for any session updated within 30 minutes
- No check for whether the session is tracked (daemon-spawned) vs untracked (ad-hoc `--no-track`)
- API test: `curl http://127.0.0.1:4096/session` showed 9 active sessions including 1 untracked

**Source:** pkg/daemon/daemon.go:416-460, API calls to OpenCode

**Significance:** This is the root cause. The daemon's capacity calculation includes sessions that were never spawned by the daemon and shouldn't count against its limits.

---

### Finding 2: Untracked agents have a distinguishable beads ID pattern

**Evidence:**
- `cmd/orch/main.go:1672-1673`: `return fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix()), nil`
- Untracked beads IDs contain `-untracked-` as a segment (e.g., `orch-go-untracked-1766786808`)
- Session titles follow format: `workspace-name [beads-id]` (e.g., `og-arch-review-url-markdown-26dec [orch-go-untracked-1766786808]`)

**Source:** cmd/orch/main.go:1672-1673, cmd/orch/review.go:287-290

**Significance:** The pattern is consistent and already used by `orch review` to identify untracked agents. Same pattern can be applied to daemon capacity counting.

---

### Finding 3: Session titles contain beads ID in brackets

**Evidence:**
- Session title example: `og-debug-daemon-capacity-count-26dec [orch-go-59m3]`
- Pattern: `{workspace-name} [{beads-id}]`
- Existing `extractBeadsIDFromTitle()` in `cmd/orch/main.go:1727-1737` uses same pattern

**Source:** OpenCode API `/session` response, cmd/orch/main.go:1727-1737

**Significance:** Beads ID can be reliably extracted from session titles to determine if a session is untracked.

---

## Synthesis

**Key Insights:**

1. **Daemon capacity was counting wrong sessions** - By counting ALL active sessions, the daemon included ad-hoc spawns that users explicitly marked as `--no-track` (untracked). These sessions shouldn't count against daemon capacity because they weren't spawned by the daemon.

2. **The pattern for detection already exists** - The codebase already has the `-untracked-` pattern for identifying untracked agents (used in `orch review` and beads ID generation). The fix just applies this same pattern to daemon capacity counting.

3. **Title-based extraction is reliable** - Session titles consistently contain the beads ID in brackets, making extraction straightforward using the existing pattern.

**Answer to Investigation Question:**

The daemon counted untracked agents toward capacity because `DefaultActiveCount()` simply counted all sessions updated within 30 minutes without examining whether they were tracked or untracked. The fix adds two new functions (`extractBeadsIDFromSessionTitle()` and `isUntrackedBeadsID()`) and modifies `DefaultActiveCount()` to skip sessions with untracked beads IDs.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The fix is minimal (adds filtering logic using existing patterns), has tests, and was verified with a smoke test showing correct behavior.

**What's certain:**

- ✅ Untracked agents have `-untracked-` in their beads ID (confirmed by code analysis)
- ✅ Session titles contain beads ID in brackets (confirmed by API inspection)
- ✅ The fix correctly filters out untracked sessions (verified with test script)
- ✅ All existing tests pass

**What's uncertain:**

- ⚠️ Edge case: Sessions without beads ID in title (rare, but they would be counted as tracked)
- ⚠️ Long-term behavior with high session counts (unlikely to be an issue in practice)

**What would increase confidence to Very High (95%+):**

- Run daemon overnight with fix and verify capacity tracking stays accurate
- Monitor in production to confirm no edge cases emerge

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐ (Implemented)

**Filter untracked sessions by checking beads ID pattern in session title**

**Why this approach:**
- Minimal code change - adds filtering to existing function
- Uses established patterns (`-untracked-` and title bracket extraction)
- Self-healing - correct immediately without manual reconciliation

**Trade-offs accepted:**
- Adds string parsing overhead per session (negligible at daemon's 60s poll interval)
- Sessions without beads IDs in title counted as tracked (conservative, avoids false exclusions)

**Implementation sequence:**
1. Add `extractBeadsIDFromSessionTitle()` - extracts beads ID from title
2. Add `isUntrackedBeadsID()` - checks for `-untracked-` pattern
3. Modify `DefaultActiveCount()` - skip sessions with untracked beads IDs

### Implementation Details (Completed)

**What was implemented:**

1. `extractBeadsIDFromSessionTitle(title string) string` in `pkg/daemon/daemon.go`
   - Extracts beads ID from session title using `[beads-id]` pattern
   - Returns empty string if no match

2. `isUntrackedBeadsID(beadsID string) bool` in `pkg/daemon/daemon.go`
   - Returns true if beads ID contains `-untracked-`

3. Modified `DefaultActiveCount()` in `pkg/daemon/daemon.go`
   - Added Title field to session struct parsing
   - Extracts beads ID from each session's title
   - Skips sessions where `isUntrackedBeadsID()` returns true

4. Added tests in `pkg/daemon/daemon_test.go`
   - `TestExtractBeadsIDFromSessionTitle` - various title formats
   - `TestIsUntrackedBeadsID` - tracked vs untracked patterns

**Success criteria:**
- ✅ Daemon excludes untracked sessions from capacity count
- ✅ All daemon tests pass
- ✅ Build compiles successfully
- ✅ Smoke test confirms correct filtering

---

## References

**Files Modified:**
- `pkg/daemon/daemon.go:416-477` - Added untracked filtering to DefaultActiveCount()
- `pkg/daemon/daemon_test.go` - Added tests for new functions

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch

# Run tests
go test ./pkg/daemon/... -v -count=1

# Smoke test - count sessions by type
curl -s http://127.0.0.1:4096/session | jq '[.[] | select((.time.updated / 1000) > (now - 1800))] | .[] | {title: .title}'
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md` - Prior fix that added recency filtering
- **Investigation:** `.kb/investigations/2025-12-26-inv-daemon-capacity-count-stuck-while.md` - Prior fix that addressed stale session inflation

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: Why does daemon capacity show 3/3 but orch status shows 3 running + 2 idle untracked?
- Context: Untracked agents counting toward daemon capacity limit

**2025-12-26:** Root cause identified
- `DefaultActiveCount()` counts all sessions, not filtering by tracked/untracked status
- Untracked agents have distinguishable `-untracked-` pattern in beads ID

**2025-12-26:** Fix implemented
- Added title parsing and untracked filtering to `DefaultActiveCount()`
- All tests passing
- Smoke test confirms correct behavior (9 sessions → 8 tracked)

**2025-12-26:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Daemon capacity now correctly excludes untracked sessions
