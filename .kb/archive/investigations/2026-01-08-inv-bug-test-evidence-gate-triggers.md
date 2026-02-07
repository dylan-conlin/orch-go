<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The test_evidence gate was triggering on markdown-only changes because `HasCodeChangesSinceSpawn` checked ALL commits since spawn time, not just commits from THIS agent.

**Evidence:** Tested with og-feat-create-kb-guides-08jan-b223 workspace - old method returned `hasCodeChanges=true`, new method returned `hasCodeChanges=false`.

**Knowledge:** When multiple agents run concurrently, `git log --since=<spawn_time>` picks up commits from ALL agents, not just the one we're verifying. The fix filters commits to only those that modified the workspace directory.

**Next:** Fix is implemented and tested. Commit and mark complete.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural)

---

# Investigation: Bug Test Evidence Gate Triggers

**Question:** Why does the test_evidence gate trigger on markdown-only changes?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Claude agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Root Cause - Concurrent Agents' Commits Included

**Evidence:** When `og-feat-create-kb-guides-08jan-b223` was spawned at 11:10:57 and only modified .md files, `git log --since=2026-01-08T11:10:57 --name-only` returned code files from OTHER concurrent agents:

```
cmd/orch/complete_cmd.go
cmd/orch/serve_system.go
pkg/events/logger.go
pkg/verify/check.go
web/src/lib/components/...
```

**Source:** `git log --name-only --since="2026-01-08T11:10:57" --format="" | sort -u`

**Significance:** The original implementation assumed spawn time was sufficient to filter commits, but concurrent agents invalidate that assumption.

---

### Finding 2: The Logic Was Correct for File Types

**Evidence:** `isCodeFile()` correctly returns `false` for .md files. The existing tests pass:
- `TestIsCodeFile/markdown` - PASS
- `TestMarkdownOnlyChangesScenario` - PASS

**Source:** `pkg/verify/test_evidence_test.go:337-372`, `pkg/verify/test_evidence_test.go:528-583`

**Significance:** The bug was not in file classification, but in which commits were being checked.

---

### Finding 3: Fix Validated with Direct Testing

**Evidence:** After implementing workspace-filtered commit checking:
```
Spawn time: 2026-01-08 11:10:57.731619 -0800 PST
Old method (all commits since spawn): hasCodeChanges=true
New method (workspace-filtered): hasCodeChanges=false
```

**Source:** Direct testing with `/tmp/test_evidence_check.go`

**Significance:** The fix correctly isolates this agent's commits from concurrent agents' commits.

---

## Synthesis

**Key Insights:**

1. **Concurrent agents share git history** - When multiple agents run simultaneously, spawn time alone doesn't isolate their commits. Each agent's commit shows up in `git log --since` for ALL other agents spawned before it.

2. **Workspace as agent identity** - The workspace path uniquely identifies an agent's commits because each agent writes SYNTHESIS.md (and other files) to its workspace. Commits that touch a workspace belong to that agent.

3. **Two-phase filtering works** - First find commits that touch the workspace, then check only those commits for code changes. This correctly scopes the check.

**Answer to Investigation Question:**

The test_evidence gate was triggering on markdown-only changes because `HasCodeChangesSinceSpawn` used `git log --since=<spawn_time>` which includes ALL commits since spawn time. When concurrent agents made commits with code changes, those were incorrectly attributed to the markdown-only agent. The fix is to filter commits to only those that modified the specific agent's workspace directory.

---

## Structured Uncertainty

**What's tested:**

- ✅ Old method returns true for concurrent-agent scenario (verified: direct test script)
- ✅ New method returns false for markdown-only workspace (verified: direct test script)
- ✅ Fallback to old behavior when workspace is empty (verified: unit test)
- ✅ Non-existent workspace returns false (verified: unit test)

**What's untested:**

- ⚠️ Performance with large git history (not benchmarked, but uses same git commands)
- ⚠️ Edge case where workspace has no commits yet (should return false)

**What would change this:**

- Finding would be wrong if agents don't write to their workspace directories
- Solution wouldn't work if workspaces were shared between agents

---

## Implementation Recommendations

### Recommended Approach ⭐

**Workspace-filtered commit checking** - Filter commits to only those that modified the workspace directory before checking for code changes.

**Why this approach:**
- Uses existing data (workspace path already available)
- No changes to git workflow or commit structure needed
- Backward compatible (empty workspace falls back to old behavior)

**Trade-offs accepted:**
- Slightly more git commands (two calls instead of one)
- Assumes workspace path is reliable (it is in current architecture)

**Implementation sequence:**
1. Add `HasCodeChangesSinceSpawnForWorkspace` function
2. Update `VerifyTestEvidenceWithComments` to use new function
3. Keep old function for backward compatibility (deprecated)

---

## References

**Files Examined:**
- `pkg/verify/test_evidence.go` - Main verification logic
- `pkg/verify/test_evidence_test.go` - Existing tests
- `pkg/spawn/session.go:135-150` - Spawn time reading

**Commands Run:**
```bash
# Reproduce the issue
git log --name-only --since="2026-01-08T11:10:57" --format="" | sort -u

# Verify workspace commit filtering
git log --since="2026-01-08T11:10:57" --format=%H -- .orch/workspace/og-feat-create-kb-guides-08jan-b223
git show <hash> --name-only --format=
```

**Related Artifacts:**
- **Issue:** orch-go-vsdz3 - Bug: test_evidence gate triggers on markdown-only changes
- **Workspace:** .orch/workspace/og-feat-create-kb-guides-08jan-b223 - Example of affected agent

---

## Investigation History

**2026-01-08 11:29:** Investigation started
- Initial question: Why does test_evidence gate trigger on markdown-only changes?
- Context: Agent og-feat-create-kb-guides-08jan-b223 only modified .md files but gate failed

**2026-01-08 11:35:** Root cause identified
- `git log --since` includes concurrent agents' commits
- The commit 305462c1 (the agent's commit) has no code files, but concurrent commits do

**2026-01-08 11:45:** Fix implemented and validated
- Added `HasCodeChangesSinceSpawnForWorkspace` function
- Updated call site to pass workspace path
- Direct testing confirms fix works

**2026-01-08 11:50:** Investigation completed
- Status: Complete
- Key outcome: Bug fixed by filtering commits to workspace-specific ones
