<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Fixed detectWorkerSession() to only cache positive results (isWorker=true), preventing workers from being permanently misclassified.

**Evidence:** Code change verified: cache now only stores true results; detection signals restored for any .orch/workspace/ filePath; broken bash workdir check removed.

**Knowledge:** Never cache negative results in per-session detection - first tool call may not be representative of session type.

**Next:** Close - implementation complete. Server restart needed to pick up changes.

**Promote to Decision:** recommend-yes - Establishes pattern: "never cache negative results in per-session detection"

---

# Investigation: Fix detectWorkerSession Caching Bug in Coaching Plugin

**Question:** How should detectWorkerSession() be fixed to stop caching negative results?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-debug-fix-detectworkersession-caching-17jan-cb26
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete

<!-- Lineage -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Root Cause Was Premature Caching of false

**Evidence:** Original code at line 1255-1256 cached every result, including false:
```typescript
// Cache the result
workerSessions.set(sessionId, isWorker)  // Always cached, even when false
```

This caused a race condition: if ANY tool call happened before a worker-identifying tool call (like reading SPAWN_CONTEXT.md), the session was permanently marked as non-worker.

**Source:** `plugins/coaching.ts:1255-1256` (before fix)

**Significance:** This was the root cause. Even if detection signals would eventually fire, the premature cache prevented re-evaluation.

---

### Finding 2: Bash workdir Check Was Invalid

**Evidence:** The original code checked for `args?.workdir`:
```typescript
if (tool === "bash" && args?.workdir) {
  if (args.workdir.includes(".orch/workspace/")) {
    isWorker = true
  }
}
```

However, the bash tool in OpenCode/Claude has no `workdir` argument. The bash tool args are only: `command`, `timeout`, `dangerouslyDisableSandbox`, `run_in_background`.

**Source:** `plugins/coaching.ts:1238-1244` (removed in fix), OpenCode tool schema

**Significance:** This detection signal never fired, providing false confidence that detection was working.

---

### Finding 3: filePath Detection Was Removed in Prior "Fix"

**Evidence:** Commit b82715c1 removed the most reliable detection signal:
```diff
-    // Detection signal 3: any tool with filePath in .orch/workspace/
-    if (args?.filePath && typeof args.filePath === "string") {
-      if (args.filePath.includes(".orch/workspace/")) {
-        isWorker = true
-      }
-    }
```

**Source:** Architect investigation 2026-01-17-inv-design-review-coaching-plugin-failures.md

**Significance:** The filePath-based detection was the most reliable signal, since workers frequently read/write files in their workspace.

---

## Implementation

**Changes Made to `plugins/coaching.ts`:**

1. **Fixed caching logic** - Only cache when `isWorker = true`:
```typescript
if (isWorker) {
  workerSessions.set(sessionId, true)
  log(`Session ${sessionId} marked as worker`)
}
// Don't cache false - keep checking
return isWorker
```

2. **Removed broken bash workdir check** - The bash tool has no workdir arg

3. **Restored filePath detection** - Re-added signal for any .orch/workspace/ path:
```typescript
if (args?.filePath && typeof args.filePath === "string") {
  if (args.filePath.includes(".orch/workspace/")) {
    isWorker = true
  }
}
```

4. **Added file_path variant** - Some tools use snake_case:
```typescript
if (args?.file_path && typeof args.file_path === "string") {
  if (args.file_path.includes(".orch/workspace/")) {
    isWorker = true
  }
}
```

---

## References

**Files Modified:**
- `plugins/coaching.ts:1224-1272` - Fixed detectWorkerSession() function

**Related Investigations:**
- `.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md` - Architect investigation that identified root cause

**Beads:**
- `orch-go-hflo3` - This fix task
- `orch-go-ls80s` - Architect investigation that identified the bug

---

## Investigation History

**2026-01-17 10:37:** Task started
- Read coaching.ts and architect investigation
- Identified the exact lines to change

**2026-01-17 10:40:** Fix implemented
- Only cache positive results
- Restored filePath detection
- Removed broken bash workdir check

**2026-01-17 10:42:** Investigation completed
- Status: Complete
- Key outcome: detectWorkerSession() now correctly identifies worker sessions by only caching true results
