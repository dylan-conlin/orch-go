# Probe: Rework Loop Design for Verification Gaps

**Date:** 2026-02-17
**Status:** Complete
**Model:** Completion Verification (`completion-verification.md`)

## Question

The completion verification model documents how gates catch incomplete work, but has no mechanism for sending work back when gaps are found. How should `orch rework` close this loop — and does the existing verification architecture support or constrain the design?

## Claims Being Tested

1. **Escalation model has no rework path** — EscalationBlock and EscalationFailed stop completion but don't trigger rework. The orchestrator must manually spawn a new agent.
2. **Workspace archival destroys rework context** — `archiveWorkspace()` moves the workspace to `archived/`, making it invisible to normal workspace lookups.
3. **Beads closure is canonical** — Reopening a closed issue should be possible but isn't used anywhere in the current codebase.
4. **Agent lifecycle has no "rework" state** — The four-state lifecycle (Spawn → Execute → Complete → Archive) has no backward path.

## What I Tested

### Claim 1: Escalation model has no rework path
- Reviewed `pkg/verify/check.go` and `cmd/orch/complete_cmd.go`
- The escalation levels are: None, Info, Review, Block, Failed
- When Block/Failed fires, `orch complete` refuses to close — but the only options are:
  - `--force` to override
  - Manual intervention (spawning a new agent)
- **No automated rework trigger exists anywhere in the codebase**

### Claim 2: Workspace archival destroys rework context
- `archiveWorkspace()` in `complete_cmd.go:1739` moves workspace to `archived/{name}`
- After archival, `findWorkspaceByBeadsID()` won't find it (only scans active workspaces)
- SYNTHESIS.md, SPAWN_CONTEXT.md, and all evidence is preserved but requires knowing the archived path
- The .beads_id file is preserved in the archived workspace, enabling reverse lookup

### Claim 3: Beads closure is canonical
- `beads.FallbackUpdate(id, "open")` exists and can reopen issues
- No code in orch-go currently calls this — all flows are one-directional (open → closed)
- `bd update <id> --status=open` works at the CLI level
- Beads comments would show the full history: original Phase: Complete, then rework

### Claim 4: Agent lifecycle has no backward path
- Confirmed: the lifecycle is strictly forward (Spawn → Execute → Complete → Archive)
- `orch resume` is for paused agents (same session, still alive)
- No command exists for "work was completed but wrong, try again"

## What I Observed

The completion verification model correctly identifies verification failures but has a **dead-end** after the Block/Failed escalation. The orchestrator currently works around this by:
1. Reading the SYNTHESIS.md manually
2. Crafting a new spawn with verbose rework instructions in the task description
3. Using `--issue` flag to link to the same beads issue (if not closed) or creating a new issue
4. Losing the connection between original and rework attempts

This is lossy because:
- The new agent doesn't automatically get the prior SYNTHESIS.md
- The rework count isn't tracked
- There's no event for "agent.reworked" in the event system
- Beads history doesn't show the rework relationship

## Model Impact

**Extends the model** in the following way:

The Completion Verification model should include a **Rework Path** as a sixth escalation outcome:
```
EscalationNone     → Auto-complete silently
EscalationInfo     → Auto-complete, log for review
EscalationReview   → Auto-complete, queue mandatory review
EscalationBlock    → Do NOT auto-complete, surface immediately
EscalationFailed   → Do NOT auto-complete, failure state
NEW → EscalationRework → Reopen issue, spawn new agent with feedback context
```

The `orch rework` command operationalizes the manual workaround the orchestrator already performs, making it:
- Traceable (event logging)
- Context-preserving (prior SYNTHESIS.md included)
- Measurable (rework count, rework rate metrics)
- Connected (same beads issue, linked workspaces)

## Design Recommendation

See investigation: `.kb/investigations/2026-02-17-design-orch-rework-command.md`
