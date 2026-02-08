<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** kb-reflect skill enhanced with structured Proposed Actions section that transforms findings into actionable proposals.

**Evidence:** Built and deployed skill successfully; new section includes Archive, Create, Promote, and Update action tables with approval checkboxes.

**Knowledge:** Gate-over-remind pattern applied to kb reflect - agent now produces proposals instead of just reports, reducing orchestrator effort from reading all findings to approving/rejecting structured actions.

**Next:** None - implementation complete, skill deployed.

**Confidence:** High (90%) - skill built and deployed, structure designed to match stated requirements.

---

# Investigation: Enhance Kb Reflect Skill Propose

**Question:** How to enhance kb-reflect skill to produce actionable proposals instead of just reports?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Current skill is procedure manual without structured output

**Evidence:** Reviewed `/Users/dylanconlin/orch-knowledge/skills/src/worker/kb-reflect/SKILL.md` - contains decision trees for each finding type (synthesis, promote, stale, drift, open) but output is free-form markdown with dispositions logged but not structured for approval.

**Source:** `SKILL.md` lines 1-374, triage-template.md, completion.md

**Significance:** Agent does the work of triaging but orchestrator must re-read all findings to understand what actions to take. This violates "surfacing over browsing" principle.

---

### Finding 2: Decision trees already exist for converting findings to actions

**Evidence:** Each finding type has a clear decision tree that leads to specific actions:
- synthesis → archive/consolidate/keep
- promote → promote to kb/add to CLAUDE.md
- stale → archive/refresh/keep
- drift → fix practice/update constraint/investigate
- open → create issue/close/update status

**Source:** decision-tree-*.md files in .skillc directory

**Significance:** The logic for determining actions already exists - just need structured output format for proposals.

---

### Finding 3: Approval workflow missing from current skill

**Evidence:** Current completion checklist only requires "Actions for actionable items logged" without structure for orchestrator approval before execution.

**Source:** completion.md lines 1-24

**Significance:** Need approval checkboxes and structured tables so orchestrator can mark `[x]` to approve without re-reading raw findings.

---

## Synthesis

**Key Insights:**

1. **Gate over remind applied to triage** - Agent now gates on producing proposals, not just reports. Orchestrator reviews proposals, not raw findings.

2. **Four action types cover all cases** - Archive (remove/supersede), Create (decision/guide/issue), Promote (kn to kb), Update (modify existing) handle all finding dispositions.

3. **Table format enables quick approval** - ID/Target/Reason/Approved columns let orchestrator scan and approve without context switching.

**Answer to Investigation Question:**

Added structured "Proposed Actions" section with four tables (Archive, Create, Promote, Update). Each proposal includes ID, target/type, reason, and approval checkbox. Agent now produces proposals that orchestrator can approve/reject with minimal reading.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Skill built and deployed successfully. Structure matches requirements from beads issue. Not 95%+ because untested with real kb reflect output.

**What's certain:**

- ✅ Skill compiles with skillc build
- ✅ Skill deploys to ~/.claude/skills
- ✅ Structure includes all required elements (finding type, action, why, draft issue)

**What's uncertain:**

- ⚠️ Real-world usage not tested
- ⚠️ Table format may need adjustment after first use

**What would increase confidence to Very High:**

- Test with actual kb reflect output
- Orchestrator confirms approval workflow works smoothly

---

## Implementation Details

**Files modified in orch-knowledge:**
- `skills/src/worker/kb-reflect/.skillc/proposed-actions.md` - NEW - Proposed Actions section definition
- `skills/src/worker/kb-reflect/.skillc/triage-template.md` - Updated with Proposed Actions tables
- `skills/src/worker/kb-reflect/.skillc/completion.md` - Updated with proposal verification
- `skills/src/worker/kb-reflect/.skillc/skill.yaml` - Added proposed-actions.md to sources
- `skills/src/worker/kb-reflect/SKILL.md` - Rebuilt with new content

---

## References

**Files Examined:**
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/kb-reflect/SKILL.md` - Current skill definition
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/kb-reflect/.skillc/*` - Skill source files

**Commands Run:**
```bash
# Build skill
cd /Users/dylanconlin/orch-knowledge/skills && skillc build src/worker/kb-reflect

# Verify Proposed Actions section
grep -n "Proposed Actions" src/worker/kb-reflect/SKILL.md
```

---

## Investigation History

**2025-12-25 20:15:** Investigation started
- Initial question: How to enhance kb-reflect skill to produce actionable proposals?
- Context: Beads issue orch-go-o2r8 requested agent produce proposals, not just reports

**2025-12-25 20:30:** Implementation complete
- Created proposed-actions.md with structured proposal format
- Updated triage-template.md with proposal tables
- Updated completion.md with proposal verification
- Built and deployed skill
- Final confidence: High (90%)
- Status: Complete
