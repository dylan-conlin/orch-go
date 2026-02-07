# Investigation: OpenCode Session.list() Performance Cliff

**Date:** 2026-02-06
**Status:** Complete
**Severity:** P0 (blocked all orchestration)
**Time to resolve:** ~1 hour

## Symptom

`orch status` degraded from sub-second to 23.6 seconds, eventually timing out entirely. This blocked all orchestration activity — status, spawning, completion, and monitoring all depend on status being fast.

## Root Cause

OpenCode's `Session.list()` (in `packages/opencode/src/session/index.ts:332-339`) performs **sequential `Storage.read()` calls per session**:

```typescript
export async function* list() {
    const project = Instance.project
    for (const item of await Storage.list(["session", project.id])) {
        const session = await Storage.read<Info>(item).catch(() => undefined)
        if (!session) continue
        yield session
    }
}
```

Each `Storage.read()` is a filesystem read of a JSON file. With 89 sessions on disk, this means 89 sequential file reads — O(n) in disk I/O with no parallelism.

## Why It Hit Now (Not Before)

Two factors converged:

1. **Unprecedented agent volume:** 47 commits in 2 days from 40+ spawned agents, each creating an OpenCode session. Previous peak was ~15 sessions.

2. **Server restart loads all disk sessions.** When OpenCode starts, `Storage.list()` enumerates all session files on disk. The prior known constraint ("x-opencode-directory header returns ALL disk sessions") became acute at this scale.

The rebase to upstream v1.1.52 did **not** change `Session.list()` — it has always been sequential. This is a scaling cliff, not a regression.

## Profiling Data

Measured with `ORCH_STATUS_DEBUG=1` (44 in-memory sessions at time of measurement):

| Operation | Wall Time | Root Cause |
|-----------|-----------|------------|
| ListSessions (no filter) | 5.8s | 44 sequential Storage.read() calls |
| Parallel enrichment (2 sessions) | 4.1s | GetMessages fetches full message history per session |
| GetCommentsBatch (7 beads IDs) | 5.2s | Beads RPC ~700ms per issue |
| GetIssuesBatch (7 beads IDs) | 4.0s | Beads RPC ~700ms per issue |
| Token fetch + risk assessment | 4.3s | Per-session token counting |
| **Total (all sequential)** | **23.6s** | |

After fix (server-side `?start=` filter + parallelization):

| Operation | Wall Time | Improvement |
|-----------|-----------|-------------|
| ListSessions (with ?start=) | 4.7ms | **1,234× faster** |
| Parallel data fetch (enrichment + comments + issues) | 597ms | 3 ops concurrent |
| Accounts (async from start) | 0ms (hidden) | Started in goroutine |
| **Total** | **730ms** | **32× faster** |

## Fix Applied

### 1. Server-side session filtering (biggest impact)

The OpenCode `/session` endpoint already supports query parameters that orch-go never used:

```
?start=<timestamp_ms>   - Filter by time.updated >= start
?limit=<n>              - Cap results
?search=<term>          - Filter by title
?roots=true             - Exclude child sessions
```

Added `ListSessionsWithOpts()` to the Go client. In compact mode (default), status now passes `?start=<1-hour-ago>` which tells the server to skip `Storage.read()` for old sessions entirely.

**Files:** `pkg/opencode/client.go` (new `ListSessionsOpts` struct + `ListSessionsWithOpts` method)

### 2. Combined session enrichment (halved HTTP calls)

Previously: `GetSessionModel()` + `IsSessionProcessing()` = 2× `GetMessages()` per session.
Now: `GetSessionEnrichment()` = 1× `GetMessages()` returning model + processing + tokens.

**Files:** `pkg/opencode/client.go` (new `SessionEnrichment` struct + `GetSessionEnrichment` method)

### 3. Parallelized all independent operations

Before: enrichment → comments → issues → accounts → infra (all sequential)
After:
- Account usage + infra health started as goroutines at function entry
- Enrichment + comments + issues run concurrently via `sync.WaitGroup`
- Token fetch skips sessions already enriched in parallel batch

**Files:** `cmd/orch/status_cmd.go` (restructured `runStatus()`)

### 4. Debug timing instrumentation

Added `ORCH_STATUS_DEBUG=1` env var that prints per-operation timing to stderr. This was instrumental in diagnosing the issue and will help catch future regressions.

## Key Insight

The OpenCode API already had the solution (`?start=` parameter) — we just weren't using it. The parameter was added to the TypeScript route handler but never documented or surfaced to API consumers. Discovery required reading the OpenCode source code directly.

## Remaining Risks

1. **`--all` mode is still slow.** It passes no `?start=` filter, so it reads all sessions. Acceptable since it's explicitly requesting everything, but could timeout with 100+ sessions.

2. **Beads RPC is slow (~700ms per issue).** The batch functions use parallelism but each individual RPC call is expensive. At 20+ tracked agents, this will become the next bottleneck.

3. **OpenCode server degrades under load.** With many active sessions doing concurrent `GetMessages` calls, the single-threaded Bun server can become a bottleneck. The restart-to-fix pattern is a band-aid.

4. **No session garbage collection.** Old sessions accumulate on disk indefinitely. Need periodic cleanup or TTL-based expiration.

## Architectural Observation

`orch status` is fundamentally a **distributed JOIN at query time** across 6 systems (OpenCode, beads, Anthropic API, tmux, registry, disk). Every invocation re-derives the full picture. The long-term fix is a **write-time materialized view** (SQLite or similar) where state changes write to a local DB and status becomes a simple SELECT. This investigation confirms the current architecture won't scale past ~50 concurrent sessions without increasingly aggressive server-side filtering.

## Related

- Constraint: `kb-62d331` — "OpenCode Session.list() does sequential Storage.read() per session"
- Beads issue: `orch-go-21361` — "orch status takes 23s - parallelize all 5 bottlenecks"
- Prior constraint: "OpenCode x-opencode-directory header returns ALL disk sessions"
