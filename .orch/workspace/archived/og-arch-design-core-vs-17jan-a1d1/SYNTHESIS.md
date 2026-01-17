# Session Synthesis

**Agent:** og-arch-design-core-vs-17jan-a1d1
**Issue:** orch-go-jiolc
**Duration:** 2026-01-17 (start) → 2026-01-17 (complete)
**Outcome:** success

---

## TLDR

Designed Core vs Reference split criteria for investigation skill progressive disclosure. Defined four-dimension test (Frequency, Stage, Type, Complexity) for content placement decisions. Investigation skill can reduce from 335 to 130-150 lines (50-60% reduction) using situation-based splitting.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-design-core-vs-reference-split.md` - Design investigation with four-dimension test criteria, reference file structure, and implementation recommendations

### Files Modified
- None (this was a design/architecture session, no code changes)

### Commits
- `b929f4a5` - architect: design core vs reference split for investigation skill

---

## Evidence (What Was Observed)

### Feature-impl Precedent
- Feature-impl achieved 77% reduction (1757 → 400 lines) using phase-based progressive disclosure
- Core SKILL.md: 458 lines with phase summaries (15-20 lines each)
- Reference files: 11 files totaling 1,963 lines with detailed phase workflows
- Discovery mechanism: Direct file path references like `~/.claude/skills/worker/feature-impl/reference/phase-investigation.md`
- Source: `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/`

### Investigation Skill Analysis
- Current structure: 335 lines total
- Essential workflow + discipline: ~135-150 lines
- Examples, templates, edge cases: ~185-200 lines
- Usage pattern: Situation-based (not phase-based like feature-impl)
- Source: `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/SKILL.md`

### Discovery Mechanism
- No custom tooling in pkg/skills/ for loading references
- Agents use standard Read tool to load reference files
- Direct file path references prevent "agents don't know reference docs exist" problem
- Source: Verification via grep/search of orch-go codebase

### Tests Run
```bash
# Verified feature-impl line counts
wc -l ~/orch-knowledge/skills/src/worker/feature-impl/SKILL.md ~/orch-knowledge/skills/src/worker/feature-impl/reference/*.md
# Result: 458 core + 1,963 reference

# Analyzed investigation skill structure
grep -n "^##" ~/orch-knowledge/skills/src/worker/investigation/SKILL.md
wc -l ~/orch-knowledge/skills/src/worker/investigation/SKILL.md
# Result: 335 lines with identifiable section boundaries
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-design-core-vs-reference-split.md` - Complete design with criteria, structure, and recommendations

### Decisions Made
- **Four-dimension test for Core vs Reference:** Content goes to Reference if it meets ANY of: (1) Sometimes needed not always, (2) Needed in specific situations not upfront, (3) Examples/templates not principles, (4) Complex detail not simple workflow
- **Situation-based splitting for investigation:** Unlike feature-impl's phase-based split, investigation needs situation-based split (error recovery, examples, templates loaded on-demand)
- **Five reference files recommended:** error-recovery.md, examples.md, template.md, self-review-guide.md, leave-it-better.md
- **Discovery mechanism: Direct file path references** - No custom tooling, just `**Reference:** See path/to/file.md` pattern

### Constraints Discovered
- Different skills need different splitting strategies - must match optimization pattern to usage pattern
- Investigation has uniform workflow (unlike feature-impl's conditional phases) requiring different approach
- Core must pass "needed by every investigation at start" test

### Key Insights
1. **Pattern matching matters:** Feature-impl uses phase-based splitting (89% of spawns use only 2-3 phases). Investigation needs situation-based splitting (all follow same workflow, need different detail at different times).

2. **Four-dimension test provides defensible criteria:** Instead of ad-hoc "this feels right" decisions, the test (Frequency, Stage, Type, Complexity) gives clear rationale for every content placement.

3. **Discovery is dead simple:** Direct file path references require zero infrastructure. Agents already know how to read files.

4. **Target 50-60% reduction:** From 335 to 130-150 lines Core + 185-200 lines Reference. Less aggressive than feature-impl's 77% but adapted to investigation's needs.

### Externalized via `kb`
- Investigation file created with recommendation to promote four-dimension test to decision
- No `kb quick` commands run (full investigation documented instead)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete - Investigation file with design criteria and recommendations
- [x] Investigation file has `**Phase:** Complete` 
- [x] SYNTHESIS.md created
- [x] Ready for `orch complete orch-go-jiolc`

**Follow-up work (separate issues):**
1. **Implement progressive disclosure for investigation skill** - Apply the four-dimension test to create Core + Reference split
2. **Consider promoting four-dimension test to decision** - Pattern is generalizable to future skill optimization

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- **Optimal reference file granularity:** Is 4-5 focused files better than 1-2 larger files for agent usability? Need empirical testing with agents reading references.

- **Reference access patterns:** Do agents read reference docs when needed, or do they try to work without them? Feature-impl has same uncertainty (90% confidence, awaiting behavior validation).

- **Dynamic reference injection at spawn time:** Could orch-go inject relevant reference content based on spawn context? Would add complexity but might improve agent experience.

- **Cross-skill reference sharing:** Could multiple skills share common reference docs (e.g., error-recovery patterns)? Or does each skill need its own?

**Areas worth exploring further:**

- Apply same pattern to codebase-audit skill (1514 lines - similar bloat as feature-impl had)
- Test investigation skill implementation with real spawns to validate reference reading behavior
- Consider if other skills (systematic-debugging, architect) would benefit from progressive disclosure

**What remains unclear:**

- Whether 130-150 line Core is sufficient or too sparse (needs implementation testing)
- If examples are truly "sometimes needed" or if they help even when not strictly necessary
- Maintenance burden of 5 reference files vs simplicity of 1 large Core

---

## Session Metadata

**Skill:** architect (autonomous mode)
**Model:** Claude (via OpenCode)
**Workspace:** `.orch/workspace/og-arch-design-core-vs-17jan-a1d1/`
**Investigation:** `.kb/investigations/2026-01-17-inv-design-core-vs-reference-split.md`
**Beads:** `bd show orch-go-jiolc`
