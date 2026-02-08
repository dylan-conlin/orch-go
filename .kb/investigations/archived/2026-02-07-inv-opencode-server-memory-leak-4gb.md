<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Memory retention is primarily tied to unbounded per-directory `Instance` caching and associated bootstrap state, not a global in-memory cache of session/message records.

**Evidence:** Isolated server experiments show RSS +50,480 KB after touching 120 unique directories, cached git context remains after directory deletion until `/instance/dispose`, and SSE connections closed via `stream.close()` on `server.instance.disposed` never emit `event disconnected`.

**Knowledge:** OpenCode server mode keeps `Instance` contexts alive indefinitely by directory key, and each context bootstraps multiple subscriptions/services that only clean up on explicit instance disposal.

**Next:** Implement architectural fixes for instance eviction and SSE close-path cleanup, then re-run multi-hour memory profiling to validate bounded growth.

**Authority:** architectural - Fixes cross subsystem boundaries (`server`, `instance`, `bus`, lifecycle policy) and require orchestrator-level design choice.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Opencode Server Memory Leak 4gb

**Question:** What OpenCode server components retain memory over long uptime (multi-hour to multi-day), and which mechanisms explain observed RSS growth from session/message/event/SSE state?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** OpenCode worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-02-07-inv-actually-kills-opencode-server-process.md | extends | yes | Prior file left leak source unknown; this investigation narrows source to instance lifecycle retention and SSE close-path behavior. |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Investigation kickoff and hypothesis framing

