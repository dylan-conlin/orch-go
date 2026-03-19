# Probe: Automatic Account Distribution for Claude CLI Spawns

**Status:** Complete
**Date:** 2026-02-24
**Model:** Orchestration Cost Economics
**Beads:** orch-go-1214

---

## Question

Does the current orch-go spawn architecture support per-spawn account selection for Claude CLI, and what are the integration points for adding tier-aware capacity routing between two Max accounts (work 20x, personal 5x)?

The cost economics model claims "Max subscription via Claude backend is now the default" and "headless swarm = batch execution + rate-limit management across accounts." This probe tests whether the infrastructure for multi-account management actually exists at the spawn level, or if it's only at the global-switch level.

---

## What I Tested

### 1. Claude CLI Account Mechanism
**Command:** Examined `~/.zshrc` for `claude-personal` alias
**Observed:** `alias claude-personal='unset CLAUDE_CODE_OAUTH_TOKEN && CLAUDE_CONFIG_DIR=~/.claude-personal claude'`

This confirms:
- `CLAUDE_CONFIG_DIR` env var selects which config directory Claude CLI uses
- `CLAUDE_CODE_OAUTH_TOKEN` must be unset to prevent it overriding the config dir
- Two config dirs exist: `~/.claude/` (work) and `~/.claude-personal/` (personal)
- Each config dir has its own `.claude.json`, `settings.json`, `history.jsonl`

### 2. Spawn Command Account Logic
**File:** `pkg/spawn/claude.go:57-78` (BuildClaudeLaunchCommand)
**Observed:** Command is `export CLAUDE_CONTEXT=X; cat CONTEXT.md | claude --dangerously-skip-permissions`
- NO CLAUDE_CONFIG_DIR injection
- NO account selection mechanism
- Command inherits tmux window's environment (system default)

### 3. Spawn Resolution System
**File:** `pkg/spawn/resolve.go:42-51` (ResolvedSpawnSettings)
**Observed:** Settings include Backend, Model, Tier, SpawnMode, MCP, Mode, Validation
- NO Account field in ResolvedSpawnSettings
- No account consideration in the resolution pipeline

### 4. Daemon Spawn Path
**File:** `pkg/daemon/issue_adapter.go:348-367` (SpawnWork)
**Observed:** `exec.Command("orch", args...)` with implicit `os.Environ()`
- NO account parameter in SpawnWork signature
- NO CLAUDE_CONFIG_DIR in subprocess environment

### 5. Existing Auto-Switch
**File:** `pkg/account/account.go:819-934` (ShouldAutoSwitch)
**Observed:** Auto-switch checks capacity thresholds and finds best alternate account
- Operates on GLOBAL auth state (`~/.local/share/opencode/auth.json`)
- Not per-spawn - switches the entire system's active account
- Only works for OpenCode backend, NOT Claude CLI backend

### 6. Accounts Configuration
**File:** `~/.orch/accounts.yaml`
**Observed:**
```yaml
accounts:
    personal:
        email: user@example.com
        refresh_token: sk-ant-ort01-...
        source: saved
    work:
        email: user@example.com
        refresh_token: sk-ant-ort01-...
        source: saved
default: personal
```
- No `tier`, `role`, or `config_dir` fields
- No capacity tier information (20x vs 5x)

---

## What I Observed

### Gap Analysis

| Component | Account-Aware? | What's Missing |
|-----------|---------------|----------------|
| accounts.yaml schema | No | tier, role, config_dir fields |
| ResolvedSpawnSettings | No | Account field with provenance |
| BuildClaudeLaunchCommand | No | CLAUDE_CONFIG_DIR env var injection |
| SpawnWork (daemon) | No | Account parameter |
| Auto-switch (account.go) | Partial | Only global switch, not per-spawn |
| User config | No | Account routing preferences |

### Key Discovery: Two Distinct Account Mechanisms

1. **OpenCode OAuth** (existing): `~/.local/share/opencode/auth.json` - global, one active account at a time. Used by OpenCode backend spawns.
2. **Claude CLI config dir** (manual): `CLAUDE_CONFIG_DIR=~/.claude-personal` - per-process, can be set per-spawn. Used by Claude backend spawns.

These are **completely independent paths**. The existing auto-switch operates on mechanism #1 (OpenCode OAuth). The design needs mechanism #2 (Claude CLI config dir) for Claude backend spawns, which are now the default path.

---

## Model Impact

### Confirms
- "Max subscription via Claude backend is now the default" - confirmed, BackendClaude is the default in resolve.go:264
- "Headless swarm = batch execution + rate-limit management across accounts" - confirmed as a decision, but **not yet implemented** for Claude backend. The infrastructure for multi-account batch execution doesn't exist at the spawn level.

### Extends
- The cost economics model should note that **per-spawn account selection requires CLAUDE_CONFIG_DIR injection**, not OpenCode auth.json switching. These are fundamentally different mechanisms for different backends.
- The model's "escalation path" (1. Opus default, 2. account switch, 3. --model flash) currently requires manual `orch account switch` for step 2. Automatic tier-aware routing would make step 2 automatic.

### Contradicts
- The model implies multi-account management is operational ("rate-limit management across accounts"). In reality, only reactive global switching exists (ShouldAutoSwitch), not proactive per-spawn routing. The Claude CLI backend (now default) has NO account distribution mechanism at all.
