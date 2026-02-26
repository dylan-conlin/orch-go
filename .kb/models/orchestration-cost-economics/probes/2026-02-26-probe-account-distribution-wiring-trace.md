# Probe: Account Distribution Heuristic Wiring Trace

**Model:** Orchestration Cost Economics
**Date:** 2026-02-26
**Status:** Complete
**Beads:** orch-go-1111

---

## Question

Is the automatic account distribution system (designed Feb 24, implemented Feb 20-21) actually wired end-to-end through the spawn flow? The prior probe (2026-02-24) found zero integration. The cost economics model now claims "per-spawn account distribution enables capacity-aware routing across multiple Max accounts via CLAUDE_CONFIG_DIR injection, with the Account field tracked as a first-class resolved setting with provenance." Is this true in the current code?

---

## What I Tested

### 1. Spawn Command Entry Point (cmd/orch/spawn_cmd.go)

**Tested:** Traced `runSpawnWithSkillInternal()` lines 477-667 for account resolution calls.

**Observed (lines 561-588):** The resolve pipeline is called at step 5 with:
```go
resolveInput := spawn.ResolveInput{
    CLI: spawn.CLISettings{
        Account: spawnAccount,  // --account flag (line 569)
    },
    CapacityFetcher: buildCapacityFetcher(),  // line 587
}
```

Then at lines 614-616:
```go
resolvedAccountName := resolved.Settings.Account.Value
resolvedAccountConfigDir := account.GetConfigDir(resolvedAccountName)
```

And injected into SpawnContext at lines 637-638:
```go
Account:          resolvedAccountName,
AccountConfigDir: resolvedAccountConfigDir,
```

**Verdict:** Fully wired. The `--account` CLI flag (line 190) and `buildCapacityFetcher()` are both passed to the resolver, and the resolved account + config dir flow into SpawnContext.

### 2. Account Resolution Logic (pkg/spawn/resolve.go)

**Tested:** Read `resolveAccount()` function (lines 429-505).

**Observed:**
- Precedence: CLI flag → heuristic (with CapacityFetcher) → default (first primary)
- Account field exists in `ResolvedSpawnSettings` (line 83)
- `resolveAccount()` is called from `Resolve()` at line 242
- Heuristic: checks primaries first (sorted), then spillovers, using >20% health threshold
- Handles: nil CapacityFetcher (fall back to default), capacity fetch failures (fail-open to primary)
- 10 tests cover the heuristic logic: CLI flag, default empty, work-first, spillover activation, both-exhausted, capacity-fail, CLI override, no-fetcher, 7-day-low, capacity-error. All pass.

### 3. Capacity Cache (pkg/account/cache.go + cmd/orch/shared.go)

**Tested:** Read `buildCapacityFetcher()` (shared.go:420-464) and `CapacityCache` (cache.go).

**Observed:**
- `buildCapacityFetcher()` returns nil if <2 accounts or no roles configured → falls through to default account
- If roles exist: creates process-level `CapacityCache` (5-min TTL), returns closure that checks cache → API → cache-set
- Called from `spawn_cmd.go:587` and `rework_cmd.go:206`
- `accounts.yaml` has work(primary, 20x, ~/.claude) and personal(spillover, 5x, ~/.claude-personal) — roles ARE configured

### 4. Claude CLI Backend (pkg/spawn/claude.go)

**Tested:** Read `BuildClaudeLaunchCommand()` (lines 59-88) and `SpawnClaude()` (lines 92-145).

**Observed:**
- `BuildClaudeLaunchCommand` injects `CLAUDE_CONFIG_DIR` when configDir is set and not `~/.claude`:
  ```go
  if configDir != "" && configDir != "~/.claude" {
      accountPrefix = fmt.Sprintf("unset CLAUDE_CODE_OAUTH_TOKEN; export CLAUDE_CONFIG_DIR=%s; ", configDir)
  }
  ```
- `SpawnClaude` passes `cfg.AccountConfigDir` to `BuildClaudeLaunchCommand` at line 130
- Tests verify: non-default configDir injects env var, default/empty does NOT inject — all pass

