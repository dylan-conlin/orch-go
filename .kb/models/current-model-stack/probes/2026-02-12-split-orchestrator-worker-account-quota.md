# Probe: Can orchestrator use personal Max 5x while workers use work Max 20x?

**Model:** .kb/models/current-model-stack.md
**Date:** 2026-02-12
**Status:** Complete

---

## Question

Can Dylan run the orchestrator on personal Max 5x account while spawning workers that use work Max 20x account, effectively splitting orchestrator quota (cheaper, better value/dollar) from worker quota (higher capacity)?

---

## What I Tested

**1. Does `orch spawn --account` already exist?**

```bash
grep -n "account" cmd/orch/spawn_cmd.go
# Line 30: spawnAccount string
# Line 197: spawnCmd.Flags().StringVar(&spawnAccount, "account", "", "Claude account name...")
```

**2. How does --account flow through the spawn pipeline?**

Traced code path: `spawn_pipeline.go:137` calls `maybeSwitchSpawnAccount(spawnAccount, resolvedModel)` → `spawn_account_isolation.go:19` → calls `account.SwitchAccount()` which writes to `~/.local/share/opencode/auth.json`.

For Claude CLI backend: `spawn_pipeline.go:632` calls `resolveSpawnClaudeConfigDir(spawnAccount, usageCheckResult)` → returns `~/.claude-{accountName}`, set on `spawn.Config.ClaudeConfigDir`. Then `spawn/claude.go:77` exports `CLAUDE_CONFIG_DIR={value}` in the tmux command.

**3. Ran existing tests to verify account isolation works:**

```bash
go test ./cmd/orch/ -run TestMaybeSwitchSpawnAccount -v
# PASS: TestMaybeSwitchSpawnAccountAnthropicModel
# PASS: TestMaybeSwitchSpawnAccountNonAnthropicModelIgnored
# PASS: TestMaybeSwitchSpawnAccountReturnsSwitchError

go test ./cmd/orch/ -run TestResolveSpawnClaudeConfigDir -v
# PASS: TestResolveSpawnClaudeConfigDirExplicitAccount → ~/.claude-work
# PASS: TestResolveSpawnClaudeConfigDirAutoSwitchNonPrimary → ~/.claude-work
# PASS: TestResolveSpawnClaudeConfigDirAutoSwitchPrimarySkipsIsolation → ""

go test ./pkg/spawn/ -run TestSpawnClaude -v
# PASS: TestSpawnClaudeIncludesClaudeConfigDirInLaunchCommand
# PASS: TestSpawnClaudeOmitsClaudeConfigDirWhenUnset
# PASS: TestSpawnClaudeInlineSetsClaudeConfigDirEnv
```

**4. Verified OpenCode auth architecture:**

Read `packages/opencode/src/auth/index.ts` — `Auth.get()` reads `auth.json` from disk on every call (no in-memory cache). Read `packages/opencode/src/provider/provider.ts:697` — provider state is cached per `Instance.state()` using `State.create()`, which caches per instance directory. New sessions on new instances read fresh auth.

**5. Checked current accounts.yaml:**

```bash
cat ~/.orch/accounts.yaml
# Only "personal" account saved. "work" not yet added.
```

**Environment:**
- Branch: master
- accounts.yaml: only personal (dylan.conlin@gmail.com) saved
- Config: backend=opencode, default_model=openai/gpt-5.3-codex

---

## What I Observed

**Three distinct auth paths with different isolation properties:**

| Backend | Auth Mechanism | Per-Spawn Isolation | Status |
|---------|---------------|-------------------|--------|
| **Claude CLI** (tmux/inline) | Keychain scoped by `CLAUDE_CONFIG_DIR` hash | YES — each process gets own env var | Working, tested |
| **OpenCode** (headless, Claude model) | `auth.json` — global file, read per instance | PARTIAL — switch before spawn works, but shared state | Working, race risk |
| **OpenCode** (headless, GPT/DeepSeek) | Separate `openai`/API key entries in auth.json | N/A — different auth, not affected | Irrelevant to probe |

**The viable split-quota pattern:**

```
Orchestrator (Claude Code CLI)
├── CLAUDE_CONFIG_DIR=~/.claude-personal → personal keychain → Max 5x
│
├── Default workers (GPT-5.3-Codex via OpenCode)
│   └── Uses openai auth entry → ChatGPT Pro → unaffected by account switching
│
├── Claude workers via Claude CLI (PREFERRED for split quota)
│   └── orch spawn --backend claude --account work investigation "task"
│       → CLAUDE_CONFIG_DIR=~/.claude-work → work keychain → Max 20x ✅
│
└── Claude workers via OpenCode headless (WORKS but has caveats)
    └── orch spawn --account work --model opus investigation "task"
        → Switches auth.json globally → new sessions use work tokens
        → ⚠️ Race: concurrent spawns may read stale cached provider state
```

**Key findings:**

1. `--account` flag already exists and is wired through both backends
2. Claude CLI backend has clean per-process isolation via `CLAUDE_CONFIG_DIR`
3. OpenCode backend has global auth.json — works for sequential spawns, race risk for concurrent
4. Default workers (GPT-5.3-Codex) use completely separate auth — no conflict
5. `accounts.yaml` only has personal account — work account needs `orch account add work`

**What's needed to enable this:**

1. `orch account add work` — save work account refresh token
2. `claude-personal` alias already exists — orchestrator uses personal
3. Workers: `orch spawn --backend claude --account work` for Claude workers (clean path)
4. Or configure default: `spawn_account: work` in `.orch/config.yaml` (doesn't exist yet, but easy)

**Potential config shape for automatic split:**

```yaml
# .orch/config.yaml addition
accounts:
  orchestrator: personal    # Which account the orchestrator runs on
  workers: work             # Default account for Claude workers
  # Per-backend overrides
  claude_workers: work      # --backend claude workers use this
  opencode_workers: ""      # OpenCode workers inherit server auth
```

---

## Model Impact

**Verdict:** extends — Multi-Account Access section

**Details:**
The model documents `CLAUDE_CONFIG_DIR` isolation as the solution to cross-account rate limit contamination. This probe extends that finding: the isolation mechanism already supports split orchestrator/worker quota via the existing `--account` flag + `CLAUDE_CONFIG_DIR`. The Claude CLI backend provides clean per-spawn isolation today. The recommended pattern is: orchestrator on personal Max 5x (via `claude-personal` alias), Claude workers on work Max 20x (via `orch spawn --backend claude --account work`). Default GPT-5.3-Codex workers are unaffected since they use separate OpenAI auth. The only gap is: work account isn't in `accounts.yaml` yet, and there's no config-level default for worker account (would need explicit `--account work` per spawn or a new config key).

**Confidence:** High — Code paths verified via test execution and source tracing. Auth isolation via CLAUDE_CONFIG_DIR is proven (3 passing tests). The only untested aspect is an end-to-end spawn with both accounts simultaneously (blocked by work account not being in accounts.yaml).
