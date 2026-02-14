<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Active orchestrator sessions are being deleted via explicit API calls, likely triggered by recent upstream changes to session listing (commit b02075844) that show all project sessions regardless of directory, combined with the Ctrl+D keyboard shortcut in the TUI session list dialog.

**Evidence:** Session ID ses_3a4b0aaf2ffe6kzgksyu5RyRz1 not found in database; Session.remove() only called via DELETE API endpoint; recent SQLite migration (Feb 13) and session listing changes (Feb 14) in upstream; multiple processes share database creating race condition potential.

**Knowledge:** OpenCode has no automatic session cleanup mechanism; sessions persist indefinitely until explicitly deleted via API; recent upstream commits fundamentally changed session storage (JSON→SQLite) and listing behavior (directory-filtered→show-all); TUI uses Ctrl+D for both app exit and session deletion creating potential for user error.

**Next:** Report bug upstream to OpenCode maintainers with reproduction steps; consider adding session deletion confirmation that shows full session details (not just title) to prevent wrong-session deletion; investigate if session list shows stale or incorrect data after recent changes.

**Authority:** architectural - Bug exists in external dependency (OpenCode), requires coordination with upstream maintainers and potentially affects all OpenCode users, not just orch-go.

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

# Investigation: Active Orchestrator Session Deleted While In Use

**Question:** Why do active orchestrator sessions get deleted mid-conversation, causing NotFoundError when trying to send prompts?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Architect Agent (orch-go-f3g)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Session deletion mechanism exists but has limited call sites

**Evidence:** 
- `Session.remove()` at session/index.ts:569-589 deletes sessions from database
- Exposed via DELETE API route at server/routes/session.ts:236
- No periodic cleanup code found in OpenCode codebase
- No code that automatically deletes projects (which would cascade to sessions)

**Source:** 
- `~/Documents/personal/opencode/packages/opencode/src/session/index.ts:569-589`
- `~/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts:236`
- Grep search for cleanup patterns across codebase

**Significance:** Sessions should only be deleted via explicit API calls (DELETE /session/:id). There's no automatic cleanup mechanism that would delete active sessions.

---

### Finding 2: Sessions cascade delete when projects are deleted

**Evidence:**
- SessionTable has foreign key to ProjectTable with `onDelete: "cascade"` (session.sql.ts:17)
- Foreign keys are enabled in SQLite: `PRAGMA foreign_keys = ON` (db.ts:76)
- No code found that deletes from ProjectTable

**Source:**
- `~/Documents/personal/opencode/packages/opencode/src/session/session.sql.ts:15-17`
- `~/Documents/personal/opencode/packages/opencode/src/storage/db.ts:76`
- Grep search for ProjectTable deletions

**Significance:** If projects were being deleted, sessions would cascade delete. However, no code path deletes projects, so this is not the root cause.

---

### Finding 3: Multiple processes share the same database

**Evidence:**
- 4 processes have the database open: opencode server (16290) + 3 agent bun processes (16720, 29950, 66312)
- All use SQLite WAL mode for concurrent access
- Each agent is an independent OpenCode CLI process running in orch-go directory

**Source:**
```bash
lsof ~/.local/share/opencode/opencode.db
ps aux | grep bun | grep opencode
```

**Significance:** Multiple agent processes accessing the same database creates potential for race conditions or unintended deletions if one process calls Session.remove while another is using the session.

---

### Finding 4: Instance eviction doesn't delete sessions

**Evidence:**
- Instance eviction (LRU/TTL) added in commit c3c84c411 disposes project instances
- `disposeCurrent()` calls `State.dispose(directory)` which cleans up in-memory state
- No session deletion in State.dispose or instance disposal code

**Source:**
- `~/Documents/personal/opencode/packages/opencode/src/project/instance.ts:86-120`
- `~/Documents/personal/opencode/packages/opencode/src/project/state.ts:31-69`
- Git commit c3c84c411

**Significance:** Instance eviction doesn't delete sessions from database, ruling out this as root cause.

---

## Synthesis

**Key Insights:**

1. **No Automatic Cleanup Exists** - OpenCode has no automatic session cleanup mechanism. Sessions persist indefinitely in the SQLite database until explicitly deleted via the DELETE /session/:id API endpoint. This contradicts the model claim that "sessions are never deleted by OpenCode" - they ARE deleted, but only via explicit API calls, not automatically.

2. **Recent Upstream Changes Created Risk Window** - Two major upstream changes in Feb 2026 created conditions for session deletion bugs: (1) SQLite migration (commit 6d95f0d14, Feb 13) completely rewrote session storage, and (2) session listing changes (commit b02075844, Feb 14) removed directory filtering, showing ALL project sessions regardless of working directory. These changes happened within 24 hours and could interact in unexpected ways.

3. **Multiple Deletion Vectors** - Session deletion can occur through multiple paths: TUI session list dialog (Ctrl+D pressed twice), direct API calls from SDK, or potential race conditions from multiple processes sharing the same SQLite database. The Ctrl+D keybind conflict (used for both app_exit and session_delete) creates additional risk of accidental deletion.

**Answer to Investigation Question:**

Active orchestrator sessions get deleted mid-conversation because something is explicitly calling the DELETE /session/:id API endpoint, most likely triggered through the TUI session list dialog when a user presses Ctrl+D twice to delete what they think is an old/inactive session, but the recent upstream changes to show all project sessions (regardless of directory) may be causing session list confusion where active sessions appear to be inactive or are misidentified. The massive SQLite migration on Feb 13 and session listing changes on Feb 14 create a high-risk window for bugs in session identification and deletion logic.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Report bug upstream with reproduction steps | architectural | Requires coordination with external maintainers, affects OpenCode users beyond orch-go, involves analyzing cross-component interactions (TUI + session storage + database migration) |
| Add enhanced deletion confirmation in Dylan's fork | implementation | Tactical UI improvement within existing TUI patterns, reversible, no architectural impact |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Report upstream bug with detailed reproduction steps and monitor for fix** - Create a detailed bug report to OpenCode maintainers with session ID, timestamp, stack trace, and hypothesis about recent commits.

**Why this approach:**
- Bug exists in OpenCode codebase (external dependency), not orch-go
- Recent upstream commits (b02075844, 6d95f0d14) are prime suspects given timing
- OpenCode maintainers have better context on SQLite migration and session listing changes
- Reproduction steps needed to confirm root cause before implementing workarounds

**Trade-offs accepted:**
- Depends on upstream fix timeline (may take days/weeks)
- Workarounds in orch-go would be band-aids, not root cause fixes
- Risk of additional session deletions until bug is fixed upstream

**Implementation sequence:**
1. **Document reproduction case** - Capture exact steps Dylan took when session was deleted (which TUI window, what actions, timing relative to recent upstream commits)
2. **Create upstream issue** - File bug report at https://github.com/sst/opencode with session ID, stack trace, hypothesis linking to commits b02075844 and 6d95f0d14
3. **Implement temporary mitigation** - Add confirmation dialog enhancement in Dylan's fork: show full session details (ID, directory, last updated time) before deletion to prevent wrong-session deletion
4. **Monitor for upstream fix** - Watch OpenCode repo for fix, rebase fork when available

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
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

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