**Evidence:** Spawn context reports severe RSS growth (8.4GB multi-day, 1.58GB after 13.5h) with 254 sessions and 768MB session data on disk, and asks for verification across four candidate retention paths: sessions, JS message objects, event listeners, and SSE cleanup.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-opencode-server-memory-07feb-58f1/SPAWN_CONTEXT.md` (read via `python3` in shell due Read tool stack overflow)

**Significance:** Establishes bounded hypotheses and target evidence needed: direct code-path verification plus runtime testing (heap/profile/behavior) before any conclusion.

---

### Finding 2: Session metadata is disk-backed and reloaded on demand, not kept in a global in-memory session cache

**Evidence:** `Session.list()` iterates `Storage.list(["session", project.id])` and reads each record from disk (`Storage.read`) per call instead of returning from an in-memory map. In runtime testing, a created session remains retrievable after `/instance/dispose`, showing session data persistence is storage-backed rather than instance-memory-backed.

**Source:** `packages/opencode/src/session/index.ts:332`, `packages/opencode/src/session/index.ts:334`, `packages/opencode/src/session/index.ts:335`, `packages/opencode/src/storage/storage.ts:169`, `packages/opencode/src/storage/storage.ts:212`; runtime command `Verify sessions persist across instance disposal from disk` (created session count before dispose=1, after dispose=1, IDs unchanged).

**Significance:** The large disk session count (254 sessions/768 MB) does not by itself prove in-memory session retention; root cause must be sought in other long-lived runtime structures.

---

### Finding 3: Server mode caches `Instance` contexts indefinitely by directory and only evicts on explicit dispose

**Evidence:** `Instance` stores contexts in `cache = new Map<string, Promise<Context>>()`, reused by `Instance.provide()` and removed only via `Instance.dispose()`/`disposeAll()`. Runtime test: a git repo path still resolves to its old git worktree after the repo directory is deleted, proving cached context survives filesystem disappearance; after calling `/instance/dispose`, same path falls back to non-git worktree `/`.

**Source:** `packages/opencode/src/project/instance.ts:15`, `packages/opencode/src/project/instance.ts:23`, `packages/opencode/src/project/instance.ts:38`, `packages/opencode/src/project/instance.ts:69`, `packages/opencode/src/project/instance.ts:72`, `packages/opencode/src/server/server.ts:187`, `packages/opencode/src/server/server.ts:197`; runtime commands `Test instance cache persistence after deleting git directory`, `Establish missing directory baseline without instance cache`, and `Verify instance dispose clears cached directory context`.

**Significance:** This is a direct memory growth mechanism in long-lived servers serving many unique workspace paths (matching orch-go multi-agent usage).

---

### Finding 4: Per-instance bootstrap allocates repeated subscriptions/services, creating listener accumulation when many directories are touched

**Evidence:** Each new instance runs `InstanceBootstrap`, which initializes plugin/share/format/lsp/filewatch/vcs and adds bus subscribers. Controlled test over 3 unique directories produced 3 bootstraps and 30 subscription log entries (~10 per instance). Stress run over 120 directories increased server RSS by 50,480 KB without any prompts/models, showing baseline per-instance retention cost.

**Source:** `packages/opencode/src/project/bootstrap.ts:17`, `packages/opencode/src/project/bootstrap.ts:30`, `packages/opencode/src/plugin/index.ts:128`, `packages/opencode/src/format/index.ts:105`, `packages/opencode/src/share/share-next.ts:22`, `packages/opencode/src/bus/index.ts:20`, `packages/opencode/src/bus/index.ts:92`; runtime commands `Measure subscription log growth across multiple instance bootstraps` and `Run isolated server memory and instance cache experiment`.

**Significance:** Event/listener accumulation is real at the instance-lifecycle level even if individual session/message records are not globally cached.

---

### Finding 5: SSE cleanup works for client-abort path but appears incomplete for server-initiated `stream.close()` path

**Evidence:** `/event` route subscribes to bus and starts heartbeat interval, with cleanup (`clearInterval`, `unsub`) only inside `stream.onAbort`. In normal disconnect test, logs show connected=1/disconnected=1. In close-via-dispose test, stream receives `server.instance.disposed` and request completes, but disconnected logs remain 0 across 40 runs (connected=40/disconnected=0), indicating onAbort path does not fire in this closure path.

**Source:** `packages/opencode/src/server/server.ts:503`, `packages/opencode/src/server/server.ts:513`, `packages/opencode/src/server/server.ts:523`, `packages/opencode/src/server/server.ts:524`, `packages/opencode/src/server/server.ts:525`, `packages/opencode/src/server/server.ts:527`, `packages/opencode/src/server/server.ts:508`; runtime commands `Run isolated server memory and instance cache experiment`, `Reproduce SSE dispose cleanup behavior with inspectable logs`, and `Stress SSE close-via-dispose path for cleanup leak signals`.

**Significance:** SSE lifecycle is partially correct but has a likely cleanup gap on server-driven close that can leak timers/subscriber closures over time.

---

## Synthesis

**Key Insights:**

1. **Primary retention is instance-scoped, not session-scoped** - Session/message records are persisted to disk and read lazily, while per-directory instance contexts stay resident until explicit disposal.

2. **Directory cardinality is the pressure multiplier** - Each unique `x-opencode-directory` path causes another bootstrap of services/subscriptions, which aligns with orch-go workspace fan-out behavior.

3. **SSE cleanup has an asymmetric lifecycle** - Client-disconnect path cleans up, but server-initiated close path likely bypasses cleanup callbacks.

**Answer to Investigation Question:**

OpenCode's observed long-uptime memory growth is best explained by unbounded per-directory `Instance` retention plus per-instance bootstrap allocations (listeners/services), not by a central cache of all sessions/messages in JS memory. Core session data is storage-backed and reloaded on demand, but server mode keeps `Instance` caches indefinitely unless `/instance/dispose` or `/global/dispose` is invoked. Event listener accumulation exists at instance bootstrap scale, and SSE cleanup appears incomplete when streams are closed by server-side `stream.close()` on instance disposal. Heap-object-level attribution (exact retained object sizes) remains unverified because Bun heap inspector was not successfully attachable to the compiled binary in this environment.

---

## Structured Uncertainty

**What's tested:**

- ✅ Instance cache persists across requests and survives directory deletion until explicit dispose (verified: git repo delete test retained previous worktree; after `/instance/dispose` worktree reset to `/`).
- ✅ Session records are available after instance disposal (verified: created session list count stayed 1 before and after dispose, with identical session IDs).
- ✅ SSE client-abort cleanup path works (verified: connected=1/disconnected=1 when client process terminated).
- ✅ SSE close-via-instance-dispose path does not emit disconnect cleanup signal (verified: connected=40/disconnected=0 while stream received `server.instance.disposed` and request completed).
- ✅ Touching many unique directories grows RSS even without model prompts (verified: +50,480 KB after 120 directories in isolated server run).

**What's untested:**

- ⚠️ Exact retained-object composition (timers, closures, LSP/filewatch internals, plugin hook objects) from heap snapshots.
- ⚠️ Multi-day production growth curve after controlling directory cardinality and explicit instance disposal.
- ⚠️ ACP session manager impact in real workloads (`ACPSessionManager.sessions` has no explicit eviction, but ACP usage was not exercised here).

**What would change this:**

- If heap snapshots show the dominant retained set is unrelated to instance cache/bootstrap state.
- If adding automatic instance eviction does not materially reduce long-run RSS slope in production-like load.
- If instrumentation proves `stream.close()` internally invokes equivalent cleanup for heartbeat/unsubscribe despite missing `event disconnected` logs.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add automatic instance eviction policy (TTL/LRU and/or max live instances) in server mode | architectural | Crosses request handling, instance lifecycle, and state cleanup semantics. |
| Refactor SSE `/event` route cleanup so `stream.close()` and abort share the same teardown path | implementation | Localized code change in route logic with clear correctness criterion. |
| Add lifecycle/memory instrumentation (instance count, subscription count, timer count, RSS trend) | architectural | Introduces observability contracts used across reliability and operations decisions. |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Bounded Instance Lifecycle + Deterministic SSE Teardown** - Limit the number/lifetime of live instances and unify SSE cleanup paths so every stream close executes teardown exactly once.

**Why this approach:**
- Directly targets the strongest tested retention vector (instance cache by directory).
- Addresses likely SSE leak gap without waiting for full heap tooling.
- Produces measurable validation signals (instance cardinality, disconnect parity, RSS slope).

**Trade-offs accepted:**
- Slightly higher latency when evicted instances are recreated.
- Additional complexity in lifecycle policy and observability wiring, justified by multi-GB crash-risk reduction.

**Implementation sequence:**
1. Add central instance-eviction policy (TTL + max entries) and emit structured logs/metrics for create/evict/dispose counts.
2. Refactor `/event` handler to use a single `cleanup()` function invoked by both abort and internal close events.
3. Run controlled long-duration load with many unique directories, compare RSS slope before/after, and iterate thresholds.

### Alternative Approaches Considered

**Option B: Periodic full server restart (`--auto-restart`) only**
- **Pros:** Fast operational mitigation.
- **Cons:** Masks retention source and still allows large memory excursions between restarts.
- **When to use instead:** Immediate reliability stopgap while lifecycle fix is in progress.

**Option C: Session/message compaction focus only**
- **Pros:** Can reduce model-context processing costs.
- **Cons:** Does not address tested instance-cache retention root cause.
- **When to use instead:** After instance lifecycle is bounded and session payload size still dominates.

**Rationale for recommendation:** Option A directly matches tested behavior, while Options B/C either defer root-cause correction or optimize a weaker signal.

---

### Implementation Details

**What to implement first:**
- Instance cardinality bounds in `Instance` cache with explicit eviction and disposal.
- Shared SSE cleanup function in `/event` route.
- Lightweight counters/logging for live instances and per-route connect/disconnect parity.

**Things to watch out for:**
- ⚠️ Evicting an active instance could interrupt in-flight operations if policy is not activity-aware.
- ⚠️ Disposal correctness depends on all `Instance.state(..., dispose)` hooks being robust and non-blocking.
- ⚠️ SSE cleanup must remain idempotent to avoid double-unsubscribe/double-clear errors.

**Areas needing further investigation:**
- Heap-object attribution with inspector/snapshot tooling on a debuggable build.
- ACP session manager lifecycle behavior under sustained ACP usage.
- Long-horizon test against production-like orch-go workspace churn.

**Success criteria:**
- ✅ Live instance count remains bounded under unbounded unique-directory traffic.
- ✅ SSE connect/disconnect counts match for both client-abort and server-initiated close paths.
- ✅ RSS growth slope is materially reduced in multi-hour runs versus current baseline.

---

## References

**Files Examined:**
- `packages/opencode/src/project/instance.ts` - Verified cache lifecycle and disposal conditions.
- `packages/opencode/src/server/server.ts` - Verified request-to-instance binding and SSE event route cleanup logic.
- `packages/opencode/src/project/bootstrap.ts` - Verified per-instance service initialization.
- `packages/opencode/src/bus/index.ts` - Verified subscription storage/unsubscribe behavior.
- `packages/opencode/src/session/index.ts` - Verified session list/messages retrieval pattern.
- `packages/opencode/src/storage/storage.ts` - Verified persistence/read/list behavior.
- `packages/opencode/src/session/prompt.ts` - Verified active prompt state and callback cleanup strategy.
- `packages/opencode/src/server/routes/global.ts` - Verified global SSE cleanup behavior.

**Commands Run:**
```bash
# Initial phase and investigation setup
orch phase orch-go-21437 Planning "Investigating opencode server memory leak behavior"
kb create investigation opencode-server-memory-leak-4gb
bd comment orch-go-21437 "investigation_path: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-07-inv-opencode-server-memory-leak-4gb.md"

