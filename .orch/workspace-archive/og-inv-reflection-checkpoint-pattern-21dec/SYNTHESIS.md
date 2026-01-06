# Session Synthesis

**Agent:** og-inv-reflection-checkpoint-pattern-21dec
**Issue:** orch-go-4kwt.8
**Duration:** 2025-12-21 16:30 → 2025-12-21 17:15
**Outcome:** success

---

## TLDR

Investigated 4 options (skill/spawn/protocol/artifact level) for implementing reflection checkpoints. Recommendation: Add "Unexplored Questions" section to SYNTHESIS.md template - minimal change that captures the value without new tooling. Evidence from existing sessions shows value comes from orchestrator reviewing completed work, not execution-time process changes.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-reflection-checkpoint-pattern-agent-sessions.md` - Full investigation with findings, analysis, and recommendations

### Files Modified
- None

### Commits
- (pending) investigation: reflection checkpoint pattern - artifact-level is minimal change

---

## Evidence (What Was Observed)

- SYNTHESIS.md template already has spawn-follow-up and escalate recommendations (lines 64-86)
- Real example: `og-arch-synthesize-findings-investigations-21dec` SYNTHESIS.md shows post-synthesis reflection with Dylan creating new epic (orch-go-ws4z with 6 children)
- Existing tooling: `orch review`, `orch send`, `orch resume` already support interactive follow-up workflow
- Cost analysis: artifact-level change is ~10 lines vs skill-level touching 87+ skill files

### Tests Run
```bash
# Analyzed real agent sessions for post-synthesis reflection patterns
cat .orch/workspace/og-arch-synthesize-findings-investigations-21dec/SYNTHESIS.md
# Found: "Post-synthesis reflection with Dylan led to new epic (orch-go-ws4z)"

# Counted follow-up work from that reflection
# Result: 6 child issues in orch-go-ws4z epic
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-reflection-checkpoint-pattern-agent-sessions.md` - Reflection checkpoint pattern analysis

### Decisions Made
- Decision 1: Artifact-level is minimal change because value comes from output quality not execution process
- Decision 2: SYNTHESIS.md is right location because orchestrator already reviews it via `orch review`

### Constraints Discovered
- Agents already reflect naturally during synthesis - forced phases add cost without proportional value
- Existing workflow (review → send) supports interactive follow-up

### Externalized via `kn`
- `kn decide "Reflection value comes from orchestrator review + follow-up, not execution-time process changes"` - kn-884584

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file produced)
- [x] Tests passing (N/A - investigation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-4kwt.8`

### Follow-up Implementation

**Issue:** Add "Unexplored Questions" section to SYNTHESIS.md template
**Skill:** feature-impl
**Context:**
```
Add section to .orch/templates/SYNTHESIS.md before Session Metadata.
Section prompts for: questions that emerged, areas worth exploring, things that remain unclear.
See investigation for proposed template text.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should we add a "questions captured" check to investigation skill's self-review? Would increase section usage.
- Could `orch review` highlight unexplored questions section specifically? Would draw orchestrator attention.

**Areas worth exploring further:**
- Whether protocol-level reinforcement (SPAWN_CONTEXT.md mention) is needed if artifact-level alone produces empty sections
- Metrics: How many sessions would benefit from structured question capture?

**What remains unclear:**
- Optimal phrasing for the section prompts
- Whether this should be mandatory or optional in SYNTHESIS.md

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-reflection-checkpoint-pattern-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-reflection-checkpoint-pattern-agent-sessions.md`
**Beads:** `bd show orch-go-4kwt.8`
