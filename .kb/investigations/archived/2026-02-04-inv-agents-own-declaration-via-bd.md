<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Issue lifecycle (open→closed) and agent lifecycle (spawned→done) are separate state machines that should remain separate, with Phase: Complete as the mapping event between them.

**Evidence:** Code analysis confirms current architecture already separates these concerns: agents report Phase: Complete via `bd comment`, orch complete runs verification gates, then closes issue via `bd close`. The proposal to add `bd complete` would just be syntactic sugar around `bd comment "Phase: Complete"`.

**Knowledge:** Three orthogonal dimensions exist: (1) Work Status (agent progress), (2) Verification Status (quality gates), (3) Issue Status (beads lifecycle). These should remain independent with explicit mapping events. The current "Phase: Complete → orch complete → bd close" chain is correct; the friction is in the orchestrator bottleneck, not the architecture.

**Next:** Close - superseded by .kb/decisions/2026-02-07-agent-completion-lifecycle-separation.md.

**Authority:** architectural - This affects cross-component boundaries (beads, orch, daemon) and multiple valid approaches exist

---

# Investigation: Agent Done Declaration and Lifecycle Separation

**Question:** Should agents own their 'done' declaration via `bd complete`, and how should issue lifecycle vs agent lifecycle be modeled as separate state machines?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Architect
**Phase:** Complete
**Next Step:** None - recommendation ready for decision
**Status:** Complete
**Superseded-By:** .kb/decisions/2026-02-07-agent-completion-lifecycle-separation.md

<!-- Lineage -->
**Related-Models:**
- `.kb/models/completion-lifecycle.md` - Current 4-layer completion chain
- `.kb/models/completion-verification.md` - Three-gate verification architecture
- `.kb/models/agent-lifecycle-state-model.md` - Four-layer agent state model
**Related-Investigation:** `.kb/investigations/2026-02-04-arch-work-graph-done-states.md` - Work Graph UI state model

---

## Findings

### Finding 1: Current Architecture Already Separates Agent Declaration from Issue Closure

**Evidence:** The current completion flow has clear separation:

```
Agent                      Orchestrator                 Beads
  |                            |                          |
  |--bd comment "Phase: Complete"------------------------>| (agent declares)
  |                            |                          |
  |                            |<--orch complete----------|
  |                            |   (verification gates)   |
  |                            |                          |
  |                            |---bd close-------------->| (issue closes)
```

Key code paths:
- Agent declaration: `bd comment <id> "Phase: Complete - <summary>"` - pure declaration, no closure
- Verification: `pkg/verify/check.go:VerifyCompletionFull()` - runs 11 gates
- Issue closure: `pkg/verify/beads_api.go:CloseIssue()` - separate from verification

**Source:**
- `cmd/orch/complete_cmd.go:315-1279` - Complete command implementation
- `pkg/verify/check.go:220-422` - Verification gate orchestration
- `pkg/verify/beads_api.go:186-206` - Issue closure wrapper

**Significance:** The proposal to add `bd complete` as an agent-callable command would not change the fundamental architecture - agents already "own" their done declaration via `bd comment "Phase: Complete"`. The `orch complete` step is orchestrator verification, not agent action.

---

### Finding 2: Three Orthogonal State Dimensions Exist

**Evidence:** The completion lifecycle involves three independent state dimensions:

**1. Work Status (Agent Progress)**
```
not_started → planning → implementing → testing → done
```
- Tracked via: Phase comments in beads (`Phase: Planning`, `Phase: Implementing`, `Phase: Complete`)
- Actor: Agent
- Authority: Agent owns this dimension

**2. Verification Status (Quality Gates)**
```
unverified → verification_passed | verification_failed → human_verified
```
- Tracked via: `orch complete` result, verification events in `~/.orch/events.jsonl`
- Actor: Orchestrator (automated) + Human (UI changes)
- Authority: System owns automated gates, human owns subjective gates

