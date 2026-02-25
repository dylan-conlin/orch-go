# Design: Automatic Account Distribution for Claude CLI Spawns

**Date:** 2026-02-24
**Type:** Architect Design
**Beads:** orch-go-1214
**Phase:** Complete

---

## Design Question

How should orch-go automatically distribute Claude CLI spawns across two Max accounts (work 20x $200/mo, personal 5x $100/mo) with tier-aware capacity routing?

## Problem Framing

### Current State
- Two Claude Max accounts: **work** (20x, sendcutsend email) and **personal** (5x, gmail)
- Manual switching via `claude-personal` alias: `unset CLAUDE_CODE_OAUTH_TOKEN && CLAUDE_CONFIG_DIR=~/.claude-personal claude`
- Two separate config directories: `~/.claude/` (work) and `~/.claude-personal/` (personal)
- accounts.yaml has both accounts but no tier/role/config_dir fields
- ResolvedSpawnSettings has no Account field
- BuildClaudeLaunchCommand injects no account env vars
- SpawnWork (daemon) passes no account parameter

### Desired State
- Daemon and `orch spawn` auto-select account based on capacity
- Work absorbs bulk (4x more capacity), personal is spillover
- Each Claude CLI subprocess gets the correct CLAUDE_CONFIG_DIR
- Interactive sessions get guidance on which account to use
- Account selection has provenance tracking (like other resolved settings)

### Success Criteria
1. Spawns use work account by default (20x capacity = primary)
2. When work is rate-limited, spawns automatically fail over to personal
3. Each Claude CLI process runs with correct config directory isolation
4. Account selection is visible in spawn output and dashboard
5. No global state mutation (per-spawn env vars, not auth.json switching)
6. Interactive sessions show recommended account

