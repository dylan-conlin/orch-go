## Summary (D.E.K.N.)

**Delta:** Defined retry/rework pattern for completed-but-broken features: reopen original issue + spawn new attempt with failure context injected via `POST-COMPLETION-FAILURE` comment.

**Evidence:** Analyzed orch-go-21226 failure case, reviewed existing infrastructure (`bd reopen`, attempt tracking in events.jsonl, `pkg/verify/attempts.go`), consulted beads state machine and principles.

**Knowledge:** Reopening preserves the causal chain and attempt history; new bugs fragment context across artifacts. The failure mode (bad spec, bad impl, bad verification) determines whether to reopen same issue or create investigation for root cause.

**Next:** Implement `orch rework` command that: (1) reopens issue, (2) adds failure context comment, (3) offers to spawn new attempt with prior context surfaced.

**Authority:** architectural - Establishes cross-component pattern affecting spawn, complete, and verification systems.

---

# Investigation: Retry/Rework Pattern for Completed-But-Broken Features

**Question:** What's the right pattern when a "completed" feature fails post-completion validation?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None - recommendation ready for orchestrator review
**Status:** Complete

---

## Context: The Triggering Example

Issue `orch-go-21226` (x-to-close feature in Work Graph):
- Worker reported `Phase: Complete` with build success evidence
- Orchestrator ran `orch complete`, closed the issue
- **Post-completion validation failed**: pressing x, entering reason, confirming does NOT close the issue - it persists after refresh
- Comment added: "POST-COMPLETION VALIDATION FAILURE: Feature doesn't work in practice"

This exposed a gap: **What's the correct workflow to rework a "completed" feature that doesn't actually work?**

---

## Findings

### Finding 1: Beads Already Supports Issue Reopening

**Evidence:** `bd reopen` command exists:
```bash
bd reopen [id...] [flags]
  -r, --reason string   Reason for reopening
```

Reopening:
- Sets status back to 'open'
- Clears `closed_at` timestamp
- Emits a `Reopened` event (for tracking)
- More explicit than `bd update --status open`

**Source:** `bd reopen --help`

**Significance:** The beads state machine already supports backward transitions. We don't need a new artifact type - we can reopen the original issue.

---

### Finding 2: Attempt Tracking Infrastructure Already Exists

**Evidence:** `pkg/verify/attempts.go` provides:
- `FixAttemptStats` struct tracking spawn/abandon/complete counts per beads ID
- `IsRetryPattern()` - detects multiple spawns with abandons
- `IsPersistentFailure()` - detects 2+ spawns, 2+ abandons, 0 completions
- `SuggestedAction()` - recommends "reliability-testing" or "investigate-root-cause"

The `orch retries` command surfaces aggregate retry patterns.

**Source:** `pkg/verify/attempts.go:1-100`

**Significance:** Reopening + respawning will naturally increment spawn counts. The existing infrastructure will surface "this issue keeps failing" patterns without additional work.

---

### Finding 3: Principles Demand Provenance and Context Preservation

**Evidence:** From `~/.kb/principles.md`:

> **Provenance (Foundational):** Every conclusion must trace to something outside the conversation.

> **Session Amnesia:** Every pattern in this system compensates for Claude having no memory between sessions. State must externalize to files.

> **Evidence Hierarchy:** Code is truth. Artifacts are hypotheses.

**Source:** `~/.kb/principles.md` - Provenance, Session Amnesia, Evidence Hierarchy principles

**Significance:** Creating a new bug issue would fragment the causal chain. The original issue contains the spec, the implementation discussion, the completion claim - all context a retry agent needs. Reopening preserves this chain; creating a new bug requires reconstructing context.

---

### Finding 4: Failure Modes Have Different Remediation Paths

**Evidence:** Analysis of what can go wrong when "complete" features don't work:

| Failure Mode | Symptom | Correct Remediation |
|--------------|---------|---------------------|
| **Bad verification** | Agent claimed tests pass but didn't run them | Reopen + retry with stricter verification gate |
| **Bad implementation** | Code has bug, tests incomplete | Reopen + retry, possibly with investigation first |
| **Bad spec** | Implementation matches spec but spec was wrong | New investigation to refine spec, then child task |
| **Integration failure** | Works in isolation, fails in context | Reliability-testing, may need architectural review |

**Source:** Analysis of orch-go-21226 and common failure patterns

**Significance:** Not all failures should be handled the same way. The pattern must account for WHY the feature failed to route appropriately.

---

## Decision Forks

### Fork 1: What Artifact Type Represents the Rework?

**Options:**
- A: Reopen original issue + spawn second attempt
- B: Create new bug issue referencing original
- C: Create child issue under original (fix)
- D: Just add comment + change status manually

**Substrate says:**
- **Provenance principle:** Reopening preserves the causal chain
- **Session Amnesia:** Agents need the original context to understand what was attempted
- **Attempt tracking model:** Events.jsonl tracks by beads ID - reopening keeps history unified

