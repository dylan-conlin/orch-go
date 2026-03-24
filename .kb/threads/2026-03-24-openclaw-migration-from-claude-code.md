---
title: "OpenClaw migration — from Claude Code lock-in to multi-model execution"
status: open
created: 2026-03-24
updated: 2026-03-24
resolved_to: ""
---

# OpenClaw migration — from Claude Code lock-in to multi-model execution

## 2026-03-24

### Problem

orch-go is locked to Anthropic: Claude Max ($200/mo) -> Claude Code CLI -> Opus 4.6. GPT-5.4 is now the top model and available at flat rate via ChatGPT Pro OAuth through any client. Anthropic explicitly bans Max subscription usage outside Claude Code. OpenAI explicitly allows Pro subscription in third-party tools.

OpenClaw (250K stars, 267 commits/day, NVIDIA/Tencent integrating) is consolidating the agentic platform layer. It has multi-backend support (Claude CLI + Codex CLI), 114+ WebSocket RPC methods for programmatic control, headless gateway mode, and a plugin SDK that maps cleanly to orch-go's 4 coordination primitives.

### Decision (2026-03-24)

Migrate worker execution to OpenClaw. Keep Claude Code CLI as orchestrator frontend (decision to replace frontend left open).

### Migration path

1. Drop OpenCode fork + opencode backend (~3,600 LoC removed, fork is 975 commits behind, dead weight)
2. GPT-5.4 stall rate test — gate before anything else. ~5 lines to enable (model alias + Codex whitelist)
3. Add OpenClaw as spawn backend (~300 LoC WebSocket client). Gives multi-model + replaces fragile SSE with agent.wait
4. Keep claude CLI backend as fallback
5. Eventually: package coordination primitives as OpenClaw plugin (distribution to 250K users)

### Architecture

```
Dylan <-> Claude Code CLI (orchestrator conversation, hooks, skills)
              |
         orch-go daemon (spawn, route, throttle, complete)
              |
         OpenClaw gateway (headless, WebSocket RPC)
           /        \
     claude -p    codex exec
     (Opus 4.6)   (GPT-5.4)
```

Frontend replacement left open — if OpenClaw's UI matures or a better coding TUI emerges, the orchestrator conversation could move there too. The OpenClaw WebSocket client should not assume Claude Code stays as frontend.

### Evidence base

- 5 investigations (2026-03-23/24): OpenClaw platform, plugin SDK, external API, GPT-5.4 routing, OpenCode fork necessity
- OpenClaw plugin SDK maps all 4 primitives: before_tool_call (Route), before_prompt_build (Align), subagent_spawning (Throttle), registerService (Sequence)
- OpenCode fork: 975 commits behind, 32 custom cherry-picks, only 2/9 API integrations need fork features, all on secondary backend
- GPT-5.4: ~5 lines to enable, flat rate via ChatGPT Pro OAuth, stall rate untested

### What this means for orch-go

orch-go's value is the methodology layer (skills, coordination primitives, knowledge system, probes, beads). The ~30K lines of execution plumbing are being replaced by OpenClaw. The methodology is fully portable. The plumbing was scaffolding.
