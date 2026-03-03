<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The lowest-weekly-usage heuristic IS executing and fetching real capacity data at spawn time, but the account selection has NO EFFECT because `config_dir` is missing from accounts.yaml — all spawns use `~/.claude` (work account) regardless of which account the heuristic picks.

**Evidence:** Traced full code path: `buildCapacityFetcher()` returns non-nil (2 accounts with roles), `resolveAccount()` fetches capacity and picks "personal" for this spawn (7d:46%, 5h:36%), but `GetConfigDir("personal")` returns `""` because config_dir isn't set in accounts.yaml, so no `CLAUDE_CONFIG_DIR` is injected.

**Knowledge:** The heuristic, capacity API, resolve pipeline, and CLAUDE_CONFIG_DIR injection are all correctly wired. The only break is the data layer: accounts.yaml is missing `config_dir` and `tier` fields. This is a configuration gap, not a code bug.

**Next:** Add `config_dir` and `tier` fields to accounts.yaml. personal needs `config_dir: "~/.claude-personal"`, work needs `config_dir: "~/.claude"`. This is a config fix, not a code change — recommend `orch account configure` command or manual yaml edit.

**Authority:** implementation — Config data fix, no architectural change needed

---

# Investigation: Account Routing Heuristic Execution

**Question:** Is the account routing heuristic actually executing at spawn time, and if so, why does `orch account list` show `tier=-` and why does Dylan see work account spawns?

**Defect-Class:** configuration-drift

**Started:** 2026-03-03
**Updated:** 2026-03-03
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/orchestration-cost-economics/probes/2026-02-26-probe-account-distribution-wiring-trace.md | extends | yes | Feb 26 probe stated accounts.yaml had tier/config_dir/roles set correctly — currently only roles are set, tier and config_dir are MISSING |
| .kb/models/orchestration-cost-economics/probes/2026-02-24-probe-automatic-account-distribution-design.md | extends | yes | Feb 24 finding (no distribution mechanism) was correct at time; Feb 26 confirmed code was added; this investigation confirms code still works but config drifted |

---

## Findings

### Finding 1: The capacity fetcher IS built and returns non-nil at spawn time

**Evidence:** `buildCapacityFetcher()` in `cmd/orch/shared.go:531-570`:
1. Loads accounts.yaml — finds 2 accounts (personal, work) → passes `len(cfg.Accounts) < 2` check
2. Checks for roles — personal has `role: primary`, work has `role: spillover` → `hasRoles = true`
3. Creates CapacityCache with 5-min TTL, returns closure that calls `GetAccountCapacity(name)` via OAuth token refresh + API call

The CapacityFetcher is passed to `spawn.ResolveInput` at `spawn_cmd.go:672`.

**Source:** `cmd/orch/shared.go:531-570`, `cmd/orch/spawn_cmd.go:672`, `~/.orch/accounts.yaml`

**Significance:** The heuristic IS running. The capacity fetcher is not nil. This disproves hypothesis that "capacity fetcher returns nil, so heuristic falls back to first primary."

---

### Finding 2: The heuristic picks accounts correctly based on capacity

**Evidence:** `resolveAccount()` in `pkg/spawn/resolve.go:483-560`:
- With CapacityFetcher set, fetches capacity for all accounts
- Sorts by highest SevenDayRemaining, then FiveHourRemaining, then alphabetical
- This spawn's SPAWN_CONTEXT.md shows: `Account: personal (source: heuristic (lowest-weekly-7d:46%-5h:36%))` — confirming capacity was fetched and personal had more headroom
- All 12 account heuristic tests pass (`go test ./pkg/spawn/ -run Account -v`)

**Source:** `pkg/spawn/resolve.go:483-560`, SPAWN_CONTEXT.md line 22, test output

**Significance:** The account selection algorithm works correctly. The selected account name ("personal") flows into SpawnContext.

---

### Finding 3: config_dir is MISSING from accounts.yaml — account selection has no effect

**Evidence:** Current `~/.orch/accounts.yaml`:
```yaml
accounts:
    personal:
        email: dylan.conlin@gmail.com
        role: primary
        # NO config_dir field
        # NO tier field
    work:
        email: dylan.conlin@sendcutsend.com
        role: spillover
        # NO config_dir field
        # NO tier field
```

`GetConfigDir("personal")` at `account.go:199-209` returns `""` (empty string) because `acc.ConfigDir` is empty.

At `spawn_cmd.go:713`: `resolvedAccountConfigDir := account.GetConfigDir(resolvedAccountName)` → returns `""`

At `claude.go:73-75`: `if configDir != "" && configDir != "~/.claude"` → condition is false → NO `CLAUDE_CONFIG_DIR` injection

Result: Every spawned agent uses the default `~/.claude` directory, which is the work account (`dylan.conlin@sendcutsend.com`).