**RECOMMENDATION:** Option A - Reopen original issue

**Reasoning:** A new bug (Option B) fragments context - the agent debugging must discover and synthesize two issues. A child issue (Option C) complicates the issue hierarchy without benefit. Manual status change (Option D) loses the "Reopened" event for tracking. Reopening is the minimal, principled choice.

**Trade-off accepted:** Completion metrics will show the issue as "completed then reopened" rather than clean completion. This is a feature, not a bug - it surfaces issues that needed rework.

---

### Fork 2: How Does Agent Context Flow Across Attempts?

**Options:**
- A: New agent starts fresh (clean slate)
- B: New agent gets full previous context (all messages)
- C: New agent gets failure-focused summary

**Substrate says:**
- **Session Amnesia:** State must externalize to files
- **Progressive Disclosure:** TLDR first, full details available
- **Surfacing Over Browsing:** Bring relevant state to the agent

**RECOMMENDATION:** Option C - Failure-focused summary

**Reasoning:** Full context (Option B) is impractical - may exceed context window and includes irrelevant messages. Clean slate (Option A) risks repeating the same mistake. A structured failure summary in the spawn context gives the new agent:
1. What was attempted
2. What was claimed (Phase: Complete comment)
3. What actually failed (POST-COMPLETION-FAILURE comment)
4. Suggested diagnostic focus

**Implementation:** Add `POST-COMPLETION-FAILURE` comment to issue before respawning. Spawn context generator should detect this and include it prominently.

---

### Fork 3: How Do We Classify the Failure Mode?

**Options:**
- A: Orchestrator classifies manually when reopening
- B: Automated classification based on evidence
- C: Don't classify - let retry agent investigate

**Substrate says:**
- **Evidence Hierarchy:** Code is truth, artifacts are hypotheses
- **Premise before solution:** "Should we X?" before "How do we X?"

**RECOMMENDATION:** Option A - Orchestrator classifies manually

**Reasoning:** Automated classification (Option B) requires understanding what the agent actually did vs claimed - complex to implement and prone to false positives. Not classifying (Option C) means the retry agent may repeat the same mistake. Orchestrator classification is simple and leverages human judgment.

**Classification options for `orch rework`:**
- `--verification-failure`: Agent claimed success but didn't verify
- `--implementation-bug`: Code doesn't work as specified
- `--spec-issue`: Spec was wrong or incomplete
- `--integration-issue`: Works in isolation, fails in context

---

### Fork 4: Who/What Initiates the Rework?

**Options:**
- A: Orchestrator discovers via manual validation
- B: Automated post-completion gate
- C: User/stakeholder reports broken feature

**Substrate says:**
- **Gate Over Remind:** Enforce through gates, not reminders
- **The caveat:** Gates must be passable by the gated party

**RECOMMENDATION:** Option A initially, with path to Option B

**Reasoning:** Automated post-completion validation (Option B) would be ideal but requires knowing HOW to validate each feature - varies by type. Start with orchestrator discipline (manual validation after `orch complete`), then evolve toward automated gates where possible.

**Future consideration:** Some features (API endpoints, CLI commands) can have automated smoke tests. UI features may need screenshot diff or manual verification.

---

## Synthesis

**Key Insights:**

1. **Reopen is the right primitive** - It exists, emits trackable events, preserves context, and integrates with existing attempt tracking. Creating new artifacts fragments the causal chain.

2. **Context must be failure-focused** - The new agent doesn't need the full conversation history, but DOES need: what was attempted, what was claimed, what actually failed, and what to investigate.

3. **Failure classification enables appropriate remediation** - Not all failures are alike. Routing a verification failure vs a spec issue vs an integration problem leads to different skills and approaches.

4. **Human classification is the right first step** - Automated failure classification is complex and error-prone. Orchestrator judgment is simple and effective.

**Answer to Investigation Question:**

When a "completed" feature fails post-completion validation:

1. **Document the failure** - Add `POST-COMPLETION-FAILURE: [description]` comment to the original issue with specific observations
2. **Reopen the issue** - `bd reopen <id> --reason "Feature doesn't work: [brief]"`
3. **Classify the failure** - Determine if it's verification, implementation, spec, or integration failure
4. **Spawn new attempt** - With failure context surfaced prominently in spawn context
5. **If persistent failure** - Escalate to investigation or reliability-testing

---

## Structured Uncertainty

**What's tested:**

- ✅ `bd reopen` exists and emits Reopened event (verified: `bd reopen --help`)
- ✅ Attempt tracking scans events.jsonl for spawn/abandon/complete (verified: read `pkg/verify/attempts.go`)
- ✅ Issues can transition closed → open (verified: `bd reopen` command exists)

**What's untested:**

- ⚠️ How spawn context generator handles reopened issues (not examined)
- ⚠️ Whether Reopened events are tracked in attempt stats (events.jsonl scanning might miss them)
- ⚠️ Dashboard display of reopened issues (might show stale "completed" state)

