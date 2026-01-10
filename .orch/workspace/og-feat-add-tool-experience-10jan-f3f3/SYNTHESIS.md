# Session Synthesis

**Agent:** og-feat-add-tool-experience-10jan-f3f3
**Issue:** orch-go-74o00
**Duration:** 2026-01-10
**Outcome:** success

---

## TLDR

Added tool experience prompts to orchestrator and investigation skills to address trust calibration meta-pattern where Dylan defers to AI despite having relevant tool experience (foreman, Docker, DevTools). Prompts ask "Have you used [tool] before?" before making recommendations or starting investigations.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-10-inv-add-tool-experience-prompts-orchestrator.md` - Investigation documenting skill structure and implementation approach

### Files Modified
- `~/orch-knowledge/skills/src/worker/investigation/.skillc/workflow.md` - Added TOOL EXPERIENCE CHECK before TEST-FIRST GATE (step 3)
- `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template` - Added Tool Experience Prompts section after Orchestrator Autonomy

### Commits
- `06bea845` - investigation: add-tool-experience-prompts-orchestrator - initial findings checkpoint
- `a6838296` - investigation: add-tool-experience-prompts-orchestrator - complete with findings and synthesis
- (orch-knowledge repo) - feat(investigation): add tool experience check before test-first gate

---

## Evidence (What Was Observed)

- Investigation skill uses modular architecture (5 source files: intro.md, workflow.md, template.md, self-review.md, completion.md)
- Orchestrator skill uses monolithic architecture (1 source file: SKILL.md.template)
- Both skills successfully deployed via `skillc deploy --target ~/.claude/skills/ ./src` (18/18 skills deployed)
- Grep confirmed prompts exist in deployed SKILL.md files:
  - Investigation: `3. **TOOL EXPERIENCE CHECK (before elaborate investigation):**`
  - Orchestrator: `## Tool Experience Prompts (Ask Before Recommending)`

### Tests Run
```bash
# Verify investigation skill deployment
grep -A 5 "TOOL EXPERIENCE CHECK" ~/.claude/skills/worker/investigation/SKILL.md
# Found at line 3 with full prompt text

# Verify orchestrator skill deployment  
grep -A 5 "Tool Experience Prompts" ~/.claude/skills/meta/orchestrator/SKILL.md
# Found with full section including examples and anti-patterns
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-10-inv-add-tool-experience-prompts-orchestrator.md` - Documents skill architecture differences and implementation approach

### Decisions Made
- **Modular vs inline placement:** Investigation skill's modular structure made it easy to add section to workflow.md. Orchestrator's monolithic structure required editing large template, but inline placement ensures visibility.
- **Placement strategy:** Tool experience prompts placed before key decision points (TEST-FIRST GATE for investigation, after Autonomy section for orchestrator) to ensure agents see them when making tool/approach choices.

### Constraints Discovered
- Orchestrator SKILL.md.template has "AUTO-GENERATED" warning but git history shows manual edits - unclear if edit approach is correct, asked orchestrator for clarification
- Skillc doesn't support conditional includes - all content must work at spawn-time via runtime references

### Externalized via `kb quick`
- (Not needed for this session - tactical implementation of existing principle)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has findings and synthesis
- [x] Skills deployed and verified via grep
- [x] Git commits made to both repos
- [x] SYNTHESIS.md created
- [x] Ready for `orch complete orch-go-74o00`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should orchestrator skill be refactored to modular architecture like investigation skill? (Monolithic template is 1400+ lines, harder to maintain)
- Should we capture Dylan's tool experience responses in kb or CLAUDE.md for automatic surfacing in future sessions? (Current approach requires asking each time)
- Is there a way to make tool experience prompts more enforceable vs relying on agents reading guidance? (OpenCode plugin hook?)

**Areas worth exploring further:**
- Pattern analysis: Do modular skills scale better than monolithic for large skills? (Investigation=62 lines workflow vs Orchestrator=1400 lines template)
- Effectiveness measurement: Track how often spawned agents actually use the prompts in next 10-20 spawns

**What remains unclear:**
- Whether orchestrator SKILL.md.template edit approach is correct (AUTO-GENERATED warning vs git history of manual edits)

---

## Session Metadata

**Skill:** feature-impl
**Model:** sonnet
**Workspace:** `.orch/workspace/og-feat-add-tool-experience-10jan-f3f3/`
**Investigation:** `.kb/investigations/2026-01-10-inv-add-tool-experience-prompts-orchestrator.md`
**Beads:** `bd show orch-go-74o00`
