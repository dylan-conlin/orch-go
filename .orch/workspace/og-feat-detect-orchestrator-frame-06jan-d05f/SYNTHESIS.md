# Session Synthesis

**Agent:** og-feat-detect-orchestrator-frame-06jan-d05f
**Issue:** orch-go-r6ep5
**Duration:** 2026-01-06 15:34 → 16:10
**Outcome:** success

---

## TLDR

Investigated how to detect orchestrator frame collapse (when orchestrator does worker work). Found that self-detection is unreliable and recommended a hybrid approach: SESSION_HANDOFF.md check at `orch complete`, skill guidance with time thresholds, and potential OpenCode plugin for real-time file edit tracking.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md` - Complete investigation with 5 findings and implementation recommendations

### Files Modified
- None (pure investigation)

### Commits
- (pending) Investigation file creation

---

## Evidence (What Was Observed)

- **Skill has self-check but no external detection:** SKILL.md lines 573-680 have ABSOLUTE DELEGATION RULE, line 1429 has "Orchestrator self-check" - all rely on self-detection which is unreliable
- **Plugin infrastructure exists:** `~/.config/opencode/plugin/` has 5 plugins, `orchestrator-session.ts` demonstrates worker detection and event hooks
- **SESSION_HANDOFF.md template lacks frame collapse section:** Template at `~/.orch/session/2026-01-06/SESSION_HANDOFF.md` doesn't explicitly ask about frame collapse
- **Good handoff example (meta-orch-resume-last-meta-06jan-1287):** Shows "Orchestrators spawned: 0", "Workers completed: 2" - proper pattern
- **Problem case from spawn context:** Price-watch orchestrator "Manual fixes by orchestrator" section - this is the detection signal

### Commands Run
```bash
# Verified plugin infrastructure
ls ~/.config/opencode/plugin/
# action-log.ts agentlog-inject.ts bd-close-gate.ts orchestrator-session.ts usage-warning.ts

# Found session handoff files
find ~/.orch -name "SESSION_HANDOFF.md"
# /Users/dylanconlin/.orch/session/2025-12-29/SESSION_HANDOFF.md
# /Users/dylanconlin/.orch/session/2026-01-06/SESSION_HANDOFF.md
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md` - Complete analysis with 5 findings

### Decisions Made
- **Multi-layer detection required:** No single heuristic is reliable; need skill + handoff check + potential plugin
- **SESSION_HANDOFF.md check is lowest cost first step:** Fits existing workflow, forces reflection, doesn't require new infrastructure

### Constraints Discovered
- **Self-detection is unreliable:** Orchestrators in frame collapse state have already rationalized their behavior
- **Detection must happen at boundaries:** Session end, handoff review, or real-time (plugin)
- **Failure-to-implementation is key trigger:** Frame collapse typically happens after agents fail

### Externalized via `kn`
- (Recommend orchestrator externalize via kn if implementing)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file complete with D.E.K.N. summary
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-r6ep5`

### Implementation Follow-ups (for orchestrator)

If implementing the recommendations, create these issues:

1. **Add skill guidance with 15-minute threshold**
   - Skill: `feature-impl`
   - File: `~/.claude/skills/meta/orchestrator/SKILL.md`
   - Add: "If you've been editing code for >15 minutes, you've frame collapsed"

2. **Add Frame Collapse Check to SESSION_HANDOFF.md template**
   - Skill: `feature-impl`
   - File: `~/.orch/templates/SESSION_HANDOFF.md`
   - Add: Required self-check section with explicit questions

3. **Update orch complete for orchestrator tier**
   - Skill: `feature-impl`
   - Add: Warning if orchestrator session has code file changes in git diff

4. **(Optional) OpenCode plugin for real-time detection**
   - Skill: `feature-impl`
   - Track Edit tool usage on code files vs orchestration artifacts

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What file extensions should trigger detection? (need to distinguish code vs orchestration artifacts)
- Should there be a threshold for line count changes?
- How should the plugin handle gradual frame collapse (multiple small edits)?
- Can we detect the failure-to-implementation pattern? (post 3 failed spawns)

**Areas worth exploring further:**
- Meta-orchestrator review patterns - how systematically are handoffs reviewed?
- Detection fatigue - how to avoid false positives causing detection blindness

**What remains unclear:**
- How widespread is frame collapse? (only have price-watch evidence)
- Does frame collapse correlate with agent failure rates?

*(Investigation complete, implementation deferred to follow-up issues)*

---

## Session Metadata

**Skill:** feature-impl (investigation mode)
**Model:** opus
**Workspace:** `.orch/workspace/og-feat-detect-orchestrator-frame-06jan-d05f/`
**Investigation:** `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md`
**Beads:** `bd show orch-go-r6ep5`
