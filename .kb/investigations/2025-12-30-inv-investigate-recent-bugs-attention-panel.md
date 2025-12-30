<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Found 4 bugs in the attention panel: (1) suppress button copies non-existent command, (2) review done hangs on stdin, (3) action-log has no project filtering, (4) pending reviews API doesn't filter closed issues.

**Evidence:** Code inspection of patterns.go:569 shows hardcoded suggestion for non-existent command; review.go:781 waits for stdin; action-log.jsonl is global file with cross-project entries; handlePendingReviews doesn't call filterClosedIssues.

**Knowledge:** The dashboard UI and CLI commands have gaps between what the UI promises and what the CLI delivers; action tracking is intentionally global but surfacing should be project-filtered.

**Next:** Create 4 beads issues for the bugs found. Priority: (1) suppress command - P1 (misleading UI), (2) stdin hang - P2 (edge case), (3) cross-project noise - P1 (UX), (4) count mismatch - P1 (confusing).

---

# Investigation: Attention Panel and Review System Bugs

**Question:** What are the root causes of 4 bugs found during a session: (1) Dashboard suppress button copies non-functional command, (2) orch review done hangs, (3) Cross-project patterns in dashboard, (4) Pending reviews count mismatch?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None - findings ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Suppress Button Copies Non-Existent CLI Command

**Evidence:** The dashboard UI at `web/src/lib/components/needs-attention/needs-attention.svelte:587` hardcodes:
```javascript
onclick={() => copyCommand(`orch patterns suppress 0`)}
```

The `orch patterns` command in `cmd/orch/patterns.go` has NO `suppress` subcommand. It only has `--json` and `--verbose` flags. The `SuppressPattern` method exists in `pkg/patterns/analyzer.go:489` but is never exposed as a CLI command.

Additionally, `pkg/patterns/analyzer.go:569` has:
```go
sb.WriteString("  Run 'orch patterns suppress <index>' to suppress a pattern\n")
```
This message is displayed in the CLI output but the command doesn't exist.

**Source:** 
- `web/src/lib/components/needs-attention/needs-attention.svelte:587`
- `cmd/orch/patterns.go:22-49` (no suppress subcommand)
- `pkg/patterns/analyzer.go:569` (misleading help text)
- `pkg/patterns/analyzer.go:489` (SuppressPattern method exists but not exposed)

**Significance:** This is a UI/CLI gap where the frontend promises functionality that doesn't exist. Users who click the button get a command that will fail.

---

### Finding 2: orch review done Hangs on Stdin Input

**Evidence:** The `runReviewDone` function in `cmd/orch/review.go` uses `bufio.Reader` to wait for stdin input at two places:
- Line 781-783: Initial confirmation prompt ("Continue? [y/N]:")
- Line 829-831: Per-agent recommendation prompt ("Create follow-up issues? [y/n/skip-all]:")

When run without interactive stdin (e.g., in a script or piped context), these reads will hang indefinitely waiting for input.

The `--yes` flag skips the first prompt, and `--no-prompt` skips the second prompts, but both must be used together for truly non-interactive execution:
```bash
orch review done orch-go -y --no-prompt  # Non-hanging version
```

**Source:**
- `cmd/orch/review.go:779-792` (first prompt)
- `cmd/orch/review.go:819-878` (recommendation prompts)

**Significance:** This is expected behavior for an interactive command but the user may have expected it to be non-blocking or didn't realize stdin was needed. The command should perhaps detect non-TTY stdin and auto-apply `--no-prompt`.

---

### Finding 3: Action Log Has Cross-Project Noise

**Evidence:** The action-log.jsonl file is stored globally at `~/.orch/action-log.jsonl` and contains actions from ALL projects, not just the current one.

Sample entries found:
```json
{"tool":"Read","target":"/Users/.../price-watch/docker-compose.yml",...}
{"tool":"Bash","target":"cd /Users/.../price-watch && docker compose...",...}
```

