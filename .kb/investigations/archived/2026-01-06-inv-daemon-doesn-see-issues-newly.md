<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Daemon was missing `--limit 0` flag in bd ready calls, causing it to only see first 10 issues by priority.

**Evidence:** Before fix: daemon found 10 issues. After fix: daemon found 20 issues. Issue 7rgz with triage:ready label is now visible.

**Knowledge:** bd ready defaults to limit 10; must pass `--limit 0` to get all issues. Fix was already in working tree but uncommitted.

**Next:** Fix committed in e6aeb559. Restart daemon to apply: `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`

---

# Investigation: Daemon Doesn't See Issues With Newly Added Labels

**Question:** Why doesn't the daemon pick up issues when labels are added after creation?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** og-debug-daemon-doesn-see-06jan-7d4d
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Daemon only saw 10 issues, not all ready issues

**Evidence:** Daemon log showed "DEBUG: Found 10 open issues" while `bd ready --json --limit 0` returned 18+ issues. Issue orch-go-7rgz was not in the list despite having triage:ready label.

**Source:** `~/.orch/daemon.log`, `bd ready --json --limit 0`

**Significance:** This explained why issues beyond the first 10 by priority were never found by the daemon, regardless of their labels.

---

### Finding 2: bd ready defaults to limit 10

**Evidence:**
```bash
bd ready --json | jq 'length'   # Returns 10
bd ready --json --limit 0 | jq 'length'   # Returns 18
```

**Source:** beads CLI behavior

**Significance:** Without explicit `--limit 0`, bd ready truncates results to 10 issues.

---

### Finding 3: Fix was already in working tree but uncommitted

**Evidence:** `git diff pkg/daemon/issue_adapter.go` showed uncommitted changes adding `--limit 0` to both RPC and CLI paths.

**Source:** pkg/daemon/issue_adapter.go (uncommitted changes)

**Significance:** Someone (likely an agent) had already identified and fixed the issue but didn't commit. The running daemon binary likely predated this fix.

---

## Synthesis

**Key Insights:**

1. **The bug was NOT about labels** - The original report assumed labels weren't being detected, but the actual issue was that issues beyond the first 10 were never fetched.

2. **Two code paths needed fixing** - Both the RPC path (`client.Ready(&beads.ReadyArgs{Limit: 0})`) and CLI fallback path (`--limit 0`) needed the explicit limit.

3. **Build vs. running binary mismatch** - The working tree had the fix but the daemon may have been running an older binary. After commit, the build auto-runs and updates the binary.

**Answer to Investigation Question:**

The daemon wasn't failing to detect labels on issues - it was failing to fetch issues beyond the first 10 because `bd ready` defaults to limit 10. The fix adds `--limit 0` to ensure all ready issues are fetched, regardless of how many exist.

---

## Structured Uncertainty

**What's tested:**

- ListReadyIssues() returns >10 issues when --limit 0 is used (verified: test script showed 20 issues)
- Issue orch-go-7rgz with triage:ready label is now visible in results (verified: test output)
- Labels are correctly populated in daemon.Issue struct (verified: test output showed labels array)

**What's untested:**

- Actual daemon loop with new binary (daemon not running during test)
- Performance impact of fetching all issues instead of 10 (likely negligible for typical backlogs)

**What would change this:**

- Finding would be wrong if daemon still sees only 10 issues after restart (would indicate different root cause)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach (DONE)

**Commit the --limit 0 fix** - Add explicit limit 0 to both RPC and CLI code paths.

**Why this approach:**
- Directly addresses root cause (bd ready default limit)
- Consistent with FallbackReady() and CLIClient.Ready() which already use --limit 0
- Minimal change, low risk

**Implementation sequence:**
1. [DONE] Commit the fix: `git commit -m "fix: add --limit 0 to bd ready calls"`
2. [TODO] Restart daemon: `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`
3. [TODO] Verify: `orch daemon preview` should show more than 10 issues

---

## References

**Files Examined:**
- pkg/daemon/issue_adapter.go:16-54 - ListReadyIssues() and listReadyIssuesCLI()
- pkg/daemon/daemon.go:198-245 - NextIssue() verbose logging
- pkg/beads/client.go:646-668 - FallbackReady() (already had --limit 0)
- pkg/beads/cli_client.go:76-102 - CLIClient.Ready() (already had limit handling)

**Commands Run:**
```bash
# Verify bd ready default limit
bd ready --json | jq 'length'  # 10
bd ready --json --limit 0 | jq 'length'  # 18

# Test fix
BEADS_NO_DAEMON=1 go run /tmp/test-limit-fix.go
# Found 20 issues (should be >10 if fix works)

# Check uncommitted changes
git diff pkg/daemon/issue_adapter.go
```

**Related Artifacts:**
- **Commit:** e6aeb559 - fix: add --limit 0 to bd ready calls to get ALL issues

---

## Investigation History

**2026-01-06 10:49:** Investigation started
- Initial question: Why doesn't daemon see issues with newly added labels?
- Context: Issue orch-go-7rgz had triage:ready label but wasn't in daemon's 10 issues

**2026-01-06 10:55:** Root cause identified
- Daemon log showed only 10 issues found
- bd ready defaults to limit 10, not all issues

**2026-01-06 11:00:** Fix found uncommitted in working tree
- git diff revealed --limit 0 changes were made but not committed
- Committed fix with proper message

**2026-01-06 11:05:** Investigation completed
- Status: Complete
- Key outcome: Bug was not label detection but issue fetch limit; fix committed
