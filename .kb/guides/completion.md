# Completion Workflow Guide

**Purpose:** Single authoritative reference for the agent completion system.

**Synthesized from:** 10 investigations (Dec 19, 2025 - Jan 7, 2026)

---

## Quick Reference

```bash
# Complete a single agent
orch complete <beads-id>

# Complete with UI approval (for web/ changes)
orch complete <beads-id> --approve

# Complete cross-project agent
orch complete <beads-id> --workdir ~/path/to/project

# Review batch completions
orch review
orch review -p <project>
orch review done <project>

# Clean stale workspaces
orch clean --stale
orch clean --stale --dry-run
```

---

## System Evolution

The completion system evolved through 4 phases:

| Phase | Date | Key Additions |
|-------|------|---------------|
| 1. Notification | Dec 2025 | Desktop notifications via beeep; session context in notifications |
| 2. Verification | Dec 2025 | Three-layer gates; `--approve` flag; 5-tier escalation model |
| 3. Cross-Project | Dec 2025 | Auto-detect from SPAWN_CONTEXT.md; `--workdir` fallback |
| 4. Metrics | Jan 2026 | Orchestrator verification path; completion rate segmentation |

---

## Verification Architecture

### Three Verification Layers

| Layer | Purpose | Blocks Completion? |
|-------|---------|-------------------|
| **Phase Gate** | Agent reported "Phase: Complete" | Yes |
| **Evidence Gate** | Visual/test evidence exists in beads comments | Yes |
| **Approval Gate** | Human explicitly approved (UI work only) | Yes (UI changes) |

### Verification by Skill Type

| Skill Type | Verification Path | Auto-Complete? |
|------------|-------------------|----------------|
| Code-only (feature-impl, systematic-debugging) | Standard (SYNTHESIS.md) | If all gates pass |
| Knowledge-producing (investigation, architect, research) | Standard + Review | Always surfaces |
| Orchestrator (orchestrator, meta-orchestrator) | SESSION_HANDOFF.md | N/A (interactive) |

### UI Verification Two-Layer Pattern

For agents that modify `web/` files:

1. **Evidence Layer:** Agent reports screenshot/browser verification in beads comments
2. **Approval Layer:** Human must explicitly approve via:
   - `orch complete <id> --approve` (single command)
   - `bd comments add <id> "APPROVED"` then `orch complete <id>`

**Why:** Agents can claim visual verification without actually doing it. Human approval gate prevents "agent renders wrong -> thinks done -> human discovers wrong" problem.

**Code reference:** `pkg/verify/visual.go` - `humanApprovalPatterns`, `HasHumanApproval()`

---

## Escalation Model

### Five-Tier Escalation

```
EscalationNone   → Auto-complete silently
EscalationInfo   → Auto-complete, log for optional review
EscalationReview → Auto-complete, queue for mandatory review
EscalationBlock  → Do NOT auto-complete, surface immediately
EscalationFailed → Do NOT auto-complete, failure state
```

### Escalation Decision Tree

```
1. VERIFICATION FAILED? → EscalationFailed
2. SKILL IS KNOWLEDGE-PRODUCING? 
   └── Has NextActions or Recommendation != "close"? → EscalationReview
   └── Otherwise → EscalationInfo
3. VISUAL VERIFICATION NEEDS APPROVAL? → EscalationBlock
4. OUTCOME != "success"? → EscalationReview
5. HAS RECOMMENDATIONS? (NextActions > 0)
   └── file_count > 10? → EscalationReview
   └── Otherwise → EscalationInfo
6. LARGE SCOPE? (file_count > 10) → EscalationInfo
7. OTHERWISE → EscalationNone
```

**Design reference:** `.kb/investigations/2025-12-27-inv-completion-escalation-model.md`

**Implementation status:** Design complete; may not be fully implemented in daemon.

---

## Cross-Project Completion

### Auto-Detection Flow

When running `orch complete <beads-id>` from a different project:

1. Find workspace by beads ID in current project's `.orch/workspace/`
2. Extract `PROJECT_DIR` from `SPAWN_CONTEXT.md`
3. If PROJECT_DIR differs from cwd, auto-set `beads.DefaultDir`
4. All beads operations use the resolved project directory

### Manual Override

```bash
# If workspace not found, use explicit --workdir
orch complete kb-cli-xyz --workdir ~/path/to/kb-cli
```

### Pattern Consistency

The `--workdir` pattern is consistent across:
- `orch spawn --workdir`
- `orch abandon --workdir`
- `orch complete --workdir`

**Design reference:** `.kb/investigations/2025-12-27-inv-design-cross-project-completion-ux.md`

---

## Orchestrator Verification

Orchestrator-type skills use a separate verification path:

| Worker Skills | Orchestrator Skills |
|---------------|---------------------|
| SYNTHESIS.md required | SESSION_HANDOFF.md required |
| Phase gates via beads comments | Session end markers |
| Visual/test evidence checks | Skipped |
| Beads issue tracking | No beads tracking |

### Session End Markers

