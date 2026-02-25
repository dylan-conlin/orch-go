# Session Synthesis

**Agent:** og-arch-investigate-orchestrator-architectural-20feb-efa9
**Issue:** orch-go-1158
**Outcome:** success

---

## Plain-Language Summary

When worker agents make architectural choices (like "cache locally for speed" vs "query directly for correctness"), no mechanism carries those tradeoffs to Dylan, who doesn't read code. The 6-week registry drift cycle (Dec 21 - Feb 18) happened because five separate cache-building attempts each seemed locally reasonable, but the "fast but drifts" tradeoff was never declared upfront or surfaced to the orchestrator.

This investigation found 7 recurring tradeoff classes in the codebase, 6 specific gaps where tradeoff information gets lost in the agent-to-orchestrator pipeline, and recommends a 3-layer defense: (1) add "Pressure Points" sections to architectural models so agents know upfront what's fragile, (2) add a required "Architectural Choices" section to SYNTHESIS.md so agents declare tradeoffs at decision time, (3) extend `orch complete` to parse and surface those tradeoffs to the orchestrator before closing. Together, these close the visibility gap that allowed the registry cycle to run for 6 weeks undetected.

## Verification Contract

See: `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- Design investigation: `.kb/investigations/2026-02-20-design-tradeoff-visibility-for-non-code-reading-orchestrator.md`
- Probe: `.kb/models/agent-lifecycle-state-model/probes/2026-02-20-tradeoff-visibility-gap-analysis.md`
- 3 blocking questions: orch-go-1159, orch-go-1160, orch-go-1161
- 3 implementation tasks: orch-go-1162 (model pressure points), orch-go-1163 (SYNTHESIS template), orch-go-1164 (completion surfacing)

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-20-design-tradeoff-visibility-for-non-code-reading-orchestrator.md` - Full architect investigation with 4 navigated forks
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-20-tradeoff-visibility-gap-analysis.md` - Probe extending the model with tradeoff visibility dimension
- `.orch/workspace/og-arch-investigate-orchestrator-architectural-20feb-efa9/SYNTHESIS.md` - This file
- `.orch/workspace/og-arch-investigate-orchestrator-architectural-20feb-efa9/VERIFICATION_SPEC.yaml` - Verification spec

### Files Modified
- None (design investigation only, no code changes)

---

## Evidence (What Was Observed)

- 7 tradeoff classes recur in this codebase (cache vs query, velocity vs verification, speed vs correctness, simplicity vs completeness, spawn modes, persistence boundaries, dedup strategies)
- The most damaging tradeoffs (#1 cache vs query, #4 velocity vs verification) share common DNA: choosing the fast path without declaring the correctness cost
- `VerifySynthesis()` in pkg/verify/check.go only checks file size > 0, never parses content
- The explain-back gate checks for non-empty text, not tradeoff comprehension
- Architect skill has excellent fork navigation format with `Trade-off accepted:` fields — but only fires when architect is spawned
- Models have Summary, Core Mechanism, Why This Fails, Constraints, Evolution, References — none designed for "what breaks if you change something"
- SYNTHESIS.md "Decisions Made" section captures tradeoffs but no gate reads it

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-20-design-tradeoff-visibility-for-non-code-reading-orchestrator.md` - Design investigation with layered defense recommendation

### Decisions Made
- Recommended 3-layer defense: model pressure points + SYNTHESIS architectural choices + completion surfacing
- Recommended skill-level gating (architect/feature-impl/systematic-debugging only) rather than universal gating
- Recommended structured table format for model pressure points (not freeform prose)

### Constraints Discovered
- SYNTHESIS.md content is never parsed by completion gates (only file size checked)
- bd comment protocol carries phase status only, not architectural content
- Model format lacks any mechanism for documenting architectural fragility

---

## Next (What Should Happen)

**Recommendation:** close (this investigation is complete)

Orchestrator should:
1. Review the investigation and answer Q3 (orch-go-1161): is the gap upstream (feature requests should flow through models) or at capture time (completion-time surfacing is sufficient)?
2. Answer Q1 (orch-go-1159): pilot pressure points on 2 models or all 11?
3. Spawn feature-impl for orch-go-1162 (model pressure points) once Q1 is resolved
4. Spawn feature-impl for orch-go-1163 (SYNTHESIS template) — can proceed independently
5. Spawn feature-impl for orch-go-1164 (completion surfacing) — depends on orch-go-1163

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-1158`

---

## Unexplored Questions

- Could `kb context` automatically match feature request descriptions against model pressure points at spawn time? (Layer 4, deferred pending Q3)
- Should the orchestrator skill include guidance on "when spawning work that touches a hot model, check pressure points first"?
- Would a `kb pressurepoints` command that lists all pressure points across all models be useful for orchestrator awareness?

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-investigate-orchestrator-architectural-20feb-efa9/`
**Investigation:** `.kb/investigations/2026-02-20-design-tradeoff-visibility-for-non-code-reading-orchestrator.md`
**Beads:** `bd show orch-go-1158`
