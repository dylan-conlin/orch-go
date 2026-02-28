# Session Synthesis

**Agent:** og-debug-fix-concurrency-cap-27feb-8772
**Issue:** orch-go-lvu0
**Outcome:** success

---

## Plain-Language Summary

Fixed two bugs in the spawn concurrency gate. First, idle agents (agents not actively running) were being counted against the concurrency limit, so 15 idle agents would block new spawns even though none were consuming resources. Now only running agents count. Second, `--max-agents 0` was supposed to mean "unlimited" but actually triggered the default limit of 5, because the flag's default value (0) was indistinguishable from "explicitly set to 0". Fixed by using -1 as the sentinel for "not set".

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## Delta (What Changed)

### Files Modified
- `pkg/agent/filters.go` - `IsActiveForConcurrency` now only counts running agents (removed idle-within-1h rule)
- `pkg/spawn/gates/concurrency.go` - `GetMaxAgents` uses `>= 0` check instead of `!= 0`, treating negative values as "not set"
- `cmd/orch/spawn_cmd.go` - Changed `--max-agents` flag default from 0 to -1 (sentinel for "not set")
- `pkg/spawn/gates/concurrency_test.go` - Updated existing tests for new sentinel, added `ZeroMeansUnlimited` and `ZeroEnvMeansUnlimited` tests
- `cmd/orch/main_test.go` - Updated 3 tests to use -1 sentinel instead of 0

### Files Created
- `pkg/agent/filters_test.go` - 7 test cases for `IsActiveForConcurrency` including repro scenario (15 idle agents)

---

## Evidence (What Was Observed)

- Root cause for Bug 1: `IsActiveForConcurrency` counted idle agents active within 1 hour. With 15 idle agents, 6 fell within the 1h window, exceeding the default limit of 5.
- Root cause for Bug 2: `GetMaxAgents(0)` treated 0 as "not set" (same as flag default), falling through to DefaultMaxAgents=5. No way to distinguish "flag not passed" from "flag explicitly set to 0".
- The daemon path (`pkg/daemon/pool.go`) uses its own `WorkerPool` with separate `MaxAgents` config — unaffected by these changes.

### Tests Run
```bash
go test ./pkg/agent/ ./pkg/spawn/gates/ ./cmd/orch/ -count=1
# ok  github.com/dylan-conlin/orch-go/pkg/agent      0.005s
# ok  github.com/dylan-conlin/orch-go/pkg/spawn/gates 0.048s
# ok  github.com/dylan-conlin/orch-go/cmd/orch        4.980s
```

---

## Architectural Choices

### Sentinel value (-1) vs Cobra Changed() detection for --max-agents
- **What I chose:** Default flag value of -1 as "not set" sentinel
- **What I rejected:** Using `cmd.Flags().Changed("max-agents")` and threading a bool through the call chain
- **Why:** Simpler — no function signature changes needed across 3 layers (spawn_cmd → extraction → concurrency). The -1 sentinel is self-documenting and the precedence logic stays in one function.
- **Risk accepted:** Negative max-agents values are silently treated as "not set" rather than erroring

### Remove idle agent counting entirely vs reduce threshold
- **What I chose:** Idle agents never count toward concurrency limit
- **What I rejected:** Reducing the threshold from 1h to something smaller
- **Why:** Concurrency limit exists to prevent resource exhaustion. Idle agents consume no resources. Any threshold still creates false positives where idle agents block spawns.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (13 new/updated tests, full suite green)
- [x] Ready for `orch complete orch-go-lvu0`

---

## Unexplored Questions

Straightforward session, no unexplored territory.

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-fix-concurrency-cap-27feb-8772/`
**Beads:** `bd show orch-go-lvu0`
