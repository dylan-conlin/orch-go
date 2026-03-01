# Investigation: Claude Code v2.1.63 Feature Audit for Orchestration System

**Date:** 2026-02-28
**Status:** Complete
**Phase:** Complete
**Beads:** orch-go-y1xm
**Type:** Design Investigation

---

## Design Question

Claude Code v2.1.63 has accumulated significant new capabilities since our spawn infrastructure was built. Which features should we adopt to improve agent orchestration, and in what order?

## Problem Framing

**Success Criteria:** A categorized set of recommendations (adopt now / evaluate / not applicable) with clear implementation paths for "adopt now" items.

**Constraints:**
- Spawning uses Claude CLI via tmux (`cat CONTEXT.md | claude --dangerously-skip-permissions`)
- Must not break existing daemon autonomous processing
- Prior worktree attempt failed (Feb 2026) — approach with caution
- Claude Max subscription, not API keys

**Scope:**
- IN: All CLI flags, hooks, settings, MCP, plugins
- OUT: Claude Desktop features, API-only features, web-only features

---

## Current Spawn Invocation Baseline

```bash
# BuildClaudeLaunchCommand() output (pkg/spawn/claude.go:98):
export ORCH_SPAWNED=1; export CLAUDE_CONTEXT=worker; cat "/path/to/SPAWN_CONTEXT.md" | claude --dangerously-skip-permissions [--mcp-config '...'] [--disallowedTools 'Task,Edit,Write,NotebookEdit']
```

**What we use today:**
- `--dangerously-skip-permissions` (always)
- `--mcp-config` (when MCP preset specified, e.g., playwright)
- `--disallowedTools` (for orchestrator/meta-orchestrator contexts)
- `CLAUDE_CONFIG_DIR` env var (for account isolation)
- `BEADS_DIR` env var (for cross-repo beads access)
- `ORCH_SPAWNED` env var (for hook detection)
- `CLAUDE_CONTEXT` env var (for hook context filtering)

**What we DON'T use:**
- `--worktree` / `-w`
- `--permission-mode`
- `--effort`
- `--model`
- `--settings`
- `--allowedTools`
- `--tools`
- `--agent` / `--agents`
- `--append-system-prompt`
- `--max-turns`
- `--fallback-model`
- `--json-schema`
- `--output-format stream-json`
- `--fork-session`
- Any sandbox settings
- Any new hook events (SubagentStart, SubagentStop, Stop, WorktreeCreate, etc.)

---

## Feature Catalogue & Evaluation

### Category 1: ADOPT NOW (High Value, Low Risk)

#### 1.1 `--permission-mode acceptEdits` instead of `--dangerously-skip-permissions`

**What:** Permission mode flag offers graduated control instead of all-or-nothing bypass.

**Why adopt:** `--dangerously-skip-permissions` is a sledgehammer. `--permission-mode acceptEdits` auto-approves file edits but still prompts for other potentially dangerous operations. However, for autonomous agents, `bypassPermissions` is the equivalent, and for our use case we need full bypass.

**Assessment:** Our current `--dangerously-skip-permissions` is functionally correct for autonomous spawned agents. The `--permission-mode` flag adds no value unless we want partial restriction (e.g., `plan` mode for investigation-only agents).

**Recommendation:** Replace `--dangerously-skip-permissions` with `--permission-mode bypassPermissions` for semantic clarity, but this is cosmetic. **Consider `--permission-mode plan` for investigation/architect skills that shouldn't modify code.**

**Implementation:**
```go
// In BuildClaudeLaunchCommand:
permFlag := " --permission-mode bypassPermissions"
if claudeContext == "investigation" || claudeContext == "architect" {
    permFlag = " --permission-mode plan"  // read-only exploration
}
```

**Scope:** ~10 lines in `pkg/spawn/claude.go`

#### 1.2 `--effort` flag for cost/speed optimization

**What:** Controls Opus 4.6's adaptive reasoning depth (low/medium/high).

