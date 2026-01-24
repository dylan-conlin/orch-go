<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Current file-path worker detection (SPAWN_CONTEXT.md reads, .orch/workspace/ paths) fires too late because detection signals only appear in tool arguments after the first tool call, but coaching alerts fire on the same first tool call before detection succeeds.

**Evidence:** Reviewed coaching.ts:1328-1377 detectWorkerSession() - detection checks tool arguments; coaching fires in tool.execute.after hook on same tool call; confirmed at orchestrator-session.ts:262 "we don't have enough info at session.created time to detect workers".

**Knowledge:** Worker detection must happen BEFORE coaching fires, requiring either: (1) early directory-based detection in session.created event, (2) deferred alert queue that waits for detection, or (3) fixing session.metadata.role which is already sent but not reliably exposed.

**Next:** Implement Option A (directory-based detection in session.created event) - add worker sessions to cache based on directory containing `.orch/workspace/` at session creation time.

**Promote to Decision:** recommend-yes - Establishes pattern: "worker detection must use signals available BEFORE first tool call (directory path), not signals only available IN tool arguments".

---

# Investigation: Review Coaching Plugin Worker Detection

**Question:** How should workers be identified early enough to skip coaching alerts? The current approach (detectWorkerSession checking for SPAWN_CONTEXT.md reads and .orch/workspace/ paths) fires too late - coaching alerts trigger on early tool calls before detection has a chance to run.

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** og-arch-review-coaching-plugin-23jan-f6bb
**Phase:** Complete
**Next Step:** None - implementation recommendation complete
**Status:** Complete

<!-- Lineage -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Detection Signals Only Appear In Tool Arguments (Too Late)

**Evidence:** The `detectWorkerSession()` function at coaching.ts:1338-1377 relies on two detection signals:
1. Read tool accessing `SPAWN_CONTEXT.md` (line 1347-1351)
2. Any tool accessing files in `.orch/workspace/` (lines 1355-1367)

Both signals require examining tool arguments (`args.filePath` or `args.file_path`). This means detection can only succeed AFTER a tool call occurs. But coaching alerts fire in the `tool.execute.after` hook (line 1592) - the same hook where detection runs.

**Timeline of a worker's first tool call:**
```
1. tool.execute.before runs
2. Tool executes
3. tool.execute.after runs:
   a. detectWorkerSession() checks args - returns false (first call)
   b. Coaching alerts may fire based on this tool
4. Second tool call - if SPAWN_CONTEXT.md read, NOW detection succeeds
```

**Source:** `plugins/coaching.ts:1338-1377, 1592-1627`

**Significance:** The race condition is fundamental to the current architecture - detection and alerting run in the same hook, but detection needs information from tool calls that haven't happened yet.

---

### Finding 2: session.created Event Has Directory But No Tool Args

**Evidence:** The orchestrator-session.ts plugin documents at line 262: "For now, we don't have enough info at session.created time to detect workers". However, the `session.created` event DOES include `event.properties.sessionID` (line 253) and the plugin has access to `directory` at init time.

Worker sessions are spawned into `.orch/workspace/` directories. The directory path IS available at session creation time - it just hasn't been used for detection.

**Source:** `plugins/orchestrator-session.ts:244-264`, `plugins/coaching.ts:1310` (directory param)

**Significance:** Directory-based detection could work at session.created time, before any tool calls.

---

### Finding 3: session.metadata.role Was Tried But Failed

**Evidence:** Investigation `2026-01-24-inv-fix-coaching-plugin-firing-workers.md` documents that session.metadata.role was previously implemented but was reverted because "metadata may not be set reliably across all spawn paths". The investigation notes "session.metadata.role which isn't reliably set".

The x-opencode-env-ORCH_WORKER=1 header IS being sent (client.go:559-561) but OpenCode apparently doesn't always expose it in session.metadata.

**Source:**
- `2026-01-24-inv-fix-coaching-plugin-firing-workers.md` (recent fix reverted to file-path detection)
- `2026-01-17-inv-update-coaching-plugin-session-metadata.md` (original session.metadata.role implementation)
- `pkg/opencode/client.go:559-561` (header is sent)

**Significance:** session.metadata.role is the "ideal" solution but has reliability issues. Fixing OpenCode to properly expose the header would be the cleanest fix, but requires upstream changes.

---

### Finding 4: Multiple Spawn Paths Set ORCH_WORKER Differently

**Evidence:** Grep shows ORCH_WORKER is set in multiple ways:
- API spawn: `x-opencode-env-ORCH_WORKER=1` header (client.go:561)
- Claude mode spawn: `ORCH_WORKER=1` env var (spawn_cmd.go:1513, 1723)
- Tmux spawns: `ORCH_WORKER=1` env var prefix in command (tmux.go:301, 327)
- Docker spawns: `ORCH_WORKER=1` in container env (claude.go:131)

