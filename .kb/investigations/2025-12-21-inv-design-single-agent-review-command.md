<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Designed `orch review <id>` command that shows comprehensive agent work summary before completing. Decision: integrate with `orch complete` flow using `--preview` flag.

**Evidence:** Analyzed existing review.go (batch mode), verify/check.go (phase status), registry.go (agent metadata). Single-agent review needs different UX than batch review.

**Knowledge:** The gap is pre-completion inspection - orchestrator needs to see what agent did BEFORE deciding to complete. Current flow is verify-then-close, but needs review-then-decide.

**Next:** Implement `orch complete <id> --preview` as recommended approach. Separate `orch review <id>` as alias for discovery.

**Confidence:** High (85%) - Clear user need, existing patterns support implementation.

---

# Investigation: Single-Agent Review Command Design

**Question:** How should we implement the ability to review a single agent's work before completing it?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Agent og-work-design-single-agent-21dec
**Phase:** Complete
**Next Step:** None - Ready for implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current orch complete flow is verify-then-close

**Evidence:** `cmd/orch/main.go:1560-1673` - The `runComplete` function:
1. Gets issue from beads
2. Verifies phase status (Phase: Complete required)
3. Updates registry to mark completed
4. Closes beads issue
5. Logs event

**Source:** `cmd/orch/main.go:1560-1673`, `pkg/verify/check.go:282-327`

**Significance:** There's no "review what happened" step. The orchestrator either trusts Phase: Complete or uses --force. No visibility into workspace/commits/artifacts before deciding.

---

### Finding 2: Existing orch review is batch-oriented

**Evidence:** `cmd/orch/review.go` shows `orch review` designed for batch review:
- Groups completions by project
- Shows verification status (OK/NEEDS_REVIEW)
- Displays SYNTHESIS.md summary via `printSynthesisCard()`
- No single-agent focus

**Source:** `cmd/orch/review.go:1-457`

**Significance:** The batch review shows synthesis info but isn't interactive for a single agent. Need complementary single-agent mode.

---

### Finding 3: Rich metadata already available

**Evidence:** The system has access to:
- **Registry:** Agent ID, beads ID, session ID, workspace path, skill, spawn time (`pkg/registry/registry.go:37-61`)
- **SYNTHESIS.md:** TLDR, Delta (files/commits), Evidence (tests), Knowledge, Next actions (`pkg/verify/check.go:116-135`)
- **Beads comments:** Phase history, any agent questions (`pkg/verify/check.go:40-58`)
- **Git state:** Can be retrieved from workspace projectDir

**Source:** Multiple files in pkg/verify/, pkg/registry/

**Significance:** All the data for a comprehensive review exists. Just needs to be assembled and displayed.

---

### Finding 4: Gap between status and complete

**Evidence:** Current commands:
- `orch status` - Shows running agents (session ID, runtime, skill)
- `orch tail <id>` - Shows recent output
- `orch complete <id>` - Verifies and closes (no preview)

Missing: "What did this agent do?" before deciding to complete.

**Source:** Command analysis in `cmd/orch/main.go`

**Significance:** Orchestrator must either trust agent or dig manually (bd show, read SYNTHESIS.md, git log). No one-liner for comprehensive review.

---

## Synthesis

**Key Insights:**

1. **Review is a pre-decision step** - The orchestrator needs to see agent work BEFORE deciding whether to complete. Current flow assumes trust.

2. **Batch vs single are different UX** - `orch review` is good for "what's pending across projects?" but not for "let me examine this specific agent."

3. **Data exists, display is missing** - SYNTHESIS.md, beads comments, git commits all exist. Need a command to assemble them.

**Answer to Investigation Question:**