**Why adopt:** We spawn agents for wildly different tasks — triage (trivial) vs architecture (complex). Using `--effort low` for triage and daemon preview would significantly reduce cost and latency. Using `--effort high` for architect/feature-impl preserves quality.

**Current behavior:** All agents get the same reasoning depth. No way to signal task complexity.

**Implementation:**
```go
// Map skill tiers to effort levels
effortMap := map[string]string{
    "investigation": "medium",
    "architect":     "high",
    "feature-impl":  "high",
    "triage":        "low",
    "code-review":   "medium",
}
effortFlag := fmt.Sprintf(" --effort %s", effortMap[skill])
```

**Scope:** ~15 lines in `pkg/spawn/claude.go`, ~5 lines in `SpawnConfig`

#### 1.3 `--append-system-prompt` for skill injection

**What:** Appends text to the default system prompt without replacing it.

**Why adopt:** Currently we pipe SPAWN_CONTEXT.md via stdin (`cat CONTEXT.md | claude`). This means Claude's first user message IS the context doc. Using `--append-system-prompt-file` would inject skill context into the system prompt instead, which is architecturally cleaner — the skill guidance becomes system-level instruction rather than a user message.

**Trade-off:** This is a significant change to the spawn flow. The system prompt approach ensures consistent framing across all turns, but our current stdin approach works well. The benefit is primarily architectural cleanliness.

**Assessment:** Evaluate further — the current `cat | claude` pattern works. This would be a nice-to-have refactor but not urgent.

#### 1.4 `--max-turns` for runaway prevention

**What:** Limits the number of agentic turns before stopping.

**Why adopt:** Our stalled agent detection currently relies on tmux liveness checks and time-based heuristics. `--max-turns` provides a hard cap that prevents agents from running indefinitely even if our monitoring misses them.

**Current gap:** Agents with infinite loops or stuck reasoning can burn through entire sessions.

**Implementation:**
```go
// In BuildClaudeLaunchCommand:
maxTurnsFlag := " --max-turns 150"  // reasonable cap for most skills
if tier == "light" {
    maxTurnsFlag = " --max-turns 30"
}
```

**Scope:** ~5 lines in `pkg/spawn/claude.go`

#### 1.5 `--settings` flag for per-spawn hook customization

**What:** Load additional settings from a file or JSON string, merged with defaults.

**Why adopt:** Currently all spawned agents inherit the user's `~/.claude/settings.json`, which includes orchestrator hooks like `gate-orchestrator-code-access.py`. Worker agents don't need (and sometimes conflict with) these hooks. Using `--settings` we could pass worker-specific settings that override or disable certain hooks.

**Implementation:**
```go
// Create .orch/settings/worker-settings.json with worker-specific hooks
settingsFlag := " --settings /Users/dylanconlin/.orch/settings/worker-settings.json"
```

**Scope:** ~3 lines in `pkg/spawn/claude.go`, plus creating the worker settings file

### Category 2: EVALUATE FURTHER (Medium Value, Needs Investigation)

#### 2.1 `--worktree` for agent isolation

**What:** Native git worktree per agent session.

**Status:** Already investigated (`.kb/investigations/2026-02-27-inv-claude-code-worktree-agent-isolation.md`). Prior custom worktree attempt failed catastrophically in Feb 2026. Claude's native `--worktree` handles lifecycle but merge-back problem remains.

**Assessment:** The phased approach from the prior investigation is sound. Wait for Phase 1 (opt-in) implementation before adopting.

**Blocking questions:**
- How does `bd sync` interact with worktree branches?
- How does `orch complete` merge back?
- What happens to concurrent agents modifying the same files?

#### 2.2 `SubagentStart` / `SubagentStop` hooks

**What:** Hook events that fire when Claude spawns/completes subagents within a session.

**Why interesting:** Could provide visibility into Claude's internal agent delegation. Currently we have no idea when Claude spawns internal subagents (Explore, Plan, etc.).

**Assessment:** Limited value for orch-go since we manage agents externally via tmux. More useful if we ever move to Claude's native agent system. Low priority.

