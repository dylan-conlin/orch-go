<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented `orch sync` command that auto-closes beads issues when commits explicitly reference them (closes/fixes/resolves markers or fix commits with issue IDs).

**Evidence:** Tests pass (14 cases); dry-run correctly excludes false positives like "Create epic xyz" commits.

**Knowledge:** Simply matching issue IDs in commits leads to false positives; must require explicit close intent (closes/fixes/resolves keywords or fix-type commits).

**Next:** Commit changes and integrate into workflow via post-commit hook or manual invocation.

---

# Investigation: Auto Close Beads Issues Related

**Question:** How to auto-close beads issues when related commits land?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing infrastructure for commit parsing

**Evidence:** `pkg/verify/stale_bug.go` already has commit parsing logic for detecting potentially stale bugs. It extracts keywords from issue titles and searches git history.

**Source:** `pkg/verify/stale_bug.go:79-125` - `searchCommits()` function

**Significance:** Provides a pattern for parsing commit history using `git log --format=%h|%s|%an|%aI`.

---

### Finding 2: Beads client interface supports issue closing

**Evidence:** `pkg/beads/interface.go` defines `CloseIssue(id, reason string) error` method, and `pkg/beads/cli_client.go` implements it with proper reason support.

**Source:** `pkg/beads/interface.go:34-35`, `pkg/beads/cli_client.go:198-207`

**Significance:** Clean API exists to close issues with descriptive reasons.

---

### Finding 3: Simple issue ID matching produces false positives

**Evidence:** Initial implementation matched any beads issue ID in commit messages. This incorrectly matched commits like "feat: Create epic orch-go-6uli" which reference but don't close the issue.

**Source:** Manual testing with `orch sync --dry-run --verbose`

**Significance:** Requires explicit close intent detection (closes/fixes/resolves keywords or fix commit types).

---

## Synthesis

**Key Insights:**

1. **Explicit intent required** - Only commits with clear close markers (closes/fixes/resolves) or fix-type commits should trigger auto-close. Mere mention of an issue ID doesn't imply completion.

2. **Create/update exclusion** - Commits that create or update issues must be explicitly excluded to prevent closing issues immediately after creation.

3. **Short IDs risky** - Short 4-char IDs can collide with common words; removed short ID matching from non-explicit-close patterns.

**Answer to Investigation Question:**

Implemented `orch sync` command that:
- Scans recent commits for explicit close markers (closes/fixes/resolves)
- Matches fix-type commits with issue IDs in scope or body
- Excludes creation/update commits
- Closes matching open/in_progress issues with commit reference as reason

---

## Structured Uncertainty

**What's tested:**

- ✅ Full issue ID matching with closes/fixes/resolves markers (verified: TestExtractIssueRefs)
- ✅ Exclusion of create/update commits (verified: TestExtractIssueRefs_ExcludesCreation)
- ✅ No duplicate matches for same issue (verified: TestExtractIssueRefs_NoDuplicates)
- ✅ Dry-run mode doesn't close issues (verified: manual test)

**What's untested:**

- ⚠️ Integration with post-commit git hook (not implemented)
- ⚠️ Cross-repo issue closing (orch sync runs in current directory only)
- ⚠️ Performance with very large commit histories (tested with 517 commits only)

**What would change this:**

- Finding would be wrong if users commonly use issue IDs in non-closing contexts that match our patterns
- Need refinement if fix commits without explicit markers should also close issues

---

## Implementation Delivered

### Files Created

1. `cmd/orch/sync.go` - Main sync command implementation
2. `cmd/orch/sync_test.go` - Unit tests for issue reference extraction

### Usage

```bash
orch sync                     # Check last 7 days of commits
orch sync --days 30           # Check last 30 days  
orch sync --dry-run           # Preview what would be closed
orch sync --verbose           # Show detailed output
orch sync --json              # Output as JSON
```

### Issue ID Patterns Detected

- Explicit close markers: `closes #orch-go-f9l5`, `fixes orch-go-f9l5`, `resolves kb-cli-abc1`
- Fix commit with scope: `fix(orch-go-f9l5): description`
- Fix commit with ID: `fix: description orch-go-f9l5`

### Patterns Excluded

- Create commits: `feat: Create epic orch-go-xyz`
- Update commits: `chore: update issue orch-go-xyz`
- Mere mentions: `feat: implement feature (orch-go-xyz)`

---

## References

**Files Examined:**
- `pkg/verify/stale_bug.go` - Existing commit parsing pattern
- `pkg/beads/interface.go` - BeadsClient interface
- `pkg/beads/cli_client.go` - CLI implementation of CloseIssue
- `cmd/orch/changelog.go` - Similar git log parsing

**Commands Run:**
```bash
# Test sync in dry-run mode
orch sync --dry-run --verbose

# Run unit tests
go test -v -run "TestExtractIssueRefs|TestTruncateSyncString" ./cmd/orch/...
```

---

## Investigation History

**2026-01-01 19:08:** Investigation started
- Initial question: How to auto-close beads issues when related commits land?
- Context: Issues gxwu and lsrj stayed open after fixes were committed

**2026-01-01 19:11:** Implementation complete
- Created `orch sync` command with conservative matching patterns
- Tests pass, false positives eliminated

**2026-01-01 19:15:** Investigation complete
- Status: Complete
- Key outcome: `orch sync` command ready for use
