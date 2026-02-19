# Probe: Daemon-Spawned Agents Stall on GPT-5.2-Codex via OpenCode Backend

**Model:** daemon-autonomous-operation
**Date:** 2026-02-19
**Status:** Complete

---

## Question

The daemon model claims spawn failures don't release slots (Capacity Starvation) and duplicate spawns happen during poll latency. But a new failure mode emerged: 3/3 daemon-spawned agents stalled in early phases after using GPT-5.2-codex model via opencode backend. Is this a model-specific issue, a config resolution bug, or an opencode backend issue?

---

## What I Tested

### 1. Traced the daemon model resolution path

```bash
# User config
cat ~/.orch/config.yaml
# Result: default_model: codex, backend: opencode

# Project config
cat .orch/config.yaml
# Result: spawn_mode: opencode, opencode.model: flash

# Model alias lookup
grep "codex" pkg/model/model.go
# Result: "codex" → {Provider: "openai", ModelID: "gpt-5.2-codex"}
```

### 2. Analyzed the config precedence in runWork()

In `cmd/orch/spawn_cmd.go:429-436`, `runWork()` loads user config `default_model` into the `spawnModel` package-level var (same var as `--model` CLI flag). This enters the resolve pipeline as `CLI.Model`, overriding all lower-priority settings including the project config's `opencode.model: flash`.

### 3. Read all 3 stalled agent session logs

- `.orch/workspace/og-feat-workspace-lookup-historical-19feb-2166/SESSION_LOG.md` (orch-go-1098)
- `.orch/workspace/og-feat-getallsessionstatus-fetches-sessions-19feb-72fd/SESSION_LOG.md` (orch-go-1099)
- `.orch/workspace/og-feat-architect-extract-daemon-19feb-2b34/SESSION_LOG.md` (orch-go-1092)

### 4. Measured SPAWN_CONTEXT sizes

```bash
wc -c .orch/workspace/og-feat-*/SPAWN_CONTEXT.md .orch/workspace/og-feat-arch*/SPAWN_CONTEXT.md
# 70732  og-feat-getallsessionstatus-fetches (orch-go-1099)
# 76318  og-feat-workspace-lookup-historical (orch-go-1098)
# 63314  og-feat-architect-extract-daemon (orch-go-1092)
```

---

## What I Observed

### Agent Behavior Breakdown

**Session 1098 (workspace-lookup-historical, 43K tokens, 30 seconds):**
- Read SPAWN_CONTEXT, reported Phase: Planning
- Then **hallucinated a constraint**: "Orchestrator policy in this session forbids reading code files or implementing changes"
- Self-blocked with BLOCKED status after just 3 tool calls
- **This constraint does not exist** in SPAWN_CONTEXT or any loaded context
- GPT-5.2 confused the "orchestrator" discussion in context with a session-level restriction

**Session 1099 (getallsessionstatus-fetches, 41K tokens, ~3 minutes):**
- Actually productive: read files, searched, applied 4 patches across files
- Modified opencode source (session.ts), orch-go client.go, serve_sessions.go, serve_agents_handlers.go
- Session ends abruptly after last apply_patch — no error message, no completion report
- May have hit API rate limit or context budget after heavy tool use

**Session 1092 (architect-extract-daemon, 145K tokens, ~4 minutes):**
- Extensive exploration: read 20+ files across Go and TypeScript
- 145K tokens consumed — approaching GPT-5.2 context window limits
- Session ends mid-exploration with an apply_patch call
- Likely hit context window exhaustion

### Config Resolution Chain

```
User config: default_model: codex
  → runWork() sets spawnModel = "codex"
    → Resolve pipeline: CLI.Model = "codex"
      → model.ResolveWithConfig("codex") = openai/gpt-5.2-codex
        → modelBackendRequirement: openai → requires opencode
          → Backend: opencode (derived)
            → Project config opencode.model: flash is BYPASSED
```

**The project config explicitly sets `opencode.model: flash` but this is never applied** because the user config `default_model` enters as CLI-level priority.

### Key Findings

1. **GPT-5.2-codex does not reliably follow the worker agent protocol**:
   - Hallucinated non-existent constraints (session 1098)
   - Consumed excessive tokens on exploration without completing tasks (session 1092)
   - Failed to complete the session close protocol on any agent

2. **Config precedence bug**: `runWork()` elevates user config `default_model` to CLI-flag priority, which makes project-level model overrides unreachable for daemon spawns.

3. **No early detection**: The daemon has no mechanism to detect stalled agents in early phases (hallucinated BLOCKED, context exhaustion). The recovery mechanism only checks idle time (10 minutes).

---

## Model Impact

- [ ] **Confirms** invariant: Capacity Starvation — spawn failures don't release slots. (These stalled sessions DO consume slots because the sessions were created successfully; the agents just didn't complete.)
- [x] **Extends** model with: New failure mode — "Model Incompatibility Stall". When the daemon uses a non-Claude model (GPT-5.2-codex) via opencode backend, agents fail in multiple ways:
  1. Hallucinated constraints (self-BLOCKED without real cause)
  2. Excessive token consumption on exploration without task completion
  3. Silent session termination on context window exhaustion
  4. The spawn counts as successful (session exists) but produces no useful output
- [x] **Extends** model with: Config resolution bug — `runWork()` injects `default_model` as CLI-level priority, bypassing project config's `opencode.model` override.

---

## Notes

### Recommendations

1. **Immediate**: Change user config `default_model` to a Claude model, or remove it and let project config control per-backend model selection
2. **Config fix**: In `runWork()`, don't set `spawnModel` from user config — let it flow through the resolve pipeline where project config `opencode.model` has appropriate priority
3. **Health check**: After spawn, verify agent reported Phase: Planning within 2 minutes. If not, flag as potentially stalled.
4. **Model suitability gate**: Block daemon spawning with models that don't support the worker protocol (GPT models with 63-76KB spawn contexts)
5. **Daemon-specific model config**: Add `daemon.model` in user config to decouple daemon model selection from interactive default

### SPAWN_CONTEXT size concern

All 3 agents received 63-76KB spawn contexts. This is ~20-25K tokens for Claude but closer to 40-50K tokens for GPT tokenizers. Combined with CLAUDE.md, skill instructions, and tool definitions, the initial context could consume 60-80% of GPT-5.2's context window before the agent even starts working.
