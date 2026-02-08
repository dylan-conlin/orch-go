<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Untracked agents couldn't be cleaned up because `orch abandon` and `orch complete` required valid beads issues, which untracked agents don't have.

**Evidence:** Added `isUntrackedBeadsID()` check to both commands; smoke test shows `orch abandon orch-go-untracked-xxx` and `orch complete orch-go-untracked-xxx` now succeed without beads errors.

**Knowledge:** The `isUntrackedBeadsID()` helper already existed in review.go - the fix was moving it to shared.go and using it in abandon_cmd.go and complete_cmd.go to skip beads-dependent operations.

**Next:** Close - fix implemented, tests pass, smoke test verified. Commit and report Phase: Complete.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Untracked Agents Cleanup Path Problem

**Question:** Why can't untracked agents (spawned with --no-track) be cleaned up via orch abandon or orch complete?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Untracked agents get synthetic beads IDs that don't exist in database

**Evidence:** When `--no-track` is used, `determineBeadsID()` returns `fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix())` - e.g., `orch-go-untracked-1735947123`

**Source:** `cmd/orch/spawn_cmd.go:1258-1260`

**Significance:** These IDs are used for tmux window names and OpenCode session titles, making agents visible in `orch status`, but they don't exist as beads issues.

---

### Finding 2: orch abandon and orch complete require valid beads issues

**Evidence:** 
- `abandon_cmd.go:86-98` calls `verify.GetIssue(beadsID)` and fails if issue not found
- `complete_cmd.go:138-150` calls `verify.GetIssue(beadsID)` and fails if issue not found

**Source:** `cmd/orch/abandon_cmd.go:86-98`, `cmd/orch/complete_cmd.go:138-150`

**Significance:** Both commands fail early with "failed to get beads issue" for untracked agents, providing no cleanup path.

---

### Finding 3: isUntrackedBeadsID helper already exists but isn't used in abandon/complete

**Evidence:** `isUntrackedBeadsID(beadsID)` returns `true` if beads ID contains "-untracked-", and it's already used in `review.go` to filter/identify untracked agents.

**Source:** `cmd/orch/review.go:366-369`

**Significance:** The detection logic exists and is tested. It just needs to be applied in abandon_cmd.go and complete_cmd.go.

---

## Synthesis

**Key Insights:**

1. **Untracked agents use synthetic beads IDs** - The `--no-track` flag generates IDs like `orch-go-untracked-1735947123` that are visible in tmux/OpenCode but don't exist in beads database.

2. **Detection logic was duplicated** - `isUntrackedBeadsID()` existed in review.go but not available to abandon_cmd.go and complete_cmd.go. Moving to shared.go makes it reusable.

3. **Skip beads operations, not cleanup** - The fix skips beads-dependent operations (issue verification, status update, closing) but still performs cleanup (tmux window kill, event logging, auto-rebuild).

**Answer to Investigation Question:**

Untracked agents couldn't be cleaned up because both `orch abandon` and `orch complete` unconditionally called `verify.GetIssue(beadsID)` as their first validation step, and this fails for untracked agents since their synthetic beads IDs don't exist in the database. The fix adds early detection using `isUntrackedBeadsID()` and skips beads-dependent operations while preserving all cleanup functionality.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch abandon orch-go-untracked-xxx` succeeds (verified: ran command, got "Abandoned agent" output)
- ✅ `orch complete orch-go-untracked-xxx` succeeds (verified: ran command, got "Cleaned up untracked agent" output)
- ✅ Existing tests pass including `TestIsUntrackedBeadsID` (verified: go test ./cmd/orch/... - 4 tests passed)

**What's untested:**

- ⚠️ Full end-to-end with actual tmux window (no active untracked agents to kill)
- ⚠️ `--reason` flag with untracked abandon (FAILURE_REPORT.md generation)
- ⚠️ Discovered work gate with untracked complete (requires workspace with SYNTHESIS.md)

**What would change this:**

- If untracked agents need beads-like tracking, a local-only tracking mechanism would be needed
- If tmux windows use different naming for untracked agents, the beads ID extraction would fail

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

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
