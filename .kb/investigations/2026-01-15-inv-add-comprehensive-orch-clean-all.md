<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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

# Investigation: Add Comprehensive Orch Clean All

**Question:** How should `orch clean --all` be implemented to comprehensively clean all 4 agent status sources (tmux, OpenCode, beads, workspaces)?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** Worker agent (orch-go-u6p99)
**Phase:** Investigating
**Next Step:** Design and implement --all flag
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Existing cleanup flags already handle all 4 status sources

**Evidence:** The clean command currently has 6 cleanup flags:
- `--windows`: Close tmux windows for completed agents
- `--phantoms`: Close phantom tmux windows
- `--verify-opencode`: Delete orphaned OpenCode disk sessions
- `--investigations`: Archive empty investigation files
- `--stale`: Archive old completed workspaces
- `--sessions`: Delete stale OpenCode sessions

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/clean_cmd.go:80-90`

**Significance:** The infrastructure to clean all sources already exists. The `--all` flag just needs to enable all of these existing flags at once.

---

### Finding 2: Each cleanup action is independent and can run together

**Evidence:** Looking at the `runClean` function, each cleanup action is controlled by a boolean flag and runs independently:
- `cleanOrphanedDiskSessions` (line 326-331)
- `cleanPhantomWindows` (line 334-340)
- `archiveEmptyInvestigations` (line 343-349)
- `archiveStaleWorkspaces` (line 352-358)
- `cleanup.CleanStaleSessions` (line 361-373)

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/clean_cmd.go:272-374`

**Significance:** The --all flag can simply set all these boolean flags to true. No additional orchestration logic is needed.

---

### Finding 3: --preserve-orchestrator flag exists and should be respected

**Evidence:** Many cleanup functions accept a `preserveOrchestrator` parameter:
- `cleanOrphanedDiskSessions` (line 476)
- `cleanPhantomWindows` (line 610)
- `archiveStaleWorkspaces` (line 850)
- `cleanup.CleanStaleSessions` (line 363)

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/clean_cmd.go`

**Significance:** The --all flag should work in combination with --preserve-orchestrator, allowing comprehensive cleanup while protecting orchestrator sessions.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Boolean flag that enables all existing cleanup flags** - Add `cleanAll` boolean flag that, when true, sets all individual cleanup flags to true before calling `runClean`.

**Why this approach:**
- Leverages existing, well-tested cleanup functions (no new cleanup logic needed)
- Simple implementation: just set 6 boolean flags to true
- Maintains compatibility with existing flags (users can still use individual flags)
- Works with --preserve-orchestrator by passing through the existing parameter

**Trade-offs accepted:**
- Users can't customize which specific cleanups to exclude when using --all (must use individual flags for that)
- This is acceptable because --all is meant for "clean everything" use case; power users can still use individual flags

**Implementation sequence:**
1. Add `cleanAll` boolean flag to cleanCmd.Flags() 
2. In cleanCmd.RunE, if cleanAll is true, set all individual cleanup flags to true
3. Add tests to verify --all flag enables all cleanup actions
4. Update help text to document the --all flag

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