**What would change this:**

- If beads doesn't support reopening (but it does)
- If attempt tracking can't distinguish reopen from fresh spawn (may need enhancement)
- If context window limits make failure-focused summary impractical (would need even more aggressive summarization)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| `orch rework` command | architectural | Creates new cross-component pattern, affects spawn/complete/verification |
| Failure comment format | implementation | Convention within existing comment system |
| Dashboard "reworking" state | implementation | UI enhancement within existing patterns |
| Automated validation gates | strategic | Requires deciding what/how to validate per feature type |

### Recommended Approach ⭐

**`orch rework <beads-id>`** - Single command that orchestrates the rework flow

**Why this approach:**
- Single entry point for the rework pattern
- Captures failure context as comment before reopening
- Can spawn new attempt with context surfaced
- Integrates with existing infrastructure (bd reopen, spawn)

**Trade-offs accepted:**
- Adds another `orch` subcommand (complexity)
- Requires orchestrator discipline (not automated)

**Implementation sequence:**

1. **Add `orch rework` command** (foundational)
   - Takes beads ID
   - Prompts for failure description
   - Prompts for failure type (verification/implementation/spec/integration)
   - Adds `POST-COMPLETION-FAILURE` comment
   - Runs `bd reopen`
   - Offers to spawn new attempt

2. **Update spawn context generator** (depends on #1)
   - Detect if issue has `POST-COMPLETION-FAILURE` comment
   - Surface failure context prominently in SPAWN_CONTEXT.md
   - Suggest appropriate skill based on failure type

3. **Update attempt tracking** (parallel with #2)
   - Ensure Reopened events are captured in stats
   - Add `ReopenedCount` to `FixAttemptStats`
   - Surface in `orch retries` output

4. **Update dashboard** (optional enhancement)
   - Show "reworking" badge for reopened issues
   - Distinguish from first attempt

### Alternative Approaches Considered

**Option B: Create new bug issue**
- **Pros:** Clean completion history, clear "bug" artifact
- **Cons:** Fragments context, agent must synthesize two issues, loses causal chain
- **When to use instead:** When the failure reveals a DIFFERENT bug (not the original feature failing)

**Option C: Create child fix issue**
- **Pros:** Keeps parent/child hierarchy, parent shows completion
- **Cons:** Overcomplicates issue tree, parent completion is misleading
- **When to use instead:** When spec is correct but needs follow-up refinement (new sub-task)

**Rationale for recommendation:** Reopening is the minimal, principled choice that preserves context and integrates with existing tracking. New artifacts create complexity without benefit for the common case.

---

### Implementation Details

**What to implement first:**
- `orch rework` command with basic flow (reopen + comment)
- This provides immediate value before spawn context enhancement

**Things to watch out for:**
- ⚠️ Dashboard may cache issue status - test that reopened issues display correctly
- ⚠️ Spawn context must handle `POST-COMPLETION-FAILURE` comment format
- ⚠️ Completion metrics reports may need adjustment to account for reopened issues

**Success criteria:**
- ✅ `orch rework <id>` reopens issue and adds failure context
- ✅ New spawn on reopened issue surfaces failure context in SPAWN_CONTEXT.md
- ✅ `orch retries` shows reopen count distinct from spawn count
- ✅ Dashboard shows "reworking" or similar badge for reopened issues

---

## References

**Files Examined:**
- `pkg/verify/attempts.go` - Existing attempt tracking infrastructure
- `~/.kb/principles.md` - Provenance, Session Amnesia, Evidence Hierarchy
- `.kb/guides/agent-lifecycle.md` - Phase reporting, completion flow
- `.kb/guides/completion.md` - Verification architecture, escalation model
- `.kb/guides/beads-integration.md` - Beads state machine, phase reporting

**Commands Run:**
```bash
# Check beads reopen capability
bd reopen --help

# Check existing attempt tracking
bd show orch-go-21226

# Query kb for related context
kb context "retry rework reopen bug"
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-26-inv-track-fix-attempts-issues-surface.md` - Prior work on attempt tracking
- **Model:** `.kb/models/completion-lifecycle.md` - Agent completion lifecycle understanding

---

## Investigation History

**2026-02-04 10:19:** Investigation started
- Initial question: What's the right pattern when completed features fail validation?
- Context: orch-go-21226 x-to-close feature marked complete but doesn't work

**2026-02-04 10:25:** Substrate consultation
- Found `bd reopen` exists with Reopened event
- Found attempt tracking infrastructure in `pkg/verify/attempts.go`
- Reviewed principles (Provenance, Session Amnesia)

**2026-02-04 10:35:** Fork navigation
- Identified 4 decision forks
- Navigated each with substrate reasoning
- Synthesized recommendation: reopen + failure comment + respawn

**2026-02-04 10:45:** Investigation completed
- Status: Complete
- Key outcome: Defined `orch rework` pattern - reopen original issue, add failure context, spawn new attempt with context surfaced
