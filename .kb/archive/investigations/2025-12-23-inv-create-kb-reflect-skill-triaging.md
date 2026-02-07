<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created kb-reflect skill with decision trees for all 5 finding types (synthesis, promote, stale, drift, open) and scheduling guidance for knowledge hygiene.

**Evidence:** Analyzed pkg/daemon/reflect.go structure, ran kb reflect to see 19 synthesis + 13 open findings, reviewed existing skills for proper format.

**Knowledge:** KB reflect has 5 distinct finding types requiring different triage approaches; proper investigation closure requires D.E.K.N. + explicit Next: disposition.

**Next:** Close - skill complete and ready for use.

**Confidence:** High (85%) - skill follows established patterns but hasn't been tested with actual triage session.

---

# Investigation: Create KB Reflect Skill for Triaging

**Question:** How should the kb-reflect skill guide agents through triaging kb reflect output?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None - skill created and documented
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: KB Reflect Outputs 5 Distinct Finding Types

**Evidence:** From `pkg/daemon/reflect.go:13-66`:
- synthesis: Topics with 3+ investigations needing consolidation
- promote: kn entries worth promoting to kb decisions
- stale: Decisions with 0 citations, >7 days old
- drift: CLAUDE.md constraints diverging from practice
- open: Investigations with unimplemented Next: actions

**Source:** `pkg/daemon/reflect.go:31-58` (type definitions), `kb reflect --help` output

**Significance:** Each finding type requires a distinct decision tree - they are not interchangeable. A synthesis finding needs consolidation logic, while a stale finding needs archive-vs-refresh logic.

---

### Finding 2: Current KB Reflect Output Shows Significant Backlog

**Evidence:** Running `kb reflect --format json` returned:
- 19 synthesis opportunities (topics with 3+ investigations)
- 13 open investigations with pending actions
- Topics like "orch" (21 investigations), "test" (17), "implement" (17)

**Source:** `kb reflect --format json` command output

**Significance:** The orch-go project has accumulated significant knowledge debt. This validates the need for a structured triage skill. The open investigations with placeholder titles ("[Investigation Title]") suggest spawn/completion issues worth separate investigation.

---

### Finding 3: Investigation Closure Has Specific Requirements

**Evidence:** From existing investigation skill (`~/.claude/skills/worker/investigation/SKILL.md:73-115`):
- D.E.K.N. summary required (Delta, Evidence, Knowledge, Next)
- Status: field must be set to Complete
- Next: field must have explicit disposition
- File must be committed

**Source:** `~/.claude/skills/worker/investigation/SKILL.md`, investigation template

**Significance:** Proper closure is a key discipline. The "open" finding type specifically catches investigations that weren't properly closed (have Next: action but Status: not Complete).

---

## Synthesis

**Key Insights:**

1. **Five-type taxonomy is fundamental** - The skill must handle each finding type distinctly because they require different actions (consolidate vs promote vs archive vs fix vs close).

2. **Scheduling enables proactive hygiene** - Session-start quick checks prevent drift accumulation; weekly full triage prevents synthesis backlog.

3. **Investigation closure is a gatekeeping function** - The open finding type exists precisely because investigations often aren't properly closed. The skill provides explicit closure procedure to prevent this.

**Answer to Investigation Question:**

The kb-reflect skill should provide:
1. Clear scheduling guidance (when to run which checks)
2. Decision trees for each of the 5 finding types with explicit actions
3. Consolidation vs archiving criteria for synthesis findings
4. Proper investigation closure procedure (D.E.K.N. + Next: + Status: + commit)
5. Triage session template for structured review

The skill has been created at `~/.claude/skills/worker/kb-reflect/SKILL.md` with all of these components.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The skill follows established patterns from other worker skills (investigation, codebase-audit) and addresses the specific requirements from the task. The decision trees are comprehensive and based on actual kb reflect output analysis.

**What's certain:**

- The 5 finding types and their meanings (from reflect.go source)
- The decision tree logic (based on practical triage needs)
- The scheduling recommendations (based on knowledge hygiene principles)

**What's uncertain:**

- Whether the skill will be discovered by Claude when needed (description optimization)
- Whether agents will follow the decision trees under pressure (no TDD testing done)
- Edge cases in synthesis consolidation (when exactly to archive vs consolidate)

**What would increase confidence to Very High (95%+):**

- Run an actual kb-reflect triage session using the skill
- Test skill discovery with different prompts
- Get feedback from orchestrator on completeness

---

## Implementation Recommendations

**Purpose:** N/A - implementation complete.

### Recommended Approach (Completed)

Created spawnable procedure skill at `~/.claude/skills/worker/kb-reflect/SKILL.md` with:
- Scheduling table (when to run)
- Decision trees for all 5 finding types
- Consolidation vs archiving guidance
- Investigation closure procedure
- Triage session template
- Self-review checklist

---

## References

**Files Examined:**
- `pkg/daemon/reflect.go` - ReflectSuggestions struct and type definitions
- `~/.claude/skills/worker/investigation/SKILL.md` - Investigation closure patterns
- `~/.claude/skills/meta/writing-skills/SKILL.md` - Skill creation guidance
- `~/.claude/skills/meta/writing-skills/phases/2-GREEN.md` - Skill structure guidelines

**Commands Run:**
```bash
# Analyzed kb reflect output
kb reflect --format json

# Checked available commands
kb reflect --help
kb chronicle --help

# Created skill directory
mkdir -p ~/.claude/skills/worker/kb-reflect

# Created symlink
ln -sf worker/kb-reflect ~/.claude/skills/kb-reflect
```

**Related Artifacts:**
- **Skill:** `~/.claude/skills/worker/kb-reflect/SKILL.md` - Created deliverable
- **Workspace:** `.orch/workspace/og-work-create-kb-reflect-23dec/SYNTHESIS.md`

---

## Investigation History

**2025-12-23 17:00:** Investigation started
- Initial question: How should kb-reflect skill guide triage of kb reflect output?
- Context: Task from orchestrator to create spawnable knowledge hygiene skill

**2025-12-23 17:10:** Analyzed existing kb reflect infrastructure
- Found 5 finding types in pkg/daemon/reflect.go
- Ran kb reflect to see actual output

**2025-12-23 17:15:** Reviewed skill creation patterns
- Read writing-skills phases for proper structure
- Examined investigation skill for closure patterns

**2025-12-23 17:20:** Created kb-reflect skill
- Wrote decision trees for all 5 finding types
- Added scheduling guidance and closure procedure

**2025-12-23 17:25:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: kb-reflect skill created with comprehensive triage guidance

---

## Self-Review

- [x] Real test performed (analyzed actual kb reflect output)
- [x] Evidence concrete (specific finding types, counts)
- [x] Conclusion factual (skill addresses identified requirements)
- [x] No speculation (all guidance based on observed patterns)
- [x] Question answered (skill created with all requested components)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled (summary section complete)
- [x] NOT DONE claims verified (N/A - this is creation not verification)

**Self-Review Status:** PASSED