**3. Issue Status (Beads Lifecycle)**
```
open → in_progress → blocked → closed
```
- Tracked via: Beads issue status field
- Actor: Orchestrator (typically) or Human
- Authority: Currently orchestrator, but could be daemon

**Current conflation problem:** The UI shows `verify` badge (agent done, needs orch complete) and `unverified` badge (issue closed, needs human check) without clearly distinguishing which dimension is represented.

**Source:**
- `pkg/verify/beads_api.go:40-46` - PhaseStatus struct (Work Status)
- `pkg/verify/check.go:30-37` - VerificationResult struct (Verification Status)
- `bd close --help` shows issue statuses: `--outcome completed|could-not-reproduce|duplicate|wont-fix|invalid`

**Significance:** The confusion in the Work Graph UI stems from presenting these three dimensions without clear visual separation. The architecture is sound; the presentation is the problem.

---

### Finding 3: The Orchestrator Bottleneck is the Real Friction Point

**Evidence:** The current model requires orchestrator intervention to close issues:

```
Agent completes work
       ↓
Agent reports Phase: Complete
       ↓
[BOTTLENECK: Waiting for orchestrator]
       ↓
Orchestrator runs orch complete
       ↓
Issue closes
```

The `.kb/models/completion-lifecycle.md:46` explicitly states:
> **The Verification Bottleneck**: Spawning is automated (Daemon), but completion is manual.

**Current daemon behavior:**
- Can spawn agents automatically via `bd ready` polling
- Cannot verify/close agents automatically
- Results in completion queue buildup

**Source:**
- `.kb/models/completion-lifecycle.md:46-47` - Verification Bottleneck constraint
- `pkg/daemon/daemon.go` - Daemon only handles spawning, not completion

**Significance:** The proposal to have agents call `bd complete` doesn't solve this bottleneck - it just moves the `bd comment "Phase: Complete"` into a wrapper command. The real solution is either:
1. Make daemon smart enough to auto-verify simple completions
2. Accept that verification is inherently bottlenecked (for quality reasons)
3. Trust agents more and reduce verification gates

---

### Finding 4: `bd close` Already Has Phase: Complete Check

**Evidence:** The beads CLI has built-in completion verification:

```bash
$ bd close --help
      --force            Force close (bypasses pinned and Phase: Complete checks)
```

This means `bd close` without `--force` already checks for Phase: Complete. The orchestrator layer (`orch complete`) adds additional gates:
- SYNTHESIS.md existence
- Test evidence in comments
- Visual verification for UI changes
- Git diff validation
- Build verification
- Decision patch limits

**Source:**
- `bd close --help` output
- `cmd/orch/complete_cmd.go:69-81` - Gate list documentation

**Significance:** The architecture supports having `bd close` be the authoritative closure command with Phase: Complete check. The `orch complete` adds orchestrator-specific gates. A `bd complete` command would be redundant unless it added beads-level verification gates.

---

## Synthesis

### State Machine Diagrams

**Issue Lifecycle State Machine (Beads)**

```
                    ┌──────────────────────────────────────────────┐
                    │            ISSUE LIFECYCLE                   │
                    │              (Beads)                         │
                    └──────────────────────────────────────────────┘

                           bd create
                              │
                              ▼
                    ┌─────────────────┐
                    │      OPEN       │
                    │  (ready queue)  │
                    └────────┬────────┘
                             │ bd update --status=in_progress
                             ▼
                    ┌─────────────────┐
                    │   IN_PROGRESS   │◄─────────────────────┐
                    │  (being worked) │                      │
                    └────────┬────────┘                      │
                             │                               │
              ┌──────────────┼──────────────┐                │
              │              │              │                │
              ▼              │              ▼                │
    ┌─────────────────┐      │    ┌─────────────────┐        │
    │     BLOCKED     │      │    │     CLOSED      │        │
    │ (dependency)    │      │    │  (completed)    │        │
    └────────┬────────┘      │    └─────────────────┘        │
             │               │              ▲                │
             │ unblock       │              │                │
             └───────────────┘              │                │
                                  bd close --reason          │
                                            │                │
                                            │                │
                                    (from IN_PROGRESS)───────┘
                                        bd reopen
```

