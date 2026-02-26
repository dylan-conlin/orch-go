# Decision: Orchestrator Lifecycle Without Beads Tracking

**Date:** 2026-01-05
**Status:** Archived (never implemented)
**Context:** Establishing how orchestrator sessions should be tracked without using beads issue lifecycle
**Archived:** 2026-02-26
**Archive Reason:** Never implemented. Orchestrators continue using beads tracking. The session registry (`~/.orch/sessions.json`) was never built. Current system works adequately with beads for all agent types.

## Decision

### The Semantic Mismatch

**Beads tracks work items** with a spawn→task→complete lifecycle.
**Orchestrator sessions have conversations** with a start→interact→end lifecycle.

These are fundamentally different things. Dylan's key insight:
> "Orchestrators aren't issues being worked on - they're interactive sessions with Dylan. Beads is for tracking work items, not collaborative sessions."

### What We're Changing

| Aspect | Before | After |
|--------|--------|-------|
| Beads issue on spawn | Created (then ignored) | Not created |
| Phase reporting | Skipped (already) | Skipped |
| Session identity | Workspace name | Workspace name (no change) |
| Completion detection | SESSION_HANDOFF.md | SESSION_HANDOFF.md (no change) |
| Status visibility | Via beads ID in window | Via session registry |

### Implementation: Workspace-Based Session Registry

Create a lightweight `~/.orch/sessions.json` registry for active orchestrator sessions:

```json
{
  "sessions": [
    {
      "workspace_name": "og-orch-ship-feature-05jan",
      "session_id": "ses_abc123...",
      "project_dir": "/Users/dylan/projects/orch-go",
      "spawn_time": "2026-01-05T10:30:00Z",
      "status": "active"
    }
  ]
}
```

### The Four Requirements

1. **Identify for `orch complete`**: Workspace name (already works)
2. **Show in `orch status`**: Read from session registry
3. **Know when ready**: Check SESSION_HANDOFF.md exists
4. **Transcript export**: tmux window by workspace name

### Why Not Keep Beads?

- **Vestigial usage**: Issue is created but never updated (no phase comments)
- **Wrong abstraction**: Issues have priority, dependencies, assignees - sessions don't need these
- **Semantic confusion**: "Closing" an issue vs "ending" a session are different concepts
- **Code complexity**: Verification code already has orchestrator-specific paths

### Why Session Registry Over Alternatives?

| Alternative | Pros | Cons | Verdict |
|-------------|------|------|---------|
| Workspace scanning | Simple, no state file | O(n) on every status | Too slow |
| Extend beads with session type | Unified | Semantic mismatch remains | Forcing fit |
| tmux as registry | Already exists | Only tmux mode | Headless breaks |
| **Session registry** | Fast, simple, cache | New file | ⭐ Chosen |

## Implementation Phases

### Phase 1: Session Registry (pkg/session/registry.go)
- JSON file at `~/.orch/sessions.json`
- Lock file for concurrent access
- CRUD operations: Register, Update, Unregister, List

### Phase 2: Spawn Updates (pkg/spawn/)
- Skip beads issue creation when `IsOrchestrator=true`
- Register session in registry instead

### Phase 3: Status Updates (cmd/orch/status_cmd.go)
- Add orchestrator sessions from registry to status output
- Show with different indicator (not "Active" since no beads)

### Phase 4: Complete Updates (cmd/orch/complete_cmd.go)
- Unregister from registry on complete
- Preserve transcript export (already workspace-based)

## Consequences

### Positive
- Cleaner model: sessions are sessions, issues are issues
- Faster status: O(1) registry lookup vs O(n) workspace scan
- Less code: Remove vestigial beads issue creation
- Better UX: No confusing "open issue" for completed sessions

### Negative
- New state file to manage
- Potential for stale entries (orphan detection needed)
- Behavioral change for existing orchestrator spawns

### Risks
- **Stale sessions**: Mitigated by status checking session liveness
- **Lock contention**: Unlikely (orchestrators spawn rarely)
- **Migration**: Old workspaces still have .beads_id files

## Related

- **Supersedes:** Implicit "orchestrators get beads issues" behavior
- **Related:** `.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md`
- **Investigation:** `.kb/investigations/2026-01-05-inv-design-orchestrator-session-lifecycle-without.md`
