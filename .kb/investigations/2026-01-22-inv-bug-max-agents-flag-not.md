## Summary (D.E.K.N.)

**Delta:** The `--max-agents 0` flag was treated as "not set" because 0 was both the sentinel value and the "unlimited" value.

**Evidence:** Tests confirm that changing sentinel from 0 to -1 allows `--max-agents 0` and `ORCH_MAX_AGENTS=0` to correctly disable the limit.

**Knowledge:** When a flag needs to distinguish "not set" from "value is zero", use a negative sentinel (-1) rather than zero.

**Next:** Close - fix implemented and tested.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural)

---

# Investigation: Bug Max Agents Flag Not Disabling Concurrency Limit

**Question:** Why does `--max-agents 0` not disable the concurrency limit as documented?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Flag default of 0 conflicts with "unlimited" semantics

**Evidence:** In `spawn_cmd.go`, the flag was defined as:
```go
spawnCmd.Flags().IntVar(&spawnMaxAgents, "max-agents", 0, "...")
```

The `getMaxAgents()` function checked:
```go
if spawnMaxAgents != 0 {
    return spawnMaxAgents
}
```

This meant both "flag not provided" and "flag explicitly set to 0" resulted in the same code path - falling through to env var check or default.

**Source:** `cmd/orch/spawn_cmd.go:193`, `cmd/orch/spawn_cmd.go:407-426`

**Significance:** This is the root cause - the sentinel value (0) collided with the desired "unlimited" value (0).

### Finding 2: Downstream code correctly handles 0 as unlimited

**Evidence:** In `checkConcurrencyLimit()`:
```go
maxAgents := getMaxAgents()
// Limit disabled (0 means unlimited)
if maxAgents == 0 {
    return nil
}
```

The logic already existed to skip concurrency checks when `maxAgents == 0`.

**Source:** `cmd/orch/spawn_cmd.go:468-473`

**Significance:** Only the flag/env var parsing needed fixing - the rest of the code was ready.

---

## Synthesis

**Key Insights:**

1. **Sentinel collision** - Using 0 as both "not set" and "unlimited" is a common Go pitfall for integer flags.

2. **Simple fix available** - Changing sentinel to -1 cleanly separates "not set" from "explicitly zero".

**Answer to Investigation Question:**

The `--max-agents 0` flag wasn't working because the flag parsing logic used 0 as its sentinel for "not set". The fix is to use -1 as the sentinel, allowing 0 to be passed through as the intended "unlimited" value.

---

## Structured Uncertainty

**What's tested:**

- ✅ `TestGetMaxAgentsDefault` - sentinel -1 falls through to default (verified: test passes)
- ✅ `TestGetMaxAgentsZeroDisablesLimit` - flag 0 returns 0 (verified: test passes)
- ✅ `TestGetMaxAgentsEnvZeroDisablesLimit` - env "0" returns 0 (verified: test passes)
- ✅ All existing max-agents tests pass (verified: `go test ./cmd/orch/... -run TestGetMaxAgents`)

**What's untested:**

- ⚠️ End-to-end spawn with `--max-agents 0` (would require running agent infrastructure)

**What would change this:**

- Finding would be wrong if other code paths also check for 0 as "not set"

---

## Implementation

**Fix applied:**

1. Changed flag default from `0` to `-1` in `spawn_cmd.go:193`
2. Updated `getMaxAgents()` to check `-1` instead of `0` in `spawn_cmd.go:412`
3. Added two new tests for zero disabling limit (flag and env var)
4. Updated existing tests to use `-1` as sentinel

**Files modified:**
- `cmd/orch/spawn_cmd.go` - flag default and getMaxAgents() logic
- `cmd/orch/main_test.go` - test updates and new tests
- `cmd/orch/serve_agents.go` - fixed unrelated duplicate condition (build blocker)

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - flag definition and parsing
- `cmd/orch/main_test.go` - existing tests

**Commands Run:**
```bash
# Run tests
go test -v ./cmd/orch/... -run "TestGetMaxAgents"

# Build
make build
```
