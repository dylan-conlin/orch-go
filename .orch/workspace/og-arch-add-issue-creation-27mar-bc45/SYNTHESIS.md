# Session Synthesis

**Agent:** og-arch-add-issue-creation-27mar-bc45
**Issue:** orch-go-aqq5a
**Duration:** 2026-03-27
**Outcome:** success

---

## Plain-Language Summary

The architect skill was creating follow-up issues via `bd create` without ever checking whether the proposed work had already been committed by another agent. The daemon's CommitDedupGate catches these duplicates at spawn-time, but by then the issue already exists in beads as a zombie — created, never spawned, never closed. I added a "Prior Art Check" procedure to the architect skill template: before every `bd create`, agents now search recent git commits and open issues for matching work. This moves dedup upstream from spawn-time (damage control) to creation-time (prevention).

## Verification Contract

See `VERIFICATION_SPEC.yaml` for acceptance criteria. Key outcomes:
- Architect SKILL.md.template updated with Prior Art Check (Phase 5d)
- Deployed via `skillc deploy`
- Architect model updated with probe findings
- Follow-up issue created for worker-base (governance-protected)

---

## TLDR

Added issue-creation-time dedup to architect skill — a "Prior Art Check" procedure (git log + bd list) before every `bd create` call. This prevents zombie issues from being created for work already committed by other agents.

---

## Delta (What Changed)

### Files Modified
- `skills/src/worker/architect/.skillc/SKILL.md.template` - Added "5d. Prior Art Check" section, renumbered subsequent sections (5e → Decomposition, 5f → Commit Artifacts), wired prior art check into decomposition steps
- `.kb/models/architect/model.md` - Updated implications (issue creation dedup), evidence table, and probes section

### Files Created
- `.kb/models/architect/probes/2026-03-27-probe-issue-creation-dedup-effectiveness.md` - Probe documenting the gap and fix
- `.kb/investigations/2026-03-27-design-issue-creation-dedup.md` - Design investigation with full analysis

---

## Evidence (What Was Observed)

- Architect SKILL.md.template had 4 `bd create` call sites with zero dedup guidance
- CommitDedupGate (pkg/daemon/prior_art_dedup.go:47-124) operates at spawn-time only
- Worker-base discovered-work.md has the same gap but is governance-protected
- Duplicate extraction provenance trace (2026-02-16) documented 9+ zombie issues from the create-then-filter pattern

---

## Architectural Choices

### Prior Art Check as skill text vs daemon code
- **What I chose:** Skill-level instructional guidance (procedure in SKILL.md.template)
- **What I rejected:** New daemon spawn gate or bd create hook
- **Why:** Task explicitly scoped as skill-level concern; instructional guidance follows established pattern for skill constraints; agent compliance with loaded skill text is high
- **Risk accepted:** Soft gate — agents could skip the check. Mitigated by CommitDedupGate as backstop.

### Placement in Phase 5d vs separate section
- **What I chose:** New Phase 5d before Decomposition (renumbered to 5e)
- **What I rejected:** Inline in decomposition, footnote, or appendix
- **Why:** Phase ordering makes it procedurally clear — run the check BEFORE creating issues. Separate section makes it referenceable from both multi-component and single-component paths.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-27-design-issue-creation-dedup.md` - Design investigation
- `.kb/models/architect/probes/2026-03-27-probe-issue-creation-dedup-effectiveness.md` - Probe

### Decisions Made
- Prior Art Check lives in architect skill template (not daemon code) because it's a skill-level concern
- Check uses git log (7-day window, 30 commits) + bd list (open issues) as dual verification

### Constraints Discovered
- Worker-base is governance-protected — broader rollout requires orchestrator direct session

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has Phase: Complete
- [x] Probe merged into model
- [x] Follow-up issue created (orch-go-95x1b for worker-base)
- [x] Ready for `orch complete orch-go-aqq5a`

---

## Unexplored Questions

- What is the false positive rate of git log keyword matching? Keywords like "spawn" or "fix" may match unrelated commits. May need tighter matching criteria after observation.
- Should question entities (Phase 3 `bd create`) also get the prior art check? Currently only implementation issues are covered.

---

## Friction

No friction — smooth session.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-add-issue-creation-27mar-bc45/`
**Investigation:** `.kb/investigations/2026-03-27-design-issue-creation-dedup.md`
**Beads:** `bd show orch-go-aqq5a`
