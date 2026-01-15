<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Worker sessions receive orchestrator handoff because the plugin checks `process.env.ORCH_WORKER` (server-level) instead of `SPAWN_CONTEXT.md` presence (session-level).

**Evidence:** Plugin at `~/.config/opencode/plugin/session-resume.js:55-58` checks server process environment, but orch-go sets `x-opencode-env-ORCH_WORKER` as HTTP header which isn't propagated to plugin environment; tests confirm SPAWN_CONTEXT.md is only created for workers.

**Knowledge:** File-based session type detection (SPAWN_CONTEXT.md presence) is more reliable than environment variables when plugins need session-specific context.

**Next:** Fix implemented in session-resume.js - restart OpenCode server and verify worker spawns don't receive handoff.

**Promote to Decision:** recommend-no (bug fix with clear solution, not architectural pattern)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Session Handoff Content Injected Into

**Question:** Why are spawned worker sessions receiving orchestrator session handoff content when they should only receive SPAWN_CONTEXT.md?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** og-arch-session-handoff-content-15jan-641e
**Phase:** Complete
**Next Step:** Verify fix works via reproduction test
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Plugin checks process.env, not session metadata

**Evidence:** The `session-resume.js` plugin at `~/.config/opencode/plugin/session-resume.js:22` checks `process.env.ORCH_WORKER === '1'` to skip handoff injection for workers. Lines 55-58 implement the skip logic.

**Source:** `~/.config/opencode/plugin/session-resume.js:22,55-58`

**Significance:** The plugin is checking the Node.js server process environment, not session-specific metadata. This means it only sees environment variables set when the OpenCode server starts, not per-session metadata.

---

### Finding 2: orch-go sends ORCH_WORKER via HTTP header

**Evidence:** When creating worker sessions, orch-go sets `x-opencode-env-ORCH_WORKER: 1` as an HTTP header at `pkg/opencode/client.go:555`.

**Source:** `pkg/opencode/client.go:553-555`

**Significance:** The header is sent with the session creation request, but there's no evidence that OpenCode propagates this header to the plugin's JavaScript environment or session properties. The mismatch between header (session-specific) and `process.env` (server-global) causes the bug.

---

### Finding 3: Worker sessions have SPAWN_CONTEXT.md

**Evidence:** Worker spawns always create `SPAWN_CONTEXT.md` in their workspace directory as the primary context injection mechanism. The bug description states "Workers should ONLY receive SPAWN_CONTEXT.md content."

**Source:** SPAWN_CONTEXT.md (this file), spawn_cmd.go context generation logic

**Significance:** SPAWN_CONTEXT.md presence could be used as a reliable indicator to skip handoff injection, independent of environment variables or HTTP headers.

---

## Synthesis

**Key Insights:**

1. **Environment variable mismatch** - The plugin checks `process.env.ORCH_WORKER` which only sees the server process environment, not session-specific metadata sent via HTTP headers. This creates a fundamental mismatch between where the flag is set (per-session header) and where it's checked (server process).

2. **SPAWN_CONTEXT.md is a perfect indicator** - Worker spawns ALWAYS create SPAWN_CONTEXT.md, while orchestrator and meta-orchestrator spawns create ORCHESTRATOR_CONTEXT.md or META_ORCHESTRATOR_CONTEXT.md instead. This file presence is a reliable, session-specific indicator that doesn't depend on environment variables.

3. **File-based detection is more robust** - Checking for file presence in the session directory works regardless of how the session was created (HTTP API, CLI, tmux) and doesn't require coordination between multiple layers (HTTP headers, OpenCode server, plugin environment).

**Answer to Investigation Question:**

Worker sessions receive orchestrator handoff content because the `session-resume.js` plugin checks `process.env.ORCH_WORKER` (server-level) instead of checking for `SPAWN_CONTEXT.md` presence (session-level). The fix is to check for `SPAWN_CONTEXT.md` in the session directory - if it exists, skip handoff injection since the worker should only use SPAWN_CONTEXT.md. This approach is reliable because SPAWN_CONTEXT.md is only created for workers, never for orchestrators.

---

## Structured Uncertainty

**What's tested:**

