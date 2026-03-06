# Workspace & Session Architecture Reference

> **Note:** Reference material for the orchestrator skill.
> The compiled skill contains a summary inline. Consult this for full details.

## Workspace Layout

Each spawned agent gets a workspace at `.orch/workspace/{name}/`:

```
.orch/workspace/{name}/
├── SPAWN_CONTEXT.md       # Full spawn context (skill + task + kb context)
├── AGENT_MANIFEST.json    # Agent metadata (skill, model, tier, issue)
├── SYNTHESIS.md           # Knowledge externalization (full tier only)
├── VERIFICATION_SPEC.yaml # Verification evidence
├── .beads_id              # Beads issue ID
├── .session_id            # OpenCode/Claude session ID
├── .spawn_mode            # Spawn backend (claude/opencode)
├── .spawn_time            # ISO timestamp of spawn
├── .tier                  # Spawn tier (light/full/orchestrator)
├── screenshots/           # Visual verification evidence
├── SESSION_LOG.md         # Exported transcript (on abandon)
└── FAILURE_REPORT.md      # Failure analysis (on abandon with --reason)
```

### Workspace Naming

Format: `{prefix}-{skill-abbrev}-{task-slug}-{date}-{hash}`

Example: `og-feat-add-dark-mode-05mar-e35c`

- `og` = orch-go project prefix
- `feat` = feature-impl skill abbreviation
- `add-dark-mode` = task slug (truncated)
- `05mar` = date
- `e35c` = random 4-char hash

### Workspace Lifecycle

1. **Created** by `orch spawn` — populated with SPAWN_CONTEXT.md, manifest, metadata files
2. **Active** while agent is running — agent adds deliverables, SYNTHESIS.md, etc.
3. **Completed** — agent reports Phase: Complete, workspace stays until `orch complete`
4. **Archived** by `orch complete` — moved to `.orch/workspace/archived/`
5. **Cleaned** by `orch clean --workspaces` — archived workspaces older than TTL (default 30 days)

### AGENT_MANIFEST.json

Contains agent metadata for the orchestrator:

```json
{
  "skill": "feature-impl",
  "model": "opus",
  "tier": "light",
  "beads_id": "orch-go-xyz",
  "spawn_mode": "claude",
  "phases": "implementation,validation",
  "mode": "tdd",
  "validation": "tests",
  "created_at": "2026-03-05T14:30:00Z"
}
```

## Spawn Tiers

| Tier | SYNTHESIS.md | Skills |
|------|-------------|--------|
| **Full** | Required | investigation, architect, research, codebase-audit, systematic-debugging |
| **Light** | Not required | feature-impl, reliability-testing, issue-creation |
| **Orchestrator** | SESSION_HANDOFF.md instead | orchestrator, meta-orchestrator |

Override with `--light` or `--full` flags on `orch spawn`.

## Session Handoff (Orchestrator Sessions)

Orchestrator sessions use a different completion signal:

```
.orch/session/{window-name}/
├── latest/
│   └── SESSION_HANDOFF.md   # Auto-injected on next session start
├── orchestrator/             # Legacy structure
└── orch-go-2/                # Legacy structure
```

**SESSION_HANDOFF.md** captures:
- Session goal and outcome
- Active/completed/blocked agents
- Decisions made
- Open questions
- Next steps

Hooks auto-inject the latest handoff at session start for continuity.

## Cross-Project Spawning

```bash
# Spawn agent in another project
orch spawn --bypass-triage --workdir ~/other-repo feature-impl "task"

# Complete cross-project agent
orch complete other-repo-123 --workdir ~/other-repo

# Abandon cross-project agent
orch abandon other-repo-123 --workdir ~/other-repo
```

Cross-project agents:
- Workspace lives in the target project's `.orch/workspace/`
- Beads issue created in target project's `.beads/`
- Use `BEADS_DIR=~/path/.beads bd close/update/list` for cross-project beads operations

## Archived Workspaces

Location: `.orch/workspace/archived/{name}/`

Archived workspaces are immutable records of completed agent work. They contain all original workspace files plus any deliverables the agent produced.

Cleanup: `orch clean --workspaces` archives workspaces older than `--workspace-days` (default 7). `--archived-ttl` controls how long archived workspaces persist (default 30 days).