#### 2.3 `Stop` hook for completion detection

**What:** Hook event that fires when Claude finishes responding. Can BLOCK stopping (exit 2 continues the conversation).

**Why interesting:** Could replace our tmux-based completion detection. A Stop hook could:
1. Check if the agent reported Phase: Complete
2. If not, block the stop and inject "You haven't reported Phase: Complete"
3. Force agents to follow the completion protocol

**Assessment:** High potential value. Would significantly improve completion reliability. Currently agents sometimes exit without reporting Phase: Complete, requiring manual intervention.

**Implementation sketch:**
```json
{
  "hooks": {
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "$HOME/.orch/hooks/enforce-phase-complete.py"
      }]
    }]
  }
}
```

The hook would check beads for Phase: Complete comment and block stopping if absent.

**Risk:** Could create infinite loops if the agent can't satisfy the condition.

#### 2.4 `TaskCompleted` hook for gate enforcement

**What:** Hook event that fires when a task is marked completed. Can block completion.

**Why interesting:** Could enforce verification gates at the Claude task level rather than at the orch-go level. When Claude's internal task system marks something done, we could validate before allowing it.

**Assessment:** Depends on how our agents use Claude's task system (TaskCreate/TaskUpdate). If they do, this is valuable. Otherwise, no impact.

#### 2.5 `--fallback-model` for graceful degradation

**What:** Automatic model fallback when primary is overloaded (print mode only).

**Why it's Category 2:** Only works in `--print` mode, not interactive mode. Our agents run interactively in tmux. If we ever move to `--print` mode spawning, this becomes immediately valuable.

#### 2.6 `--json-schema` for structured output

**What:** Get validated JSON matching a schema after agent completes (print mode only).

**Why it's Category 2:** Same limitation as fallback — print mode only. Would be extremely valuable for daemon triage (structured decision output) if we switched triage to print mode.

**Potential use:** Triage decisions could return:
```json
{
  "skill": "feature-impl",
  "priority": 2,
  "rationale": "..."
}
```

#### 2.7 `--agents` flag for dynamic subagent definitions

**What:** Define custom subagents via JSON on the command line.

**Why interesting:** Could inject our skill-specific subagent configurations at spawn time rather than relying on `~/.claude/agents/*.md` files.

**Assessment:** Our current approach (injecting everything via SPAWN_CONTEXT.md) works well. The `--agents` flag would be an alternative path but doesn't solve a current problem.

#### 2.8 `--tools` for capability restriction

**What:** Restrict which built-in tools Claude can use.

**Why interesting:** Could enforce strict capability boundaries:
- Investigation agents: `--tools "Read,Grep,Glob,WebFetch,WebSearch"` (no write access)
- Feature-impl agents: `--tools "Bash,Edit,Write,Read,Grep,Glob"` (full access)

**vs. current approach:** We use `--disallowedTools` for orchestrators. `--tools` is an allowlist (opposite approach). The allowlist is more secure (fail-closed).

**Assessment:** Worth adopting for investigation/architect agents. Would provide genuine isolation rather than trust-based "don't modify files" instructions.

#### 2.9 HTTP Hooks for external orchestration

**What:** Hooks can POST to HTTP endpoints instead of running shell scripts.

**Why interesting:** `orch serve` (port 3348) could receive hook events directly from Claude sessions. This would give the dashboard real-time visibility into agent tool usage, permission requests, and lifecycle events without parsing tmux output.

**Assessment:** High potential. Would replace our tmux-based monitoring with proper event-driven architecture. Requires adding webhook endpoints to `orch serve`.

#### 2.10 Prompt/Agent Hooks for LLM-powered gates

**What:** Hooks can delegate decisions to an LLM (prompt) or a subagent (agent).

**Why interesting:** Complex validation gates (like "does this code change violate architectural patterns?") could be evaluated by a fast model rather than pattern-matching scripts.

**Assessment:** Novel capability but adds complexity and cost. Keep on radar.

