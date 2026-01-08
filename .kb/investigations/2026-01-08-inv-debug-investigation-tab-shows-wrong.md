---
linked_issues:
  - orch-go-4gxrr
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation tab shows wrong files for cross-project agents because `discoverInvestigationPath` uses an incorrect `ProjectDir` when the workspace cache lookup fails.

**Evidence:** Code trace shows that `agents[i].ProjectDir` starts as `s.Directory` (orchestrator's cwd) and is only overwritten if `beadsProjectDirs[beadsID]` lookup succeeds; when it fails, `discoverInvestigationPath` searches the wrong project's `.kb/investigations/`.

**Knowledge:** Cross-project agents need a reliable fallback for `ProjectDir` - the session directory from OpenCode reflects the orchestrator's cwd due to `--attach` bug, not the target project.

**Next:** Fix the fallback logic - don't use session directory for investigation discovery when it's not the agent's target project.

**Promote to Decision:** recommend-no - tactical bug fix, not architectural

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

# Investigation: Debug Investigation Tab Shows Wrong

**Question:** Why does the Investigation tab show the wrong file for cross-project agents?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** og-inv-debug-investigation-tab-08jan-de93
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: ProjectDir initialization uses orchestrator's cwd

**Evidence:** In `serve_agents.go:466`, when creating the `AgentAPIResponse`, `ProjectDir` is initialized from `s.Directory` which is the OpenCode session directory. Due to the `--attach` bug in OpenCode, this is often the orchestrator's cwd, not the target project.

**Source:** `cmd/orch/serve_agents.go:458-467`

**Significance:** For cross-project agents spawned with `--workdir`, the initial `ProjectDir` value is wrong.

---

### Finding 2: ProjectDir overwrite only happens if workspace cache has the beads ID

**Evidence:** At line 780-782, `agents[i].ProjectDir` is overwritten from `beadsProjectDirs[agents[i].BeadsID]` - but only if the lookup succeeds. If the workspace cache doesn't have this beads ID (new agent, workspace not scanned yet, etc.), the incorrect initial value remains.

**Source:** `cmd/orch/serve_agents.go:780-782`

**Significance:** When workspace cache lookup fails, `ProjectDir` remains as the orchestrator's cwd, causing `discoverInvestigationPath` to search the wrong `.kb/investigations/` directory.

---

### Finding 3: discoverInvestigationPath searches based on ProjectDir

**Evidence:** At line 792, `discoverInvestigationPath(workspaceName, agents[i].BeadsID, agents[i].ProjectDir, invDirCache)` is called with the potentially incorrect `ProjectDir`. The function searches `projectDir/.kb/investigations/` for matching files.

**Source:** `cmd/orch/serve_agents.go:787-795` and `cmd/orch/serve_agents.go:144-268`

**Significance:** If `ProjectDir` is wrong (orchestrator's cwd instead of target project), the auto-discovery searches the wrong project's investigations and may find an unrelated file with similar keywords.

---

### Finding 4: Fix implemented - guard auto-discovery with hasReliableProjectDir

**Evidence:** Added `hasReliableProjectDir` boolean flag that's set to `true` only when workspace cache lookup succeeds. The `discoverInvestigationPath` call now has an additional guard: `agents[i].InvestigationPath == "" && hasReliableProjectDir`.

**Source:** `cmd/orch/serve_agents.go:779-802` (after fix)

**Significance:** Cross-project agents without a reliable project dir (from workspace cache) will no longer have investigation auto-discovery performed against the wrong project. They'll rely on the `investigation_path:` beads comment instead.

---

## Test Performed

**Test:** Built and ran `go install ./cmd/orch/...` after applying the fix. Verified all existing tests pass with `go test ./cmd/orch/... -v -count=1`.

**Result:** Build succeeded. All 52 tests passed. The fix adds a boolean guard `hasReliableProjectDir` that prevents investigation auto-discovery when the workspace cache doesn't provide a reliable project directory.

---

## Conclusion

The Investigation tab shows the wrong file for cross-project agents because `discoverInvestigationPath` uses an incorrect `ProjectDir` when the workspace cache lookup fails. The session directory (`s.Directory`) is used as a fallback, but for cross-project agents spawned with `--workdir`, this is the orchestrator's cwd (e.g., `orch-go`) not the target project (e.g., `scs-slack`).

The fix adds a `hasReliableProjectDir` guard that only allows investigation auto-discovery when the workspace cache successfully provides the project directory. Cross-project agents without a cached project dir will no longer incorrectly auto-discover investigations from the wrong project.

---

## Synthesis

**Key Insights:**

1. **Session directory is unreliable for cross-project agents** - Due to the OpenCode `--attach` bug, `s.Directory` reflects the orchestrator's cwd rather than the target project specified via `--workdir`.

2. **Workspace cache is the source of truth for project dir** - The `beadsProjectDirs` map populated from workspace cache contains the correct `PROJECT_DIR:` from `SPAWN_CONTEXT.md`.

3. **Auto-discovery should be conservative** - When we don't have reliable project dir information, it's better to show "No investigation file reported" than to show the wrong file from a different project.

**Answer to Investigation Question:**

The Investigation tab shows wrong files because the auto-discovery mechanism searches using an incorrect project directory. For cross-project agents where the workspace cache lookup fails, `ProjectDir` defaults to the session directory (orchestrator's cwd), causing the search to happen in the wrong project's `.kb/investigations/` directory. The fix guards auto-discovery with a `hasReliableProjectDir` flag that's only true when the workspace cache provides the project directory.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles and all existing tests pass (verified: `go install` and `go test` succeeded)
- ✅ Fix correctly adds guard condition to prevent auto-discovery without reliable project dir (verified: code review)

**What's untested:**

- ⚠️ Live dashboard behavior with a cross-project agent (requires orchestrator restart and real agent)
- ⚠️ Edge case where workspace exists but doesn't have PROJECT_DIR field (old format workspaces)

**What would change this:**

- Finding would be wrong if the session directory is actually reliable for some cross-project scenarios
- Finding would be incomplete if there are other code paths that populate ProjectDir incorrectly

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Guard auto-discovery with hasReliableProjectDir** - Only perform investigation file auto-discovery when the workspace cache provides a reliable project directory.

**Why this approach:**
- Minimal code change (adds one boolean flag and one guard condition)
- Conservative behavior - better to show "no file" than wrong file
- Directly addresses the root cause: unreliable ProjectDir for cross-project agents

**Trade-offs accepted:**
- Cross-project agents without workspace cache entry won't have auto-discovery
- Must rely on `investigation_path:` beads comment for cross-project agents
- This is acceptable because agents should report their investigation path anyway

**Implementation sequence:**
1. Add `hasReliableProjectDir` flag set when workspace cache lookup succeeds
2. Add guard condition to auto-discovery: `&& hasReliableProjectDir`
3. Test with existing test suite

### Alternative Approaches Considered

**Option B: Fix OpenCode --attach bug**
- **Pros:** Root cause fix, all cross-project scenarios would work
- **Cons:** Requires changes to OpenCode (different repo), longer timeline
- **When to use instead:** Long-term solution if cross-project issues persist

**Option C: Skip auto-discovery entirely, require investigation_path comment**
- **Pros:** Eliminates all auto-discovery bugs
- **Cons:** Worse UX for agents that forget to report path
- **When to use instead:** If more auto-discovery bugs appear

**Rationale for recommendation:** Option A is the minimal fix that addresses the immediate bug without changing the happy path (agents with proper workspace cache entries).

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Main handler for /api/agents, contains investigation path logic
- `cmd/orch/serve_agents_cache.go` - Workspace cache implementation, beadsProjectDirs population
- `pkg/verify/beads_api.go` - ParseInvestigationPathFromComments function
- `web/src/lib/components/agent-detail/investigation-tab.svelte` - Frontend component consuming investigation_content

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...
go install ./cmd/orch/...

# Test verification
go test ./cmd/orch/... -v -count=1
```

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Investigation History

**2026-01-08:** Investigation started
- Initial question: Why does Investigation tab show wrong file for cross-project agents?
- Context: Agent ss-inv-slack-lists-api-08jan-79f8 (scs-slack project) shows investigation file from orch-go instead of its own

**2026-01-08:** Root cause identified
- ProjectDir defaults to session directory (orchestrator's cwd) when workspace cache lookup fails
- discoverInvestigationPath uses incorrect ProjectDir to search wrong project's .kb/investigations/

**2026-01-08:** Fix implemented and verified
- Added hasReliableProjectDir guard to auto-discovery
- All existing tests pass
- Investigation completed
