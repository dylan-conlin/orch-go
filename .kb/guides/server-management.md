# Server Management

**Purpose:** Clarify the boundary between `orch servers` and `orch-dashboard` to eliminate confusion in spawn context and documentation.

**Last verified:** 2026-02-04

---

## Two Tools, Different Purposes

| Aspect | `orch servers` | `orch-dashboard` |
|--------|----------------|------------------|
| **Scope** | ALL projects | orch-go only |
| **Tool** | tmuxinator | overmind |
| **Config** | `.orch/config.yaml` + `~/.tmuxinator/workers-{project}.yml` | `Procfile` |
| **For orch-go** | 2 processes (api, web) | 3 processes (api, web, **opencode**) |
| **Purpose** | Project dev servers | Orchestrator infrastructure |
| **Cleanup** | None | Orphan cleanup, stale socket handling |

---

## When to Use Which

### Use `orch servers` for:
- Any project's development servers (frontend, backend, databases)
- Projects other than orch-go
- Generic "start my dev environment" needs

```bash
# Start servers for any project
orch servers start price-watch
orch servers start beads

# List all projects with server configs
orch servers list

# Open web port in browser
orch servers open price-watch
```

### Use `orch-dashboard` for:
- orch-go ONLY
- When you need OpenCode server running (port 4096)
- When starting the full orchestrator stack

```bash
# Start all orch-go services (kills orphans first)
orch-dashboard start

# Stop all services
orch-dashboard stop

# Full restart with cleanup
orch-dashboard restart

# Check status
orch-dashboard status
```

---

## Why the Separation?

**orch-go is special** - it's the orchestrator itself and needs OpenCode running. Other projects are targets of orchestration.

| Project Type | OpenCode Needed? | Use |
|--------------|------------------|-----|
| orch-go | Yes (it IS the orchestrator) | `orch-dashboard` |
| Other projects | No (targets of orchestration) | `orch servers` |

**Why not unify?** Could add overmind support + OpenCode to `orch servers`, but:
- Increases complexity significantly
- Only one project (orch-go) needs it
- Clear separation is easier to reason about

---

## The Confusion Source

`pkg/spawn/context.go` generates server context for ALL spawns. Previously it told ALL workers to use `orch servers start {project}`, which was wrong for orch-go workers.

**Fixed (Feb 2026):** `GenerateServerContext()` now detects orch-go and recommends `orch-dashboard` instead. Other projects still get `orch servers` guidance.

---

## Agent Constraint

**Agents CANNOT start/stop orch-go services** because:
1. Claude Code runs in a Linux sandbox
2. orch-go services are macOS ARM binaries
3. Agents don't have access to host system management

Workers should NOT try to run `orch-dashboard start` - it must be run from a macOS terminal by the human operator or via automation outside the agent sandbox.

---

## Service Ports

| Service | Port | Started By |
|---------|------|------------|
| OpenCode API | 4096 | `orch-dashboard` only |
| orch serve (API) | 3348 | Both |
| Web UI | 5188 | Both |

---

## Troubleshooting

### "Dashboard not loading"
Check all three services: `lsof -i :4096 -i :3348 -i :5188`

If services are down, run `orch-dashboard start` from macOS terminal.

### "orch servers start orch-go doesn't start OpenCode"
That's expected. `orch servers` uses tmuxinator, which only knows about 2 processes. Use `orch-dashboard` instead.

### "Agent tried to start servers and failed"
Expected. Agents can't manage host services. Document that services need to be running before spawn, or have human start them.

---

## References

- **Investigation:** `.kb/investigations/2026-02-03-inv-orch-go-21218.md`
- **Dashboard guide:** `.kb/guides/dashboard.md`
- **CLAUDE.md:** `## Dashboard Server Management` section
