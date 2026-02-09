## Summary (D.E.K.N.)

**Delta:** At 20+ concurrent agents, the most immediate breakpoints are state/registry drift, spawn-context token bloat, and unbounded event log growth; several older hotspots (756-line `handleAgents`, 626-line `runDaemonLoop`) were already reduced but their load profile still degrades with per-session enrichment.

**Evidence:** Live checks showed `state_open_rows=166` vs `live_agents=18` (`open_minus_live=148`), `~/.orch/events.jsonl` at 12,693,779 bytes / 32,293 lines, and spawn context size `chars=72223` (~18k tokens) in this session; code confirms single socket path resolution per beads client and global tmux socket targeting.

**Knowledge:** The architecture has moved from monolithic functions toward split handlers, but scale risk now concentrates in cross-boundary state reconciliation and context/log payload growth rather than single giant functions.

**Next:** Prioritize three fixes in order: state reconciliation/closure hardening, spawn-context budget enforcement, and events.jsonl retention/compaction; queue tmux/beads multi-socket changes behind those unless >50-agent load is imminent.

**Authority:** architectural - Recommendations cross daemon, spawn, serve, state, and events subsystems.

---

# Investigation: Scaling Assumptions Audit Systematically Examine

**Question:** Which early architecture assumptions break, degrade, or remain latent at 20+ concurrent agents, and what is the severity/blast-radius/fix-order plan?

**Started:** 2026-02-09
**Updated:** 2026-02-09
**Owner:** architect worker (orch-go-svs0p)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation                                                                    | Relationship     | Verified | Conflicts                                                           |
| -------------------------------------------------------------------------------- | ---------------- | -------- | ------------------------------------------------------------------- |
| `.kb/investigations/2026-02-06-audit-architecture-audit-orch-go-codebase.md`     | extends          | yes      | Partially: prior giant-function counts are improved in current code |
| `.kb/investigations/2026-02-09-inv-beads-reliability-synthesis-cohesive-root.md` | confirms         | yes      | None                                                                |
| `.kb/models/daemon-autonomous-operation.md`                                      | confirms/extends | yes      | None                                                                |

---

## Findings

### Finding 1: State tracking drift is currently the highest-severity scaling break

**Evidence:**

- Live comparison: `state_open_rows=166`, `live_agents=18`, `open_minus_live=148`.
- Registry snapshot: `total=88 active=4 completed=75 abandoned=9` (`stale_non_active=84`).
- `pkg/session/registry.go` stores all orchestrator sessions in flat JSON with no automatic historical compaction.

**Source:**

- Command: `python3` queries against `~/.orch/state.db`, `/tmp/orch_status.json`, and `~/.orch/sessions.json`
- `pkg/session/registry.go:18`

**Significance:**
This is a **broken** assumption (local state remains close to runtime truth). At 20+ agents, stale rows inflate dashboards/recovery logic and make automated completion/reconciliation noisier and less trustworthy.

---

### Finding 2: Spawn context payload is already beyond safe prompt budget

**Evidence:**

- This workspace's `SPAWN_CONTEXT.md` is `chars=72223` / `lines=1748` / `approx_tokens=18055`.
- Template and context assembly intentionally include many sections, but no hard end-to-end token budget gate on final context size.
- Existing KB truncation guard (`MaxKBContextChars = 80000`) protects only one input stream, not total prompt footprint.

**Source:**

- Command: `python3` size check on `.orch/workspace/og-arch-scaling-assumptions-audit-09feb-f6d3/SPAWN_CONTEXT.md`
- `pkg/spawn/context_template.go:11`
- `pkg/spawn/kbcontext.go:31`

**Significance:**
This is **degraded** and approaching **broken** for smaller context-window models and long-lived worker sessions. Prompt bloat directly reduces effective reasoning room and increases completion-risk variance.

---

### Finding 3: events.jsonl growth is unbounded and increasingly query-expensive

**Evidence:**

