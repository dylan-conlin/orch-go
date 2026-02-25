# Session Synthesis

**Agent:** og-arch-orientation-frame-dylan-20feb-9c0f
**Issue:** orch-go-1157
**Outcome:** success

---

## Plain-Language Summary

Designed a verification level system (V0-V3) that gives Dylan and the orchestrator a shared vocabulary for "how much verification does this change need?" The current system has 14 completion gates that all fire by default, requiring skip flags or `--force` to bypass — leading to gate proliferation and `--force` becoming the happy path. The new design unifies three existing implicit level systems (spawn tier, checkpoint tier, skill-based auto-skips) into four named levels: V0 (Acknowledge — did agent finish?), V1 (Artifacts — are deliverables present?), V2 (Evidence — is there proof of testing?), V3 (Behavioral — did a human observe it working?). The level is declared at spawn time, inferred from skill+issue type with orchestrator override, and determines which gates fire at completion. Common case requires zero skip flags.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for evidence and gate support.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-20-inv-architect-verification-levels.md` - Full design investigation with V0-V3 taxonomy, gate mapping, 6 navigated forks, 3 blocking questions, implementation phasing
- `.kb/models/completion-verification/probes/2026-02-20-probe-verification-levels-design.md` - Probe confirming model staleness, documenting that "independent and cumulative" gates are actually level-selective via auto-skip logic

### Files Modified
- None (design investigation, no code changes)

---

## Evidence (What Was Observed)

- 14 gates inventoried from `pkg/verify/check.go` — all fire by default but 5-8 auto-skip per completion based on scattered heuristics
- Auto-skip logic distributed across 6 files with no centralized documentation
- Three implicit level systems already converge on ~4 natural levels
- Build gate (`go build`) is the only gate that ignores all level signals — fires regardless of tier/skill
- Checkpoint tier system (`TierForIssueType`) and spawn tier system (`SkillTierDefaults`) encode the same spectrum in different vocabularies

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-20-inv-architect-verification-levels.md` - Verification levels design
- `.kb/models/completion-verification/probes/2026-02-20-probe-verification-levels-design.md` - Model probe

### Decisions Made
- V0-V3 level taxonomy chosen over numeric (L0-L4) for human legibility
- Infer + override chosen over pure inference or pure declaration
- SkipConfig kept as escape hatch rather than removed
- Tradeoff visibility embedded in levels (not a separate gate) per orch-go-1158 conclusion

### Constraints Discovered
- Build gate must remain unconditional (a broken build should always be caught regardless of level)
- `--verify-level` should be spawn-only (not changeable at completion time) to maintain the "level declared upfront" invariant

---

## Next (What Should Happen)

**Recommendation:** close (design complete, implementation is separate work)

### If Close
- [x] Investigation has `**Phase:** Complete`
- [x] Probe has `Status: Complete`
- [x] All 6 forks navigated with substrate trace
- [x] 3 blocking questions surfaced (storage location, completion-time override, auto-elevation for web changes)
- [x] Implementation phasing documented (4 phases)

### Implementation Follow-up
When promoted to decision, implementation should follow the 4-phase plan in the investigation's Phasing section. Estimated as 4 separate feature-impl spawns (one per phase).

---

## Unexplored Questions

- **Should EscalationLevel (existing in escalation.go) be unified with VerifyLevel?** They cover different concerns (post-verification routing vs. pre-verification gate selection) but share the intuition of "how much attention does this need." Worth investigating whether they should be one concept.
- **What happens when an agent's work turns out to be higher-risk than the declared level?** E.g., investigation that accidentally modifies code. The level was V1 but should be V2. Current design handles this via SkipConfig (gates would fail, orchestrator uses skip flags). Future: completion could auto-detect and warn.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-orientation-frame-dylan-20feb-9c0f/`
**Investigation:** `.kb/investigations/2026-02-20-inv-architect-verification-levels.md`
**Beads:** `bd show orch-go-1157`