**Source:** `~/.orch/accounts.yaml`, `pkg/account/account.go:199-209`, `cmd/orch/spawn_cmd.go:713`, `pkg/spawn/claude.go:73-75`

**Significance:** This is the root cause. The heuristic selects the right account name, but the name-to-config-dir mapping is broken because config_dir is not populated. All spawns land on `~/.claude` regardless.

---

### Finding 4: Two Claude config dirs exist with different accounts

**Evidence:**
- `~/.claude/` — NO `.claude.json` with oauthAccount (but is the default config dir, so Claude CLI uses it)
- `~/.claude-personal/` — HAS `.claude.json` with `emailAddress: dylan.conlin@gmail.com` (personal account)

This means:
- Default `~/.claude` = work account (sendcutsend.com)
- `~/.claude-personal` = personal account (gmail.com)

The CLAUDE_CONFIG_DIR mechanism works — `claude-personal` alias in `~/.zshrc` uses `CLAUDE_CONFIG_DIR=~/.claude-personal`. But orch spawn doesn't inject it because config_dir is not in accounts.yaml.

**Source:** `ls -la ~/.claude-personal/`, `python3 -c "..." ~/.claude-personal/.claude.json`

**Significance:** The infrastructure for per-account config dirs exists. Only the accounts.yaml data is missing.

---

### Finding 5: `orch account list` shows tier=- because tier field is not set (separate from capacity)

**Evidence:** `account_cmd.go:110-113`:
```go
tier := acc.Tier
if tier == "" {
    tier = "-"
}
```
`acc.Tier` reads the `tier` yaml field. It's empty → displays `-`.

`orch account list` calls `account.RecommendAccount(accounts, nil)` — note **nil** capacity fetcher. It does NOT fetch live capacity data. The recommendation is role-based (first primary), not capacity-based.

**Source:** `cmd/orch/account_cmd.go:104-129`, `pkg/account/account.go:323-380`

**Significance:** The `tier=-` display is expected — it's a static yaml field, not a live capacity indicator. The `orch account list` command was not designed to show live capacity. This is a display gap, not a routing failure.

---

## Synthesis

**Key Insights:**

1. **The heuristic works but the config_dir mapping is broken** — The entire code chain (buildCapacityFetcher → resolveAccount → GetConfigDir → BuildClaudeLaunchCommand) is correctly wired. The failure is purely at the data layer: accounts.yaml lacks `config_dir` values.

2. **The Feb 26 probe's description of accounts.yaml was inaccurate or has since drifted** — That probe stated "accounts.yaml has work(primary, 20x, ~/.claude) and personal(spillover, 5x, ~/.claude-personal)". The actual yaml shows personal=primary, work=spillover, with NO tier or config_dir. Either the config was changed after the probe, or the probe described intended state rather than actual state.

3. **Two separate display issues vs one routing issue** — `tier=-` is a display gap (static yaml field not populated). The real problem is `config_dir` not being set, which makes the entire account routing system inert despite functioning correctly in logic.

**Answer to Investigation Question:**

The lowest-weekly-usage heuristic **IS executing** at spawn time. Capacity data IS fetched via OAuth token refresh + API call. The heuristic correctly picks the account with most remaining weekly capacity. However, the selection has **no effect on which account the spawned agent actually uses** because `config_dir` is missing from accounts.yaml for both accounts. Without config_dir, `GetConfigDir()` returns empty, and `BuildClaudeLaunchCommand()` skips `CLAUDE_CONFIG_DIR` injection. All spawned agents default to `~/.claude` (work account) regardless of the heuristic's choice.

---

## Structured Uncertainty

**What's tested:**

- ✅ `buildCapacityFetcher()` returns non-nil closure (code trace + accounts.yaml has 2 accounts with roles)
- ✅ `resolveAccount()` fetches capacity and picks correct account (SPAWN_CONTEXT shows "personal" with 7d:46%, 5h:36%)
- ✅ `GetConfigDir()` returns empty for both accounts (accounts.yaml confirmed missing config_dir)
- ✅ `BuildClaudeLaunchCommand()` skips CLAUDE_CONFIG_DIR when configDir is empty (code at claude.go:73)
- ✅ `~/.claude-personal` exists and has personal account OAuth (verified .claude.json)
- ✅ All 12 account heuristic tests pass (`go test ./pkg/spawn/ -run Account`)

**What's untested:**

- ⚠️ Whether manually adding config_dir to accounts.yaml actually fixes the routing (not tested, would require editing yaml and spawning)
- ⚠️ Whether the work account's Claude CLI in `~/.claude` actually has valid OAuth (no .claude.json found, may use different auth mechanism)

**What would change this:**

