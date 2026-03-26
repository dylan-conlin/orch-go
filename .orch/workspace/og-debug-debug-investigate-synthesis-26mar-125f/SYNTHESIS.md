# Session Synthesis

**Agent:** og-debug-debug-investigate-synthesis-26mar-125f
**Issue:** orch-go-n4uwb
**Duration:** 2026-03-26 → 2026-03-26
**Outcome:** success

---

## Plain-Language Summary

The 4% `SYNTHESIS.md` rate for `feature-impl` is mostly not a model obedience failure. Current `feature-impl` spawns default to `light` tier, and light-tier contexts explicitly say synthesis is not required while the tier cap drops verification to `V0`, so archive-based synthesis counting is measuring an optional artifact for most runs. There is a secondary protocol problem in the rare full-tier `feature-impl` path: the skill text still ends with "Phase: Complete" and commit, while full-tier worker protocol adds `SYNTHESIS.md`, `BRIEF.md`, and `VERIFICATION_SPEC.yaml`, creating a conflicting contract.

## TLDR

I traced the `feature-impl` completion path across spawn defaults, verification gates, archived manifests, and the benchmark investigation that reported 4% synthesis compliance. The low rate is primarily a configuration/measurement mismatch (`feature-impl` defaults to `light` + `V0`), with a smaller but real protocol-weight conflict when `feature-impl` is upgraded to full tier and the skill text still does not teach the extra artifact requirements.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-debug-debug-investigate-synthesis-26mar-125f/SYNTHESIS.md` - Investigation synthesis for orchestrator review
- `.orch/workspace/og-debug-debug-investigate-synthesis-26mar-125f/BRIEF.md` - Dylan-facing comprehension brief
- `.orch/workspace/og-debug-debug-investigate-synthesis-26mar-125f/VERIFICATION_SPEC.yaml` - Verification contract for this investigation

### Files Modified
- No repository source files modified; this session produced investigation artifacts only.

### Commits
- Pending commit at session close.

---

## Evidence (What Was Observed)

- The benchmark that raised the question reports `feature-impl` at `3/79 = 4%` SYNTHESIS coverage and explicitly calls it a protocol-weight problem, while also noting `100% Phase:Complete` for the same skill: `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:72`.
- Current tier defaults classify `feature-impl` as `light`, not `full`: `pkg/spawn/config.go:43`.
- Tier capping drops light-tier verification to `V0`, which skips the synthesis gate entirely: `pkg/spawn/verify_level.go:97` and `pkg/verify/level.go:7`.
- The current `feature-impl` skill still describes completion as self-review -> phases complete -> `Phase: Complete` -> commit, with no mention of `SYNTHESIS.md`, `BRIEF.md`, or `VERIFICATION_SPEC.yaml`: `skills/src/worker/feature-impl/SKILL.md:253`.
- A current `feature-impl` light-tier spawn context says `SYNTHESIS.md is NOT required`, confirming the default runtime contract seen by workers: `.orch/workspace/og-feat-add-thin-issue-26mar-70b4/SPAWN_CONTEXT.md:8`.
- Manifest sampling across current and archived workspaces found 28 `feature-impl` manifests at `(tier=light, verify_level=V0)` versus only 4 older `(tier=full, verify_level unset)` examples, which supports the interpretation that synthesis is usually out of contract rather than ignored.
- Workspace scan over current and archived feature-like workspaces found 34 manifested workspaces with only 5 synthesis files, matching the benchmark symptom while also aligning with the light-tier default.

### Tests Run
```bash
pwd
# /Users/dylanconlin/Documents/personal/orch-go

python3 - <<'PY'
# summarized all current+archived feature-impl manifests by tier/verify_level
PY
# counts: ('light', 'V0') = 28, ('full', None) = 4

python3 - <<'PY'
# counted manifested feature-like workspaces with/without SYNTHESIS.md
PY
# workspace-like total = 34, with synthesis = 5

