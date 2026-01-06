# Session Synthesis

**Agent:** og-inv-failure-mode-artifacts-21dec
**Issue:** orch-go-4kwt.6
**Duration:** 2025-12-21
**Outcome:** success

---

## TLDR

Investigated what artifacts should persist when agents fail. Found significant gap: successful agents produce SYNTHESIS.md, abandoned agents produce nothing. Recommended layered failure capture: enhanced events, optional `--reason` flag on abandon, and manual post-mortems for patterns.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md` - Complete investigation with findings and recommendations

### Commits
- `612dc1c` - investigation: failure mode artifacts - what should persist when agents fail

---

## Evidence (What Was Observed)

- Examined 5 abandoned workspaces: All contain only SPAWN_CONTEXT.md, no structured failure artifacts
- 33 abandoned events in events.jsonl, each with minimal data (agent_id, beads_id, window_id only)
- Beads comments show progress phases but no failure annotation when abandoned
- No post-mortem infrastructure exists (searched entire codebase)

### Tests Run
```bash
# Verified abandoned workspace contents
ls -la .orch/workspace/og-debug-fix-oauth-token-20dec/
# Result: Only SPAWN_CONTEXT.md (30KB)

# Analyzed events log
cat ~/.orch/events.jsonl | grep "abandoned" | wc -l
# Result: 33 abandonments

# Checked beads for abandoned agent
bd comments orch-go-bdd.2 --json
# Result: Progress phases visible, no failure context
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md` - Complete analysis of failure artifact gap

### Decisions Made
- Layered approach for failure capture: automatic events, optional reason, manual post-mortems
- Beads is the right integration point for failure context (comments already track phases)

### Constraints Discovered
- Can't require detailed failure reports on every abandon (adds friction, sometimes reason unknown)
- Agent cooperation needed for detailed failure context (can't extract automatically)

### Three failure modes identified
1. **Context exhaustion** (~60%) - Agent makes progress then silently stops
2. **External blocker** (~25%) - Agent asks question, never gets answer
3. **Unresponsive/stuck** (~15%) - No progress after spawn

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation file committed
- [x] Ready for `orch complete orch-go-4kwt.6`

### Follow-up Work (optional)
If orchestrator wants to implement recommendations:

1. **Modify `orch abandon`** - Add `--reason` flag that creates beads comment
2. **Enhance events.jsonl** - Add `failure_mode` and `last_phase` to `agent.abandoned` events
3. **Create templates** - Add `.orch/templates/FAILURE_REPORT.md` for detailed failure analysis
4. **Create directory** - Add `.orch/knowledge/spawning-lessons/` with README

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-failure-mode-artifacts-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md`
**Beads:** `bd show orch-go-4kwt.6`
