# SYNTHESIS: LIGHT Tier / V2 Verification Level Conflict Resolution

## Plain-Language Summary

Every feature-impl agent gets told "you don't need SYNTHESIS.md" (because it's LIGHT tier), but then `orch complete` fails with a missing SYNTHESIS.md error (because V2 verification requires it). The orchestrator has to use `--skip-synthesis` every time — violating the V0-V3 design goal of "zero skip flags for well-configured spawns." The root cause is that the V0-V3 level system was designed to replace the tier system, but the migration was incomplete: feature-impl kept its V2 default (which includes the synthesis gate from V1) while also keeping its LIGHT tier default (which tells agents to skip synthesis). The fix is to change feature-impl's default verification level from V2 to V1. Issue type minimums (`feature` → min V2, `bug` → min V2) automatically escalate tracked work back to V2 — so feature-impl agents backed by real issues still get all evidence gates. Only untracked ad-hoc spawns would drop to V1, which is appropriate.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

**Key outcomes:**
- Recommended Option A: change `SkillVerifyLevelDefaults["feature-impl"]` from V2 to V1
- Issue type minimums preserve V2 for tracked feature/bug work
- Eliminates mandatory `--skip-synthesis` for every feature-impl completion
- Moves toward completing V0-V3 migration (tiers → levels as single authority)

## Artifacts Produced

- **Investigation:** `.kb/investigations/2026-02-27-design-light-tier-v2-verification-conflict-resolution.md`
- **Probe:** `.kb/models/completion-verification/probes/2026-02-27-probe-light-tier-v2-verification-conflict.md`

## Decision Fork Summary

| Fork | Recommendation | Reasoning |
|------|---------------|-----------|
| Option A: feature-impl → V1 default | ⭐ Recommended | Fixes conflict at source; issue type minimums preserve V2 for tracked work |
| Option B: GatesForLevel checks tier | Rejected | Enshrines tier as shadow authority alongside levels; contradicts V0-V3 decision |
| Option C: Remove LIGHT tier entirely | Deferred | Correct long-term but too large; Option A is the stepping stone |

## Discovered Work

No discovered work — this is a focused decision issue.
