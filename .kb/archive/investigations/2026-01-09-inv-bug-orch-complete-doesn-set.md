<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** orch complete closes beads issues but doesn't delete OpenCode sessions, causing completed agents to appear as "running" in orch status.

**Evidence:** Investigation .kb/investigations/2026-01-09-inv-pw-oicj-ghost-agent-postmortem.md shows session had status null. Code review shows complete_cmd.go lines 686-689 only invalidates cache, doesn't delete session. orch status (line 188) fetches all sessions via client.ListSessions and filters by age, not by completion status.

**Knowledge:** OpenCode API doesn't have a "status" field to update. The only way to prevent completed sessions from appearing in orch status is to delete them via client.DeleteSession(). Session ID is available via state.GetLiveness() or from workspace .session_id file.

**Next:** Add client.DeleteSession(sessionID) call in complete_cmd.go after beads issue closes (line 563) but before cache invalidation (line 686).

**Promote to Decision:** recommend-no (tactical bug fix, not architectural)

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

# Investigation: Bug Orch Complete Doesn Set

**Question:** Why doesn't orch complete set OpenCode session status to 'done', causing ghost agents in status?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** Agent og-debug-bug-orch-complete-09jan-04ba
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: complete_cmd.go doesn't delete OpenCode sessions

**Evidence:** Reviewed complete_cmd.go lines 1-1112. Function runComplete() closes beads issues (line 531) and invalidates cache (line 688) but never calls client.DeleteSession(). The only OpenCode client interaction is cache invalidation via HTTP POST to /api/cache/invalidate.

**Source:** cmd/orch/complete_cmd.go:686-689, cmd/orch/complete_cmd.go:529-563

**Significance:** This is the root cause - completed agents leave stale sessions in OpenCode API, causing orch status to show them as "running".

---

### Finding 2: OpenCode API doesn't have a "status" field to update

**Evidence:** Checked OpenCode Session struct (pkg/opencode/types.go:52-62). Fields are: ID, Version, ProjectID, Directory, Title, ParentID, Time, Summary. No "status" field. Verified via curl http://localhost:4096/session - actual sessions have no status field either.

**Source:** pkg/opencode/types.go:52-76, curl http://localhost:4096/session

**Significance:** The bug description mentions "set session status to 'done'" but this endpoint doesn't exist. The solution is to DELETE the session instead.

---

### Finding 3: Session ID is accessible via state.GetLiveness()

**Evidence:** complete_cmd.go line 374 calls state.GetLiveness() which returns LivenessResult with SessionID field (pkg/state/reconcile.go:36). This sessionID is already used for liveness warnings at line 386-387.

**Source:** cmd/orch/complete_cmd.go:374, pkg/state/reconcile.go:19-47, pkg/state/reconcile.go:77-111

**Significance:** We have the sessionID available to pass to DeleteSession(). No additional API calls needed.

---

## Synthesis

**Key Insights:**

1. **Deletion vs Status Update** - The OpenCode API doesn't expose a "status" field for sessions. The only way to prevent completed sessions from appearing in orch status is to delete them entirely via the /session/:id DELETE endpoint.

2. **Session ID Availability** - The session ID is already accessible in complete_cmd.go via the workspace .session_id file. No additional API calls or lookups needed.

3. **Cache Invalidation Alone is Insufficient** - The existing invalidateServeCache() call only clears TTL cache but doesn't remove sessions from OpenCode's session list. The dashboard refetches from OpenCode API, so stale sessions persist until deleted.

**Answer to Investigation Question:**

orch complete doesn't set OpenCode session status to 'done' because such a status field doesn't exist in the OpenCode API. The fix is to delete the OpenCode session after successful completion by calling client.DeleteSession(sessionID). Session ID is read from workspace/.session_id file. Implementation added at complete_cmd.go:565-580, between beads closure and transcript export.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles without errors (verified: go build ./cmd/orch succeeded)
- ✅ OpenCode API has DeleteSession method (verified: pkg/opencode/client.go:752)
- ✅ Session ID is available in workspace .session_id file (verified: complete_cmd.go:374 already uses state.GetLiveness which reads this)

**What's untested:**

- ⚠️ Fix resolves ghost agents in production (not yet deployed)
- ⚠️ Error handling covers all edge cases (session already deleted, network errors)
- ⚠️ Performance impact of DeleteSession call during completion (assumed negligible)

**What would change this:**

- Finding would be wrong if OpenCode sessions had a separate "status" field that could be set to "done"
- Fix would fail if DeleteSession requires special permissions or fails on completed sessions
- Implementation would need adjustment if .session_id file is not reliably present in all workspaces

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Delete OpenCode Session After Completion** - Call client.DeleteSession(sessionID) after beads issue closes but before cache invalidation.

**Why this approach:**
- Prevents completed sessions from appearing in OpenCode API /session endpoint
- orch status filters by session age, not completion status, so old sessions still appear
- Deletion is the only way to remove sessions (no "status" field to update)
- Session ID already available from workspace .session_id file

**Trade-offs accepted:**
- Sessions are permanently deleted (no history in OpenCode)
- Transcripts are exported before deletion (for orchestrator sessions)
- Non-fatal error if deletion fails (warns but doesn't block completion)

**Implementation sequence:**
1. Add opencode import to complete_cmd.go imports
2. After beads closure (line 563), read sessionID from workspace/.session_id
3. Call client.DeleteSession(sessionID) with error handling
4. Continue with existing flow (transcript export, tmux cleanup, cache invalidation)

### Alternative Approaches Considered

**Option B: Update session metadata instead of deleting**
- **Pros:** Preserves session history in OpenCode
- **Cons:** OpenCode API doesn't expose metadata update endpoint. Session struct has no "status" field to set.
- **When to use instead:** Never - this endpoint doesn't exist

**Option C: Modify orch status to filter out completed agents**
- **Pros:** Preserves all session data
- **Cons:** Requires cross-referencing beads status with OpenCode sessions (expensive). Doesn't solve root cause (stale sessions accumulate).
- **When to use instead:** Could complement deletion, but deletion alone is sufficient

**Rationale for recommendation:** Deletion is the only viable approach given OpenCode API constraints. Alternative approaches require API changes or add complexity without benefits.

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
- cmd/orch/complete_cmd.go - Main completion flow, identified missing DeleteSession call
- pkg/opencode/client.go - Confirmed DeleteSession method exists at line 752
- pkg/opencode/types.go - Verified Session struct has no status field
- pkg/state/reconcile.go - Confirmed GetLiveness provides SessionID
- cmd/orch/status_cmd.go - Verified orch status uses OpenCode API /session endpoint

**Commands Run:**
```bash
# Check OpenCode session structure
curl -s http://localhost:4096/session | jq '.[0]'

# Verify DeleteSession exists in codebase
grep -n "func.*DeleteSession" pkg/opencode/client.go

# Build to verify changes compile
go build ./cmd/orch
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-09-inv-pw-oicj-ghost-agent-postmortem.md - Original ghost agent discovery
- **Workspace:** .orch/workspace/og-debug-bug-orch-complete-09jan-04ba/ - This debugging session

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
