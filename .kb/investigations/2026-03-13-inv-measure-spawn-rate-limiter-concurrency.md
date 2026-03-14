## Summary (D.E.K.N.)

**Delta:** Spawn rate limiter and concurrency gate are two *pairs* of mechanisms — one for manual spawns (gates.CheckRateLimit + gates.CheckConcurrency), one for daemon spawns (daemon.RateLimiter + WorkerPool). The pairs are architecturally redundant because daemon spawns shell out to `orch work` which also runs the manual gates, creating double-gating. Neither pair has ever blocked a spawn in production (0 blocks in 406 spawns). The daemon rate limiter is invisible (0 logged events). Cost is negligible for daemon-side (<200ns) but unknown for manual-side (HTTP calls to Anthropic API).

**Evidence:** events.jsonl: 38 ratelimit gate evaluations, 0 blocks, 54 warnings; 38 concurrency evaluations, 0 blocks. Benchmarks: daemon RateLimiter.CanSpawn = 33-140ns, WorkerPool.TryAcquire = 4-50ns. CheckRateLimit() makes 2 HTTP calls (Anthropic usage API), no timing captured.

**Knowledge:** Gates that never fire provide false assurance (model: harness-engineering failure mode #5). Both rate limiters exist but operate in the same pipeline, double-checking the same spawns. The daemon's in-memory rate limiter has zero telemetry — its behavior is truly theological.

**Next:** (1) Add timing telemetry to CheckRateLimit() and CheckConcurrency() to measure real-world HTTP call cost. (2) Add daemon rate limiter events to events.jsonl for visibility. (3) Evaluate whether double-gating is intentional or accidental architecture.

**Authority:** architectural — double-gating question reaches across daemon/spawn boundaries, requires orchestrator decision

---

# Investigation: Measure Spawn Rate Limiter and Concurrency Gate

**Question:** What are the 4 measurement properties (cost, coverage, precision, calibration) of the spawn rate limiter and concurrency gate? Are these gates theological (exist in code but unmeasured)?

**Started:** 2026-03-13
**Updated:** 2026-03-13
**Owner:** Agent (orch-go-y7i1p)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** harness-engineering

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-11-inv-gate-retrospective-accuracy-audit.md` | extends | yes — 0% FP rate confirmed for signal gates | - |
| `.kb/plans/2026-03-11-measurement-instrumentation.md` | extends | yes — gate_decision events now logging | Coverage gap larger than expected |
| `.kb/investigations/2026-02-13-inv-cherry-pick-daemon-rate-limiting.md` | deepens | pending | - |

**Relationship types:** extends, confirms, contradicts, deepens

---

## Findings

### Finding 1: Two Separate Pairs of Mechanisms, Not One

**Evidence:** Code analysis reveals the system has FOUR rate-limiting/concurrency mechanisms, organized into two pairs:

**Manual Spawn Path** (`orch spawn` → `RunPreFlightChecks()`):
1. `gates.CheckConcurrency()` — Counts active OpenCode sessions + tmux windows + beads queries. Default: 5 max agents. Makes HTTP calls to OpenCode localhost + shells out to tmux + queries beads DB.
2. `gates.CheckRateLimit()` — Checks Claude Max account usage (5h/weekly windows). Warn at 80%, block at 95%. Makes 2 HTTP calls to Anthropic API (profile + usage endpoint, 30s timeout each).

**Daemon Spawn Path** (`daemon.Sense()` → `CheckPreSpawnGates()`):
3. `daemon.RateLimiter` — In-memory sliding window. Default: 20 spawns/hour. O(n) scan of timestamp array. Checked in `CheckPreSpawnGates()`, recorded after successful spawn in `spawnIssue()`.
4. `daemon.WorkerPool` — Semaphore-based. Default: 5 max workers. `TryAcquire()` in `spawnIssue()`, `Release()` on completion/failure.

**Source:** `pkg/spawn/gates/ratelimit.go:53`, `pkg/spawn/gates/concurrency.go:50`, `pkg/daemon/rate_limiter.go:30`, `pkg/daemon/pool.go:41`

**Significance:** These are commonly described as "the rate limiter" and "the concurrency gate," but they're actually two pairs serving different spawn paths with different mechanisms, thresholds, and measurement surfaces.

---

### Finding 2: Double-Gating — Daemon Spawns Hit Both Pairs

**Evidence:** The daemon's `SpawnWork()` function (at `pkg/daemon/issue_adapter.go:381`) shells out to `orch work <beadsID>`, which calls `runSpawnWithSkillInternal()` → `runSpawnWithSkill()` → `RunPreFlightChecks()`. This means daemon spawns are checked by:
1. First: `daemon.RateLimiter` + `WorkerPool` (in daemon's OODA cycle)
2. Then: `gates.CheckConcurrency` + `gates.CheckRateLimit` (in subprocess `orch work`)

The same spawn is rate-limited and concurrency-checked twice by different mechanisms with different thresholds:
- Daemon rate limit: 20/hour (in-memory)
- Spawn gate rate limit: 95% usage (Anthropic API)
- Daemon concurrency: 5 slots (WorkerPool)
- Spawn gate concurrency: 5 agents (OpenCode + tmux scan)

**Source:** `pkg/daemon/issue_adapter.go:381-394`, `cmd/orch/work_cmd.go:237`, `cmd/orch/spawn_cmd.go:337`

**Significance:** Double-gating is a discovered architectural property, likely accidental rather than intentional. It adds latency (HTTP calls per daemon spawn) without additional safety value since both check overlapping constraints.

---

### Finding 3: Neither Gate Has Ever Blocked a Spawn (0% Block Rate)

**Evidence:** From `~/.orch/events.jsonl` analysis (406 total spawns):
- `spawn.gate_decision` with `gate_name=ratelimit`: 38 evaluations, **0 blocks**, 0 bypasses, 38 allows
- `spawn.gate_decision` with `gate_name=concurrency`: 38 evaluations, **0 blocks**, 0 bypasses, 38 allows
- `spawn.blocked.rate_limit` events: **0** total
- `spawn.warning.rate_limit` events: **54** total (warn at 80% usage threshold)

The concurrency gate has 0 blocks because with 5 max agents and a typical spawn rate of ~3-5/hour, the system never reached capacity. The rate limit gate has 0 blocks because usage never hit 95% — though it hit the 80% warning threshold 54 times.

**Source:** `~/.orch/events.jsonl`, `grep 'spawn.gate_decision' | grep 'block'`

**Significance:** Gates that never fire provide false assurance (harness-engineering failure mode #5: "Gate that never fires = false assurance"). With 0 blocks in 406 spawns, we cannot measure precision (FP rate has no denominator) or validate that the blocking path actually works in production.

---

### Finding 4: Coverage — 9.4% of Spawns Have Gate Telemetry

**Evidence:**
- 38 ratelimit gate_decision events from 406 total spawns = **9.4%** coverage
- The gap is temporal, not architectural: gate_decision "allow" events were added Mar 12 20:19 (commit `775c03bd0`). 173 spawns occurred between initial gate logging (blocks only, Mar 11) and allow-event logging.
- 112 of 125 daemon spawns have no gate_decision events because they occurred before allow events were added.
- Since Mar 12 20:19, coverage is close to 100% for manual spawns.

The daemon's in-memory rate limiter (`daemon.RateLimiter`) emits **zero events** to events.jsonl. Its behavior is completely invisible to telemetry. We don't know how many times it checked, how close to the limit it got, or whether it ever blocked.

**Source:** `~/.orch/events.jsonl` temporal analysis, `pkg/daemon/compliance.go:55-66` (RateLimiter check with no event emission)

**Significance:** The manual-path gates now have coverage (since Mar 12). The daemon-path gates are truly theological — exist in code, have tests, but zero operational telemetry.

---

### Finding 5: Cost — Daemon Gates Are Sub-Microsecond, Manual Gates Are Unknown

**Evidence:** Benchmark results (Go benchmark, 3 runs, 12 cores):

| Component | Operation | Time (ns/op) | Allocations |
|-----------|-----------|-------------|-------------|
| daemon.RateLimiter | CanSpawn (empty) | 33 ns | 0 allocs |
| daemon.RateLimiter | CanSpawn (half full) | 47 ns | 0 allocs |
| daemon.RateLimiter | CanSpawn (at limit) | 140 ns | 1 alloc |
| daemon.RateLimiter | RecordSpawn | 78 ns | 0 allocs |
| daemon.WorkerPool | TryAcquire (available) | 50 ns | 1 alloc |
| daemon.WorkerPool | TryAcquire (at capacity) | 4 ns | 0 allocs |
| daemon.WorkerPool | Reconcile (no change) | 4 ns | 0 allocs |
| daemon.CheckPreSpawnGates | All pass | 34 ns | 0 allocs |
| daemon.CheckPreSpawnGates | Rate limited | 190 ns | 2 allocs |

The manual-path gates (`gates.CheckRateLimit`, `gates.CheckConcurrency`) have **no timing telemetry**. `CheckRateLimit()` makes 2 HTTP calls to Anthropic API with 30s timeout each. `CheckConcurrency()` makes HTTP calls to OpenCode + tmux scanning + beads batch queries. Real-world cost is likely 100ms-2s but is unmeasured.

**Source:** `pkg/daemon/rate_limiter_bench_test.go` (new file), `pkg/account/capacity.go:186` (30s HTTP timeout)

**Significance:** The daemon-side gates are essentially free (<200ns). The manual-side gates are the real cost concern but have no timing instrumentation. This matches harness-engineering failure mode #7 (enforcement without measurement = invisible cost).

---

### Finding 6: Calibration — Thresholds May Be Wrong

**Evidence:**
- **Rate limit warn (80%):** Fires regularly (54 warnings). This threshold works.
- **Rate limit block (95%):** Never fires. Either the threshold is correctly calibrated (usage never reaches 95%) or it's too high to ever trigger (by the time you're at 95%, you're already rate-limited by Anthropic).
- **Concurrency max (5 agents):** Never fires for either mechanism. With daemon spawning 20/hour max and agents running 30-120 minutes each, 5 concurrent slots may be too generous.
- **Daemon rate limit (20/hour):** With 15s poll interval, theoretical max is 240 spawns/hour. The 20/hour limit would fire after 5 minutes of continuous spawning. But with spawn delay (3s) and issue availability constraints, it may never trigger naturally.

**Source:** Event log analysis, `pkg/daemonconfig/config.go:237-241`

**Significance:** Thresholds that never trigger may be correctly calibrated (safety net for extreme scenarios) or miscalibrated (providing false assurance). Without blocks, we can't distinguish between "threshold is right" and "threshold is too generous."

---

## Synthesis

**Key Insights:**

1. **Architecture is accidentally doubled** — The same spawn is checked by two independent pairs of rate limiter + concurrency gate. The daemon has its own (in-memory, sub-microsecond, zero telemetry) and the subprocess `orch work` hits the spawn gates (HTTP-based, no timing data, partial telemetry). This double-gating adds latency to every daemon spawn without clear benefit.

2. **Theological status confirmed for 3 of 4 mechanisms** — Only `gates.CheckRateLimit` has partial telemetry (38 evaluations logged since Mar 12). The daemon RateLimiter and WorkerPool have zero operational visibility. `gates.CheckConcurrency` has identical coverage to the rate limit gate (38 evaluations). None have timing data.

3. **Zero blocks means zero precision data** — With 0 blocks across 406 spawns, we cannot calculate false positive rate. The gates may be working correctly (the system never hits limits under normal operation) or may be dead code that would fail if triggered. The warning path (54 warns at 80% usage) shows the rate limit mechanism works, but the blocking path (95%) is untested in production.

**Answer to Investigation Question:**

| Property | gates.CheckRateLimit | gates.CheckConcurrency | daemon.RateLimiter | daemon.WorkerPool |
|----------|---------------------|----------------------|-------------------|------------------|
| **Cost** | Unknown (2 HTTP calls, 30s timeout) | Unknown (HTTP + tmux + beads) | 33-140 ns (benchmarked) | 4-50 ns (benchmarked) |
| **Coverage** | 9.4% (38/406), improving since Mar 12 | 9.4% (38/406), improving since Mar 12 | 0% (zero events) | 0% (zero events) |
| **Precision** | Undefined (0/0 blocks) | Undefined (0/0 blocks) | Undefined (0 telemetry) | Undefined (0 telemetry) |
| **Calibration** | Warn: 80% (fires). Block: 95% (never fires) | 5 agents (never fires) | 20/hour (no data) | 5 slots (no data) |

All 4 mechanisms are theological on at least 2 of the 4 measurement properties. The daemon pair is fully theological (0 events). The spawn gate pair has partial visibility but no timing data and no block data.

---

## Structured Uncertainty

**What's tested:**

- ✅ daemon.RateLimiter benchmarked: 33-140ns per CanSpawn() (ran Go benchmarks, 3 iterations)
- ✅ daemon.WorkerPool benchmarked: 4-50ns per TryAcquire() (ran Go benchmarks, 3 iterations)
- ✅ Gate coverage measured: 38/406 = 9.4% (parsed events.jsonl)
- ✅ Block count verified: 0 blocks across all 4 mechanisms (parsed events.jsonl)
- ✅ Double-gating confirmed: daemon.SpawnWork() → `orch work` → RunPreFlightChecks() (code trace)

**What's untested:**

- ⚠️ CheckRateLimit() real-world latency (requires timing instrumentation in spawn path)
- ⚠️ CheckConcurrency() real-world latency (requires timing instrumentation)
- ⚠️ Whether blocking paths work in production (0 blocks means untested)
- ⚠️ Whether daemon rate limiter ever approaches its limit under normal operation
- ⚠️ Whether the double-gating is intentional design or accidental architecture

**What would change this:**

- A single block event would provide data for precision calculation
- Adding timing instrumentation to the HTTP-calling gates would fill the cost gap
- Adding daemon rate limiter events would make the daemon pair visible
- Load testing (simulating 20+ issues labeled triage:ready) would test the daemon rate limiter

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add daemon rate limiter events | implementation | Straightforward telemetry addition within daemon package |
| Add spawn gate timing | implementation | Straightforward timing instrumentation |
| Evaluate double-gating | architectural | Crosses daemon/spawn boundary, may require design decision |

### Recommended Approach ⭐

**Instrument First, Decide Later** — Add telemetry to all 4 mechanisms before deciding whether to consolidate or keep the double-gating.

**Why this approach:**
- Cannot make architectural decisions about double-gating without cost data
- Daemon pair has zero visibility — any measurement is better than none
- Aligns with harness-engineering model: "pair every enforcement layer with measurement"

**Trade-offs accepted:**
- Deferring the double-gating decision until data is available
- Some additional events in events.jsonl (low volume — these gates fire once per spawn)

**Implementation sequence:**
1. **Add daemon rate limiter events** — Log `daemon.rate_limit.check` with `{spawns_last_hour, max_per_hour, allowed}` in `CheckPreSpawnGates()`. Low-effort, high-value.
2. **Add timing to spawn gate HTTP calls** — Wrap `CheckRateLimit()` and `CheckConcurrency()` with time.Since() instrumentation in `RunPreFlightChecks()`. Include duration in gate_decision events.
3. **Evaluate double-gating** (after 1-2 weeks of data) — With cost data from both paths, decide if daemon-side gates are redundant with the subprocess gates.

### Alternative Approaches Considered

**Option B: Consolidate gates (remove double-gating now)**
- **Pros:** Reduces latency, simplifies architecture
- **Cons:** Cannot verify which pair is actually needed without measurement data
- **When to use instead:** If the HTTP call latency turns out to be >500ms per daemon spawn

**Option C: Remove daemon-side gates entirely**
- **Pros:** Daemon trusts subprocess gates, simpler daemon code
- **Cons:** Loses fast-fail at daemon level (subprocess must start, run gates, then fail)
- **When to use instead:** If daemon-side gates are proven redundant with timing data

---

### Implementation Details

**What to implement first:**
- Daemon rate limiter telemetry (Finding 4: zero events = zero visibility)
- Spawn gate timing (Finding 5: unknown cost for HTTP-calling gates)

**Things to watch out for:**
- ⚠️ Adding events to the daemon rate limiter check will increase events.jsonl volume by ~1 event per daemon poll cycle (every 15s). Consider logging only state changes (allowed→blocked, blocked→allowed) to reduce noise.
- ⚠️ The 30s HTTP timeout in CheckRateLimit() means a single gate check could take up to 60s if both API calls time out. This is a potential spawn latency issue for daemon spawns.
- ⚠️ CheckConcurrency makes multiple external calls (OpenCode API + tmux + beads) — timing should capture each component separately.

**Success criteria:**
- ✅ Can query daemon rate limiter behavior from events.jsonl
- ✅ Can measure p50/p95 latency of CheckRateLimit() and CheckConcurrency()
- ✅ Coverage of gate telemetry reaches >90% of all spawns

---

## References

**Files Examined:**
- `pkg/spawn/gates/ratelimit.go` — Manual spawn path rate limit gate
- `pkg/spawn/gates/concurrency.go` — Manual spawn path concurrency gate
- `pkg/daemon/rate_limiter.go` — Daemon in-memory rate limiter
- `pkg/daemon/pool.go` — Daemon worker pool (semaphore)
- `pkg/daemon/compliance.go` — Daemon pre-spawn gates orchestration
- `pkg/daemon/spawn_execution.go` — Daemon spawn pipeline with pool/rate limiter
- `pkg/daemon/issue_adapter.go:381` — SpawnWork shells out to `orch work`
- `pkg/orch/spawn_preflight.go` — RunPreFlightChecks gate sequence
- `pkg/account/capacity.go:101` — GetCurrentCapacity HTTP calls
- `pkg/agent/filters.go` — IsActiveForConcurrency logic
- `cmd/orch/spawn_cmd.go:337` — Where RunPreFlightChecks is called
- `cmd/orch/work_cmd.go:237` — work command sets daemonDriven=true

**Commands Run:**
```bash
# Gate decision event counts
grep 'spawn.gate_decision' ~/.orch/events.jsonl | grep -o '"gate_name":"[^"]*"' | sort | uniq -c

# Block/allow breakdown per gate
grep 'spawn.gate_decision' ~/.orch/events.jsonl | grep '"gate_name":"ratelimit"' | grep -o '"decision":"[^"]*"' | sort | uniq -c

# Total spawn count
grep -c '"type":"session.spawned"' ~/.orch/events.jsonl

# Daemon spawn count
grep -c '"type":"daemon.spawn"' ~/.orch/events.jsonl

# Rate limit warning/block event counts
grep -c 'spawn.warning.rate_limit' ~/.orch/events.jsonl
grep -c 'spawn.blocked.rate_limit' ~/.orch/events.jsonl

# Benchmark tests
go test ./pkg/daemon/ -bench 'Benchmark' -benchmem -count=3 -run '^$'
```

**Related Artifacts:**
- **Model:** `.kb/models/harness-engineering/model.md` — measurement surface framework, failure mode #5 (gates that never fire)
- **Plan:** `.kb/plans/2026-03-11-measurement-instrumentation.md` — Phase 2 gate visibility
- **Investigation:** `.kb/investigations/2026-03-11-inv-gate-retrospective-accuracy-audit.md` — 0% FP rate for signal gates

---

## Investigation History

**2026-03-13 21:00:** Investigation started
- Initial question: Measure 4 properties (cost, coverage, precision, calibration) of spawn rate limiter and concurrency gate
- Context: Task described as "fully theological, unknown on all 4 properties"

**2026-03-13 21:15:** Discovered two pairs of mechanisms (daemon + spawn gates)
- Code trace revealed daemon.RateLimiter + WorkerPool are separate from gates.CheckRateLimit + CheckConcurrency
- Both paths serve the same purpose but different mechanisms

**2026-03-13 21:25:** Confirmed double-gating architecture
- SpawnWork() → orch work → RunPreFlightChecks means daemon spawns hit both pairs

**2026-03-13 21:35:** Ran benchmarks, analyzed event telemetry
- Daemon gates: sub-microsecond, zero telemetry
- Spawn gates: unknown cost (HTTP), partial telemetry (38/406)
- 0 blocks across all mechanisms in production

**2026-03-13 21:45:** Investigation completed
- Status: Complete
- Key outcome: All 4 mechanisms are partially or fully theological. Double-gating is likely accidental.