rg -n '"skill":"feature-impl"|"tier":|"verify_level":' .orch/archive/workspace/*/AGENT_MANIFEST.json
# confirmed older archived full-tier examples exist
```

---

## Architectural Choices

### Root cause framing
- **What I chose:** Treat the 4% metric as a spawn-contract investigation, not a model-compliance investigation.
- **What I rejected:** Assuming the low rate meant feature workers were broadly ignoring explicit synthesis requirements.
- **Why:** The runtime contract is mostly determined by tier + verify-level inference, and current defaults make synthesis optional for `feature-impl`.
- **Risk accepted:** Archive counts are still a proxy; I did not reclassify every historical workspace manually.

### Recommendation scope
- **What I chose:** Recommend architect follow-up instead of direct implementation.
- **What I rejected:** Patching prompts or metrics directly from this investigation session.
- **Why:** `synthesis` is flagged as a hotspot area in the spawn context, and the real problem spans metrics, tier semantics, and skill wording.
- **Risk accepted:** The follow-up issue is intentionally narrow and still needs a richer design description before implementation.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.orch/workspace/og-debug-debug-investigate-synthesis-26mar-125f/SYNTHESIS.md` - Root cause synthesis for the 4% feature-impl signal
- `.orch/workspace/og-debug-debug-investigate-synthesis-26mar-125f/BRIEF.md` - Reader-facing explanation of the finding
- `.orch/workspace/og-debug-debug-investigate-synthesis-26mar-125f/VERIFICATION_SPEC.yaml` - Verification steps and evidence ledger

### Decisions Made
- Decision 1: Classify the problem as primarily configuration/measurement mismatch, not raw agent noncompliance.
- Decision 2: Treat the full-tier `feature-impl` wording conflict as secondary but real, because it can still depress compliance when scope upgrades require synthesis artifacts.

### Constraints Discovered
- `feature-impl` is configured as light-tier by default, so synthesis presence is not a valid universal completion metric for that skill under the current contract.
- Full-tier worker protocol and `feature-impl` skill-local completion text diverge, which weakens the prompt when a `feature-impl` task is upgraded to full tier.

### Externalized via `kb quick`
- No new `kb quick` entry created in this session.

---

## Verification Contract

See `.orch/workspace/og-debug-debug-investigate-synthesis-26mar-125f/VERIFICATION_SPEC.yaml`.

Key outcomes:
- Verified the repo location matched the required project path.
- Verified `feature-impl` defaults to `light` tier and effective `V0` verification.
- Verified the current light-tier spawn prompt explicitly says synthesis is not required.
- Verified benchmark + workspace counts are consistent with the config-driven explanation.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** `orch-go-cwj26` - Align feature-impl synthesis semantics across skill, tier, and compliance metrics
**Skill:** `architect`
**Context:**
```text
The low feature-impl SYNTHESIS rate is mostly a contract mismatch: default light-tier/V0 runs intentionally skip synthesis, but metrics and some full-tier expectations still treat synthesis as if it were universal. Design a single semantics for when feature-impl should require synthesis, how the skill text should teach it, and which metric should be used for compliance reporting.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should feature-impl full-tier upgrades remain possible, or should synthesis-heavy implementation work route to a different skill entirely?
- Should benchmark dashboards report `Phase: Complete` as the canonical implementation completion signal and reserve synthesis metrics for full-tier work only?

**Areas worth exploring further:**
- Historical breakpoints where `feature-impl` moved from full-tier to light-tier defaults
- Whether `BRIEF.md` / `VERIFICATION_SPEC.yaml` should also be tier-gated in shared worker protocol text

**What remains unclear:**
- How many of the historical 79 benchmarked workspaces were intentionally light-tier versus scope-upgraded full-tier runs with missing synthesis

---

## Friction

No friction — smooth session.

---

## Session Metadata

**Skill:** `systematic-debugging`
**Model:** `openai/gpt-5.4`
**Workspace:** `.orch/workspace/og-debug-debug-investigate-synthesis-26mar-125f/`
**Investigation:** Not created; findings are captured in workspace artifacts.
**Beads:** `bd show orch-go-n4uwb`
