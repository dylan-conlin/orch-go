# Session Synthesis

**Agent:** og-work-orchestrator-pre-spawn-25dec
**Issue:** orch-go-untracked-1766695797 (beads issue not found - untracked spawn)
**Duration:** ~45 minutes
**Outcome:** success

---

## TLDR

Clarified the boundary between orchestrator context-gathering (allowed) and deep investigation (delegate). Proposed a new "Context Gathering vs Investigation" section for the orchestrator skill with explicit table, 5-minute rule, and purpose-based test.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-25-inv-orchestrator-pre-spawn-context-gathering.md` - Full investigation with D.E.K.N., findings, and proposed skill update

### Files Modified
- None (investigation artifact contains proposed changes for orchestrator to review)

### Commits
- Will commit investigation file with this synthesis

---

## Evidence (What Was Observed)

- The orchestrator skill (1320 lines) contains contradictory guidance:
  - Line 279: "ANY investigation (even 'quick' ones)" should be delegated
  - Lines 905-928: Requires `kb context` and "Include key findings (2-3 sentence summary)"
  - Line 293: "Read completed artifacts to synthesize" is allowed
- The skill uses broad language ("never investigate") that accidentally prohibits legitimate spawn context work
- Three categories of context work exist: routing (< 30s), enrichment (1-5 min), exploration (15+ min)
- Only category 3 (exploration) should be delegated; skill conflates categories 2 and 3

### Tests Run
```bash
# Verified skill location and read full content
read ~/.claude/skills/meta/orchestrator/SKILL.md
# Confirmed contradictory language exists

# Checked knowledge base
kb context "orchestrator delegation spawn context"
# No existing guidance on this specific distinction
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-orchestrator-pre-spawn-context-gathering.md` - Complete investigation with proposed skill update

### Decisions Made
- Time-boxing (5 minutes) is the practical boundary between allowed and forbidden context work
- Purpose-based distinction: "reading to write spawn prompt" vs "reading to answer codebase question"
- Explicit table format best communicates allowed vs forbidden activities

### Constraints Discovered
- Skill updates must not lose the core "never investigate deeply" message
- Time-boxing must be a guideline, not a hard rule (judgment still required)

### Externalized via `kn`
- None yet - pending orchestrator review of proposed skill update

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation artifact created)
- [x] Investigation file has `**Status:** Complete`
- [x] Proposed skill update embedded in investigation artifact
- [ ] Ready for `orch complete {issue-id}` - Note: beads issue not found, may need manual close

### Orchestrator Action Required

The investigation artifact at `.kb/investigations/2025-12-25-inv-orchestrator-pre-spawn-context-gathering.md` contains:

1. **Proposed new section** (~30 lines): "Context Gathering vs Investigation" with explicit table
2. **Updated line 299**: More nuanced test that allows reading kb context results
3. **5-minute rule**: Practical threshold for when context-gathering becomes investigation

**To implement:** Edit the orchestrator skill source at `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc` and run `skillc build`.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How do orchestrators in practice handle this boundary? (no behavioral observation data)
- Should there be automated detection when orchestrators cross into investigation mode?
- Is the 5-minute threshold actually the right number? (could be 3 or 10)

**Areas worth exploring further:**
- Other orchestration systems' coordinator/worker responsibility divisions
- Whether `kb context` results should come with an explicit "include this in spawn" section

**What remains unclear:**
- Whether purpose-based distinctions are enforceable or just rationalizations
- Actual orchestrator pain points (this investigation was theoretical analysis)

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-work-orchestrator-pre-spawn-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-orchestrator-pre-spawn-context-gathering.md`
**Beads:** Issue not found (orch-go-untracked-1766695797)