- Finding would be wrong if config_dir IS set but the yaml parser ignores it (unlikely — Account struct has `yaml:"config_dir,omitempty"`)
- Finding would be wrong if there's another CLAUDE_CONFIG_DIR injection path not in BuildClaudeLaunchCommand (searched — there isn't)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add config_dir and tier to accounts.yaml | implementation | Config data fix, no code change, reversible |
| Add `orch account configure` command | architectural | New command, changes CLI surface, orchestrator decision |
| Show live capacity in `orch account list` | architectural | Changes command behavior, may have performance implications |

### Recommended Approach ⭐

**Add config_dir and tier fields to accounts.yaml** — Manual YAML edit to unblock account routing

**Why this approach:**
- Zero code changes needed — the code already handles these fields correctly
- Immediately fixes account routing for all spawn paths (manual, daemon, orch work)
- The Feb 26 probe confirmed the full code path works when these fields are populated

**Required accounts.yaml update:**
```yaml
accounts:
    personal:
        email: dylan.conlin@gmail.com
        refresh_token: ...
        source: saved
        role: primary
        config_dir: "~/.claude-personal"
        tier: "5x"       # or whatever the personal subscription tier is
    work:
        email: dylan.conlin@sendcutsend.com
        refresh_token: ...
        source: saved
        role: spillover
        config_dir: "~/.claude"
        tier: "20x"      # or whatever the work subscription tier is
```

**Trade-offs accepted:**
- Manual yaml edit is fragile — `orch account add` doesn't prompt for config_dir/tier
- If accounts are re-added, these fields will be lost

**Implementation sequence:**
1. Edit `~/.orch/accounts.yaml` — add config_dir for both accounts
2. Verify with `orch spawn --dry-run` or trace log (if available)
3. Follow up: add `orch account configure` to set fields persistently

### Alternative Approaches Considered

**Option B: Auto-discover config_dir by scanning ~/.claude* directories**
- **Pros:** No manual config needed, self-healing
- **Cons:** Fragile heuristic, wrong directory = wrong account, adds code complexity
- **When to use instead:** If config_dir changes frequently or more accounts are added

**Option C: Add config_dir prompting to `orch account add`**
- **Pros:** Prevents config drift for future account additions
- **Cons:** Doesn't fix existing accounts, requires code change
- **When to use instead:** As a follow-up after the manual fix

---

### Implementation Details

**What to implement first:**
- Edit `~/.orch/accounts.yaml` to add config_dir and tier for both accounts
- Verify with a test spawn that CLAUDE_CONFIG_DIR is injected

**Things to watch out for:**
- ⚠️ Ensure `~/.claude-personal/.claude.json` has valid OAuth tokens before routing agents there
- ⚠️ The `~/.claude` directory may need a valid `.claude.json` too — currently none found
- ⚠️ Roles (primary/spillover) are swapped from what Feb 26 probe described — verify intended assignment

**Success criteria:**
- ✅ `GetConfigDir("personal")` returns `"~/.claude-personal"`
- ✅ `GetConfigDir("work")` returns `"~/.claude"`
- ✅ When heuristic picks "personal", spawned agent command includes `CLAUDE_CONFIG_DIR=~/.claude-personal`
- ✅ `orch account list` shows tier values instead of `-`

---

## References

**Files Examined:**
- `cmd/orch/shared.go:531-570` — buildCapacityFetcher() implementation
- `pkg/spawn/resolve.go:483-560` — resolveAccount() heuristic
- `pkg/account/account.go:199-209` — GetConfigDir() function
- `pkg/spawn/claude.go:68-75` — CLAUDE_CONFIG_DIR injection logic
- `cmd/orch/spawn_cmd.go:645-713` — Spawn resolution and context building
- `cmd/orch/account_cmd.go:90-133` — orch account list display logic
- `~/.orch/accounts.yaml` — Current account configuration

**Commands Run:**
```bash
# Verify accounts.yaml state
cat ~/.orch/accounts.yaml

# Run orch account list
orch account list

# Run account heuristic tests
go test ./pkg/spawn/ -run "Account" -v

# Run BuildClaudeLaunchCommand tests
go test ./pkg/spawn/ -run "BuildClaudeLaunchCommand" -v

# Check which Claude config dirs exist and their accounts
ls -la ~/.claude-personal/
python3 -c "import json; ..." ~/.claude-personal/.claude.json
```

**Related Artifacts:**
- **Probe:** .kb/models/orchestration-cost-economics/probes/2026-02-26-probe-account-distribution-wiring-trace.md — Confirmed code wiring, but described config state inaccurately
- **Probe:** .kb/models/orchestration-cost-economics/probes/2026-02-24-probe-automatic-account-distribution-design.md — Original gap analysis (all gaps were closed by code, but config_dir was never populated)
