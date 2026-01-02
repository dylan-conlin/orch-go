# Session Synthesis

**Agent:** og-work-synthesis-review-workflow-26dec
**Issue:** orch-go-untracked-1766782315
**Duration:** ~45 minutes
**Outcome:** success

---

## TLDR

Designed the synthesis review workflow to address the gap where orchestrators batch-close agents without extracting value from SYNTHESIS.md recommendations. The solution has three integration points: `orch complete` (single agent), `orch review done` (batch with prompts), and dashboard (visual review with issue creation).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-design-synthesis-review-workflow.md` - Design investigation with three-phase implementation plan

### Files Modified
- None (design session, no implementation)

### Commits
- None yet (investigation artifact created, needs commit)

---

## Evidence (What Was Observed)

- `runReviewDone()` in `review.go:516-674` closes issues without surfacing synthesis recommendations
- `verify.ParseSynthesis()` already extracts `NextActions`, `Recommendation`, `AreasToExplore`, `Uncertainties`
- Dashboard has `POST /api/issues` endpoint for creating follow-up issues
- Constraint exists: "orch complete must verify SYNTHESIS.md exists and is not placeholder before closing"
- Decision exists: "orch complete prompts for each SYNTHESIS.md recommendation individually"

### Tests Run
```bash
# Explored codebase structure
/tmp/orch-test review --help
/tmp/orch-test complete --help
# Read existing investigation artifacts for synthesis patterns
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-design-synthesis-review-workflow.md` - Design for synthesis review experience

### Decisions Made
- Three integration points needed: CLI single, CLI batch, dashboard
- Batch operations are the primary value leak - orchestrators prefer `review done` over individual `complete`
- Review state tracking enables "unreviewed" badges and prevents re-prompting

### Constraints Discovered
- Must not block batch-close (too much friction) - use prompts with skip option
- Should share review state across CLI and dashboard

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

**Issue 1:** Add recommendation prompts to `orch review done`
**Skill:** feature-impl
**Context:**
```
Modify runReviewDone() in review.go to prompt for each synthesis recommendation
before closing. Add --no-prompt flag for automated batch-close escape hatch.
Reference: .kb/investigations/2025-12-26-design-synthesis-review-workflow.md Phase 1.
```

**Issue 2:** Add review state tracking to synthesis
**Skill:** feature-impl
**Context:**
```
Add .review-state.json to workspace tracking which recommendations were
acted on vs dismissed. Enables dashboard to show "unreviewed" badges.
Reference: .kb/investigations/2025-12-26-design-synthesis-review-workflow.md Phase 2.
```

**Issue 3:** Add dashboard "Pending Reviews" section
**Skill:** feature-impl
**Context:**
```
Create dedicated dashboard section showing unreviewed synthesis recommendations
with "Create Issue" and "Dismiss" buttons. Not buried in agent detail.
Reference: .kb/investigations/2025-12-26-design-synthesis-review-workflow.md Phase 3.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should escalations in synthesis block batch-close entirely? (probably yes)
- How to handle stale recommendations from agents completed days ago? (expiration?)
- Should dashboard aggregate synthesis from all projects or just current?

**Areas worth exploring further:**
- Whether `orch complete` actually implements recommendation prompts (verify the constraint is implemented)
- Integration with daemon for auto-surfacing recommendations at spawn time

**What remains unclear:**
- Optimal UX for many recommendations (>5 items) - pagination vs summary
- How to detect duplicate recommendations that are already beads issues

---

## Session Metadata

**Skill:** design-session
**Model:** claude (via opencode)
**Workspace:** `.orch/workspace/og-work-synthesis-review-workflow-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-design-synthesis-review-workflow.md`
**Beads:** `bd show orch-go-untracked-1766782315`
