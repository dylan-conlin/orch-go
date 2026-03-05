# Tools & Commands Reference

> **Note:** This is reference material extracted from the orchestrator skill.
> The compiled skill contains the most-used commands inline.
> Consult this file for tool ecosystem details, config locations, and daemon internals.

## Tool Ecosystem

```
orch        → agent coordination     (spawn, monitor, complete, daemon)
beads (bd)  → what work needs doing  (issues, dependencies, tracking)
kb          → knowledge management   (investigations, decisions, quick entries)
skillc      → skill compilation      (modular skills → SKILL.md)
opencode    → agent execution        (Claude frontend, session management)
```

**Cross-repo architecture:** See `~/.orch/ECOSYSTEM.md`

## Search Tool Selection

| Question | Tool |
|----------|------|
| "What do we know about X?" | `kb context "X"` |
| "Find all mentions of X in .kb/" | `kb search "X"` or Grep |
| "Find X in code" | Grep on `pkg/` `cmd/` |

## Config Locations

- Orch: `~/.orch/config.yaml` | Accounts: `~/.orch/accounts.yaml`
- Daemon: `~/Library/LaunchAgents/com.orch.daemon.plist`
- OpenCode: `{project}/opencode.json`
- Plugins: `.opencode/plugin/` (project) or `~/.config/opencode/plugin/` (global)

## Daemon Behavior That Changes Orchestration Timing

- **30s grace period before spawn:** when an issue is first seen with `triage:ready`, daemon records first-seen time and waits `GracePeriod` (default `30s`) before it becomes spawnable.
- **ProcessedIssueCache prevents duplicate spawns:** daemon checks `~/.orch/processed-issues.jsonl`, active OpenCode sessions, and `Phase: Complete` comments before spawning; issues are marked before spawn and unmarked on spawn failure.
- **Idle sessions auto-expire from capacity gates:** stale idle agents age out of active-count filters (1h in spawn concurrency checks), so ghosts stop blocking new work.
- **Concurrency cap 5, round-robin fairness:** daemon spawns max 5 agents, alternating between projects at same priority level. Focus-aware: focused project gets priority boost.
- **Self-check invariants:** daemon pauses spawning when invariant violations exceed threshold (e.g., agents > cap, active count unreachable). Resumes after violations clear.
- **Auto-complete for auto-tier agents:** capture-knowledge agents are auto-completed by daemon when they report Phase: Complete. No `orch complete` needed.
- **Stuck detection with notifications:** agents running >2h with no phase updates trigger desktop notification. Orchestrator receives STUCK signal.

## Attention Signals To Act On

- **UNBLOCKED:** dependencies resolved; issue is actionable again.
- **STUCK:** runtime >2h with low/no activity; intervene.
- **VERIFY FAILED:** auto-complete verification failed; rerun `orch complete <id>`.

## System Maintenance

**Skill editing:** Edit `src/SKILL.md.template` or `.skillc/` files, then `skillc build`. Never edit `SKILL.md` directly (auto-generated, will be overwritten).

## Daemon Operations

```bash
launchctl list | grep orch                               # Status
launchctl kickstart -k gui/$(id -u)/com.orch.daemon      # Restart
tail -f ~/.orch/daemon.log                               # Logs
```
