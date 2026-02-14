# Model: Completion Verification Architecture

**Domain:** Completion / Verification / Quality Gates
**Last Updated:** 2026-01-14
**Synthesized From:** 31 investigations + completion.md guide on verification layers, UI approval gates, cross-project detection, escalation model, targeted bypasses

---

## Summary (30 seconds)

Completion verification operates through **three independent gates** (Phase, Evidence, Approval) that check different aspects of "done". Phase gate verifies agent claims completion, Evidence gate requires visual/test proof in beads comments, Approval gate (UI changes only) requires human sign-off. Verification is **tier-aware**: light tier checks Phase + commits, full tier adds SYNTHESIS.md, orchestrator tier checks SESSION_HANDOFF.md instead. The **5-tier escalation model** surfaces knowledge-producing work (investigation/architect/research) for mandatory orchestrator review before auto-closing. Cross-project detection uses SPAWN_CONTEXT.md to determine which directory to verify in. **Targeted bypasses** (`--skip-{gate} "reason"`) replace blanket `--force`, allowing specific gates to be skipped while others still run.

---

## Core Mechanism

### Three-Layer Verification

Each layer checks a different question:

| Layer | Question | Blocks Completion? | Checked By |
|-------|----------|-------------------|------------|
| **Phase Gate** | Did agent claim completion? | Yes | `bd comment` search for "Phase: Complete" |
| **Evidence Gate** | Does proof exist? | Yes | Beads comment search for screenshots/test output |
| **Approval Gate** | Did human verify? | Yes (UI only) | Beads comment search for "APPROVED" or --approve flag |

**Key insight:** Gates are **independent and cumulative**. All must pass. Phase without Evidence = incomplete. Evidence without Approval (for UI) = incomplete.

### Phase Gate

**What:** Verifies agent reported completion via beads comment.

**Check:**
```go
comments := beads.GetComments(beadsID)
for _, comment := range comments {
    if strings.Contains(comment.Text, "Phase: Complete") {
        return true
    }
}
return false  // Gate fails
```

**Why it exists:** Agent's claim of completion is the first signal. Without this, we're guessing if agent thinks it's done.

**What it doesn't check:** Whether work is actually complete, just whether agent claims it is.

**Source:** `pkg/verify/phase.go`

### Evidence Gate

**What:** Verifies visual or test evidence exists for UI/feature work.

**Check:**
```go
comments := beads.GetComments(beadsID)
hasScreenshot := false
hasTestOutput := false

for _, comment := range comments {
    if containsImageURL(comment.Text) || contains(comment.Text, "screenshot") {
        hasScreenshot = true
    }
    if contains(comment.Text, "test output") || contains(comment.Text, "✓") {
        hasTestOutput = true
    }
}

// UI work requires screenshot, feature work requires tests
if isUIWork && !hasScreenshot {
    return ErrMissingEvidence
}
if isFeatureWork && !hasTestOutput {
    return ErrMissingEvidence
}
```

**Why it exists:** Agents can claim "I tested it" without actually testing. Evidence gate requires proof in comments.

**What it doesn't check:** Whether evidence is correct, just whether it exists.

**Source:** `pkg/verify/evidence.go`

### Approval Gate (UI Changes Only)

**What:** Requires explicit human approval for UI modifications.

**Check:**
```go
// UI work = modified files under web/
if !modifiedWebFiles(workspace) {
    return nil  // Not UI work, skip approval
}

comments := beads.GetComments(beadsID)
for _, comment := range comments {
    if matchesApprovalPattern(comment.Text) {
        return nil  // Approved
    }
}

// Also check for --approve flag
if cliArgs.Approve {
    addComment(beadsID, "APPROVED by --approve flag")
    return nil
}

return ErrRequiresApproval  // Gate fails
```

**Approval patterns:**
- "APPROVED"
- "lgtm"
- "looks good"
- "ship it"

**Why it exists:** Agents can claim visual verification without actually doing it. Human approval gate prevents "agent renders wrong → thinks done → human discovers wrong" problem.

**Why only UI changes:** Code changes have test evidence, refactors have no behavior change. UI requires subjective judgment ("does this look right?").

**Source:** `pkg/verify/visual.go`

### Tier-Aware Verification

Different workspace tiers have different requirements:

| Tier | Artifact Required | Beads Checks | Phase Reporting | Handoff Update |
|------|-------------------|--------------|-----------------|----------------|
| **light** | None | Yes (status, comments) | Yes | Optional prompt |
| **full** | SYNTHESIS.md | Yes | Yes | Optional prompt |
| **orchestrator** | SESSION_HANDOFF.md | No | No | N/A |

**Implementation:**
```go
func VerifyCompletionWithTier(workspace string) error {
    tier := readTierFile(workspace)

    switch tier {
    case "light":
        return verifyLight(workspace)  // Phase + commits
    case "full":
        return verifyFull(workspace)   // Phase + commits + SYNTHESIS.md
    case "orchestrator":
        return verifyOrchestrator(workspace)  // SESSION_HANDOFF.md only
    default:
        return verifyFull(workspace)   // Default to full
    }
}
```

