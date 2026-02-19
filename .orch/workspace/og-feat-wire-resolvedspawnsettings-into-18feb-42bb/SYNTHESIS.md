# Session Synthesis

**Agent:** og-feat-wire-resolvedspawnsettings-into-18feb-42bb
**Issue:** orch-go-1070
**Outcome:** success

## Plain-Language Summary
Updated the spawn pipeline to use the centralized ResolvedSpawnSettings resolver so backend/model/tier/mode/validation are resolved consistently across spawn and rework paths. This removes the old ad-hoc backend/model selection logic and limits the infrastructure escape hatch to cases with no explicit config, fixing the precedence bugs targeted by this step.

## Verification Contract
- See `/.orch/workspace/og-feat-wire-resolvedspawnsettings-into-18feb-42bb/VERIFICATION_SPEC.yaml`
