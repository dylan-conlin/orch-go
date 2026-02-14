# Session Synthesis

**Agent:** og-arch-design-solution-model-14feb-6846
**Issue:** orch-go-rdd
**Duration:** 2026-02-14 → 2026-02-14
**Outcome:** success

---

## TLDR

Designed solution for model artifact staleness: **Detect-Annotate-Queue** using git-diff-based detection at spawn time. Produced decision record (`.kb/decisions/2026-02-14-model-staleness-detection.md`) and investigation with 5 findings, 4 design forks navigated, and 3-phase implementation plan.

---

## Delta (What Changed)

### Files Created
- `.kb/decisions/2026-02-14-model-staleness-detection.md` - Decision record with recommended approach, implementation plan, and tradeoffs
- `.kb/investigations/2026-02-14-inv-design-solution-model-artifact-staleness.md` - Full investigation with evidence and fork analysis

### Files Modified
- None (design-only session)

### Commits
- (pending - will commit before completion)

---

## Evidence (What Was Observed)

- 12+ stale file references found across 24 models (50% staleness rate) - verified by cross-referencing model paths against filesystem
- `kb reflect --type stale` returns 0 results for models - only checks decisions
- `pkg/spawn/kbcontext.go` serves models without any freshness check
- Models already reference code in "Primary Evidence" sections but in 4 inconsistent, unparseable formats
- TEMPLATE.md has `{file path}:{lines}` format but models don't follow it consistently
- 6 principles (Evidence Hierarchy, Gate Over Remind, Infrastructure Over Instruction, Surfacing Over Browsing, Capture at Context, Verification Bottleneck) directly constrain the solution design

### Tests Run
```bash
# No code changes - design session only
# Verified staleness detection gap:
kb reflect --type stale
# Result: "No stale opportunities found." (doesn't cover models)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/decisions/2026-02-14-model-staleness-detection.md` - Detect-Annotate-Queue decision
- `.kb/investigations/2026-02-14-inv-design-solution-model-artifact-staleness.md` - Full analysis

### Decisions Made
- **Detect-Annotate-Queue** over Block, Periodic-only, Auto-fix, Git hooks, or Bidirectional linkage
  - Annotate (not block) because models are approximately correct even when stale
  - Spawn-time (not periodic) because Surfacing Over Browsing principle
  - Queue (not auto-fix) because Verification Bottleneck principle
- **code_refs HTML comment markers** over external manifest or content-derived references
  - Self-describing, machine-parseable, backward-compatible, invisible in rendered markdown
- **3-phase implementation**: (1) convention, (2) spawn-time detection, (3) kb reflect integration

### Constraints Discovered
- Models reference code in 4 inconsistent formats - formalization needed before automation
- `kb reflect` only supports decisions for staleness, not models
- Performance budget: staleness check must be <200ms to not slow spawn
- Auto-fix violates Understanding Through Engagement (models require orchestrator synthesis)

### Externalized via `kn`
- N/A (findings captured in investigation and decision records)

---

## Verification Contract

Verification specification not applicable (design-only session, no code changes). The decision record includes success criteria for each implementation phase.

---

## Next (What Should Happen)

**Recommendation:** close + spawn-follow-up

### If Close
- [x] All deliverables complete (decision record + investigation)
- [x] N/A - no tests (design session)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-rdd`

### If Spawn Follow-up

**Issue 1:** "Backfill code_refs blocks in all 24 models"
**Skill:** feature-impl
**Context:**
```
Add <!-- code_refs: --> / <!-- /code_refs --> markers to all 24 .kb/models/*.md files.
Extract file paths from existing "Primary Evidence" sections, verify against filesystem.
See .kb/decisions/2026-02-14-model-staleness-detection.md Phase 1.
```

**Issue 2:** "Implement spawn-time model staleness detection"
**Skill:** feature-impl
**Context:**
```
Add staleness check to pkg/spawn/kbcontext.go. Parse code_refs from model files,
check git log --since={Last Updated}, prepend annotation if stale.
See .kb/decisions/2026-02-14-model-staleness-detection.md Phase 2.
```

**Issue 3:** "Add --type model-drift to kb reflect"
**Skill:** feature-impl
**Context:**
```
Extend kb reflect to detect model-code drift. Scan .kb/models/ code_refs blocks,
check referenced files against git history.
See .kb/decisions/2026-02-14-model-staleness-detection.md Phase 3.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch complete` warn when an agent modifies code referenced by a model? (Would catch drift at source)
- How should cross-repo model references work? (e.g., opencode-fork.md references opencode codebase)
- What is the false positive rate when using file-level detection vs semantic detection?

**Areas worth exploring further:**
- Performance profiling of git log --since at spawn time with 10+ file references
- Whether agents actually adjust behavior based on staleness annotations
- Optimal cadence for kb reflect model-drift sweeps

**What remains unclear:**
- Whether staleness check should live in orch-go (kbcontext.go) or upstream in kb-cli
- How to handle models that reference non-file evidence (architecture diagrams, behavior descriptions)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-solution-model-14feb-6846/`
**Investigation:** `.kb/investigations/2026-02-14-inv-design-solution-model-artifact-staleness.md`
**Beads:** `bd show orch-go-rdd`
