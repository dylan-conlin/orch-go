# Agent State Architecture Model (Feb 2026)

**Summary:** Agent state is currently assembled at query time via a distributed JOIN across 6 systems (OpenCode API, Beads, Tmux, Registry, Workspace disk, Anthropic API). A proposed SQLite single-source migration has been investigated, evaluated, and scoped down to a "Projection-First" approach. This model captures the current architecture, its failure modes, and the validated path forward.

**Synthesized from:** 5 investigations, 1 prior decision, direct orchestrator engagement.

---

## Core Mechanism: The Distributed JOIN

`orch status` currently queries 6 systems to assemble the complete picture of an agent:

```
OpenCode API (session, tokens, model, processing state)  ~5-50ms/call, 5.8s unfiltered
    +
Beads (issue status, labels, phase via comments)          ~700ms/issue
    +
Tmux (window exists, pane activity)                       <50ms/call
    +
Registry (~/.orch/agent-registry.json)                    <10ms
    +
Workspace disk (.orch/workspace/{name}/)                  <5ms
    +
Anthropic API (account usage)                             ~200ms
    =
Wall-clock: 730ms (optimized) to 23.6s (pre-optimization)
```

### Field Categories

The field-level audit identified **~30 fields** organized by source:

| Category | Count | Source | Freshness | Stability |
|----------|-------|--------|-----------|-----------|
| Core Identity | 5 | Mixed (spawn-time) | Stale-ok (immutable) | ✅ Stable |
| Session State | 11 | OpenCode API | Seconds (real-time) | ⚠️ Derived fields |
| Beads State | 9 | Beads RPC/CLI | Minutes | ⚠️ Phase parsed from comments |
| Tmux State | 4 | Tmux subprocess | Real-time | ✅ Authoritative |
| Workspace | Varies | Disk | Stale-ok | ✅ Stable |

**Key insight:** Most latency comes from the OpenCode session list (sequential file reads) and Beads RPC (~700ms per issue). Core identity fields are immutable after spawn and shouldn't need network calls.

---

## Why This Fails

### Failure Mode 1: Performance Cliff at Scale
- OpenCode `Session.list()` does sequential `Storage.read()` per session on disk
- 89 sessions → 5.8s for unfiltered list (O(n) in disk I/O, no parallelism)
- **Mitigated:** Server-side `?start=` filter + parallelization (5.8s → 4.7ms)

### Failure Mode 2: State DB Identity Drift
- `state.db` stores single row per beads ID (`beads_id UNIQUE`)
- When issues are respawned, new workspace is created but state.db keeps old row
- `orch abandon` trusted stale row → resolved to wrong workspace → deleted wrong session
- **Root cause:** Insert-only writes, no respawn-safe update path, session/tmux fields never populated

### Failure Mode 3: Registry Drift (Historical)
- `agent-registry.json` was supposed to be spawn cache
- Became pseudo-authoritative without reconciliation
- Decision (2026-01-12): Registry is explicitly "spawn cache, not source of truth"
- **Lesson:** Any local state without hard reconciliation drifts to unreliable

### Failure Mode 4: Phase Derived from Comments
- Agent phase is parsed from beads comments via regex `Phase:\s*(\w+)`
- Comments are append-only, so "latest phase" requires scanning all comments
- Phase is the most queried runtime field but has the worst access pattern

---

## The Validated Path Forward: Projection-First Architecture

### What Was Proposed
A full SQLite single-source of truth with SSE event materialization, replacing the distributed JOIN.

### What the Review Found
The review tested the design against 7 critical questions and found:

| Question | Verdict |
|----------|---------|
| Over-architecting past simpler fix? | **Yes** — Session filter already killed the main cliff |
| Operational surface area too large? | **Yes** — SSE + backfill + polling adds complexity |
| Repeating registry drift pattern? | **Risk** — Without reconciliation, it's a new registry |
| Dual-write timeline realistic? | **Needs forcing functions** — Otherwise permanent shadow mode |
| Will `orch phase` be adopted? | **Not yet** — Command doesn't exist, templates still say `bd comment` |
| UX improvement vs complexity? | **Reframe** — Primary beneficiary is dashboard/daemon, not CLI |
| Beads-audit vs runtime separation clean? | **Conceptually yes, operationally incomplete** |