Valid markers in SESSION_HANDOFF.md:
- `## Session Summary`
- `## Handoff`
- `## Next Steps`
- `**Status:** Complete`
- Or substantial content (>100 chars)

**Code reference:** `pkg/verify/check.go` - `verifySessionEndedProperly()`

---

## Completion Rate Metrics

### Understanding the Metrics

The raw completion rate can be misleading due to:

1. **Coordination skills** (orchestrator, meta-orchestrator) - Interactive sessions, not completable tasks
2. **Test spawns** - Ad-hoc/untracked spawns pollute metrics
3. **Rate limiting** - 14-21% of abandonments are rate-limit-related

### Healthy Metrics

| Category | Expected Rate |
|----------|---------------|
| **Tracked task work** | ~80% |
| **Raw overall** | ~66-68% (misleading) |
| **Coordination skills** | N/A (wrong metric) |

### Stats Segmentation

When analyzing completion rates:
- Filter untracked spawns (beads_id contains "untracked")
- Exclude coordination skills from task completion rate
- Track rate-limiting as a separate abandonment category

**Diagnostic reference:** `.kb/investigations/2026-01-06-inv-diagnose-overall-66-completion-rate.md`

---

## Workspace Lifecycle

### Completion vs Archival

`orch complete` performs:
- Phase verification
- Beads issue closure
- Tmux window cleanup
- Auto-rebuild if Go changes

`orch complete` does NOT:
- Archive workspace directory
- Clean up OpenCode disk sessions

### Stale Workspace Cleanup

```bash
# Preview what would be archived
orch clean --stale --dry-run

# Archive stale workspaces (>7 days old)
orch clean --stale
```

### Archival Criteria

A workspace is stale if:
- `.spawn_time` file is older than 7 days
- Has completion indicators (SYNTHESIS.md, light tier, .beads_id)

**Recommendation:** Add auto-archive to `orch complete` workflow or integrate into daemon poll cycle.

**Reference:** `.kb/investigations/2026-01-07-inv-address-340-active-workspaces-completion.md`

---

## Notification System

### Desktop Notifications

Completion notifications via `beeep` library:
- Triggered when agent reports Phase: Complete
- Includes workspace name (session title)
- Abstracted via `pkg/notify` package

### Notification Content

```
Title: "Agent Complete"
Body: "{workspace-name} completed successfully"
```

**Code reference:** `pkg/notify/notify.go` - `SessionComplete()`

---

## Common Workflows

### Single Agent Completion

```bash
# Standard completion
orch complete <beads-id>

# With UI approval
orch complete <beads-id> --approve

# Cross-project
orch complete <beads-id> --workdir ~/other/project
```

### Batch Review

```bash
# After overnight daemon run
orch review                    # See all pending
orch review -p <project>       # Filter by project
orch review --needs-review     # Failures only
orch review done <project>     # Mark batch reviewed
```

### Verification Failures

When completion fails:
1. Check `orch status` for agent state
2. Review beads comments: `bd show <id>`
3. Check verification output in `orch complete` output
4. Address specific gate failure:
   - Missing Phase: Complete → Agent needs to report phase
   - Missing evidence → Agent needs to report test/visual evidence
   - Missing approval → Use `--approve` flag

---

## Key Code References

| Component | File | Purpose |
|-----------|------|---------|
| Main verification | `pkg/verify/check.go` | `VerifyCompletionFull()`, `VerifyCompletionWithTier()` |
| Visual verification | `pkg/verify/visual.go` | `HasWebChanges()`, `HasHumanApproval()` |
| Complete command | `cmd/orch/complete_cmd.go` | `runComplete()` |
| Review command | `cmd/orch/review.go` | `getCompletionsForReview()` |
| Notification | `pkg/notify/notify.go` | `SessionComplete()`, `BeeepBackend` |

---

## Investigations (Archived)

The following investigations were synthesized into this guide:

| Date | Investigation | Status |
|------|---------------|--------|
| 2025-12-19 | Desktop Notifications Completion | Archived |
| 2025-12-26 | UI Completion Gate - Require Screenshot | Archived |
| 2025-12-27 | Implement Cross-Project Completion | Archived |
| 2026-01-04 | Phase Completion Verification Orchestrator Spawns | Archived |
| 2026-01-04 | Test Completion Works 04jan | Archived |
| 2026-01-04 | Test Completion Works Say Hello | Archived |

### Investigations (Active Reference)

| Date | Investigation | Why Kept |
|------|---------------|----------|
| 2025-12-27 | Completion Escalation Model | Design reference, implementation status unclear |
| 2025-12-27 | Cross-Project Completion UX Design | Design options reference |
| 2026-01-06 | Diagnose Overall 66% Completion Rate | Recent diagnostic, pending actions |
| 2026-01-07 | Address 340 Active Workspaces | Recent, auto-archive recommendation pending |

---

## See Also

- `.kb/guides/daemon.md` - Daemon polling and batch completion
- `.kb/guides/cli.md` - CLI command reference
- `pkg/verify/` - Verification implementation