The environment variable approach works for tmux/claude mode because the agent process inherits the env var. The header approach for API spawns depends on OpenCode processing it correctly.

**Source:** `pkg/opencode/client.go:559-561`, `pkg/tmux/tmux.go:278-301`, `cmd/orch/spawn_cmd.go:1512-1513`

**Significance:** The detection mechanism must work across ALL spawn paths. Directory-based detection is the most universal since all workers operate in `.orch/workspace/` regardless of spawn mode.

---

### Finding 5: Only Caching Positive Results Fixed Permanent Misclassification

**Evidence:** Investigation `2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md` fixed a bug where `isWorker=false` was cached, permanently misclassifying workers whose first tool call didn't trigger detection. The fix: only cache `isWorker=true`.

Current code at coaching.ts:1369-1377:
```typescript
// Only cache positive results - don't cache false
if (isWorker) {
  workerSessions.set(sessionId, true)
}
return isWorker
```

**Source:** `plugins/coaching.ts:1369-1377`, `2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md`

**Significance:** The caching fix helps workers get detected eventually, but doesn't solve the early-firing problem. Workers still receive coaching alerts until their first detection-triggering tool call.

---

## Synthesis

**Key Insights:**

1. **Detection timing is the root problem** - Current detection runs in `tool.execute.after` using tool arguments, but coaching alerts also fire in `tool.execute.after`. Detection succeeds only AFTER a worker-identifying tool call, but alerts fire on every tool call including the first one.

2. **Directory path is available early** - Workers are spawned into `.orch/workspace/{workspace-name}/` directories. This directory is known at session creation time (in the `session.created` event) and could be used for early detection.

3. **session.metadata.role is the ideal but unreliable path** - OpenCode should expose `ORCH_WORKER=1` from the x-opencode-env header, but it doesn't work reliably. Fixing this in OpenCode would be the cleanest solution but requires upstream changes.

4. **Multiple spawn paths require universal detection** - Detection must work for API spawns (headless), Claude mode spawns (tmux), and Docker spawns. Directory-based detection is universal across all paths.

**Answer to Investigation Question:**

Workers should be identified early using **directory-based detection in the session.created event**. When a session is created, check if the directory path contains `.orch/workspace/`. If so, immediately add the sessionID to the `workerSessions` cache. This ensures workers are marked BEFORE any tool calls occur, completely eliminating the race condition.

The current tool-argument-based detection can remain as a backup signal, but the primary detection should happen at session creation time.

---

## Structured Uncertainty

**What's tested:**

- ✅ **Current detection relies on tool args** (verified: read coaching.ts:1338-1377 detectWorkerSession implementation)
- ✅ **session.created event fires before tool calls** (verified: OpenCode event lifecycle in opencode-plugins.md guide)
- ✅ **Workers operate in .orch/workspace/ directories** (verified: spawn creates workspace at pkg/spawn/config.go)
- ✅ **x-opencode-env-ORCH_WORKER header is sent** (verified: client.go:559-561)

**What's untested:**

- ⚠️ **session.created event includes directory** (hypothesis: properties should include directory, needs verification)
- ⚠️ **Directory-based detection has no false positives** (hypothesis: only workers have .orch/workspace/ in path)
- ⚠️ **Performance impact of event hook processing** (hypothesis: minimal, but not measured)

**What would change this:**

- Recommendation would change if session.created event doesn't include directory - would need to use different hook
- Recommendation would change if OpenCode is fixed to reliably expose session.metadata.role - would prefer that approach
- Recommendation would change if there are non-worker sessions with .orch/workspace/ in path - would need additional signals

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Option A: Directory-based detection in session.created event** - Add event hook to coaching.ts that marks sessions as workers when directory contains `.orch/workspace/`.

**Why this approach:**
- Detects workers BEFORE any tool calls occur (Finding 1, 2)
- Uses signal that's available at session creation time (Finding 2)
- Works across ALL spawn paths - API, Claude mode, Docker (Finding 4)
- No upstream OpenCode changes required (Finding 3)

**Trade-offs accepted:**
- Relies on workspace directory convention (but this is stable)
- Doesn't fix the underlying session.metadata.role issue (out of scope)

**Implementation sequence:**
1. Add `event` hook to coaching plugin that handles `session.created`
2. Extract sessionID and directory from event properties
3. If directory contains `.orch/workspace/`, add sessionID to workerSessions cache
4. Existing tool-argument detection remains as backup

### Alternative Approaches Considered