- Current file size: `12,693,779 bytes` and `32,293` lines.
- Logger append path has no rotation or retention policy.
- Multiple serve endpoints repeatedly scan/read from this file (`readLastNEvents`, service/agent/event health handlers).

**Source:**

- Command: `python3` file size/line count on `~/.orch/events.jsonl`
- `pkg/events/logger.go:91`
- `cmd/orch/serve_services_events.go:39`
- `cmd/orch/serve_agents_events.go:235`

**Significance:**
This is **degraded** today and trends toward **broken** under sustained high-volume daemon + dashboard polling. Growth amplifies I/O and latency across observability endpoints.

---

### Finding 4: Beads subprocess amplification is mitigated; single-socket topology remains latent risk

**Evidence:**

- Concurrency cap exists (`defaultMaxBDSubprocess = 12`) and stale-retry tests pass.
- Socket discovery still resolves one `.beads/bd.sock` path per invocation; client connects to a single unix socket path.

**Source:**

- `pkg/beads/client.go:28`
- `pkg/beads/client.go:541`
- `pkg/beads/client.go:616`
- Command: `go test ./pkg/beads/... -run 'TestRunBDCommand_AddsQuietByDefault|TestRunBDCommand_RetriesAllowStaleWhenOutOfSyncJSONErrorPayload' -count=1`

**Significance:**
Subprocess stampede is mostly **fixed** (from prior reliability saga), but single-daemon/socket dependency is **latent** at higher parallelism and cross-project contention.

---

### Finding 5: Dashboard/serve monolith has improved structurally, but enrichment still scales with agent count

**Evidence:**

- Prior hotspot reduced: `handleAgents` is now 57 lines delegating to split collectors/enrichers.
- Serve endpoint surface is broad (`routes_registered=53`), with `runServe` still coordinating many subsystems.
- Per-session message fetch remains in enrichment path (`GetMessages(sessionID)` in goroutines), so cost still grows with active session count.

**Source:**

- Command: AST function-length check (`handleAgents ... lines=57`, `runServe ... lines=179`, `runDaemonLoop ... lines=138`)
- Command: `python3` route count on `cmd/orch/serve.go`
- `cmd/orch/serve_agents.go:87`
- `cmd/orch/serve_agents_enrich.go:259`

**Significance:**
The earlier "single giant function" assumption is now mostly corrected. Remaining risk is **degraded** runtime scaling from per-agent enrichment and broad serve responsibilities rather than line-count monoliths.

---

### Finding 6: Tmux control path assumes one main socket context

**Evidence:**

- Global `mainSocket` is detected once and reused; command wrapper prepends one `-S` target when inside overmind.
- No pool/sharding of tmux servers in orchestration layer.

**Source:**

- `pkg/tmux/tmux.go:17`
- `pkg/tmux/tmux.go:110`

**Significance:**
This remains **latent**. It is not today's top bottleneck, but at 50+ workers it can become a control-plane chokepoint and failure domain concentration.

---

## Synthesis

**Key Insights:**

1. **Shift from structural to operational scaling risk** - the codebase already addressed some giant-function risks; now failures are dominated by stale state, payload growth, and long-tail reconciliation.

2. **Three assumptions fail earliest under load** - (a) state rows stay in sync with live sessions, (b) spawn context remains reasonably sized, and (c) append-only events stay cheap to read.

3. **Socket-level singletons are second-wave risks** - single beads and tmux socket assumptions are less urgent than drift/bloat growth, but become serious as concurrency moves toward 50+.

**Answer to Investigation Question:**

At 20+ concurrent agents, the most credible breakpoints are state/registry drift (**broken**), spawn-context bloat (**degraded→broken**), and unbounded events growth (**degraded**). Prior suspected monolith issues are partially outdated (key handlers are now split), while single-socket beads/tmux assumptions remain **latent** and should be queued after the first three fixes unless near-term scaling targets exceed current operating envelope.

---

## Structured Uncertainty

**What's tested:**