The `GenerateBehavioralPatternsContext` function in `pkg/spawn/context.go:963-1019` loads patterns from this global file without filtering by project directory. There IS a `GenerateBehavioralPatternsContextForWorkspace` function (line 1021) that tries to filter by workspace name, but:
1. It falls back to global patterns if no workspace-specific ones exist
2. It doesn't filter by project directory

**Source:**
- `~/.orch/action-log.jsonl` - global file with 6213 entries
- `pkg/spawn/context.go:963-1019` - no project filtering
- `pkg/action/action.go:249-255` - defines global path `~/.orch/action-log.jsonl`

**Significance:** When viewing the dashboard for orch-go, patterns from price-watch and other projects appear, creating noise and confusion. The patterns shown are accurate (they did happen) but irrelevant to the current project context.

---

### Finding 4: Pending Reviews Count Mismatch (API vs Display)

**Evidence:** The `handlePendingReviews` function in `cmd/orch/serve.go:2960-3124` does NOT filter out workspaces whose beads issues are closed. It counts ALL workspaces with SYNTHESIS.md that have unreviewed NextActions.

In contrast, `getCompletionsForReview` in `cmd/orch/review.go:139-257` explicitly calls `filterClosedIssues` to exclude workspaces with closed beads issues.

This means:
- API `/api/pending-reviews` returns: 23 pending reviews (includes closed issues)
- `orch review` command shows: 17 pending reviews (excludes closed issues)

The discrepancy occurs because old workspaces with SYNTHESIS.md still exist even after their beads issues were closed. Without filtering, these are counted as "pending" forever.

**Source:**
- `cmd/orch/serve.go:2960-3124` - handlePendingReviews, no closed-issue filtering
- `cmd/orch/review.go:254-256` - uses filterClosedIssues
- `cmd/orch/review.go:259-305` - filterClosedIssues implementation

**Significance:** The dashboard shows an inflated count that doesn't match CLI output, creating user confusion. Users might think there's more work pending than actually exists.

---

## Synthesis

**Key Insights:**

1. **UI/CLI Gaps** - The dashboard UI and CLI have diverged in functionality. The suppress button and patterns output promise a command that was never implemented. This is a common pattern when backend capabilities are planned but not exposed.

2. **Global vs Project Scope** - The action-log is intentionally global for cross-session learning, but surfacing patterns should be project-aware. The current design trades noise for completeness.

3. **Filtering Inconsistency** - Different code paths (API vs CLI) handle closed-issue filtering differently. The API should reuse the same filtering logic as the CLI.

4. **Interactive Assumptions** - The review done command assumes interactive use but doesn't gracefully handle non-interactive contexts.

**Answer to Investigation Question:**

All 4 bugs are confirmed as real issues with identifiable root causes:

1. **Suppress button**: Implementation gap - the CLI command was never created despite UI and help text referencing it
2. **Review done hang**: Expected stdin behavior but missing TTY detection for graceful non-interactive handling
3. **Cross-project noise**: Architectural decision (global action log) without project-scoped filtering on display
4. **Count mismatch**: Missing `filterClosedIssues` call in the API handler that the CLI has

---

## Structured Uncertainty

**What's tested:**

- ✅ Suppress command doesn't exist (verified: checked patterns.go for subcommand registration, none found)
- ✅ Review done waits for stdin (verified: traced code path, found bufio.Reader.ReadString calls)
- ✅ Action log is global (verified: inspected ~/.orch/action-log.jsonl, found cross-project entries)
- ✅ API doesn't filter closed issues (verified: compared handlePendingReviews with getCompletionsForReview)

**What's untested:**

- ⚠️ Exact scenario that triggered the original hang (not reproduced, analyzed from code)
- ⚠️ Whether adding filtering to API causes performance issues (not benchmarked)
- ⚠️ Whether suppress functionality is actually needed (user research not done)

**What would change this:**

