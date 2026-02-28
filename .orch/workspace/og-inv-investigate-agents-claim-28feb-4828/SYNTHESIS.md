# Session Synthesis

**Agent:** og-inv-investigate-agents-claim-28feb-4828
**Issue:** orch-go-4czz
**Outcome:** success

---

## Plain-Language Summary

All three agents that failed the synthesis gate on Feb 28 (orch-go-i9qr, orch-go-vem4, orch-go-amty) were correctly following their instructions. The spawn system told them "SYNTHESIS.md is NOT required" because they were feature-impl agents assigned light tier. But the verification system (`orch complete`) checked them at verify_level=V2, which includes the synthesis gate. This is a known contradiction between the tier system and the verify-level system — identified in a Feb 27 probe (orch-go-i9qi) but never fixed. Every feature-impl completion either requires `--skip-synthesis` or fails.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for evidence.

---

## TLDR

3/3 feature-impl completions on Feb 28 failed because the tier system (light=no SYNTHESIS.md) and verify-level system (V2=requires SYNTHESIS.md) disagree. Agents were told one thing, verified against another. This is an architectural inconsistency in orch-go's spawn/verify pipeline, not an agent behavior problem.

---

## Delta (What Changed)

### Files Created
- `.kb/models/completion-verification/probes/2026-02-28-probe-synthesis-gate-light-tier-empirical-failures.md` - Probe confirming Feb 27 prediction with Feb 28 empirical evidence

### Commits
- N/A (investigation only, no code changes)

---

## Evidence (What Was Observed)

- All 3 agents had `tier: "light"` in .tier file AND `verify_level: "V2"` in AGENT_MANIFEST.json
- All 3 SPAWN_CONTEXT.md files had 5+ mentions of "SYNTHESIS.md is NOT required"
- All 3 reported detailed, legitimate Phase: Complete comments with test evidence
- All 3 workspaces had no SYNTHESIS.md (agents followed instructions)
- `pkg/verify/check.go:551` runs `ShouldRunGate(verifyLevel, GateSynthesis)` — uses verify_level, ignores tier
- `pkg/spawn/config.go` maps `feature-impl → TierLight`
- `pkg/spawn/verify_level.go` maps `feature-impl → VerifyV2`
- Feb 27 probe (orch-go-i9qi) identified this exact conflict — no fix was applied

---

## Architectural Choices

No architectural choices — investigation only. Two fix options recommended for architect:
1. Add `&& tier != "light"` to synthesis gate (quick fix, like line 388 does for architectural choices)
2. Align tier defaults with verify levels — change feature-impl to TierFull since V2 requires synthesis

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Tier and verify_level must agree on SYNTHESIS.md requirement (currently they don't for feature-impl)

### Externalized via `kb`
- `kb quick constrain` (kb-596b66) — tier/verify_level disagreement constraint

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix tier/verify_level SYNTHESIS.md disagreement for feature-impl
**Skill:** architect
**Context:**
```
Tier system says feature-impl=light (no SYNTHESIS.md). Verify-level says feature-impl=V2 (synthesis gate fires).
Fix the contradiction. See probes: 2026-02-27-probe-light-tier-v2-verification-conflict.md and
2026-02-28-probe-synthesis-gate-light-tier-empirical-failures.md
```

---

## Unexplored Questions

- How many total feature-impl completions have required --skip-synthesis since V2 was introduced?
- Should the tier system be fully retired in favor of verify levels?
- Are there other gates with similar tier/level disagreements besides synthesis and architectural choices?

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-agents-claim-28feb-4828/`
**Probe:** `.kb/models/completion-verification/probes/2026-02-28-probe-synthesis-gate-light-tier-empirical-failures.md`
**Beads:** `bd show orch-go-4czz`