**Why tiers matter:**
- Light tier: Quick fixes, no investigation needed
- Full tier: Complex work requiring understanding artifacts
- Orchestrator tier: Different artifact (SESSION_HANDOFF.md), no beads tracking

**Source:** `pkg/verify/check.go:VerifyCompletionWithTier()`

### Activity Feed Persistence

The activity feed for completed agents remains viewable after completion via a hybrid persistent layer:
*   **Storage:** Proxied from OpenCode's `/session/:sessionID/messages` API.
*   **Reconciliation:** Historical messages are transformed into SSE-compatible events and merged with any real-time events stored in the frontend cache.
*   **Caching:** The dashboard uses a per-session Map cache for historical events to reduce redundant API calls during a browser session.

**Source:** `cmd/orch/serve_agents.go:handleSessionMessages()`

### Progressive Handoff Updates

To prevent knowledge loss at session end (**Capture at Context**), `orch complete` triggers interactive prompts for active orchestrator sessions.

**The Flow:**
1.  **Verify:** standard verification gates run for the worker agent.
2.  **Prompt:** Orchestrator is prompted for the worker's outcome and a 1-line key finding.
3.  **Inject:** The outcome is automatically inserted into the active session's `SESSION_HANDOFF.md` Spawns table.
4.  **Close:** Beads issue is closed only after handoff update (or skip).

**Source:** `cmd/orch/session.go:UpdateHandoffAfterComplete()`

---

## Why This Fails

### 1. Evidence Gate False Positive


**What happens:** Agent passes Evidence gate without actual visual verification.

**Root cause:** Agent generates screenshot placeholder text ("Screenshot attached") without actually attaching screenshot. Evidence gate searches for keyword "screenshot", finds it, passes.

**Why detection is hard:** Text-based keyword matching can't distinguish placeholder from actual proof.

**Fix:** Approval gate for UI changes. Even if Evidence passes, human must verify via --approve.

**Why this matters:** False positive on Evidence gate means broken UI ships thinking it's verified.

### 2. Approval Gate Bypass

**What happens:** Non-UI changes accidentally avoid approval gate.

**Root cause:** File path detection (`modifiedWebFiles()`) misclassifies files. `web-utils/` not under `web/`, approval skipped.

**Why detection is hard:** File structure varies across projects. Heuristics (path contains "web") can miss edge cases.

**Fix:** Explicit skill-based detection. `feature-impl` with UI flag requires approval, regardless of file paths.

**Future:** Skill manifest declares "requires_ui_approval: true".

### 3. Cross-Project Verification Wrong Directory

**What happens:** Verification runs in wrong directory, checks wrong tests, reports false failure.

**Root cause:** `SPAWN_CONTEXT.md` missing PROJECT_DIR, fallback uses workspace location (orch-go), but agent worked in orch-cli.

**Why detection is hard:** Workspace location != work location. No guaranteed signal of where work happened.

**Fix:** `orch spawn --workdir` explicitly sets PROJECT_DIR in SPAWN_CONTEXT.md. Verification reads it.

**Prevention:** Make --workdir mandatory for cross-project spawns, fail spawn if missing.

---

## Constraints

### Why Three Gates Instead of One?

**Constraint:** Verification checks Phase AND Evidence AND Approval separately.

**Implication:** Agent can pass Phase gate but fail Evidence gate. Each failure has different fix.

**Workaround:** None needed - this is the design.

**This enables:** Precise diagnostics for each failure mode (phase/evidence/approval)
**This constrains:** Cannot simplify to single pass/fail check

---

### Why Approval Only for UI Changes?

**Constraint:** Approval gate only applies to files under `web/`.

**Implication:** Backend changes auto-close even if agent claims visual verification.

**Workaround:** Manually check if suspicious, or spawn with custom verification requirements.

**This enables:** Backend work to complete without human bottleneck
**This constrains:** Cannot require human approval for non-UI changes without custom config

---

### Why Knowledge Work Surfaces, Not Auto-Closes?

**Constraint:** investigation/architect/research agents surface for review even if all gates pass.

**Implication:** Can't batch-close knowledge work overnight. Orchestrator must review next session.

**Workaround:** None needed - synthesis is the point.

**This enables:** Knowledge synthesis opportunity, findings integration into mental model
**This constrains:** Cannot batch-close knowledge work without orchestrator review

---

### Why Tier-Aware Verification?

**Constraint:** Orchestrator tier skips beads checks entirely.

**Implication:** Can't use standard verification flow for orchestrator sessions.

**Workaround:** `VerifyCompletionWithTier()` routes to tier-specific verification.

**This enables:** Different verification logic for orchestrator vs worker tiers
**This constrains:** Cannot use single verification flow for all work types

---

## Evolution

