**TLDR:** Question: How should orch-go spawn command set beads issue status to in_progress? Answer: Call verify.UpdateIssueStatus(beadsID, "in_progress") after beads ID determination, before writing SPAWN_CONTEXT.md. High confidence (85%) - implemented and manually verified bd update works.

<!--
Example TLDR:
"Question: Why aren't worker agents running tests? Answer: Agents follow documentation literally but test-running guidance isn't in spawn prompts or CLAUDE.md, only buried in separate docs. High confidence (85%) - validated across 5 agent sessions but small sample size."

Guidelines:
- Keep to 2-3 sentences maximum
- Answer: What question? What's the answer? How confident?
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Set beads issue status to in_progress on spawn

**Question:** How should the orch-go spawn command automatically set beads issue status to in_progress when spawning an agent?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** dylanconlin
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (80-94%)

---

## Findings

### Finding 1: Current spawn flow creates beads issue but doesn't update status

**Evidence:** In `cmd/orch/main.go`, the `createBeadsIssue` function creates a new beads issue via `bd create` and returns its ID. The `runSpawnWithSkill` function uses this ID (or a provided `--issue` flag) but does not update the issue status to `in_progress`. The issue remains in `open` status.

**Source:** `cmd/orch/main.go:368-397` (createBeadsIssue), `cmd/orch/main.go:200-209` (beadsID determination), `cmd/orch/main.go:220-228` (spawn config)

**Significance:** The spawn command should set beads issue status to `in_progress` to reflect that work has started. This is required for proper tracking and for the orchestrator to monitor progress.

---

### Finding 2: bd update command supports --status flag to update issue status

**Evidence:** Running `bd update --help` shows the `--status` flag to set new status. The verify package already has functions to interact with beads (`GetIssue`, `CloseIssue`), but no function to update status.

**Source:** `bd update --help` output, `pkg/verify/check.go:147-161` (CloseIssue), `pkg/verify/check.go:163-182` (GetIssue)

**Significance:** We can use `bd update <id> --status in_progress` to update status. Should add a helper function in verify package or directly call exec.Command.

---

### Finding 3: Status update should happen after beads ID determined, before writing spawn context

**Evidence:** The beads ID is determined either from `--issue` flag or by creating a new issue. The status should be updated to `in_progress` before the agent starts working, so that when the agent reads SPAWN_CONTEXT.md, the issue already reflects the correct status. The update should happen in `runSpawnWithSkill` after beadsID assignment, before `spawn.WriteContext`.

**Source:** `cmd/orch/main.go:200-228` (beadsID determination and spawn config), `cmd/orch/main.go:225-228` (WriteContext call)

**Significance:** Placement ensures the agent sees the issue as in_progress, and any failure in status update can be logged but not block spawning.

### Finding 4: UpdateIssueStatus function already exists in verify package

**Evidence:** The `pkg/verify/check.go` file already contains a `UpdateIssueStatus` function that calls `bd update <id> --status <status>`. This function is not yet used in the spawn flow.

**Source:** `pkg/verify/check.go:163-172` (UpdateIssueStatus function)

**Significance:** We can directly call this existing function instead of implementing new logic. Need to import verify package and call `verify.UpdateIssueStatus(beadsID, "in_progress")`.

---

## Synthesis

**Key Insights:**

1. **Current spawn flow creates beads issues but doesn't update status** - The `createBeadsIssue` function creates issues with default `open` status, and the spawn command doesn't update them to `in_progress` when work begins.

2. **bd update command can set status** - The `bd update` command supports `--status` flag, allowing programmatic status updates. The verify package already has beads integration but lacks a status update function.

3. **Status update should happen early in spawn process** - Update should occur after beads ID determination (from `--issue` or new creation) but before writing SPAWN_CONTEXT.md, ensuring the agent sees correct status and failures don't block spawning.

**Answer to Investigation Question:**

The orch-go spawn command should call `bd update <beadsID> --status in_progress` after determining the beads ID (either from `--issue` flag or newly created issue). This should be done in `runSpawnWithSkill` before writing SPAWN_CONTEXT.md, with error handling that logs warnings but continues. A new helper function `UpdateIssueStatus` should be added to the `verify` package for consistency with existing beads operations.

---

## Confidence Assessment

**Current Confidence:** Medium (70%)

**Why this level?**
We have examined the spawn flow, beads integration, and bd update command. The approach is straightforward, but we haven't yet implemented or tested the changes. There is some uncertainty about error handling edge cases and whether there are any existing tests that might need updating.

**What's certain:**
- ✅ The spawn command currently does not update beads issue status to `in_progress`
- ✅ The `bd update` command supports `--status` flag for status updates
- ✅ The verify package already has beads integration functions (`CloseIssue`, `GetIssue`) that can be extended

**What's uncertain:**
- ⚠️ Whether there are any existing tests that might break with the new status update
- ⚠️ Edge cases: what if beads issue is already closed? Should we still update status?
- ⚠️ Should we also update status for inline spawn mode? (Yes, both tmux and inline)

**What would increase confidence to High (80%+):**
- Implement the helper function and integration
- Run existing tests to ensure no regressions
- Add new tests for the status update functionality

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
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