# Code inspection sweeps
rg -n "x-opencode-directory|session|Bus.subscribe|streamSSE|onAbort|Instance.state" packages/opencode/src

# Runtime experiments (isolated servers)
# 1) 120 unique directories memory delta
# 2) cache persistence across deleted git directory
# 3) missing-directory baseline
# 4) dispose clears cached git context
# 5) sessions persist across dispose
# 6) SSE normal disconnect parity
# 7) SSE close-via-dispose disconnect gap
# 8) SSE stress (40 cycles) connected/disconnected counts
```

**External Documentation:**
- N/A (primary-source code and runtime behavior only).

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-07-inv-actually-kills-opencode-server-process.md` - Prior incident analysis identifying memory-pressure death without source attribution.
- **Workspace:** `.orch/workspace/og-inv-opencode-server-memory-07feb-58f1/SPAWN_CONTEXT.md` - Task framing and constraints.

---

## Investigation History

**[2026-02-07 17:40]:** Investigation started
- Initial question: Which OpenCode components are retaining memory over long uptime (sessions/messages/listeners/SSE)?
- Context: Prior investigation proved memory-pressure death (~8.4 GB RSS) but did not identify the retention mechanism.

**[2026-02-07 18:05]:** Core retention path identified
- Determined `Instance` cache is unbounded by directory key and persists until explicit dispose; reproduced stale cached git context after directory deletion.

**[2026-02-07 18:15]:** Investigation completed
- Status: Complete
- Key outcome: Memory growth is primarily instance-lifecycle retention (plus potential SSE close-path cleanup gap), not a global session/message cache.
