**TLDR:** Implemented `orch review` command for batch completion workflow. Created review.go with commands: `orch review` (show pending completions grouped by project), `orch review -p project` (filter by project), `orch review --needs-review` (show failures only), `orch review done project` (mark project completions as reviewed). High confidence (95%) - all tests pass and command works as expected.

---

# Investigation: Add orch review command for batch completion workflow

**Question:** How to implement a batch completion review workflow for managing multiple completed agents?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Registry already has ListCompleted() method

**Evidence:** The `pkg/registry/registry.go` file already has a `ListCompleted()` method at line 389-400 that returns all agents with status `StateCompleted`.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/registry/registry.go:389-400`

**Significance:** No new registry methods needed - can use existing infrastructure directly.

---

### Finding 2: Verification infrastructure exists in verify package

**Evidence:** The `pkg/verify/check.go` file has `VerifyCompletion()` function that checks if an agent has reported "Phase: Complete" via beads comments.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go:116-147`

**Significance:** Can reuse existing verification logic to determine which completions need review vs are OK.

---

### Finding 3: Command patterns established in daemon.go and clean_test.go

**Evidence:** The daemon.go file shows the pattern for subcommands (daemon run, daemon once, daemon preview). The clean_test.go shows thorough testing patterns for registry-based operations.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/daemon.go`, `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/clean_test.go`

**Significance:** Followed established patterns for consistency.

---

## Synthesis

**Key Insights:**

1. **Reuse existing infrastructure** - ListCompleted() and VerifyCompletion() already exist, just needed to compose them into a new command.

2. **Grouping by project** - Used filepath.Base(agent.ProjectDir) to extract project name and group completions for the UI.

3. **Two-tier review status** - Completions can be OK (Phase: Complete reported) or NEEDS_REVIEW (verification failed or no phase reported).

**Answer to Investigation Question:**

Implemented a new `review` command in `cmd/orch/review.go` that:

- Lists completed agents from the registry
- Checks verification status for each using verify.VerifyCompletion()
- Groups by project for batch review
- Supports filtering by project (-p) and needs-review status (--needs-review)
- Provides `review done <project>` to mark all project completions as reviewed (deleted from registry)

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All tests pass (16 tests for review_test.go), command builds and runs correctly, follows established patterns.

**What's certain:**

- ✅ Command builds and tests pass
- ✅ Review workflow functions as specified
- ✅ Follows existing codebase patterns

**What's uncertain:**

- ⚠️ No integration test with actual beads CLI (unit tests mock this)
- ⚠️ UI output format might need adjustment based on real usage

---

## Implementation Recommendations

### Recommended Approach ⭐

**Review command with batch workflow** - Show pending completions grouped by project, allow filtering, and mark as reviewed.

**Why this approach:**

- Matches the orchestrator workflow described in SPAWN_CONTEXT.md
- Groups by project for efficient batch review
- Verification status helps prioritize attention

**Implementation sequence:**

1. `orch review` - List all pending completions
2. `orch review -p <project>` - Filter to specific project
3. `orch review --needs-review` - Show only failures
4. `orch review done <project>` - Mark project completions as reviewed

---

## References

**Files Created/Modified:**

- `cmd/orch/review.go` - New review command
- `cmd/orch/review_test.go` - Tests for review command
- `cmd/orch/main.go` - Wired reviewCmd into root command

**Commands Run:**

```bash
# Build
go build ./cmd/orch/...

# Test
go test ./cmd/orch/... -v

# Verify command works
/tmp/orch-test review --help
/tmp/orch-test review done --help
/tmp/orch-test review
```

---

## Investigation History

**2025-12-20 22:41:** Investigation started

- Initial question: How to implement `orch review` command
- Context: Spawned from beads issue orch-go-jqv

**2025-12-20 22:50:** Implementation complete

- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Implemented review.go with full batch completion workflow support