**Agent Lifecycle State Machine (Orch)**

```
                    ┌──────────────────────────────────────────────┐
                    │            AGENT LIFECYCLE                   │
                    │               (Orch)                         │
                    └──────────────────────────────────────────────┘

                         orch spawn
                              │
                              ▼
                    ┌─────────────────┐
                    │     SPAWNED     │
                    │(workspace ready)│
                    └────────┬────────┘
                             │ agent starts working
                             ▼
                    ┌─────────────────┐
                    │     ACTIVE      │
                    │(phases reported)│─────────┐
                    └────────┬────────┘         │
                             │                  │
              ┌──────────────┴──────────────┐   │
              │                             │   │ orch abandon
              ▼                             │   ▼
    ┌─────────────────┐           ┌─────────────────┐
    │  PHASE_COMPLETE │           │    ABANDONED    │
    │(agent says done)│           │  (stuck/lost)   │
    └────────┬────────┘           └─────────────────┘
             │ orch complete (verification)
             ▼
    ┌─────────────────┐
    │    VERIFIED     │
    │(work confirmed) │
    └────────┬────────┘
             │ (archives workspace, deletes session)
             ▼
    ┌─────────────────┐
    │    COMPLETED    │
    │  (terminal)     │
    └─────────────────┘
```

**Mapping Between Lifecycles**

```
                    AGENT LIFECYCLE              ISSUE LIFECYCLE
                    ──────────────               ───────────────

                        SPAWNED    ───────────►    IN_PROGRESS
                           │                          │
                           ▼                          │
                        ACTIVE     ◄──────────────────┤
                           │                          │
                           ▼                          │
                    PHASE_COMPLETE ──────────────────►│ (no state change)
                           │                          │
                           │ orch complete            │
                           ▼                          ▼
                       VERIFIED    ───────────►     CLOSED
                           │
                           ▼
                       COMPLETED

    Key mapping events:
    1. orch spawn:        creates agent state + sets issue to in_progress
    2. Phase: Complete:   agent declares done (no issue state change)
    3. orch complete:     verifies work + closes issue
```

### Key Insights

1. **Separation Already Exists** - The current architecture correctly separates agent declaration (`bd comment "Phase: Complete"`) from issue closure (`bd close`). The `orch complete` command is the mapping function that connects the two lifecycles with verification gates.

2. **Three Dimensions, Not Two** - The completion lifecycle has three orthogonal dimensions: Work Status (agent progress), Verification Status (quality gates), and Issue Status (beads lifecycle). Proposals to "simplify" by merging these will create new conflation problems.

3. **The Bottleneck is the Design** - The orchestrator verification step is intentionally a bottleneck. The question isn't "how do we remove it?" but "what level of automation is appropriate?" For simple tasks, daemon auto-verification may be appropriate. For complex/UI tasks, human verification is necessary.

### Answer to Investigation Question

**Should agents call `bd complete` themselves to declare done?**

No. Agents already own their done declaration via `bd comment "Phase: Complete"`. A `bd complete` command would be syntactic sugar that doesn't change the architecture. The value would be:
- Slightly cleaner agent code: `bd complete <id> --summary "..."` vs `bd comment <id> "Phase: Complete - ..."`
- Standardized completion format (less variation in Phase: Complete comments)
- Potential for beads-level completion hooks

**How should issue lifecycle vs agent lifecycle be modeled?**

They should remain **orthogonal state machines** with explicit mapping events:
- Agent spawning maps to issue in_progress (already true)
- Agent Phase: Complete does NOT map to issue state change (correct)
- Orchestrator verification maps to issue closure (the bottleneck)