**Option B: Deferred Alert Queue**
- **Pros:** No changes to detection logic; alerts only fire after N tool calls
- **Cons:** Adds complexity; delays coaching for legitimate orchestrator sessions; buffering logic error-prone
- **When to use instead:** If session.created event doesn't include directory

**Option C: Fix session.metadata.role in OpenCode**
- **Pros:** Cleanest architectural solution; detection at session creation via metadata
- **Cons:** Requires upstream OpenCode changes; uncertain timeline; header processing may have reliability issues
- **When to use instead:** If contributing to OpenCode is feasible and fix is accepted

**Option D: Environment variable at plugin init**
- **Pros:** Simple check for ORCH_WORKER=1
- **Cons:** Plugin runs in server process, not agent process; env var not visible (Finding 4)
- **When to use instead:** Never - fundamentally doesn't work for this use case

**Rationale for recommendation:** Option A (directory-based detection) is the most reliable and requires no upstream changes. It uses a signal (directory path) that's inherently available for all worker sessions and available before any tool calls. The existing tool-argument detection provides redundancy.

---

### Implementation Details

**What to implement first:**
1. Add `event` hook to coaching plugin
2. Handle `session.created` event type
3. Extract directory from event (likely `event.properties?.directory` or from session lookup)
4. If directory includes `.orch/workspace/`, add to workerSessions cache

**Example implementation:**
```typescript
event: async ({ event }) => {
  if (event.type !== "session.created") return

  const sessionId = (event as any).properties?.sessionID
  const directory = (event as any).properties?.directory

  if (!sessionId) return

  // Early worker detection via directory path
  if (directory && directory.includes(".orch/workspace/")) {
    workerSessions.set(sessionId, true)
    log(`Worker detected (session.created, directory): ${sessionId}`)
  }
}
```

**Things to watch out for:**
- ⚠️ Verify event.properties includes directory - may need to fetch session info via API
- ⚠️ Handle case where directory is undefined (fall back to existing tool-arg detection)
- ⚠️ Test with all spawn modes (API headless, Claude mode, Docker)

**Areas needing further investigation:**
- What exactly is in session.created event properties? (may need to log and inspect)
- Could OpenCode be patched to reliably expose session.metadata.role?
- Are there edge cases where orchestrators have `.orch/workspace/` in their path?

**Success criteria:**
- ✅ No coaching alerts fire on worker sessions
- ✅ Worker sessions show zero entries in coaching-metrics.jsonl for orchestrator patterns
- ✅ Orchestrator sessions continue to receive coaching as expected
- ✅ Detection works for API spawns, Claude mode spawns, and Docker spawns

---

## References

**Files Examined:**
- `plugins/coaching.ts:1310-1877` - Coaching plugin with worker detection logic
- `plugins/orchestrator-session.ts:1-283` - Similar detection pattern, session.created handling
- `pkg/opencode/client.go:534-580` - CreateSession with x-opencode-env-ORCH_WORKER header
- `pkg/spawn/config.go` - Workspace creation
- `.kb/guides/opencode-plugins.md` - Plugin hooks reference

**Commands Run:**
```bash
# Check ORCH_WORKER usage across codebase
grep -rn "ORCH_WORKER\|x-opencode-env" *.go

# View recent commits on coaching
git log --oneline -5
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-10-inv-add-worker-filtering-coaching-ts.md` - Original worker filtering approach
- **Investigation:** `.kb/investigations/2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md` - Cache bug fix
- **Investigation:** `.kb/investigations/2026-01-17-inv-update-coaching-plugin-session-metadata.md` - session.metadata.role attempt
- **Investigation:** `.kb/investigations/2026-01-24-inv-fix-coaching-plugin-firing-workers.md` - Recent revert to file-path detection
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Comprehensive architecture analysis
- **Guide:** `.kb/guides/opencode-plugins.md` - Plugin hook reference

---

## Investigation History

**2026-01-23 09:00:** Investigation started
- Initial question: How should workers be identified early enough to skip coaching alerts?
- Context: Current file-path detection fires too late, causing coaching alerts on workers

**2026-01-23 09:15:** Reviewed prior investigations
- Found 4 prior investigations on this topic
- Identified the race condition: detection and alerting both happen in tool.execute.after
- session.metadata.role was tried but reverted due to reliability issues

**2026-01-23 09:30:** Identified 4 architectural options
- Option A: Directory-based detection in session.created event (recommended)
- Option B: Deferred alert queue
- Option C: Fix session.metadata.role in OpenCode
- Option D: Environment variable (not viable)

**2026-01-23 09:45:** Investigation completed
- Status: Complete
- Key outcome: Recommend directory-based detection in session.created event to detect workers BEFORE any tool calls occur