### Phase 1: Basic Verification (Dec 2025)

**What existed:** Phase gate only. Check for "Phase: Complete" comment, close beads issue.

**Gap:** No evidence checking, no UI approval, auto-closed everything.

**Trigger:** Agents claimed "tested, works" but shipped broken UI.

### Phase 2: Evidence Gate (Dec 26-28, 2025)

**What changed:** Added Evidence gate for visual/test proof. Search comments for screenshots, test output.

**Investigations:** 4 investigations on false claims, evidence patterns, keyword matching.

**Key insight:** Agents can claim verification without doing it. Evidence gate requires proof.

### Phase 3: Approval Gate (Dec 29-31, 2025)

**What changed:** Added human approval requirement for UI changes. --approve flag or "APPROVED" comment.

**Investigations:** 6 investigations on UI verification failures, approval patterns, bypass attempts.

**Key insight:** Even with Evidence gate, agents can attach wrong screenshot. Human approval is final gate for subjective quality.

### Phase 4: 5-Tier Escalation (Jan 2-4, 2026)

**What changed:** Knowledge-producing work (investigation/architect/research) surfaces for review instead of auto-closing.

**Investigations:** 8 investigations on completion rates, synthesis gaps, knowledge loss.

**Key insight:** Auto-closing knowledge work means findings never get synthesized. Surfacing forces orchestrator engagement.

### Phase 5: Cross-Project Verification (Jan 5-7, 2026)

**What changed:** Detection of project directory from SPAWN_CONTEXT.md, verification runs in correct directory.

**Investigations:** 4 investigations on verification failures, wrong directory detection, test path issues.

**Key insight:** Workspace location != work location. Must read spawn context to know where work happened.

### Phase 6: Targeted Bypasses (Jan 14, 2026)

**What changed:** Replaced blanket `--force` with targeted `--skip-{gate}` flags. Each gate can be bypassed individually with a required reason.

**New flags:**
- `--skip-phase "reason"` - Skip phase completion check
- `--skip-commits "reason"` - Skip git commits check
- `--skip-test-evidence "reason"` - Skip test evidence requirement
- `--skip-visual "reason"` - Skip visual verification
- `--skip-synthesis "reason"` - Skip SYNTHESIS.md check
- `--skip-decision-patch "reason"` - Skip decision impact check

**Constraint:** Reason must be ≥10 characters. Bypass events logged for observability.

**Key insight:** 55% of completions used `--force` to bypass ALL gates due to false positives. Targeted bypasses let agents skip specific failing gates while still running others.

**Verification metrics:** `orch stats` now shows pass/fail/bypass rates per gate, enabling data-driven improvement.

**Additional fixes in this phase:**
- Cross-repo file detection via mtime (files outside project verified by modification time)
- Markdown-only work exempted from test_evidence gate
- Zero spawn_time handled gracefully (skip with warning for legacy workspaces)

### Phase 7: Pure-Noise Gate Removal (Feb 2026)

**What changed:** Removed three gates identified as pure noise through friction analysis: `agent_running`, `model_connection`, and `commit_evidence`.

**Investigation:** Probe 2026-02-13 analyzed 1,008 bypass events and 403 failure events across all gates. Three gates showed extreme bypass:fail ratios indicating they never caught real defects:
- `agent_running`: ∞:1 ratio (183 bypasses, 0 failures) - never caught anything, 94% bypassed for GPT model compatibility
- `model_connection`: 71:1 ratio (71 bypasses, 1 failure) - almost never caught anything
- `commit_evidence`: 11.8:1 ratio (59 bypasses, 5 failures) - redundant with `git_diff` gate which already validates commits

**Key insight:** Gates that generate only bypass noise without catching defects should be removed entirely, not softened. These three gates were removed from both the verification code and CLI skip flags.

---

## References

**Guide:**
- `.kb/guides/completion.md` - Procedural guide (commands, workflows, troubleshooting)

**Investigations:**
- Completion.md references 10 investigations from Dec 2025 - Jan 2026
- Additional 16+ investigations on evidence gates, approval patterns, cross-project detection

**Decisions:**
- (Check for completion-related decisions in .kb/decisions/)

**Models:**
- `.kb/models/agent-lifecycle-state-model.md` - Where completion fits in agent lifecycle
- `.kb/models/orchestrator-session-lifecycle.md` - How orchestrator completion differs
- `.kb/models/spawn-architecture.md` - How SPAWN_CONTEXT.md sets PROJECT_DIR

**Source code:**
- `pkg/verify/check.go` - Main verification entry point, VerifyCompletionWithTier()
- `pkg/verify/phase.go` - Phase gate implementation
- `pkg/verify/evidence.go` - Evidence gate implementation
- `pkg/verify/visual.go` - Approval gate implementation (UI verification)
- `pkg/verify/cross_project.go` - Project directory detection
- `pkg/verify/escalation.go` - 5-tier escalation model
- `cmd/orch/complete.go` - Complete command orchestration
