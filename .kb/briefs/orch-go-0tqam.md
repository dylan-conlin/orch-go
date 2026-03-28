# Brief: orch-go-0tqam

## Frame

When a spawned worker runs `claude --print "route this prompt"`, the child process inherits `ORCH_SPAWNED=1` from the parent. The stop hook sees a "worker" exiting without Phase: Complete and blocks it, injecting a JSON block decision into stdout. That extra output contaminates the print result, breaking skill routing and A/B testing flows that depend on clean `--print` output.

## Resolution

Three lines. The hook already had a skip-conditions section (not a spawned worker, no beads ID, escape hatch). The missing case was "this is a print-mode subprocess, not a real worker." Adding `ORCH_PRINT_MODE=1` to the skip conditions — right after the `is_spawned_worker()` check — lets print calls pass through cleanly while preserving enforcement for actual worker sessions. The test suite already had the failing test (`TestEnforcePhaseCompleteSkipsPrintMode`) plus a guard test ensuring real workers are still blocked. Both pass.

## Tension

The governance protection worked exactly as designed — it took two escalations and an `orch harness unlock` to apply a 3-line fix. Whether that friction is proportionate for hooks that affect every agent exit is worth noticing: the protection prevented accidental modification, but the cost was real time on a blocking bug.