**Key observation:** `~/.claude` (the work/primary account's configDir) is the default, so it's never injected — the work account uses the system default Claude CLI config. Only `~/.claude-personal` (the spillover) gets explicit env var injection. This is correct because:
- When spillover is NOT activated: resolves to "work" → configDir="~/.claude" → no injection → uses system default ✓
- When spillover IS activated: resolves to "personal" → configDir="~/.claude-personal" → env var injected ✓

### 5. OpenCode Backend Path (pkg/orch/spawn_modes.go)

**Tested:** Read `runSpawnHeadless()` and `runSpawnTmux()` in spawn_modes.go.

**Observed:**
- Neither function uses `cfg.Account` or `cfg.AccountConfigDir`
- OpenCode backend uses its own auth mechanism (`~/.local/share/opencode/auth.json`)
- Account distribution via CLAUDE_CONFIG_DIR is a Claude CLI concept — not applicable to OpenCode
- This is architecturally correct: OpenCode has its own auth, Claude CLI has its own auth

### 6. Daemon Path

**Tested:** Read `SpawnWork()` in `pkg/daemon/issue_adapter.go:352-367`.

**Observed:**
- Daemon shells out to `orch work <beadsID>` (optionally with `--model` and `--workdir`)
- NO `--account` flag passed to `orch work`
- BUT: `orch work` calls `runSpawnWithSkillInternal()` which calls `buildCapacityFetcher()`
- So the daemon spawns DO get capacity-aware account routing via the heuristic
- The daemon doesn't need to pass `--account` because the heuristic runs inside `orch work`

### 7. End-to-End Verification

**Command:**
```bash
go test ./pkg/spawn/ -run "Account|BuildClaudeLaunchCommand" -v
```
**Result:** All 22 tests pass (10 account resolution + 12 launch command tests including configDir injection).

---

## What I Observed

### Complete Call Chain (Claude Backend)

```
orch spawn/work
  └─ runSpawnWithSkillInternal()                    [cmd/orch/spawn_cmd.go:477]
      ├─ buildCapacityFetcher()                     [cmd/orch/shared.go:425]
      │   └─ account.LoadConfig()                   [checks roles exist]
      │   └─ account.NewCapacityCache(5min)         [lazy init]
      │   └─ returns closure: cache.Get → API → cache.Set
      ├─ spawn.ResolveInput{CapacityFetcher: ...}   [cmd/orch/spawn_cmd.go:587]
      ├─ orch.ResolveSpawnSettings(resolveInput)    [calls spawn.Resolve()]
      │   └─ resolveAccount(input)                  [pkg/spawn/resolve.go:429]
      │       ├─ CLI --account flag? → use it
      │       ├─ CapacityFetcher set? → heuristic routing
      │       │   ├─ primaries healthy? → use first healthy
      │       │   ├─ spillovers healthy? → use first healthy
      │       │   └─ all exhausted → use first primary
      │       └─ fallback → first primary account
      ├─ account.GetConfigDir(resolvedName)         [cmd/orch/spawn_cmd.go:616]
      ├─ SpawnContext{Account, AccountConfigDir}    [cmd/orch/spawn_cmd.go:637-638]
      ├─ BuildSpawnConfig(ctx) → Config{Account, AccountConfigDir}
      │                                             [pkg/orch/extraction.go:927-928]
      └─ DispatchSpawn() → runSpawnClaude()         [pkg/orch/spawn_modes.go:43-44]
          └─ spawn.SpawnClaude(cfg)                 [pkg/spawn/claude.go:92]
              └─ BuildClaudeLaunchCommand(..., cfg.AccountConfigDir)
                                                    [pkg/spawn/claude.go:130]
                  └─ "unset CLAUDE_CODE_OAUTH_TOKEN; export CLAUDE_CONFIG_DIR=~/.claude-personal; ..."
                                                    [pkg/spawn/claude.go:65]
```

### Gap Analysis: Prior Probe vs Current State

| Component | Prior Probe (Feb 24) | Current Code (Feb 26) |
|-----------|---------------------|----------------------|
| accounts.yaml schema | No tier/role/config_dir | ✅ tier, role, config_dir all present |
| ResolvedSpawnSettings | No Account field | ✅ Account field with provenance |
| BuildClaudeLaunchCommand | No CLAUDE_CONFIG_DIR | ✅ Injects when non-default |
| Daemon spawn path | No account parameter | ✅ Heuristic runs inside orch work |
| CapacityCache | Didn't exist | ✅ 5-min TTL, process-level |
| resolveAccount() | Didn't exist | ✅ CLI → heuristic → default |
| Test coverage | None | ✅ 22 tests, all passing |

---

## Model Impact

- [x] **Confirms** invariant 9 (spawn-architecture model): "Account routing is capacity-aware — Primary accounts checked first; spillover activated when primaries exhausted (>20% threshold)" — Confirmed by code trace and 10 passing tests.

- [x] **Confirms** model claim (orchestration-cost-economics): "Per-spawn account distribution enables capacity-aware routing across multiple Max accounts via CLAUDE_CONFIG_DIR injection, with the Account field tracked as a first-class resolved setting with provenance." — Fully confirmed. The entire chain from CLI flag through resolve pipeline through capacity heuristic through config dir lookup through env var injection is wired.

- [x] **Contradicts** prior probe finding (2026-02-24): "The Claude CLI backend has NO account distribution mechanism at all." — This was accurate at the time (Feb 24) but the implementation was completed between Feb 24-25. Every gap identified in the prior probe's gap analysis table has been closed.

- [x] **Extends** model: OpenCode backend correctly does NOT participate in account distribution. This is architecturally correct (OpenCode has its own auth mechanism), not a gap. The two auth mechanisms are intentionally independent.

---

## Notes

- The `buildCapacityFetcher()` short-circuit (returns nil when <2 accounts or no roles) means single-account setups have zero overhead — the heuristic is completely bypassed.
- Daemon path works through orch CLI re-invocation (`orch work`), so it naturally gets the heuristic without needing special daemon-side wiring.
- The current live config has work=primary(20x) and personal=spillover(5x) — this matches the intended design where the higher-tier account is preferred.
- This probe's SPAWN_CONTEXT.md shows `Account: personal (source: heuristic (spillover-activated-5h:95%-7d:81%))` — confirming the heuristic ran live for this very spawn and activated spillover because the work account's 5h usage was at 95%.