The best approach is to add a `--preview` flag to `orch complete` that shows comprehensive agent work summary before prompting for completion. This:
- Integrates with existing completion flow
- Doesn't require orchestrator to remember new command
- Allows "review then decide" in one step
- Can be aliased as `orch review <id>` for discoverability

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Clear user need (gap in workflow), existing patterns to follow (review.go's printSynthesisCard), data already available. Only uncertainty is exact UX for approval prompt.

**What's certain:**

- ✅ The gap exists - no pre-completion review
- ✅ Data is available (SYNTHESIS.md, beads, git)
- ✅ --preview flag pattern matches existing CLI conventions

**What's uncertain:**

- ⚠️ Exact format for git diff/commit display (too verbose?)
- ⚠️ Whether to block on missing SYNTHESIS.md or just warn
- ⚠️ How to handle agents without beads tracking

**What would increase confidence to Very High:**

- Build prototype and test with real agent completions
- Get orchestrator feedback on output format
- Validate edge cases (abandoned agents, missing artifacts)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**`orch complete <id> --preview` with optional approval prompt**

Adds `--preview` flag that shows:
1. Agent metadata (workspace, skill, duration)
2. SYNTHESIS.md TLDR and Delta
3. Beads phase history (last 3 comments)
4. Git commit summary (commits in workspace, file stats)
5. Artifacts produced (.kb/investigations, deliverables)
6. Prompts "Complete this agent? [y/N]"

Without --preview: current behavior (verify + close)
With --preview: show summary + prompt + optional close

**Why this approach:**
- Integrates naturally with existing complete flow
- No new command to remember
- Preview before decision, not as separate step
- Can abort if review reveals issues

**Trade-offs accepted:**
- Adds flag to existing command (slight complexity)
- Approval prompt requires stdin (not fully scriptable with --preview)

**Implementation sequence:**
1. Create `pkg/verify/review.go` with `AgentReview` struct and `GetAgentReview()` function
2. Add `--preview` flag to completeCmd in main.go
3. Implement display logic similar to printSynthesisCard
4. Add approval prompt with --yes to skip
5. Optionally add `orch review <id>` as alias for discoverability

### Alternative Approaches Considered

**Option B: Separate `orch review <id>` command**
- **Pros:** Clear separation, discoverable via `orch help`
- **Cons:** Fragmented workflow (review then complete separately), more commands to remember
- **When to use instead:** If `--preview` flag feels too hidden or users want review without completing

**Option C: Extend existing `orch review` to handle single ID**
- **Pros:** One command for both batch and single
- **Cons:** Confuses mental model (`orch review` = batch, `orch review <id>` = single), different output formats
- **When to use instead:** If strict command count minimization is priority

**Rationale for recommendation:** Option A (--preview flag) integrates review into the decision point where it's needed. The orchestrator doesn't need to remember "first review, then complete" - they just "complete with preview."

---

### Implementation Details

**What to implement first:**
1. `pkg/verify/review.go` with data gathering functions
2. `--preview` flag implementation
3. Display formatting (reuse printSynthesisCard patterns)

**Things to watch out for:**
- ⚠️ Git operations should be scoped to workspace directory, not project root
- ⚠️ Handle case where SYNTHESIS.md doesn't exist (show warning, still allow complete)
- ⚠️ Beads comment fetch can fail - graceful degradation

**Areas needing further investigation:**
- Exact git log format (commits since spawn? all in workspace?)
- How to summarize file diffs without verbosity

**Success criteria:**
- ✅ `orch complete <id> --preview` shows comprehensive summary
- ✅ Orchestrator can see TLDR, commits, test results before approving
- ✅ Works for agents with/without SYNTHESIS.md
- ✅ `--yes` flag allows scriptable completion

---

## References

**Files Examined:**
- `cmd/orch/main.go:1560-1673` - Complete command implementation
- `cmd/orch/review.go` - Batch review command
- `pkg/verify/check.go` - Verification and SYNTHESIS.md parsing
- `pkg/registry/registry.go:37-61` - Agent struct with metadata
- `.orch/templates/SYNTHESIS.md` - Template structure

**Commands Run:**
```bash
# Check recent git history
git log --oneline -n 10

# Find SYNTHESIS.md examples
find .orch/workspace -name "SYNTHESIS.md"
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-work-design-single-agent-21dec/`
- **Issue:** `bd show orch-go-3anf`

---

## Investigation History

**2025-12-21 12:42:** Investigation started
- Initial question: How to review single agent before completing?
- Context: Gap between batch review and completion flow

**2025-12-21 12:50:** Analyzed existing codebase
- Found review.go, verify/check.go, main.go complete command
- Identified data sources for review

**2025-12-21 13:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend --preview flag on orch complete
