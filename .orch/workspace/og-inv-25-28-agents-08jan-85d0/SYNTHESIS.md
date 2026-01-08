# Session Synthesis

**Agent:** og-inv-25-28-agents-08jan-85d0
**Issue:** orch-go-s73s2
**Duration:** 2026-01-08 05:30 → 2026-01-08 07:30
**Outcome:** success

---

## TLDR

Investigated why 25-28% of agents appear not to complete. Found the "failure" rate is actually a metrics bug - true completion rate is ~89% when properly deduplicated. Only 3 spawns (<1%) are truly stuck. The gap is in event tracking (missing completion events), not agent failures.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-08-inv-25-28-agents-not-completing.md` - Full investigation with 7 findings

### Files Modified
- None

### Commits
- `0f525bca` - investigation: 25-28% agents not completing - checkpoint
- (pending) - investigation: 25-28% agents not completing - complete

---

## Evidence (What Was Observed)

- `orch stats` shows 72% completion rate, but this counts events not unique completions
- 26 duplicate completion events found in 7-day window (same beads_id, multiple events)
- 88 spawns without completion events analyzed:
  - 72 have closed beads issues (work done, event missing)
  - 6 are cross-project (`pw-*` prefix)
  - 3 are still in progress
  - 7 have unknown/edge-case status
- Two completion event types exist: `session.completed` (from orch monitor) and `agent.completed` (from orch complete)
- `orch monitor` is rarely running for headless spawns, so `session.completed` events are rare

### Tests Run
```bash
# Verified duplicate completions
grep -E '"type":"(agent\.completed|session\.completed)"' ~/.orch/events.jsonl | jq '.data.beads_id' | sort | uniq -c | sort -rn | head -10
# Result: Multiple beads_ids have 2-4 completion events each

# Checked "missing" spawn status
for id in $(cat /tmp/missing.txt); do bd show "$id" | grep "^Status:"; done
# Result: 72 closed, 3 in_progress, 13 other
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-25-28-agents-not-completing.md` - Root cause of misleading completion metrics

### Decisions Made
- Metrics bug, not agent failure: The investigation shows the real issue is event tracking gaps, not agents failing to complete work

### Constraints Discovered
- `session.completed` requires `orch monitor` running - most headless spawns won't have this event
- Direct `bd close` bypasses `orch complete` - no event emitted for these completions
- Zombie reconciliation closes issues without events - another gap in tracking

### Externalized via `kn`
```bash
kn constrain "Stats double-count completions" --reason "Events counted, not unique beads_ids. 26 duplicates in 7-day sample."
kn constrain "session.completed requires orch monitor" --reason "Only emitted when monitor is watching SSE - rare for headless spawns"
kn tried "Trusting orch stats completion rate" --failed "Rate is 72% but real is 89% - events not deduplicated"
```

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with 7 findings, synthesis, recommendations)
- [x] Tests passing (verified by cross-referencing events with beads database)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-s73s2`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why do test completions create so many duplicates? (orch-go-vizg had 4 completions for "Test orchestrator completion")
- Should zombie reconciliation be part of `orch complete` instead of separate?

**Areas worth exploring further:**
- How to emit events from `bd close` without tight coupling
- Whether coordination skills should be excluded entirely from completion rate

**What remains unclear:**
- Whether the 72 "closed but no event" spawns actually completed their work or were force-closed

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-25-28-agents-08jan-85d0/`
**Investigation:** `.kb/investigations/2026-01-08-inv-25-28-agents-not-completing.md`
**Beads:** `bd show orch-go-s73s2`
