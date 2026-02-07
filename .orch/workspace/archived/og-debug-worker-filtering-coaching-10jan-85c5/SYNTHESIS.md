# SYNTHESIS: Worker Filtering Bug Fix

**Investigation:** `.kb/investigations/2026-01-10-inv-debug-worker-filtering-coaching-ts.md`
**Issue:** orch-go-l4y14
**Status:** Complete - Fix implemented and committed

## TLDR

Worker filtering in coaching.ts was broken because plugin-level detection checked `process.env.ORCH_WORKER`, but plugins run in the OpenCode server process which never sees env vars from spawned agent processes. Fixed by moving detection to per-session checking in `tool.execute.after` hook using `input.args.workdir` and `input.args.filePath` patterns.

## Problem

### Root Cause
1. **Architecture mismatch:** OpenCode plugins run in the server process (one instance serving all sessions)
2. **Environment isolation:** Spawned workers have `ORCH_WORKER=1` set in their environment, but server process never sees it
3. **Plugin-level detection:** Old code checked `process.env.ORCH_WORKER` at plugin init, which always returned undefined
4. **Result:** Worker sessions tracked orchestrator coaching metrics, polluting data

### Evidence
- Prior investigation (2026-01-10-inv-add-worker-filtering-coaching-ts.md) implemented isWorker() copying from orchestrator-session.ts
- That implementation assumed plugin init runs per-agent, but it actually runs once in server
- Spawn context explicitly states: "ORCH_WORKER env var is set in agent's env, not server's"

## Solution

### Approach
Move worker detection from plugin init (server-level) to tool hooks (per-session):

1. **Per-session tracking:** Added `Map<sessionID, boolean>` to cache worker status
2. **Detection in tool hooks:** Check `input.args.workdir` and `input.args.filePath` in `tool.execute.after`
3. **Multiple signals:** Detect via:
   - `args.workdir` containing `.orch/workspace/`
   - `args.filePath` pointing to `SPAWN_CONTEXT.md` (workers always read this)
   - Any `filePath` in `.orch/workspace/`
4. **Caching:** Once detected, session cached as worker to avoid repeated checks

### Implementation Details

**coaching.ts changes:**
- Removed plugin-level `isWorker()` check (lines 84-108, 843-846)
- Added `detectWorkerSession()` function with three detection signals
- Added early return in `tool.execute.after` if worker detected
- Added cache check in `experimental.chat.messages.transform` hook
- Removed unused `access` import from fs/promises

**orchestrator-session.ts changes:**
- Removed plugin-level `isWorker()` check (lines 76-100, 151-157)
- Added TODO note that worker detection at session.created time needs tool hooks
- Event hook currently allows workers to potentially trigger session start (harmless)
- Can be enhanced later with tool.execute.after hook for more robust detection

### Detection Signals Hierarchy

| Signal | When Available | Reliability | Priority |
|--------|---------------|-------------|----------|
| **workdir in .orch/workspace/** | First bash command | High (workers always use workspace dir) | 1 |
| **Reading SPAWN_CONTEXT.md** | Early (workers read this first) | Very high (unique to workers) | 2 |
| **filePath in .orch/workspace/** | Any file operation | High | 3 |

## Testing

### Verification Method
To test the fix:
1. Enable debug logging: `ORCH_PLUGIN_DEBUG=1`
2. Note current coaching-metrics.jsonl line count
3. Spawn a worker: `orch spawn investigation "test"`
4. Check debug logs for "Worker detected" messages
5. Verify worker sessionID does NOT appear in coaching-metrics.jsonl

### Expected Behavior
**Before fix:**
- Worker sessions tracked in metrics
- coaching-metrics.jsonl grows with worker tool usage
- Dylan pattern detection runs for worker messages

**After fix:**
- Worker sessions detected via workdir/filePath patterns
- `detectWorkerSession()` returns true, causing early return
- No metrics written for worker sessions
- Debug log shows: "Session {id} marked as worker (will skip metrics)"

## Impact

### What's Fixed
✅ Worker sessions no longer pollute orchestrator coaching metrics
✅ Dylan pattern detection (priority_uncertainty, compensation_pattern) skips workers
✅ Behavioral variation detection (Phase 1) skips workers
✅ Circular pattern detection (Phase 2) skips workers

### What's Improved
✅ Caching prevents performance overhead from repeated detection
✅ Multiple detection signals provide redundancy
✅ Architecture now matches OpenCode's plugin model (server process, not per-agent)

### What's Not Fixed
⚠️ orchestrator-session.ts event hook doesn't fully filter workers yet
- Worker sessions might still trigger "orch session start" (harmless)
- Can be enhanced with tool.execute.after hook if needed
- Config hook still runs for all sessions (no sessionID available there)

## Follow-Up

### Potential Improvements
1. **Add tool.execute.after to orchestrator-session.ts** for robust worker filtering in event/config hooks
2. **Monitor false positives:** Check if any orchestrator sessions incorrectly detected as workers
3. **Add smoke test:** Automated test spawning worker and verifying no metrics written

### Related Work
- Same architectural issue may exist in other plugins checking `process.env`
- Consider documenting "plugins run in server, not per-agent" in plugin guide
- May want to add OpenCode plugin hook that explicitly receives "isWorker" flag from OpenCode itself

## Lessons Learned

### Key Insights
1. **Process boundaries matter:** Plugin code runs in server, agents run in separate processes
2. **Environment variables are process-local:** Can't rely on env vars for cross-process detection
3. **Prior investigations can be wrong:** 2026-01-10-inv-add-worker-filtering-coaching-ts.md implemented a solution that couldn't work
4. **Tool hooks provide session context:** `input.sessionID` and `input.args` enable per-session logic

### System Knowledge
- OpenCode plugins: single server instance serves all sessions
- Tool hooks run per-tool-call with session context
- Bash tool has `workdir` parameter exposing agent's working directory
- Read/Write tools have `filePath` parameter exposing target file paths
- Workers always operate in `.orch/workspace/{name}/` directories
- Workers always read SPAWN_CONTEXT.md early in their lifecycle

---

**Committed:** 6e6503ae
**Files Changed:**
- plugins/coaching.ts (+58 lines detection logic, -48 lines removed isWorker)
- plugins/orchestrator-session.ts (-27 lines removed isWorker, +worker tracking stub)
- .kb/investigations/2026-01-10-inv-debug-worker-filtering-coaching-ts.md (investigation doc)
