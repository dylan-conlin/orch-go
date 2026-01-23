## Summary (D.E.K.N.)

**Delta:** Moved auto-rebuild logic to run BEFORE verification in `orch complete`, ensuring verification runs against fresh binaries.

**Evidence:** Build succeeds (`go build ./cmd/orch/...`), all auto-rebuild tests pass. Implementation adds cross-project support via `rebuildGoProjectsIfNeeded()`.

**Knowledge:** Previous implementation ran rebuild AFTER verification, causing verification to run against stale binaries. Cross-project agents (spawned from orch-go but modifying kb-cli) were not handled.

**Next:** Close - feature implemented and tested.

**Promote to Decision:** recommend-no (tactical fix extending existing behavior, not architectural)

---

# Investigation: Auto Rebuild Go Binaries During Verification

**Question:** How to auto-rebuild Go binaries BEFORE verification in `orch complete` so verification runs against fresh binaries?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A (extends 2025-12-24 investigation)

---

## Findings

### Finding 1: Existing auto-rebuild runs AFTER verification

**Evidence:** In `complete_cmd.go`, the auto-rebuild logic at lines 1056-1095 (before this change) ran after:
- Verification (lines 512-716)
- Beads issue closure (lines 901-933)
- Workspace archival (lines 985-1008)
- Tmux cleanup (lines 1029-1053)

**Source:** `cmd/orch/complete_cmd.go:1056-1095` (original code)

**Significance:** Verification was running against stale binaries when agents committed Go code changes. The fresh binary was only available AFTER completion, which defeats the purpose of verification.

---

### Finding 2: Cross-project agents not handled

**Evidence:** The existing `hasGoChangesInRecentCommits(beadsProjectDir)` only checked the beads project directory. If an agent spawned from orch-go modified code in kb-cli, the rebuild would not be triggered.

**Source:** `cmd/orch/complete_cmd.go:1057` - only checks `beadsProjectDir`

**Significance:** Cross-project work (agent working on different repo than where issue lives) would not trigger auto-rebuild of the affected repo.

---

### Finding 3: Solution structure

**Evidence:** Added `rebuildGoProjectsIfNeeded(beadsProjectDir, workspacePath)` function that:
1. Collects unique project directories (beads project + workspace PROJECT_DIR)
2. Checks each for Go changes via `hasGoChangesInRecentCommits`
3. Rebuilds each affected Go project
4. Restarts `orch serve` only if orch-go was rebuilt

**Source:** `cmd/orch/complete_cmd.go:1439-1491` (new function)

**Significance:** This ensures all affected Go repos are rebuilt before verification runs.

---

## Synthesis

**Key Insights:**

1. **Timing matters** - Rebuild must happen BEFORE verification to ensure verification tests the actual committed code, not stale binaries.

2. **Cross-project support** - Agents can be spawned from one project but work in another. Both must be checked for Go changes.

3. **Minimal duplication** - Removed duplicate rebuild logic from post-completion, kept only the CLI command detection feature.

**Answer to Investigation Question:**

The solution adds a `rebuildGoProjectsIfNeeded()` function called immediately after setup but before verification. It checks both the beads project directory and the workspace's PROJECT_DIR for Go changes, rebuilds any affected Go repos, and restarts `orch serve` if orch-go was rebuilt.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build compilation succeeds (`go build ./cmd/orch/...`)
- ✅ All auto-rebuild tests pass (TestHasGoChangesDetection, TestAutoRebuildLockPath, TestAutoRebuildIntegrationSkip)
- ✅ Cross-project error message test passes (TestCompleteCrossProjectErrorMessage)

**What's untested:**

- ⚠️ E2E test with actual Go changes and verification (requires real agent workflow)
- ⚠️ Multi-repo scenario (agent modifying both orch-go and kb-cli)
- ⚠️ Service restart behavior in overmind-managed environment

**What would change this:**

- If `extractProjectDirFromWorkspace` returns incorrect paths for cross-project workspaces
- If `restartOrchServe` conflicts with overmind process management

---

## Implementation Recommendations

### Recommended Approach ⭐

**Move rebuild before verification + cross-project support** - Add `rebuildGoProjectsIfNeeded()` that handles both scenarios.

**Why this approach:**
- Ensures verification runs against fresh binaries
- Handles cross-project agents (spawned from one project, working in another)
- Minimal changes to existing flow

**Trade-offs accepted:**
- Rebuild happens before verification passes (potentially wastes build if verification fails)
- Not a blocking gate - rebuild failures produce warnings, not errors

**Implementation sequence:**
1. Add `rebuildGoProjectsIfNeeded()` function
2. Call it before verification in `runComplete()`
3. Remove duplicate rebuild logic from post-completion (keep CLI command detection)

### Implementation Details

**What was implemented:**
- New `rebuildGoProjectsIfNeeded(beadsProjectDir, workspacePath string)` function
- Called at line 505 (after setup, before verification)
- Removed duplicate `runAutoRebuild` call from lines 1057-1069

**Things to watch out for:**
- ⚠️ `extractProjectDirFromWorkspace` must correctly parse SPAWN_CONTEXT.md
- ⚠️ Rebuild runs before verification, so verification failures happen after rebuild (acceptable trade-off)

**Success criteria:**
- ✅ Build compiles successfully
- ✅ Existing tests pass
- ✅ Verification runs against fresh binaries (verified by code inspection)

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go:300-550` - runComplete function flow
- `cmd/orch/complete_cmd.go:1439-1506` - existing runAutoRebuild and restartOrchServe
- `.kb/investigations/2025-12-24-inv-auto-rebuild-after-go-changes.md` - prior investigation

**Commands Run:**
```bash
/usr/local/go/bin/go build ./cmd/orch/...
/usr/local/go/bin/go test ./cmd/orch/... -run "TestHasGoChanges|TestAutoRebuild|TestComplete" -v
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-24-inv-auto-rebuild-after-go-changes.md` - Original implementation
- **Decision:** N/A (extends existing behavior)

---

## Investigation History

**2026-01-23 19:06:** Investigation started
- Initial question: How to auto-rebuild BEFORE verification

**2026-01-23 19:15:** Found existing implementation timing issue
- Rebuild at lines 1056-1095 runs AFTER verification

**2026-01-23 19:25:** Implementation complete
- Added `rebuildGoProjectsIfNeeded()` function
- Moved call to before verification
- Removed duplicate rebuild logic
- All tests pass

**2026-01-23 19:30:** Investigation completed
- Status: Complete
- Key outcome: Auto-rebuild now runs before verification with cross-project support
