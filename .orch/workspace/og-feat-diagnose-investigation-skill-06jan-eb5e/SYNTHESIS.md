# Session Synthesis

**Agent:** og-feat-diagnose-investigation-skill-06jan-eb5e
**Issue:** orch-go-1qjvb
**Duration:** 2026-01-06 ~18:45 → 2026-01-06 ~19:XX
**Outcome:** success

---

## TLDR

Investigation skill's 29% completion rate is caused by TWO bugs: test spawn pollution (45% of spawns never intend to complete) AND missing completion events (37.5% of real work produces SYNTHESIS.md but no agent.completed event recorded). True completion rate is ~94%.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-diagnose-investigation-skill-29-completion.md` - Full investigation with findings

### Files Modified
- None

### Commits
- (pending - investigation file)

---

## Evidence (What Was Observed)

- 29 investigation spawns in last 7 days (from events.jsonl)
- 13 spawns (45%) are test/verify/hello patterns - infrastructure validation, not real work
- 16 spawns are real investigation work
- Of 16 real: 9 have agent.completed event, 7 do not
- Of 7 "not completed": 6 have SYNTHESIS.md in workspace! They finished but event wasn't recorded
- Only 1 true failure (workspace not found)
- True completion rate: 15/16 = 93.75%

### Tests Run
```bash
# Count investigation spawns
grep '"skill":"investigation"' ~/.orch/events.jsonl | grep '"type":"session.spawned"' | wc -l
# 298 all time, 29 in last 7 days

# Check for SYNTHESIS.md in "not completed" workspaces
ls .orch/workspace-archive/og-inv-dashboard-port-confusion-03jan/SYNTHESIS.md
# Exists! But no completion event

# Check events for a specific beads_id
grep '"orch-go-untracked-1767476493"' ~/.orch/events.jsonl | jq '.type'
# Output: "session.spawned" only - no completion event
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-06-inv-diagnose-investigation-skill-29-completion.md` - Supersedes prior investigation, adds new finding about completion event bug

### Decisions Made
- Confirmed prior investigation's test spawn filtering recommendation
- NEW: Identified completion event recording bug as second root cause

### Constraints Discovered
- Completion events are not reliably recorded for all successful work
- Test spawns using investigation skill inflate failure metrics
- Stats calculation depends on event recording fidelity

### Externalized via `kn`
- (Will run kn command before completing)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue 1:** Implement --exclude-test flag for orch stats (prior recommendation)
**Skill:** feature-impl
**Context:**
```
Add --exclude-test flag to orch stats that filters:
- beads_ids containing "untracked"  
- workspaces matching patterns: test, verify, hello, quick, exit
Already designed in prior investigation. Quick implementation.
```

**Issue 2:** Investigate completion event recording gap
**Skill:** systematic-debugging
**Context:**
```
6 investigations have SYNTHESIS.md but no agent.completed event in events.jsonl.
Trace the path from Phase: Complete beads comment → agent.completed event emission.
Check: Is daemon auto-complete handling these? Was orch complete called? Timing issue?
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Did the 6 "missed" investigations actually report Phase: Complete via beads? (would clarify if it's agent behavior or event recording)
- Does this completion event bug affect other skills or just investigation? (broader scope)
- Is there a race condition between OpenCode session cleanup and event recording?

**Areas worth exploring further:**
- The exact code path from beads comment → event emission
- Whether daemon auto-complete vs orch complete have different event recording behavior

**What remains unclear:**
- Why specifically these 6 investigations have missing events but others are fine
- Whether the agents called /exit properly

---

## Session Metadata

**Skill:** feature-impl (mismatched - should have been investigation)
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-feat-diagnose-investigation-skill-06jan-eb5e/`
**Investigation:** `.kb/investigations/2026-01-06-inv-diagnose-investigation-skill-29-completion.md`
**Beads:** `bd show orch-go-1qjvb`