- ✅ SPAWN_CONTEXT.md only created for workers (verified: pkg/spawn/orchestrator_context_test.go:177-181 shows orchestrators don't create it)
- ✅ Plugin checks process.env.ORCH_WORKER (verified: read ~/.config/opencode/plugin/session-resume.js:22)
- ✅ orch-go sets x-opencode-env-ORCH_WORKER header (verified: pkg/opencode/client.go:555)

**What's untested:**

- ⚠️ Fix prevents handoff injection for workers (requires OpenCode server restart and new spawn)
- ⚠️ Fix still allows handoff injection for orchestrators (requires testing orchestrator session)
- ⚠️ sessionDirectory is correctly resolved in plugin (assumed from event.properties)

**What would change this:**

- Finding would be wrong if SPAWN_CONTEXT.md exists in orchestrator workspaces (contradicts tests)
- Fix would fail if sessionDirectory doesn't point to workspace root where SPAWN_CONTEXT.md lives
- Fix would fail if SPAWN_CONTEXT.md is created AFTER session.created event fires

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Check for SPAWN_CONTEXT.md presence** - Replace `process.env.ORCH_WORKER` check with file system check for SPAWN_CONTEXT.md in the session directory.

**Why this approach:**
- SPAWN_CONTEXT.md is only created for workers, never for orchestrators (verified by tests)
- File presence is session-specific, not server-global like environment variables
- No coordination needed between HTTP headers, server, and plugin layers
- Works regardless of how the session was created (API, CLI, tmux)

**Trade-offs accepted:**
- Small file system check overhead (async, non-blocking, cached by OS)
- Assumes workspace directory structure remains stable (already foundational to orch-go)

**Implementation sequence:**
1. Import Node.js fs and path modules in plugin
2. Check for `SPAWN_CONTEXT.md` in `sessionDirectory` before injecting handoff
3. Skip injection if file exists, proceed if it doesn't
4. Remove obsolete `IS_WORKER` constant and add comment about why it doesn't work

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- Update `~/.config/opencode/plugin/session-resume.js` to check for SPAWN_CONTEXT.md
- Restart OpenCode server to load updated plugin
- Test with worker spawn to verify handoff is skipped

**Things to watch out for:**
- ⚠️ Session directory must be correctly resolved from event.properties
- ⚠️ Async file access requires proper error handling (SPAWN_CONTEXT.md absence is normal for orchestrators)
- ⚠️ Race condition if handoff check happens before SPAWN_CONTEXT.md is written (unlikely since file is written before session creation)

**Areas needing further investigation:**
- None - fix is straightforward and well-scoped

**Success criteria:**
- ✅ Worker sessions do NOT receive session handoff content
- ✅ Orchestrator sessions still receive session handoff when resuming
- ✅ Plugin logs show correct skip/proceed decisions when ORCH_PLUGIN_DEBUG=1

---

## Verification Test Plan

**Pre-conditions:**
1. OpenCode server must be restarted to load updated plugin
2. An orchestrator session with SESSION_HANDOFF.md must exist for resume testing

**Test 1: Worker spawn does NOT receive handoff**
```bash
# Restart OpenCode server
pkill -f "opencode serve"
opencode serve --port 4096 &

# Wait for server to start
sleep 2

# Spawn a worker with debug logging enabled
ORCH_PLUGIN_DEBUG=1 orch spawn feature-impl "test worker spawn" --no-track

# EXPECT: No handoff content in initial prompt
# EXPECT: Plugin log shows "Skipping injection for worker session (SPAWN_CONTEXT.md found)"
```

**Test 2: Orchestrator session DOES receive handoff**
```bash
# Start a new interactive orchestrator session in a project with prior SESSION_HANDOFF.md
cd /path/to/project/with/session/history
oc

# EXPECT: Session displays "📋 Session Resumed" with handoff content
# EXPECT: Plugin log shows "No SPAWN_CONTEXT.md found, proceeding with handoff injection"
```

**Success criteria:**
- ✅ Worker spawns show no handoff content
- ✅ Orchestrator sessions show handoff content
- ✅ Plugin logs confirm correct skip/proceed logic

## References

**Files Examined:**
- `~/.config/opencode/plugin/session-resume.js` - Plugin that injects handoff
- `pkg/opencode/client.go:553-555` - Where ORCH_WORKER header is set
- `pkg/spawn/orchestrator_context_test.go:177-181` - Test proving SPAWN_CONTEXT.md exclusivity

**Commands Run:**
```bash
# Search for plugin files
find ~/.config/opencode -type f -name "*.ts" -o -name "*.js" | grep -v node_modules

# Search for ORCH_WORKER usage
rg "ORCH_WORKER" --type go

# Search for SPAWN_CONTEXT.md creation
rg "SPAWN_CONTEXT" cmd/orch/spawn_cmd.go
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-15-inv-session-handoff-content-injected-into.md` - This file
- **Plugin:** `~/.config/opencode/plugin/session-resume.js` - Fixed plugin file

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