- ✅ State drift quantified from live system: `state_open_rows=166` vs `live_agents=18`.
- ✅ Registry staleness measured from `~/.orch/sessions.json` (`stale_non_active=84`).
- ✅ events.jsonl growth measured (`12,693,779 bytes`, `32,293` lines).
- ✅ Spawn context footprint measured in active workspace (`~18k token estimate`).
- ✅ Beads mitigation behavior validated via targeted tests in `pkg/beads`.
- ✅ Dashboard handler architecture delta validated via AST function-length checks.

**What's untested:**

- ⚠️ Throughput/latency benchmark at true 20/50/100-agent synthetic load.
- ⚠️ Failure behavior with multiple beads daemons or tmux server shards (not implemented).
- ⚠️ End-to-end SLO impact of events compaction strategy options.

**What would change this:**

- If load-test data shows stable latency with current drift and file sizes, severity for events/context could be lowered.
- If state reconciliation already runs elsewhere and closes stale rows automatically, Finding 1 severity drops.
- If per-agent enrichment is moved to push/SSE materialization, dashboard degradation risk drops significantly.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation                                                                 | Authority     | Rationale                                                      |
| ------------------------------------------------------------------------------ | ------------- | -------------------------------------------------------------- |
| Add hard reconciliation/closure path for stale state rows and registry records | architectural | Crosses daemon, status, completion, and storage components     |
| Add final spawn-context token budget gate with deterministic truncation order  | architectural | Crosses spawn context assembly and model execution reliability |
| Add events retention/rotation + compacted read path                            | architectural | Crosses logger, serve endpoints, and observability assumptions |
| Add optional multi-socket strategy for beads/tmux                              | architectural | Cross-component control-plane design change                    |

### Recommended Approach ⭐

**Risk-first scaling hardening** - fix the three highest blast-radius growth failures first, then address latent socket topology.

**Why this approach:**

- Resolves currently observed breakage (`open_minus_live=148`) before latent concerns.
- Reduces immediate failure probability for long-context model workers.
- Improves serve/dashboard stability under sustained event volume.

**Trade-offs accepted:**

- Defers more ambitious architecture rewrites (event bus, distributed scheduler).
- Keeps single-socket control paths in place short term.

**Implementation sequence (prioritized by severity + blast radius):**

1. **State reconciliation hardening (broken / system-wide blast radius):** periodic close-or-archive pass that reconciles `state.db` and `sessions.json` against live OpenCode sessions and workspace markers; emit events for each correction.
2. **Spawn context budget enforcement (degraded / worker reliability blast radius):** enforce a strict final size budget at context write time (not just KB slice), with role-aware truncation and explicit omitted-section metadata.
3. **events.jsonl lifecycle controls (degraded / dashboard+analytics blast radius):** implement rotation policy and compact index/summary path for serving recent and aggregate queries.
4. **Socket topology evolution (latent / high-concurrency blast radius):** design optional per-project or pool-based beads/tmux socket routing; gate rollout behind >50-agent benchmark need.

### Alternative Approaches Considered

**Option B: Multi-socket first**

- **Pros:** Future-proofs high concurrency.
- **Cons:** Does not fix current observed drift/bloat regressions.
- **When to use instead:** If near-term target is immediate 50-100 concurrent workers.

**Option C: Incremental tuning only (cache/TTL tweaks)**

- **Pros:** Lower implementation effort.
- **Cons:** Treats symptoms, not root growth assumptions.
- **When to use instead:** Short emergency window before major release.

**Rationale for recommendation:** Current evidence shows drift and payload growth are active now; fixing them yields immediate reliability gains with lower risk than first tackling socket topology.

---

### Implementation Details

**What to implement first:**

- Reconciliation job + command (`orch doctor state-reconcile` style) that marks/archives stale rows.
- Spawn context final-size checker with deterministic reduction order and warnings.
- events logger retention policy (size and age thresholds) plus reader fallback for rotated files.

**Things to watch out for:**

