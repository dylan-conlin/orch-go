# Completion Workflow Guide

**Purpose:** Single authoritative reference for the agent completion system.

**Synthesized from:** 28 investigations (Dec 19, 2025 - Jan 17, 2026)

**Last updated:** Jan 17, 2026

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

## Resource Cleanup (Four-Layer Model)

Agent state exists in four layers that must be cleaned up on completion:

| Layer | What to Clean | When | Command/Action |
|-------|---------------|------|----------------|
| **Beads** | Close issue | `orch complete` | `bd close <id>` |
| **OpenCode Session** | Delete session | `orch complete` | `client.DeleteSession()` |
| **Tmux Window** | Close window | `orch complete` | `tmux kill-window` |
| **Workspace** | Archive to archived/ | `orch complete` | `os.Rename()` |

### Cleanup Order (Critical)

```
1. Verify completion (all gates pass)
2. Close beads issue
3. Delete OpenCode session
4. Export transcript (if orchestrator)
5. Archive workspace to archived/
6. Close tmux window
7. Invalidate serve cache
```

**Why order matters:**
- Beads closure is the authoritative signal for "done"
- OpenCode session must be deleted BEFORE status checks (prevents "ghost agents")
- Transcript export reads from workspace, so archive must come after
- Tmux cleanup last (window may be needed for debugging)

### Ghost Agent Prevention

**Symptom:** `orch status --all` shows completed agents as "running"

**Root cause:** `orch complete` closed beads issue but didn't delete OpenCode session. Status filters by session age (30 min), so recently-completed agents persist.

**Fix:** `orch complete` now calls `client.DeleteSession(sessionID)` after closing beads issue.

**Code reference:** `cmd/orch/complete_cmd.go:565-580`

---

## Workspace Lifecycle

### Completion Operations

`orch complete` performs:
- Phase verification (all gates)
- Beads issue closure
- OpenCode session deletion (prevents ghost agents)
- Tmux window cleanup
- Workspace archival (moved to `archived/`)
- Auto-rebuild if Go changes
- Serve cache invalidation

### Automated Archival

As of Jan 17, 2026, `orch complete` automatically archives workspaces:

```bash
# After completion:
.orch/workspace/og-feat-xyz/  →  .orch/workspace/archived/og-feat-xyz/
```

**Name collision handling:** If archived/ already contains that workspace name, appends timestamp suffix:
```
archived/og-feat-xyz/  →  archived/og-feat-xyz-1737123456/
```

**Opt-out:** `orch complete <id> --no-archive`

**Registry update:** `ArchivedPath` field tracks where workspace was archived.

### Manual Stale Cleanup

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

---

## Session Handoff Updates

When completing worker agents, `orch complete` triggers session handoff updates for the orchestrator:

### What Gets Captured

| Section | Source | Prompt |
|---------|--------|--------|
| **Spawns Table** | Agent metadata | Outcome, key finding |
| **Evidence** | Agent observation | Pattern observation (optional) |
| **Knowledge** | Agent learning | Decision/constraint (optional) |

### Capture Flow

```
orch complete <id>
       │
       ▼
┌──────────────────────────────────────┐
│ Prompt: "Outcome? (success/partial)" │
│ Prompt: "Key finding (1 line)?"      │
└──────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│ Prompt: "Pattern observation?"       │  ← Optional (Enter to skip)
│ Prompt: "Decision/constraint?"       │  ← Optional (Enter to skip)
└──────────────────────────────────────┘
       │
       ▼
   Updates SESSION_HANDOFF.md
```

### Why Capture at Completion

**Principle:** Capture knowledge at the moment of context, not later when context is lost.

- Agent just finished → orchestrator has fresh context
- Synthesis recommendations are visible → easy to extract
- Session handoff is open → natural place to record

**Code reference:** `cmd/orch/session.go:1736-2002`

---

## Daemon Auto-Completion

The daemon can auto-complete agents that report Phase: Complete, governed by the escalation model.

### When Daemon Auto-Completes

| Escalation Level | Action | Example |
|------------------|--------|---------|
| `None` | Auto-complete silently | Clean code-only work |
| `Info` | Auto-complete, log for review | Has minor recommendations |
| `Review` | Auto-complete, queue mandatory review | Knowledge-producing skills |
| `Block` | Do NOT auto-complete | Needs visual approval |
| `Failed` | Do NOT auto-complete | Verification failed |

### Expected Distribution

- ~60% auto-complete silently (None/Info levels)
- ~30% auto-complete with review flag (Review level)
- ~10% require human decision (Block/Failed levels)

### Integration with Run Loop

```
Daemon Poll Cycle:
1. Reconcile with OpenCode (free stale slots)
2. Process completions (CompletionOnce)  ← NEW
3. Run periodic reflection
4. Write daemon status
5. Check capacity
6. Spawn new agents
```

**Code reference:** `pkg/daemon/completion.go`

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

## Investigations Synthesized (28 total)

This guide synthesizes 28 investigations on agent completion from Dec 2025 - Jan 2026:

### Core Implementation
- CLI orch complete command (2025-12-19)
- Orch complete closes beads issue (2025-12-21)
- Agents being marked completed registry (2025-12-21)
- Reconciliation should check completed work (2025-12-21)
- Implement orch complete preview update (2025-12-21)

### Verification & Gates
- Add liveness warning orch complete (2025-12-22)
- Pre-spawn phase complete check (2025-12-25)
- Orch complete prompt follow-up (2025-12-25)
- Debug orch complete force sets close (2025-12-26)
- Improve orch complete cross-project (2025-12-26)

### Escalation Model
- Completion escalation model completed agents (2025-12-27)

### Daemon Integration
- Daemon capacity stale after complete (2026-01-02)
- Test spawned agents complete work (2026-01-03)
- Agents report phase complete via (2026-01-03)
- Debug orch complete fails orchestrator sessions (2026-01-05)
- Daemon auto complete agents report (2026-01-06)

### Dashboard/Status
- Dashboard distinguish completed agents active (2026-01-06)
- Orch status shows completed agents (2026-01-06)

### Event Tracking
- Add agent completed event emission (2026-01-08)

### Session Cleanup
- Bug orch complete doesn set (2026-01-09) - Ghost agents fix

### Progressive Capture
- Orch complete triggers handoff updates (2026-01-14)

### Archival
- Persist activity feed completed agents (2026-01-16, 2026-01-17)
- Implement automated archival orch complete (2026-01-17)

### Knowledge Preservation
- What knowledge context lives completed (2025-12-22)
- Fix archive section sort completed (2025-12-24)
- Orchestrator skill says complete agents (2025-12-24)

---

## See Also

- `.kb/guides/daemon.md` - Daemon polling and batch completion
- `.kb/guides/cli.md` - CLI command reference
- `pkg/verify/` - Verification implementation