**What should `bd complete` mean?**

If implemented, `bd complete` should be a **convenience wrapper** that:
1. Adds "Phase: Complete - {summary}" comment to beads
2. Does NOT close the issue
3. Optionally triggers a webhook/notification to orchestrator

**Where does human verification fit?**

Human verification is part of the **Verification Status** dimension, not Issue Status:
1. `orch complete` runs automated gates (test evidence, build, etc.)
2. For UI changes, `--approve` flag or "APPROVED" comment required
3. After verification passes, issue closes
4. Post-closure, "human verified" badge can be added (for high-confidence confirmation)

**How does this affect `orch complete`?**

`orch complete` should remain the primary completion command. It:
1. Checks Phase: Complete (agent done)
2. Runs verification gates (automated)
3. Requires --approve for UI changes (human gate)
4. Closes beads issue (lifecycle transition)
5. Archives workspace (cleanup)

---

## Structured Uncertainty

**What's tested:**

- ✅ Current completion flow uses `bd comment "Phase: Complete"` then `orch complete` (verified: code analysis of `cmd/orch/complete_cmd.go`)
- ✅ `bd close` already checks for Phase: Complete (verified: `bd close --help` shows `--force` bypasses "Phase: Complete checks")
- ✅ Three orthogonal state dimensions exist (verified: code shows PhaseStatus, VerificationResult, Issue.Status as separate structs)

**What's untested:**

- ⚠️ Daemon auto-verification would reduce completion bottleneck (not implemented)
- ⚠️ `bd done` wrapper command would improve agent DX (not prototyped)
- ⚠️ Current architecture scales to 100+ concurrent agents (load test not performed)

**What would change this:**

- Finding would be wrong if: agents currently have a way to close issues directly that I missed
- Finding would be wrong if: there's an existing daemon completion feature that's disabled
- Finding would be wrong if: Dylan's proposal was about something other than `bd complete` command

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Keep current architecture, add conveniences | architectural | Crosses beads/orch boundary, affects daemon behavior |

### Recommended Approach ⭐

**Keep Current Architecture + Add Conveniences** - Maintain the Phase: Complete → orch complete → bd close chain with minor UX improvements.

**Why this approach:**
- Current architecture correctly separates agent declaration from verification
- Phase: Complete is already the authoritative agent-done signal
- Adding `bd done` wrapper is low-risk UX improvement
- Daemon auto-verification is opt-in enhancement, not architectural change

**Trade-offs accepted:**
- Orchestrator bottleneck remains (verification is inherently expensive)
- Agents still use `bd comment` format (or new `bd done` wrapper)
- No single command combines declaration + verification + closure

**Implementation sequence:**

1. **Add `bd done` convenience command (beads-cli)**
   ```bash
   bd done <id> --summary "Test passed, PR ready"
   # Equivalent to: bd comment <id> "Phase: Complete - Test passed, PR ready"
   ```
   - Why first: Low-risk, improves agent DX, establishes pattern
   - Scope: kb-cli change only

2. **Add verification_status label convention (beads-cli + orch)**
   ```bash
   # After orch complete succeeds:
   bd label add <id> verification:passed
   # After human verification:
   bd label add <id> verification:human_verified
   ```
   - Why second: Enables dashboard to show verification dimension clearly
   - Scope: Convention, minimal code change

3. **Add daemon auto-verification for simple tasks (orch-go)**
   ```go
   // In daemon.go, after detecting Phase: Complete:
   if issue.Labels.Contains("auto-verify") && !issue.HasWebChanges() {
       result := verify.VerifyCompletionLight(beadsID, workspace)
       if result.Passed {
           closeIssue(beadsID, "Auto-verified: " + result.Summary)
       }
   }
   ```
   - Why third: Builds on label convention, opt-in per-issue
   - Scope: Daemon enhancement, requires careful testing

### Alternative Approaches Considered