### Category 3: NOT APPLICABLE (Low Value / No Current Use Case)

#### 3.1 `--chrome` / `--no-chrome`
Browser automation via Chrome extension. We use Playwright MCP for browser testing. No advantage to switching.

#### 3.2 `--from-pr`
Resume sessions linked to PRs. We don't use PR-based workflows for agent sessions.

#### 3.3 `--fork-session`
Branch conversations. No current use case for conversation forking in orchestrated agents.

#### 3.4 `--input-format stream-json`
Streaming JSON input. Only useful for print mode. Our agents run interactively.

#### 3.5 `claude mcp serve`
Run Claude Code as an MCP server. We don't need Claude-as-a-tool — we use Claude as the agent itself.

#### 3.6 Plugin marketplace
Distributing plugins. We use skills via CLAUDE.md and SPAWN_CONTEXT.md. Plugin system is orthogonal.

#### 3.7 `--no-session-persistence`
Disable session saving. We want sessions saved for debugging/monitoring.

#### 3.8 `--file` flag
Doesn't exist — file attachment is via `@` mentions in interactive mode. We already pipe via stdin.

#### 3.9 Sandbox mode
Network/filesystem sandboxing. Our agents need full system access for builds, tests, git operations.

#### 3.10 `ConfigChange` hook
Fires when settings change mid-session. No use case for dynamic config changes during agent runs.

---

## Recommendations Summary

### Adopt Now (create implementation issues)

| # | Feature | Effort | Impact | Risk |
|---|---------|--------|--------|------|
| 1.2 | `--effort` flag | ~15 lines | Cost savings, speed for triage | None (additive flag) |
| 1.4 | `--max-turns` | ~5 lines | Runaway prevention | Low (needs good default) |
| 1.5 | `--settings` for workers | ~10 lines + file | Hook isolation | Low |
| 2.3 | `Stop` hook | ~50 lines (hook script) | Completion reliability | Medium (loop risk) |

### Evaluate Next Sprint

| # | Feature | Why Wait |
|---|---------|----------|
| 2.1 | `--worktree` | Needs phased approach from existing investigation |
| 2.8 | `--tools` allowlist | Needs skill→toolset mapping design |
| 2.9 | HTTP hooks → orch serve | Needs webhook endpoint design |
| 1.1 | `--permission-mode plan` | Needs investigation/architect skill updates |

### Monitor / Backlog

| # | Feature | Trigger |
|---|---------|---------|
| 2.5 | `--fallback-model` | If we adopt `--print` mode spawning |
| 2.6 | `--json-schema` | If daemon triage moves to print mode |
| 2.7 | `--agents` flag | If subagent config becomes painful |
| 2.10 | Prompt/Agent hooks | If script-based gates prove insufficient |

---

## Open Questions for Orchestrator

1. **Effort mapping:** Should effort levels be per-skill or per-tier? Skills are more granular but tiers are simpler.

2. **Stop hook loop risk:** How do we prevent infinite loops when a Stop hook blocks exit? Timer-based escape hatch?

3. **HTTP hooks priority:** Is real-time event visibility (via orch serve webhooks) more important than the items in "Adopt Now"? It's architecturally bigger but would transform monitoring.

4. **Print mode migration:** Several high-value features (fallback-model, json-schema, max-budget-usd) only work in print mode. Should we consider a print-mode spawn path for some use cases?

---

## Evidence Sources

- Claude Code v2.1.63 binary: `claude --version` → `2.1.63 (Claude Code)`
- Claude Code CLI reference: https://code.claude.com/docs/en/cli-usage
- Claude Code hooks reference: https://code.claude.com/docs/en/hooks
- Current spawn code: `pkg/spawn/claude.go:BuildClaudeLaunchCommand()`
- Current settings: `~/.claude/settings.json`
- Prior worktree investigation: `.kb/investigations/2026-02-27-inv-claude-code-worktree-agent-isolation.md`
- Claude Code changelog (GitHub): version timeline from v1.0.0 through v2.1.63