### Recommended Phased Approach

**Phase A (low-risk):** SQLite as projection of immutable + command-owned fields (spawn, complete, abandon) with periodic read-reconciliation. No authority claims.

**Phase B (measured):** Add phase/runtime ingestion only AFTER `orch phase` command exists and template migration reaches target adoption. This is currently blocked.

**Phase C (gated):** Promote SQLite to primary runtime read source only when shadow discrepancy SLO is met for a fixed window and bail-out path is tested.

### Blocking Dependencies

| Dependency | Status | Required For |
|-----------|--------|-------------|
| `orch phase` command | ❌ Not implemented | Phase B (agent writes phase to state.db) |
| Template migration (bd comment → orch phase) | ❌ Not started | Phase B (agent adoption) |
| Shadow discrepancy telemetry | ❌ Not built | Phase C (cutover safety) |
| Reconciliation job | ❌ Not built | Phase A (drift prevention) |

---

## OpenCode Integration Points

The fork audit mapped the complete write path:

| Lifecycle Event | Storage Path | Bus Event | Integration Option |
|----------------|-------------|-----------|-------------------|
| Session create | `storage/session/{projID}/{sesID}.json` | `session.created` | SSE subscribe or webhook |
| Session update | Same | `session.updated` | SSE subscribe |
| Message write | `storage/message/{sesID}/{msgID}.json` | `message.updated` | SSE subscribe (high frequency!) |
| Part write | `storage/part/{msgID}/{partID}.json` | `part.updated` | SSE subscribe (very high frequency) |

**Key finding:** OpenCode is disk-backed JSON + in-process Bus + SSE broadcast. No database, no WAL. Every write is `Bun.write()`, every read is `Bun.file().json()`.

**Integration options evaluated:**
1. **SSE subscription** — Low-latency, but high-frequency part events create noise
2. **Polling** — Simpler, but adds latency and missed events
3. **Direct file watch** — Brittle, OS-dependent

The recommended approach is SSE for session lifecycle events only (create/update/status), NOT for message/part events (too frequent, and we only care about aggregate state).

---

## Constraints

- **OpenCode doesn't expose session state change events via structured format** — Phase, tokens, model are all derived from message content
- **Beads comments are the canonical phase record** — Until `orch phase` exists, this won't change
- **state.db must remain non-fatal** — Current writes are explicitly non-fatal; making them fatal without reconciliation would create new failure modes
- **Registry will eventually be deprecated** — But only after state.db reaches Phase C authority

---

## Actionable Next Steps

1. **Implement respawn-safe state semantics** — state.db needs historical attempts + current pointer (not single row per beads ID)
2. **Wire `RecordSessionID`/`RecordTmuxWindow`** — Currently declared but never called at runtime
3. **Harden `orch abandon` resolution** — Prefer live workspace/session coherence checks over stale cache rows
4. **Design `orch phase` command** — Prerequisite for Phase B; should write to both state.db and beads comment
5. **Build reconciliation job** — Periodic check that state.db matches reality; auto-correct drift

---

## References

### Investigations (Provenance Chain)
- `2026-02-06-inv-agent-state-field-level-audit.md` — Complete field map across 6 systems
- `2026-02-06-inv-evaluate-single-source-agent-state.md` — 7-question review of proposed design
- `2026-02-06-inv-opencode-fork-audit-session-lifecycle-integration.md` — Write paths and hooks
- `2026-02-06-inv-opencode-session-list-performance-cliff.md` — Performance cliff diagnosis
- `2026-02-06-inv-determine-worker-orch-go-21398.md` — State drift causing wrong-workspace abandon

### Decisions
- `2026-01-12-registry-is-spawn-cache.md` — Registry drift lesson
- `2026-02-06-completion-pipeline-parallel-redesign.md` — Completion bottleneck (related: batch processing)