- Finding would be wrong if there's another patterns subcommand file I didn't find
- Finding would be wrong if stdin hang was caused by something other than the code paths identified
- Finding would be wrong if there's project-scoped action-log loading I missed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach (Fix All 4 Bugs)

**Multi-issue Fix** - Create 4 separate beads issues for each bug, prioritized by user impact

**Why this approach:**
- Each bug is independent and can be fixed separately
- Allows parallel work if desired
- Clear scope per issue

**Trade-offs accepted:**
- 4 separate spawns instead of one combined fix
- Slightly more coordination overhead

**Implementation sequence:**
1. **Bug 3 (Cross-project noise)** - Add project_dir field to pattern loading, filter by current project. Most visible user impact.
2. **Bug 4 (Count mismatch)** - Add filterClosedIssues to handlePendingReviews. Simple fix, high confusion reduction.
3. **Bug 1 (Suppress button)** - Either implement the CLI command or remove the button. Medium effort.
4. **Bug 2 (Stdin hang)** - Add TTY detection, auto-apply --no-prompt when non-interactive. Edge case fix.

### Alternative Approaches Considered

**Option B: Remove suppress functionality entirely**
- **Pros:** Simpler, no maintenance
- **Cons:** Users lose ability to silence known patterns
- **When to use instead:** If pattern suppression is rarely needed

**Option C: Fix only critical bugs**
- **Pros:** Faster
- **Cons:** Technical debt remains
- **When to use instead:** Time pressure, these specific issues not impacting users

---

### Implementation Details

**What to implement first:**
- Bug 3: Add project_dir filtering to GenerateBehavioralPatternsContext
- Bug 4: Add beads ID batch lookup and filtering to handlePendingReviews

**Things to watch out for:**
- ⚠️ Performance: Adding beads lookups to the API could slow it down - consider caching
- ⚠️ Backward compatibility: Suppress command implementation should match the documented format

**Areas needing further investigation:**
- Is there a use case for global patterns across projects?
- Should pattern suppression be persisted or temporary?

**Success criteria:**
- ✅ Suppress button either works or is removed
- ✅ review done works non-interactively with -y --no-prompt, or auto-detects
- ✅ Dashboard patterns are project-scoped
- ✅ API pending reviews count matches CLI output

---

## References

**Files Examined:**
- `cmd/orch/patterns.go` - patterns command implementation
- `cmd/orch/review.go` - review done implementation
- `cmd/orch/serve.go` - pending reviews API
- `pkg/patterns/analyzer.go` - pattern analysis and suppression
- `pkg/spawn/context.go` - behavioral patterns context generation
- `pkg/action/action.go` - action tracking
- `web/src/lib/components/needs-attention/needs-attention.svelte` - dashboard UI

**Commands Run:**
```bash
# Check action-log contents and size
cat ~/.orch/action-log.jsonl | head -20
wc -l ~/.orch/action-log.jsonl

# Check for cross-project entries
cat ~/.orch/action-log.jsonl | grep -E "price-watch|beads" | head -20

# Count workspaces with SYNTHESIS.md
find .orch/workspace -name "SYNTHESIS.md" | wc -l
```

---

## Self-Review

- [x] Real test performed (verified code paths, inspected files)
- [x] Conclusion from evidence (all 4 bugs confirmed with specific code locations)
- [x] Question answered (root causes identified for all bugs)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-30 13:49:** Investigation started
- Initial question: What are root causes of 4 bugs found during session?
- Context: Bugs identified during orchestrator session, needed investigation before fixing

**2025-12-30 14:15:** All 4 bugs analyzed
- Bug 1: Confirmed suppress command doesn't exist
- Bug 2: Confirmed stdin blocking behavior
- Bug 3: Confirmed global action log without project filtering
- Bug 4: Confirmed missing filterClosedIssues in API

**2025-12-30 14:30:** Investigation completed
- Status: Complete
- Key outcome: All 4 bugs confirmed with root causes identified, ready for implementation