**Option B: Agent calls `bd complete` which closes issue**
- **Pros:** Simpler mental model, agents "own" their closure
- **Cons:** Bypasses verification gates, removes orchestrator oversight
- **When to use instead:** If we trust agents completely (we don't)

**Option C: Remove orchestrator bottleneck, daemon auto-closes everything**
- **Pros:** Faster completion, no queue buildup
- **Cons:** Quality gates bypassed, UI changes could ship broken
- **When to use instead:** If we add agent-side verification (Phase 2)

**Option D: Add "agent-done" beads status between in_progress and closed**
- **Pros:** Explicit state in beads for Work Graph display
- **Cons:** Schema change in beads, complicates bd commands
- **When to use instead:** If label-based verification_status proves insufficient

**Rationale for recommendation:** The current architecture is sound. The friction is in the orchestrator bottleneck, which is a feature (quality gates) not a bug. The recommendation adds conveniences without changing the fundamental model.

---

### Implementation Details

**What to implement first:**
- `bd done` command in kb-cli (simple wrapper, establishes pattern)
- Update Work Graph UI to show verification dimension clearly (see orch-go-21254, 21252, 21255)

**Things to watch out for:**
- ⚠️ Daemon auto-verification could close issues before human review for complex work
- ⚠️ `bd done` must not allow `--force` or closure - agents shouldn't bypass verification
- ⚠️ Label-based verification_status needs to be kept in sync with actual state

**Areas needing further investigation:**
- What criteria define "simple task" that can be auto-verified?
- Should auto-verification be skill-based (investigation=yes, feature-impl=no)?
- How to handle cross-project completion with daemon auto-verify?

**Success criteria:**
- ✅ Agents can use `bd done <id> --summary "..."` instead of `bd comment "Phase: Complete - ..."`
- ✅ Work Graph shows clear distinction between "agent done" and "verified" states
- ✅ Daemon can optionally auto-verify tasks with `auto-verify` label
- ✅ Orchestrator bottleneck is reduced for simple tasks

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go` - Complete command implementation (1987 lines)
- `pkg/verify/check.go` - Verification gate orchestration (647 lines)
- `pkg/verify/beads_api.go` - Beads API wrapper (738 lines)
- `.kb/models/completion-lifecycle.md` - Current completion model
- `.kb/models/completion-verification.md` - Verification architecture
- `.kb/models/agent-lifecycle-state-model.md` - Agent state model

**Commands Run:**
```bash
# Check beads CLI completion capabilities
bd --help
bd close --help

# Verify Phase: Complete detection
grep -r "Phase: Complete" pkg/verify/
```

**Related Artifacts:**
- **Model:** `.kb/models/completion-lifecycle.md` - Current 4-layer completion chain
- **Model:** `.kb/models/completion-verification.md` - Three-gate verification architecture
- **Model:** `.kb/models/agent-lifecycle-state-model.md` - Four-layer agent state model
- **Investigation:** `.kb/investigations/2026-02-04-arch-work-graph-done-states.md` - Work Graph UI state model
- **Issues:** orch-go-21254, orch-go-21252, orch-go-21255 - Work Graph badge/UI improvements

---

## Investigation History

**2026-02-04 11:45:** Investigation started
- Initial question: Should agents own their done declaration via bd complete?
- Context: Dylan proposed collapsing orchestrator-in-the-middle step

**2026-02-04 12:00:** Found current architecture already separates concerns
- Agent declaration: `bd comment "Phase: Complete"`
- Orchestrator verification: `orch complete` with gates
- Issue closure: `bd close` (called by orch complete)

**2026-02-04 12:30:** Identified three orthogonal state dimensions
- Work Status (agent progress)
- Verification Status (quality gates)
- Issue Status (beads lifecycle)

**2026-02-04 13:00:** Investigation completed
- Status: Complete
- Key outcome: Current architecture is sound; recommend adding `bd done` convenience wrapper and optional daemon auto-verification, not architectural change
