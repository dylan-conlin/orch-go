<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added Phase: Complete verification gate to `bd close` command - now requires a "Phase: Complete" comment before allowing issue closure.

**Evidence:** Tests pass. Build succeeds. validatePhaseComplete function checks comments for "Phase: Complete" string.

**Knowledge:** The gate prevents manual `bd close` from bypassing the orchestrator's verification workflow. Use `--force` flag to override when necessary.

**Next:** Commit changes to beads repo and rebuild/install bd binary.

---

# Investigation: Gate bd close on Phase: Complete

**Question:** How to prevent manual `bd close` from bypassing verification - specifically requiring "Phase: Complete" comment exists before allowing closure?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** .kb/investigations/2025-12-30-inv-investigate-went-wrong-session-dec.md Finding 5
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: bd close validates via validateIssueClosable function

**Evidence:** The `close.go` command uses `validateIssueClosable` function in `show_unit_helpers.go` to check:
- Issue is not a template (read-only)
- Issue is not pinned (unless `--force`)

**Source:** `/Users/dylanconlin/Documents/personal/beads/cmd/bd/close.go:81`, `show_unit_helpers.go:21-32`

**Significance:** This is the existing validation pattern we should follow. The function takes `force` bool to allow bypass.

---

### Finding 2: Comments available via store.GetIssueComments or daemon RPC

**Evidence:** 
- Direct mode: `store.GetIssueComments(ctx, id)` returns `[]*types.Comment`
- Daemon mode: `daemonClient.ListComments(&rpc.CommentListArgs{ID: id})` returns comments via RPC

**Source:** `/Users/dylanconlin/Documents/personal/beads/cmd/bd/comments.go:41-69`

**Significance:** We can access comments in both daemon and direct modes to validate Phase: Complete.

---

### Finding 3: Comment struct has Text field for content

**Evidence:** `types.Comment` struct has `Text string` field containing the comment content.

**Source:** `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:608-614`

**Significance:** Simple string check with `strings.Contains(c.Text, "Phase: Complete")` is sufficient.

---

## Synthesis

**Key Insights:**

1. **Existing pattern works well** - Following `validateIssueClosable` pattern with separate validation function keeps code clean
2. **Force flag already exists** - Can reuse `--force` to bypass the new check (updated flag description)
3. **Both modes supported** - Comments fetched via daemon RPC or direct store access depending on mode

**Answer to Investigation Question:**

Added `validatePhaseComplete(id string, comments []*types.Comment, force bool) error` function that:
1. Returns nil if `force` is true (bypass validation)
2. Returns nil if any comment contains "Phase: Complete"
3. Returns error with clear message and hint to use `--force`

---

## Structured Uncertainty

**What's tested:**

- ✅ validatePhaseComplete returns error with no comments (verified: TestValidatePhaseComplete)
- ✅ validatePhaseComplete returns error with comments lacking "Phase: Complete" (verified: TestValidatePhaseComplete)
- ✅ validatePhaseComplete returns nil when "Phase: Complete" present (verified: TestValidatePhaseComplete)
- ✅ validatePhaseComplete returns nil when force=true (verified: TestValidatePhaseComplete)
- ✅ Build succeeds (verified: go build ./cmd/bd/...)

**What's untested:**

- ⚠️ Integration test of full `bd close` with Phase: Complete check in daemon mode
- ⚠️ Integration test of full `bd close` with Phase: Complete check in direct mode

**What would change this:**

- Finding would be wrong if comments retrieval fails silently (but error handling is in place)
- Finding would be wrong if "Phase: Complete" needs to be exact match (currently substring)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Implemented

**Added validatePhaseComplete function** - Checks comments for "Phase: Complete" string, respects force flag.

**Why this approach:**
- Follows existing validation pattern (validateIssueClosable)
- Reuses existing --force flag (no new flags needed)
- Simple string matching is robust (handles "Phase: Complete - summary" format)

**Trade-offs accepted:**
- Substring match may have false positives if someone writes "After Phase: Complete..." (acceptable)
- No check for comment author (agents should report their own phase)

**Implementation sequence:**
1. Add validatePhaseComplete to show_unit_helpers.go ✅
2. Call from close.go in daemon mode ✅
3. Call from close.go in direct mode ✅
4. Add unit tests ✅
5. Build and verify ✅

---

### Implementation Details

**Files changed:**
- `cmd/bd/show_unit_helpers.go` - Added `validatePhaseComplete` function
- `cmd/bd/show_unit_helpers_test.go` - Added tests for `validatePhaseComplete`
- `cmd/bd/close.go` - Call `validatePhaseComplete` in both daemon and direct modes
- Updated `--force` flag description to mention Phase: Complete check

**Things to watch out for:**
- ⚠️ Existing scripts using `bd close` will now fail without Phase: Complete comment (use --force)
- ⚠️ The check runs client-side, not on daemon server (consistent with other validations)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/close.go` - Close command implementation
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/show_unit_helpers.go` - Validation helpers
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go` - Comment struct definition

**Commands Run:**
```bash
# Build
go build ./cmd/bd/...

# Test
go test -v ./cmd/bd/... -run TestValidatePhaseComplete
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-30-inv-investigate-went-wrong-session-dec.md` - Original investigation that identified this gap

---

## Self-Review

- [x] Real test performed (unit tests pass)
- [x] Conclusion from evidence (substring match in comments)
- [x] Question answered (gate added to bd close)
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-30 18:15:** Investigation started
- Initial question: How to gate bd close on Phase: Complete?
- Context: Finding 5 from session investigation identified bypass

**2025-12-30 18:30:** Implementation complete
- Added validatePhaseComplete function
- Added unit tests
- Build and tests pass
- Status: Complete
