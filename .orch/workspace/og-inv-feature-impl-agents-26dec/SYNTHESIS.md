# Session Synthesis

**Agent:** og-inv-feature-impl-agents-26dec
**Issue:** orch-go-q9tk
**Duration:** 2025-12-26 ~30 minutes
**Outcome:** success

---

## TLDR

Feature-impl agents not producing SYNTHESIS.md is BY DESIGN - the tier system intentionally assigns feature-impl to "light" tier which skips synthesis. The gap is in review tooling (`/api/pending-reviews`, dashboard) which only scans for SYNTHESIS.md, making light tier completions invisible.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-feature-impl-agents-not-producing.md` - Investigation documenting the tier system and review tooling gap

### Files Modified
- None (investigation only)

### Commits
- (pending) Investigation file with comprehensive findings

---

## Evidence (What Was Observed)

- Both example workspaces (`og-feat-debounce-gold-processing-26dec`, `og-feat-fix-duplicate-key-26dec`) have `.tier` file containing "light"
- `pkg/spawn/config.go:31` explicitly maps `"feature-impl": TierLight`
- SPAWN_CONTEXT.md in both workspaces says "⚡ LIGHT TIER: SYNTHESIS.md is NOT required"
- `cmd/orch/serve.go:2355-2359` skips workspaces without SYNTHESIS.md in pending reviews endpoint
- Feature-impl skill (390 lines) has no mention of SYNTHESIS.md

### Tests Run
```bash
# Verified tier values
cat .orch/workspace/og-feat-debounce-gold-processing-26dec/.tier
# Output: light

cat .orch/workspace/og-feat-fix-duplicate-key-26dec/.tier
# Output: light
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-feature-impl-agents-not-producing.md` - Complete investigation with findings, evidence, and recommendations

### Decisions Made
- This is working as designed; the tier system is intentional

### Constraints Discovered
- Review tooling only scans for SYNTHESIS.md, creating blind spot for light tier completions
- Light tier exists to reduce overhead for implementation-focused work

### Externalized via `kn`
- (Recommend orchestrator run): `kn decide "Feature-impl tier gap requires tooling update" --reason "Review tooling only scans for SYNTHESIS.md, light tier completions are invisible"`

---

## Next (What Should Happen)

**Recommendation:** escalate

This investigation reveals a design decision is needed:

### If Escalate
**Question:** Should we update review tooling to handle light tier, or change feature-impl to full tier?

**Options:**
1. **Option A: Update review tooling** - Modify `/api/pending-reviews` to detect light tier completions via `.tier` + `Phase: Complete` in beads comments
   - Pros: Preserves tier system, less overhead for quick tasks
   - Cons: More complex detection logic

2. **Option B: Change feature-impl to full tier** - Change `pkg/spawn/config.go:31` from `TierLight` to `TierFull`
   - Pros: All completions produce synthesis, simpler review logic
   - Cons: Adds overhead to quick implementation tasks, reverses deliberate design

**Recommendation:** Option A - The tier system was deliberately implemented with tests. Better to complete the system by updating review tooling rather than reverting the design decision.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch review` CLI have the same gap? (likely yes, worth checking)
- What percentage of spawns are light vs full tier? (usage data question)

**Areas worth exploring further:**
- Whether light tier agents SHOULD produce some minimal completion artifact
- Whether beads comments are sufficient as completion evidence

**What remains unclear:**
- Original reasoning for tier split (no decision record found)

---

## Session Metadata

**Skill:** investigation
**Model:** Claude
**Workspace:** `.orch/workspace/og-inv-feature-impl-agents-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-feature-impl-agents-not-producing.md`
**Beads:** `bd show orch-go-q9tk`
