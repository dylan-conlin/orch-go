# Session Synthesis

**Agent:** og-inv-measure-spawn-rate-13mar-aa94
**Issue:** orch-go-y7i1p
**Duration:** 2026-03-13 21:00 → 2026-03-13 21:50
**Outcome:** success

---

## Plain-Language Summary

The spawn rate limiter and concurrency gate are actually **four mechanisms** organized into two pairs: one for manual spawns (HTTP-based, calls Anthropic API and OpenCode) and one for daemon spawns (in-memory, sub-microsecond). Daemon spawns are double-gated because the daemon shells out to `orch work`, which runs the manual gates too. None of the four mechanisms have ever blocked a spawn in production (0 blocks across 406 spawns). The daemon pair has zero operational telemetry — it's the definition of theological enforcement. Benchmarks show daemon gates cost <200ns, but manual gate costs are unknown because they make HTTP calls with no timing instrumentation.

---

## TLDR

Measured the 4 properties (cost, coverage, precision, calibration) of spawn rate limiters and concurrency gates. Found two pairs of mechanisms (daemon + spawn gates) creating accidental double-gating. Zero blocks in production across all mechanisms. Daemon-side gates have zero telemetry. Benchmarked daemon gates at 33-190ns.

---

## Delta (What Changed)

### Files Created
- `pkg/daemon/rate_limiter_bench_test.go` — Benchmarks for RateLimiter, WorkerPool, CheckPreSpawnGates
- `.kb/investigations/2026-03-13-inv-measure-spawn-rate-limiter-concurrency.md` — Full investigation findings

### Files Modified
- None (measurement-only investigation)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- 406 total spawns in events.jsonl, 0 rate limit blocks, 0 concurrency blocks
- 54 rate limit warnings (80% usage threshold fires regularly)
- 38 gate_decision events for ratelimit/concurrency (9.4% coverage due to temporal gap)
- daemon.RateLimiter CanSpawn: 33-140ns, WorkerPool TryAcquire: 4-50ns
- CheckRateLimit makes 2 HTTP calls to Anthropic API (30s timeout each) — no timing data
- CheckConcurrency makes HTTP calls to OpenCode + tmux scan + beads queries — no timing data
- Daemon SpawnWork() shells out to `orch work` which runs RunPreFlightChecks() — confirmed double-gating

### Tests Run
```bash
# Existing rate limiter tests
go test ./pkg/daemon/ -run 'TestRateLimiter|TestDaemon_OnceExcluding_RateLimited' -v -count=1
# PASS: 8 tests passing

# Existing spawn gates tests
go test ./pkg/spawn/gates/ -run 'TestDefault|TestUsage' -v -count=1
# PASS: 3 tests passing

# Worker pool tests
go test ./pkg/daemon/ -run 'TestWorkerPool' -v -count=1
# PASS: 14 tests passing

# Benchmarks (new)
go test ./pkg/daemon/ -bench 'Benchmark' -benchmem -count=3 -run '^$'
# 11 benchmarks, all passing
```

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for gate measurement outcomes.

---

## Architectural Choices

No architectural choices — investigation-only session. The double-gating finding surfaces an architectural question but this investigation does not make the decision.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-13-inv-measure-spawn-rate-limiter-concurrency.md` — Full 4-property measurement of rate limiter and concurrency gate

### Constraints Discovered
- daemon.RateLimiter emits zero events — its behavior is invisible to telemetry
- CheckRateLimit() has 30s HTTP timeout per API call — potential 60s latency per spawn
- Double-gating means daemon spawns pay HTTP latency for gates that daemon already checked

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + benchmarks)
- [x] Tests passing
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-y7i1p`

---

## Unexplored Questions

- **Is the double-gating intentional?** The daemon has its own rate limiter and pool, but also spawns via subprocess which hits independent gates. This may be defense-in-depth or accidental redundancy.
- **What's the real-world latency of CheckRateLimit()?** HTTP calls to Anthropic API with 30s timeout — could be 100ms or 10s.
- **Would load testing trigger the daemon rate limiter?** Labeling 25 issues as triage:ready simultaneously would test the 20/hour limit.

---

## Friction

- No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-measure-spawn-rate-13mar-aa94/`
**Investigation:** `.kb/investigations/2026-03-13-inv-measure-spawn-rate-limiter-concurrency.md`
**Beads:** `bd show orch-go-y7i1p`