### Constraints
- CLAUDE_CONFIG_DIR + unset CLAUDE_CODE_OAUTH_TOKEN is the proven mechanism
- accounts.yaml is the source of truth for accounts
- Must not violate Anthropic ToS (this is Dylan's own accounts, not sharing)
- Hotspot areas (daemon, spawn, orch) need minimal surgical changes
- Existing auto-switch (account.go:ShouldAutoSwitch) operates on OpenCode auth.json — different mechanism from CLAUDE_CONFIG_DIR

### Scope
- **IN:** accounts.yaml schema, account selection algorithm, env var injection, interactive guidance
- **OUT:** Dashboard changes, cost tracking, cross-account session handoff, new accounts

---

## Exploration: Decision Forks

### Fork 1: Where to store account tier information?

**Options:**
- **A: Add fields to accounts.yaml** — tier, role, config_dir fields alongside existing email/token
- **B: Derive from capacity API** — query usage endpoint to determine tier
- **C: Separate routing config** — new accounts-routing.yaml

**SUBSTRATE:**
- Principle: "Local-First" — files over databases, accounts.yaml is already the local file
- Principle: "Session Amnesia" — config must be discoverable in standard locations
- Model: Cost Economics — tier (20x vs 5x) is a subscription property, changes rarely
- Decision: "Dashboard account name lookup uses email reverse-mapping from accounts.yaml"

**RECOMMENDATION:** Option A — add fields to accounts.yaml

The tier is a static subscription property, not dynamic. Querying capacity API (B) adds latency (OAuth token refresh) at every spawn and conflates "what tier am I" with "how much is left." Separate config (C) splits related data unnecessarily.

**Trade-off accepted:** Manual config update when subscription tier changes (rare — maybe yearly)
**When this would change:** If subscription tiers become dynamic or if there are 10+ accounts requiring auto-discovery

---

### Fork 2: Account selection algorithm?

**Options:**
- **A: Work-first, personal-spillover** — simple priority with health check
- **B: Capacity-weighted routing** — proportional distribution based on remaining capacity
- **C: Round-robin** — alternate between accounts with health check

**SUBSTRATE:**
- Decision: "Headless swarm = batch execution + rate-limit management across accounts"
- Model: Cost Economics — work has 4x more capacity (20x vs 5x)
- Task: "Work absorbs bulk, personal is spillover when work is rate-limited"
- Principle: "Compose Over Monolith" — simple, focused algorithm

**RECOMMENDATION:** Option A — work-first with spillover

The algorithm:
```
1. Check work account capacity (cached, 5-min TTL)
2. If work is healthy (>20% remaining on both limits): use work
3. If work is low (<20% on either limit): check personal
4. If personal is healthy: use personal
5. If both limited: use work (still has more headroom with 20x)
6. If capacity check fails: use work (fail-open to primary)
```

**Trade-off accepted:** Personal account idles when work is healthy (wastes potential capacity)
**When this would change:** If there are 5+ accounts where proportional routing would better utilize capacity

---

### Fork 3: Where does account selection happen in the architecture?

**Options:**
- **A: In spawn resolution** — add Account to ResolvedSpawnSettings via resolveAccount()
- **B: In launch command** — inject at BuildClaudeLaunchCommand level
- **C: In daemon** — select before calling SpawnWork

**SUBSTRATE:**
- Architecture: ResolvedSpawnSettings has provenance tracking (Source, Detail) for all settings
- Pattern: resolveBackend, resolveModel, resolveTier follow identical precedence cascade
- Principle: "Compose Over Monolith" — account is a spawn-time decision like model/backend
- Hotspot warning: minimal changes to each file

**RECOMMENDATION:** Option A — add to spawn resolution

New `resolveAccount()` function follows the same precedence pattern:
1. CLI flag (`--account work`)
2. User config (`default_account: work`)
3. Heuristic (capacity-aware routing: work-first, personal-spillover)
4. Default (work)

The resolved account flows through to BuildClaudeLaunchCommand as the config_dir path. This gives full provenance tracking ("account: work, source: heuristic, detail: work-healthy-87%").

**Trade-off accepted:** Adds another field to ResolvedSpawnSettings (growing struct)
**When this would change:** If account selection becomes backend-specific (different algorithms for Claude vs OpenCode)

---

### Fork 4: How to inject account identity to Claude CLI subprocess?

**Options:**
- **A: CLAUDE_CONFIG_DIR env var** — per-process, matches existing alias
- **B: CLAUDE_CODE_OAUTH_TOKEN** — direct token injection
- **C: Global auth switch** — mutate ~/.local/share/opencode/auth.json before spawn

**SUBSTRATE:**
- Evidence: `alias claude-personal='unset CLAUDE_CODE_OAUTH_TOKEN && CLAUDE_CONFIG_DIR=~/.claude-personal claude'` — proven mechanism
- Principle: "No Local Agent State" — don't maintain global mutable state
- Architecture: BuildClaudeLaunchCommand already sets CLAUDE_CONTEXT env var

**RECOMMENDATION:** Option A — CLAUDE_CONFIG_DIR env var injection

Modify BuildClaudeLaunchCommand to prepend:
```bash
unset CLAUDE_CODE_OAUTH_TOKEN; export CLAUDE_CONFIG_DIR=~/.claude-personal; export CLAUDE_CONTEXT=worker; cat CONTEXT.md | claude --dangerously-skip-permissions
```

For the default work account (config_dir = `~/.claude`), no env var needed since that's the default.

**Trade-off accepted:** Each account needs a pre-configured config directory (one-time setup)
**When this would change:** If Claude CLI adds a `--config-dir` flag or `--account` flag

---

### Fork 5: When does capacity check happen?

**Options:**
- **A: Live check at every spawn** — call GetAccountCapacity() per spawn
- **B: Cached with 5-minute TTL** — periodic refresh
- **C: No capacity check** — always work-first, fail-over on error

**SUBSTRATE:**
- Model (Cost Economics): GetAccountCapacity() rotates OAuth tokens as side effect
- Investigation: "Auto-Switch Account Failing Silently" — token mismatch bugs from frequent rotation
- Constraint: Daemon polls every 60s, may spawn 3+ agents per cycle
- Principle: "No Local Agent State" — but this is a cache, not a registry

**RECOMMENDATION:** Option B — cached with 5-minute TTL

Token rotation on every spawn (Option A) risks the token mismatch bug documented in the auto-switch investigation. No check (Option C) means personal never gets used. 5-minute TTL is sufficient — the 5-hour rate window is the finest granularity that matters.

Implementation: `AccountCapacityCache` struct in `pkg/account/` with `GetCapacity(name string) (*CapacityInfo, error)` that returns cached result if fresh, or fetches + caches if stale.

**Trade-off accepted:** Up to 5 minutes stale data (may send 1-2 spawns to a just-exhausted account)
**When this would change:** If spawn rate exceeds 10/minute where staleness matters more

---

### Fork 6: Interactive session guidance?

**Options:**
- **A: New `orch account recommend` command** — dedicated recommendation command
- **B: Extend `orch account list`** — add recommended indicator to existing command
- **C: Document alias pattern** — just update guides

**SUBSTRATE:**
- User Interaction Model: "Dylan does NOT type CLI commands directly" — commands are for orchestrator
- Architecture: `orch account list` already shows accounts with capacity
- Principle: "Surfacing Over Browsing" — bring recommendation to the agent

**RECOMMENDATION:** Option B — extend `orch account list` with recommendation indicator

Add `[RECOMMENDED]` tag and capacity summary to `orch account list` output. Orchestrators already use this command to check accounts. The recommendation follows the same work-first algorithm. For interactive sessions, the orchestrator can say: "Launch `claude` for work account (recommended, 87% remaining) or `claude-personal` for personal."

Also: the spawn command itself should log which account was selected, so the orchestrator sees it in spawn output.

**Trade-off accepted:** No new command to maintain (keeps existing surface area)
**When this would change:** If recommendation logic becomes complex enough to warrant its own command

---

## Synthesis: Design Specification

### 1. accounts.yaml Schema Changes

```yaml
accounts:
    work:
        email: dylan.conlin@sendcutsend.com
        refresh_token: sk-ant-ort01-...
        source: saved
        tier: 20x           # NEW: subscription tier (5x, 20x)
        role: primary        # NEW: routing role (primary, spillover)
        config_dir: ~/.claude  # NEW: Claude CLI config directory
    personal:
        email: dylan.conlin@gmail.com
        refresh_token: sk-ant-ort01-...
        source: saved
        tier: 5x
        role: spillover
        config_dir: ~/.claude-personal
default: personal  # Note: this is the OpenCode default, separate from spawn routing
```

**Backward compatibility:** Accounts without `tier`/`role`/`config_dir` default to: tier="" (unknown), role="" (eligible for selection), config_dir="" (system default ~/.claude). The algorithm treats accounts without role as "primary" candidates.

### 2. Account Selection Algorithm (resolveAccount)

```
resolveAccount(input ResolveInput) -> ResolvedSetting

Precedence:
1. CLI flag: --account work                    → Source: cli-flag
2. User config: default_account: work          → Source: user-config
3. Heuristic: capacity-aware routing           → Source: heuristic
4. Default: first account with role=primary    → Source: default

Heuristic (capacity-aware routing):
  primary = accounts where role == "primary" (or role == "" for backward compat)
  spillover = accounts where role == "spillover"

  for each primary account (sorted by tier desc):
    capacity = cache.GetCapacity(name)
    if capacity.IsHealthy():  // >20% remaining on both limits
      return account, "primary-healthy"

  for each spillover account:
    capacity = cache.GetCapacity(name)
    if capacity.IsHealthy():
      return account, "spillover-activated"

  // All exhausted: use highest-tier primary (still has most headroom)
  return highestTierPrimary, "all-exhausted-using-primary"
```

**Only applies to Claude backend.** When backend is OpenCode, account selection is N/A (OpenCode uses its own auth.json). The resolver should return empty account for non-Claude backends.

### 3. Env Var Injection (BuildClaudeLaunchCommand)

Current:
```go
func BuildClaudeLaunchCommand(contextPath, claudeContext, mcp string) string {
    return fmt.Sprintf("export CLAUDE_CONTEXT=%s; cat %q | claude --dangerously-skip-permissions%s%s",
        claudeContext, contextPath, mcpFlag, disallowFlag)
}
```

New signature:
```go
func BuildClaudeLaunchCommand(contextPath, claudeContext, mcp, configDir string) string
```

When `configDir` is non-empty and differs from default (`~/.claude`):
```bash
unset CLAUDE_CODE_OAUTH_TOKEN; export CLAUDE_CONFIG_DIR=/path/to/config; export CLAUDE_CONTEXT=worker; cat CONTEXT.md | claude --dangerously-skip-permissions
```

When `configDir` is empty or `~/.claude` (the default):
```bash
export CLAUDE_CONTEXT=worker; cat CONTEXT.md | claude --dangerously-skip-permissions
```

### 4. Daemon Integration (SpawnWork)

Current:
```go
func SpawnWork(beadsID, model, workdir string) error
```

New (extends via resolved settings flow):
- SpawnWork itself doesn't change. Account selection happens in `orch work` → spawn resolution → BuildClaudeLaunchCommand.
- The `orch work` command already calls `Resolve()` which will now include `resolveAccount()`.
- Resolved account flows through Config → SpawnClaude → BuildClaudeLaunchCommand → CLAUDE_CONFIG_DIR.

Alternative: If we want explicit daemon control, add `--account` flag to `orch work`:
```go
func SpawnWork(beadsID, model, workdir, account string) error
```
But this is unnecessary if the resolver handles it automatically.

### 5. Interactive Session Guidance

Extend `orch account list` output:
```
  NAME       EMAIL                          TIER   ROLE       5H-REMAINING  7D-REMAINING  STATUS
  work       dylan.conlin@sendcutsend.com   20x    primary    87%           72%           [RECOMMENDED]
  personal   dylan.conlin@gmail.com         5x     spillover  95%           88%           available
```

When orchestrator needs to guide interactive sessions:
- "Use `claude` (work account, recommended)" or "Use `claude-personal` if work is rate-limited"
- The `orch spawn --inline` path should log which account was auto-selected

### 6. Capacity Cache

```go
// pkg/account/cache.go
type CapacityCache struct {
    mu      sync.Mutex
    entries map[string]*cacheEntry
    ttl     time.Duration
}

type cacheEntry struct {
    capacity  *CapacityInfo
    fetchedAt time.Time
}

func NewCapacityCache(ttl time.Duration) *CapacityCache
func (c *CapacityCache) Get(name string) (*CapacityInfo, error)
func (c *CapacityCache) Invalidate(name string)
```

Default TTL: 5 minutes. The daemon creates one cache instance at startup.

---

## File Targets

| File | Change | Lines |
|------|--------|-------|
| `pkg/account/account.go` | Add Tier, Role, ConfigDir fields to Account struct | ~10 |
| `pkg/account/cache.go` | **NEW** — CapacityCache with TTL | ~60 |
| `pkg/spawn/resolve.go` | Add Account to ResolvedSpawnSettings, resolveAccount() | ~50 |
| `pkg/spawn/claude.go` | Add configDir param to BuildClaudeLaunchCommand | ~10 |
| `pkg/spawn/config.go` | Add Account field to Config struct | ~5 |
| `cmd/orch/account_cmd.go` | Extend list output with recommendation | ~20 |
| `cmd/orch/spawn_cmd.go` | Add --account flag, pass through | ~10 |
| Tests | account_test, resolve_test, claude_test | ~100 |

**Total estimated:** ~265 lines across 7 files + tests

---

## Acceptance Criteria

1. `orch spawn --account personal architect "task"` forces personal account
2. `orch spawn architect "task"` auto-selects work (when healthy)
3. When work is at >80% 5-hour usage, auto-selects personal (if healthy)
4. Spawn output shows: `Account: work (source: heuristic, detail: primary-healthy-87%)`
5. `orch account list` shows `[RECOMMENDED]` tag
6. Claude CLI subprocess has correct CLAUDE_CONFIG_DIR env var
7. Daemon spawns auto-route without configuration beyond accounts.yaml
8. Backward compatible: existing accounts.yaml without new fields works (defaults to primary, system config dir)

## Out of Scope

- Dashboard UI changes for account display
- Per-spawn cost tracking
- Cross-account session handoff (agent started on work, continued on personal)
- OpenCode backend account routing (different mechanism entirely)
- Automatic accounts.yaml population (user configures tier/role/config_dir once)

---

## Recommendations

⭐ **RECOMMENDED:** Phased implementation

**Phase 1: Schema + CLI flag** (~2 hours)
- Add tier/role/config_dir to Account struct and accounts.yaml
- Add --account flag to spawn command
- Add resolveAccount() to resolver with CLI source only
- Modify BuildClaudeLaunchCommand to accept and inject configDir
- Tests for all new code

**Phase 2: Capacity routing** (~2 hours)
- Implement CapacityCache
- Add heuristic routing (work-first, personal-spillover) to resolveAccount()
- Wire cache into daemon
- Tests for routing algorithm

**Phase 3: Polish** (~1 hour)
- Extend `orch account list` with recommendation
- Add spawn output logging for account selection
- Update guides

**Why phased:** Phase 1 gives immediate value (explicit `--account` flag) with minimal risk. Phase 2 adds the automatic routing. Phase 3 adds discoverability.

**Alternative: Big-bang implementation**
- **Pros:** All features at once
- **Cons:** Harder to test, review, and debug; higher risk
- **When to choose:** If the phased approach creates too much overhead

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the manual account switching pain point
- Future spawns might need to know about account routing
- Account management changes should consult this design

**Suggested blocks keywords:**
- account distribution
- CLAUDE_CONFIG_DIR
- account routing
- capacity routing
- work personal account