- ⚠️ Avoid deleting historical records needed for audit; archive instead of destructive purge.
- ⚠️ Truncation policy must preserve safety-critical instructions before historical context.
- ⚠️ Rotation should keep SSE consumers stable while file handles roll.

**Areas needing further investigation:**

- Benchmark-driven threshold selection (context max chars/tokens, events rollover size, reconciliation cadence).
- Whether to promote state.db as sole operational source with explicit materializers.

**Success criteria:**

- ✅ `open_minus_live` consistently near zero after reconciliation windows.
- ✅ Spawn contexts remain under configured token budget without critical instruction loss.
- ✅ Dashboard/event endpoints stay within latency targets while events continue growing over time.

---

## References

**Files Examined:**

- `.kb/investigations/2026-02-06-audit-architecture-audit-orch-go-codebase.md` - prior architecture baseline and hotspot claims.
- `.kb/investigations/2026-02-09-inv-beads-reliability-synthesis-cohesive-root.md` - beads reliability root cause synthesis.
- `.kb/models/daemon-autonomous-operation.md` - daemon scaling invariants and failure model.
- `pkg/session/registry.go` - flat-file registry mechanics.
- `pkg/events/logger.go` - append-only event log behavior.
- `pkg/spawn/context_template.go` and `pkg/spawn/context.go` - spawn context assembly path.
- `pkg/beads/client.go` - subprocess caps and socket lookup behavior.
- `pkg/tmux/tmux.go` - main socket detection/reuse.
- `cmd/orch/serve.go`, `cmd/orch/serve_agents.go`, `cmd/orch/serve_agents_enrich.go` - current serve/dashboard architecture.

**Commands Run:**

```bash
# Phase + project verification
orch phase orch-go-svs0p Planning "Starting scaling assumptions audit and evidence collection"
pwd

# Create investigation artifact and report path
kb create investigation scaling-assumptions-audit-systematically-examine
bd comment orch-go-svs0p "investigation_path: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-09-inv-scaling-assumptions-audit-systematically-examine.md"

# Measure file/function scale and runtime state
wc -l cmd/orch/serve.go cmd/orch/serve_agents.go cmd/orch/serve_agents_collect.go cmd/orch/serve_agents_enrich.go pkg/events/logger.go pkg/session/registry.go pkg/spawn/context_template.go
python3 (sessions.json counts)
python3 (events.jsonl bytes/lines)
python3 (SPAWN_CONTEXT size/tokens)
python3 (route count in serve.go)
python3 (state.db open rows vs live agents)

# Validate known mitigations still hold
go test ./pkg/beads/... -run 'TestRunBDCommand_AddsQuietByDefault|TestRunBDCommand_RetriesAllowStaleWhenOutOfSyncJSONErrorPayload' -count=1
go test ./cmd/orch -run 'TestHandleAgents|TestReadLastNEvents' -count=1
```

**Related Artifacts:**

- **Investigation:** `.kb/investigations/2026-02-06-audit-architecture-audit-orch-go-codebase.md` - superseded hotspot sizing baseline.
- **Investigation:** `.kb/investigations/2026-02-09-inv-beads-reliability-synthesis-cohesive-root.md` - beads mitigation status.
- **Model:** `.kb/models/daemon-autonomous-operation.md` - daemon behavioral constraints.

---

## Investigation History

**2026-02-09 10:38:** Investigation started

- Initial question: scaling assumptions that break/degrade/latent at 20+ concurrent agents.
- Context: spawned architecture audit with explicit confirmed/suspected gap list.

**2026-02-09 10:43:** Baseline evidence captured

- Collected live counts for state.db, sessions registry, events log size, and current spawn-context payload size.

**2026-02-09 10:49:** Model/code validation completed

- Verified beads mitigation tests pass and confirmed dashboard/daemon functions were split compared to prior baseline.

**2026-02-09 10:54:** Investigation completed

- Status: Complete
- Key outcome: prioritized fix order established with severity/blast-radius classification; three immediate hardening targets identified before latent socket-topology work.
