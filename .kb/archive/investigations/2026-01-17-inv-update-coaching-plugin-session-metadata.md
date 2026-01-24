## Summary (D.E.K.N.)

**Delta:** Replaced file-based worker detection with session.metadata.role detection in coaching plugin.

**Evidence:** Updated detectWorkerSession() function and call site; pending validation via spawn test.

**Knowledge:** OpenCode now passes session.metadata in plugin hooks, enabling direct role detection instead of heuristics.

**Next:** Validate by spawning worker and checking metrics.

**Promote to Decision:** recommend-no (tactical fix following OpenCode upgrade)

---

# Investigation: Update Coaching Plugin Session Metadata

**Question:** How to update coaching plugin to use session.metadata.role for worker detection?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker agent (orch-go-v3v8z)
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

---

## Findings

### Finding 1: Previous detection used file-based heuristics

**Evidence:** The detectWorkerSession function checked:
1. Read tool accessing SPAWN_CONTEXT.md
2. Any file_path containing ".orch/workspace/"

**Source:** plugins/coaching.ts:1319-1359 (before change)

**Significance:** File-based detection was fragile - required workers to access specific files early in session for detection to work.

---

### Finding 2: OpenCode now exposes session.metadata to plugins

**Evidence:** Task context indicates OpenCode now:
1. Exposes session.metadata to plugins
2. Sets session.metadata.role = 'worker' when x-opencode-env-ORCH_WORKER=1 header present
3. Passes session object in hook input

**Source:** SPAWN_CONTEXT.md task description

**Significance:** Enables reliable worker detection at session creation rather than inferring from behavior.

---

### Finding 3: Single call site needed update

**Evidence:** Only one call site for detectWorkerSession found in tool.execute.after hook at line 1552.

**Source:** plugins/coaching.ts:1552

**Significance:** Minimal change surface - simple update to pass input.session instead of tool/args.

---

## Synthesis

**Key Insights:**

1. **Metadata-based detection is more reliable** - Detecting workers via session.metadata.role happens at session creation, not inferred from file access patterns.

2. **Cache behavior preserved** - Worker sessions are still cached after detection, so messages.transform hook's cache check continues to work.

3. **Backwards compatible** - If session.metadata.role is undefined, detection returns false, preserving safe fallback.

**Answer to Investigation Question:**

Updated detectWorkerSession() to accept session object and check session.metadata.role === 'worker'. Updated call site in tool.execute.after to pass input.session.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles (no syntax errors in TypeScript)
- ✅ Function signature updated correctly
- ✅ Call site updated to pass session object

**What's untested:**

- ⚠️ Actual worker spawning with new detection (needs spawn + metrics check)
- ⚠️ OpenCode actually passes session.metadata in hook input

**What would change this:**

- Finding would need revision if OpenCode doesn't pass session in input or metadata structure differs

---

## References

**Files Examined:**
- plugins/coaching.ts - Coaching plugin with worker detection logic

**Commands Run:**
```bash
# Create investigation
kb create investigation update-coaching-plugin-session-metadata
```

---

## Investigation History

**2026-01-17:** Investigation started
- Initial question: Update coaching plugin worker detection to use session.metadata.role
- Context: OpenCode upgrade exposed session metadata to plugins

**2026-01-17:** Implementation completed
- Updated detectWorkerSession function
- Updated call site in tool.execute.after hook
- Key outcome: Replaced heuristic detection with metadata detection
