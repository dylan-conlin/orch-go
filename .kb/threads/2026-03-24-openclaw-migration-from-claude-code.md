---
title: "OpenClaw migration — from Claude Code lock-in to multi-model execution"
status: open
created: 2026-03-24
updated: 2026-03-26
resolved_to: ""
active_work:
resolved_by:
  - "orch-go-8l4h9"
  - "orch-go-h8tcb"
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

## 2026-03-26

We clarified the decision boundary: the lock-in problem is not simply using Claude Code, but coupling worker execution to Anthropic's subscription-bound path. The current recommendation is to keep Claude Code as the orchestrator frontend for now, stop treating OpenCode as the strategic destination, and evaluate OpenClaw as the likely worker execution substrate.

Today's concrete next steps:
- orch-go-1dhv8 benchmarks worker reliability across Claude, Codex/GPT-5.4, and a fallback path on real orch-go tasks
- orch-go-y4k6w designs the OpenClaw worker-backend migration while preserving Claude Code as frontend
- orch-go-btd1g resolves the resulting long-term worker-routing policy

This keeps the methodology/frontend/backend layers distinct so we do not rewrite the system on vibes or keep maintenance-heavy execution plumbing by inertia.

Operational update: Dylan canceled the second Claude Max subscription and subscribed to ChatGPT Pro. The system is no longer provisioned around dual-Claude capacity; it is now provisioned for one Anthropic subscription path plus one OpenAI subscription path.

Implication: the model-routing and worker-backend questions are now more urgent and less hypothetical. Benchmarking Codex/GPT worker reliability is no longer speculative R&D; it is validating an already-funded execution path.

Interim routing policy resolved from the completed benchmark and migration design work: Claude Code/Opus remains the default worker path because it is the only empirically validated route. GPT-5.4/OpenAI stays manual-only until it completes a small real-task benchmark. OpenClaw remains the execution-direction decision, but OpenCode stays temporarily as a bridge until direct pkg/opencode consumers are migrated away from backend-specific session types.

Empirical update from orch-go-1dhv8: GPT-5.4 via Codex OAuth / ChatGPT Pro completed the first real worker benchmark at 80% first-attempt and 100% with retry on N=5 tasks. Feature-impl is now validated as an overflow route. The Anthropic monoculture is no longer mandatory for implementation work.

Caveat: reasoning-heavy work is still under-tested. Investigation showed one transient silent death in two attempts, and GPT-5.4 showed weaker scope control than Opus on at least one task. Default routing remains Claude/Opus; GPT-5.4 is promoted only to feature-impl overflow pending a focused benchmark on investigation/architect/debugging skills.

Follow-up from untracked DAO-13 verification: current SPAWN_CONTEXT sizes are materially smaller than the historical GPT-5.2-era framing used in DAO-13. This matters because prompt-size inflation is no longer the best explanation for GPT-5.4 viability questions; current routing decisions should weight protocol compliance, silent-death frequency, and scope control more heavily than context-window pressure. A cleanup task has been filed to update the shared DAO-13/model wording so future benchmark work inherits the corrected frame.
