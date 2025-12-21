# Decision: Single-Agent Review Command

**Date:** 2025-12-21
**Status:** Proposed
**Decision Maker:** Dylan (orchestrator review needed)

## Context

The orchestrator needs to review individual agent work before completing them. Currently:
- `orch complete <id>` verifies Phase: Complete and closes the beads issue
- `orch review` shows batch completions grouped by project
- No command shows "what did this specific agent do?" in detail

**Gap:** Orchestrator must manually dig through SYNTHESIS.md, beads comments, and git logs to understand what an agent accomplished before deciding to complete it.

## Question

How should we implement the ability to review a single agent's work before completing it?

## Options

### Option A: `--preview` flag on `orch complete` ⭐ RECOMMENDED

Add `--preview` flag that shows comprehensive review before prompting for completion.

```bash
orch complete orch-go-3anf --preview
```

**Output:**
```
AGENT REVIEW: og-feat-implement-port-allocation-21dec
────────────────────────────────────────────────────

Beads: orch-go-lqll.2
Skill: feature-impl  
Duration: 15m (12:20 → 12:35)
Status: Phase: Complete

TLDR: Implemented port allocation registry for orch-go that prevents 
      port conflicts across projects.

DELTA:
  Files:    +2 created, 1 modified
  Commits:  78caa41, 5fbd648
  Tests:    17 passing

BEADS COMMENTS:
  • Phase: Planning - Starting implementation
  • Phase: Complete - All tests passing, ready for review

ARTIFACTS:
  • .kb/investigations/2025-12-21-inv-implement-port-allocation-*.md
  • SYNTHESIS.md (success, rec=close)

Complete this agent? [y/N]: 
```

**Pros:**
- Integrates review into decision point
- One command flow: review → approve → complete
- No additional command to remember
- Can be scripted with `--yes` flag

**Cons:**
- Flag might be overlooked (not default behavior)
- Adds complexity to existing command

### Option B: Separate `orch review <id>` command

New command that only displays review, doesn't complete.

```bash
orch review orch-go-3anf  # Shows review
orch complete orch-go-3anf  # Then complete separately
```

**Pros:**
- Clear separation of concerns
- Discoverable via `orch help`
- Matches mental model of "review" vs "complete"

**Cons:**
- Fragmented workflow (two commands)
- Easy to forget review step
- `orch review` already exists for batch mode

### Option C: Extend `orch review` with optional ID

Make existing `orch review` handle both batch and single modes.

```bash
orch review              # Batch mode (current)
orch review orch-go-3anf # Single agent mode (new)
```

**Pros:**
- No new command
- Reuses existing review concept

**Cons:**
- Confusing: same command, different output formats
- Batch vs single are different mental models
- Clutters existing implementation

## Recommendation

**Option A: `--preview` flag on `orch complete`**

### Rationale

1. **Integration over separation:** The review happens AT the decision point. Orchestrator sees summary and immediately decides. No context switch.

2. **Progressive disclosure:** Default `orch complete` works as before (fast path). `--preview` provides deep dive when needed.

3. **Scriptable:** With `--yes` flag, can be used in automation while still supporting interactive review.

4. **Discoverable via alias:** Add `orch review <id>` as an alias for `orch complete <id> --preview --dry-run` for users who think in terms of "review."

## Implementation Sequence

1. **Create `pkg/verify/review.go`:**
   - `AgentReview` struct (metadata, TLDR, delta, artifacts)
   - `GetAgentReview(beadsID string)` function
   - Git operations (commits since spawn, file stats)

2. **Update `cmd/orch/main.go`:**
   - Add `--preview` flag to completeCmd
   - Add `--yes` flag for non-interactive mode
   - Display logic (reuse printSynthesisCard patterns)

3. **Optional: Add alias command:**
   - `orch review <id>` → `orch complete <id> --preview --dry-run`

## Consequences

**If accepted:**
- Orchestrators can review agent work in one command
- Less friction in completion workflow
- Better visibility into agent deliverables

**Trade-offs:**
- Adds flags to existing command
- Interactive prompt requires stdin

## Decision

[TO BE FILLED BY ORCHESTRATOR]

---

**Related:**
- Investigation: `.kb/investigations/2025-12-21-inv-design-single-agent-review-command.md`
- Existing review command: `cmd/orch/review.go`
- Verify package: `pkg/verify/check.go`
