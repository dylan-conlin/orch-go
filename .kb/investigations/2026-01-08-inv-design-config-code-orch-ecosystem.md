<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Designed config-as-code system with ~/.orch/system.yaml as declarative source, generation commands for plist/symlinks, and drift detection via `orch doctor --config`.

**Evidence:** Audited 6 external config locations; identified plist bug class root cause (manual flag edits without version control); verified existing config.yaml and accounts.yaml provide partial foundation.

**Knowledge:** External config splits into declarative (can regenerate), secrets (credentials), and ephemeral (runtime state). Generation + drift detection covers the plist bug case without over-engineering.

**Next:** Implement in 3 phases: (1) Add daemon config to system.yaml, (2) Add `orch config generate plist`, (3) Add drift detection to `orch doctor`.

**Promote to Decision:** recommend-yes - This establishes the config-as-code pattern for the orch ecosystem.

---

# Investigation: Config-as-Code Design for Orch Ecosystem

**Question:** How should external config that affects orch system behavior be managed to prevent invisible bugs like the plist flag drift?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent og-feat-design-config-code-08jan-5f13
**Phase:** Complete
**Next Step:** None - design complete, ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Complete External Config Inventory

**Evidence:** Audited all external config locations that affect orch system behavior:

| Location | Purpose | Type | Version Controlled? |
|----------|---------|------|---------------------|
| `~/Library/LaunchAgents/com.orch.daemon.plist` | Daemon flags, PATH, working dir | Generated | ❌ No |
| `~/.orch/config.yaml` | User preferences (backend, tier, notifications) | Declarative | ❌ No |
| `~/.orch/accounts.yaml` | OAuth tokens (refresh_token), account list | Secrets | ❌ No |
| `~/.config/opencode/plugin/` | Plugin symlinks | Generated | ❌ No |
| `~/.bun/bin/` | CLI symlinks (bd, kb, orch, etc.) | Generated | ❌ No |
| `{project}/.orch/config.yaml` | Project servers (port mappings) | Declarative | ✅ Yes |
| `~/.orch/focus.json` | Current orchestrator focus | Ephemeral | ❌ No |
| `~/.orch/session.json` | Orchestrator session state | Ephemeral | ❌ No |
| `~/.orch/daemon-status.json` | Daemon runtime state | Ephemeral | ❌ No |
| Environment variables (BEADS_NO_DAEMON, etc.) | Runtime config | Ephemeral | ❌ No |

**Source:** Direct examination of files, `ls -la` commands, grep of codebase for config patterns.

**Significance:** Config naturally falls into categories: (1) Generated from source, (2) Secrets/credentials, (3) Ephemeral runtime state. Only category 1 benefits from config-as-code.

---

### Finding 2: The Plist Bug Root Cause

**Evidence:** The com.orch.daemon.plist contains:
```xml
<string>--reflect-issues=false</string>
```

The bug was that this flag was manually changed from `--reflect` to `--reflect-issues=false` (or vice versa), and there was no record of the change, no way to detect drift, and no single source of truth.

Current plist structure:
- `ProgramArguments`: daemon flags including `--poll-interval`, `--max-agents`, `--label`, `--verbose`, `--reflect-issues`
- `WorkingDirectory`: set to orch-go
- `EnvironmentVariables`: PATH, BEADS_NO_DAEMON
- Standard launchd keys: RunAtLoad, KeepAlive, StandardOutPath

**Source:** `/Users/dylanconlin/Library/LaunchAgents/com.orch.daemon.plist`

**Significance:** The plist is derived config - it SHOULD be generated from a declarative source. Manual edits create invisible drift that persists across restarts.

---

### Finding 3: Existing Config Infrastructure

**Evidence:** 

The codebase already has:
1. `pkg/userconfig/userconfig.go` - Manages `~/.orch/config.yaml` with typed Config struct
2. `pkg/account/account.go` - Manages `~/.orch/accounts.yaml` with OAuth tokens
3. `pkg/config/config.go` - Manages project-level `.orch/config.yaml` for server ports

Current `~/.orch/config.yaml` structure:
```yaml
backend: opencode
auto_export_transcript: true
default_tier: full
notifications:
  enabled: true
reflect:
  enabled: true
  interval_minutes: 60
  create_issues: true
```

**Source:** `pkg/userconfig/userconfig.go:44-59`, `~/.orch/config.yaml`

**Significance:** The infrastructure exists to extend `~/.orch/config.yaml` with daemon configuration. We don't need to invent a new config system.

---

### Finding 4: Symlinks Are Already Tracked (Partially)

**Evidence:** 

The `~/.bun/bin/` symlinks are documented in `~/.claude/CLAUDE.md`:
```bash
ln -sf ~/go/bin/bd ~/.bun/bin/bd
ln -sf ~/.local/bin/kb ~/.bun/bin/kb
ln -sf ~/bin/orch ~/.bun/bin/orch
ln -sf /opt/homebrew/bin/tmux ~/.bun/bin/tmux
ln -sf /opt/homebrew/bin/go ~/.bun/bin/go
```

But there's no automated verification that these symlinks exist or point to the right targets.

**Source:** `~/.claude/CLAUDE.md` (CLI PATH Fix section), `ls -la ~/.bun/bin/`

**Significance:** Documentation is not verification. We need `orch doctor` to check symlink integrity.

---

### Finding 5: Plugin Symlinks Are Ad-Hoc

**Evidence:** `~/.config/opencode/plugin/` contains:
```
bd-close-gate.ts -> /Users/dylanconlin/Documents/personal/orch-cli/.opencode/plugin/bd-close-gate.ts
orchestrator-session.ts -> /Users/dylanconlin/Documents/personal/orch-go/plugins/orchestrator-session.ts
usage-warning.ts -> /Users/dylanconlin/Documents/personal/orch-cli/.opencode/plugin/usage-warning.ts
agentlog-inject.ts -> /Users/dylanconlin/Documents/personal/orch-cli/.opencode/plugin/agentlog-inject.ts
```

Plus 3 non-symlink files: `action-tracker.ts`, `friction-capture.ts`, `guarded-files.ts`

**Source:** `ls -la ~/.config/opencode/plugin/`

**Significance:** Plugin deployment is inconsistent - some symlinks to source repos, some direct files. A config-as-code system should standardize this.

---

## Synthesis

**Key Insights:**

1. **Natural Config Categories** - External config falls into 3 clear categories:
   - **Generated:** Plist, symlinks → Should be generated from declarative source
   - **Secrets:** OAuth tokens → Keep separate, never in version control
   - **Ephemeral:** Focus, session state → Runtime state, not config

2. **Extend Existing Infrastructure** - `~/.orch/config.yaml` already exists with typed Go structs. Adding daemon config there (not a new file) is the path of least resistance.

3. **Generation + Drift = Solution** - The plist bug wasn't that we lacked config, it was that:
   - No single source of truth
   - No way to regenerate from source
   - No way to detect when actual != expected
   
   Generation + drift detection solves this completely.

4. **Don't Over-Engineer** - Secrets and ephemeral state don't need config-as-code. Only generated config does. Keep scope narrow.

**Answer to Investigation Question:**

External config should be managed via a **generation + drift detection** pattern:

1. **Declarative source:** `~/.orch/config.yaml` (extended with daemon section)
2. **Generation commands:** `orch config generate [plist|symlinks|all]`
3. **Drift detection:** `orch doctor --config` shows expected vs actual
4. **Secrets stay separate:** `accounts.yaml` remains as-is (not generated, contains credentials)

This addresses the plist flag bug class by:
- Making the expected config explicit and declarative
- Allowing regeneration from source
- Detecting when manual edits cause drift

---

## Structured Uncertainty

**What's tested:**

- ✅ Config file locations verified via ls and file reads
- ✅ Existing config infrastructure examined in Go source
- ✅ Plist structure analyzed for what needs to be configurable
- ✅ Symlink patterns identified from actual system state

**What's untested:**

- ⚠️ Plist generation XML templating (not implemented yet)
- ⚠️ Launchd reload after plist changes (needs testing)
- ⚠️ Whether symlink drift actually causes user-visible bugs (assumed but not proven)

**What would change this:**

- If launchd plist regeneration requires user logout/login, the workflow may need adjustment
- If symlink drift is never actually a problem, we could skip that part

---

## Implementation Recommendations

**Purpose:** Bridge from investigation to implementation.

### Recommended Approach ⭐

**Extend ~/.orch/config.yaml with daemon config, add generation commands**

**Why this approach:**
- Uses existing infrastructure (`pkg/userconfig/userconfig.go`)
- Single source of truth in one file
- Familiar YAML format already in use
- Directly addresses the plist flag drift issue

**Trade-offs accepted:**
- More fields in config.yaml (acceptable complexity)
- Need to teach users about `orch config generate`

**Implementation sequence:**
1. **Phase 1: Daemon config in config.yaml** (~2 hours)
   - Add DaemonConfig struct to userconfig.go
   - Fields: poll_interval, max_agents, label, verbose, reflect_issues, working_directory, path_additions
   - Update config.yaml loading/saving

2. **Phase 2: Plist generation** (~2 hours)
   - `orch config generate plist` command
   - Template for com.orch.daemon.plist
   - Reads from config.yaml, generates plist
   - Prompts for `launchctl kickstart` if changed

3. **Phase 3: Drift detection** (~1 hour)
   - `orch doctor --config` (or extend existing doctor)
   - Compare expected (from config.yaml) vs actual (from plist)
   - Show drift with actionable fix: "Run: orch config generate plist"

4. **Phase 4 (optional): Symlink management** (~2 hours)
   - Add symlinks section to config.yaml
   - `orch config generate symlinks`
   - Doctor checks for missing/broken symlinks

### Alternative Approaches Considered

**Option B: Separate ~/.orch/system.yaml for generated config**
- **Pros:** Clear separation of concerns
- **Cons:** Another file to manage, cognitive overhead
- **When to use instead:** If config.yaml becomes unwieldy (>100 lines)

**Option C: Version control ~/Library/LaunchAgents/com.orch.daemon.plist directly**
- **Pros:** Simple, direct tracking
- **Cons:** Ties system config to single machine, doesn't solve generation
- **When to use instead:** If you're tracking all dotfiles in version control already

**Rationale for recommendation:** Option A is simplest, reuses existing infrastructure, and solves the actual problem (flag drift) without over-engineering.

---

### Implementation Details

**What to implement first:**
- DaemonConfig struct in pkg/userconfig/userconfig.go
- This is foundational - generation depends on it

**Things to watch out for:**
- ⚠️ Plist has XML escaping requirements
- ⚠️ PATH in plist needs to be built correctly (multiple sources)
- ⚠️ Launchd reload may require specific incantation (`launchctl kickstart -k`)

**Areas needing further investigation:**
- Whether OpenCode plugin symlinks should be managed here or separately
- Whether project-level `.orch/config.yaml` also needs generation (probably not initially)

**Success criteria:**
- ✅ `orch config generate plist` produces valid, functional plist
- ✅ `orch doctor --config` detects drift from the plist flag bug case
- ✅ Config changes are now visible via git diff of config.yaml

---

## Design Specification

### Extended Config Schema

```yaml
# ~/.orch/config.yaml (proposed additions)

# ... existing fields ...
backend: opencode
auto_export_transcript: true
default_tier: full
notifications:
  enabled: true
reflect:
  enabled: true
  interval_minutes: 60
  create_issues: true

# NEW: Daemon configuration
daemon:
  poll_interval: 60        # seconds
  max_agents: 3
  label: "triage:ready"
  verbose: true
  reflect_issues: false    # THE FLAG THAT CAUSED THE BUG
  working_directory: ~/Documents/personal/orch-go
  
  # PATH additions beyond system PATH
  path:
    - ~/.bun/bin
    - ~/bin
    - ~/go/bin
    - /opt/homebrew/bin

# NEW (optional): CLI symlinks to verify
symlinks:
  ~/.bun/bin/bd: ~/go/bin/bd
  ~/.bun/bin/kb: ~/.local/bin/kb
  ~/.bun/bin/orch: ~/bin/orch
  ~/.bun/bin/tmux: /opt/homebrew/bin/tmux
  ~/.bun/bin/go: /opt/homebrew/bin/go
  ~/.bun/bin/opencode: ~/Documents/personal/opencode/packages/opencode/dist/opencode-darwin-arm64/bin/opencode

# NEW (optional): OpenCode plugins to deploy
plugins:
  global:  # ~/.config/opencode/plugin/
    - ~/Documents/personal/orch-go/plugins/orchestrator-session.ts
    - ~/Documents/personal/orch-cli/.opencode/plugin/bd-close-gate.ts
    - ~/Documents/personal/orch-cli/.opencode/plugin/usage-warning.ts
```

### New Commands

```bash
# Generate plist from config
orch config generate plist
# Output: Updated ~/Library/LaunchAgents/com.orch.daemon.plist
# Output: Run: launchctl kickstart -k gui/$(id -u)/com.orch.daemon

# Generate all derived config
orch config generate all

# Check for drift (extend existing doctor)
orch doctor --config
# Output: 
#   ✓ plist: daemon.max_agents matches (3)
#   ✗ plist: daemon.reflect_issues DRIFT: config=false, plist=true
#     Fix: orch config generate plist
#   ✓ symlinks: 6/6 valid
```

### Migration Path

1. **No breaking changes** - Existing config.yaml continues to work
2. **Additive** - New daemon section is optional
3. **Gradual adoption:**
   ```bash
   # First time: inspect current plist
   orch config show plist
   
   # Add daemon section to config.yaml manually (or via orch config init)
   
   # Generate and verify
   orch config generate plist --dry-run
   orch config generate plist
   
   # Enable drift detection
   orch doctor --config
   ```

---

## References

**Files Examined:**
- `/Users/dylanconlin/Library/LaunchAgents/com.orch.daemon.plist` - Current daemon plist
- `/Users/dylanconlin/.orch/config.yaml` - User config
- `/Users/dylanconlin/.orch/accounts.yaml` - Account credentials (structure only)
- `~/.config/opencode/plugin/` - Plugin deployment
- `~/.bun/bin/` - CLI symlinks
- `pkg/userconfig/userconfig.go` - Config infrastructure
- `pkg/account/account.go` - Account management
- `cmd/orch/doctor.go` - Existing health checks

**Commands Run:**
```bash
# Inventory external config
ls -la ~/Library/LaunchAgents/com.orch.daemon.plist
ls -la ~/.orch/
ls -la ~/.config/opencode/plugin/
ls -la ~/.bun/bin/
cat ~/.orch/config.yaml
cat ~/.orch/accounts.yaml

# Search codebase
grep -r "config.yaml" --include="*.go"
grep -r "plist" --include="*.go"
```

**Related Artifacts:**
- **Issue:** orch-go-xzr2q (this work)
- **Root cause:** Synthesis issue bug (plist --reflect flag drift)

---

## Investigation History

**[2026-01-08 10:32]:** Investigation started
- Initial question: How to prevent invisible config bugs like plist flag drift?
- Context: Synthesis issue bug persisted 2 days due to untracked plist change

**[2026-01-08 10:45]:** External config inventory complete
- Identified 6 key config locations
- Categorized into generated/secrets/ephemeral

**[2026-01-08 11:00]:** Design complete
- Recommended: Extend config.yaml + generation + drift detection
- Rejected: Separate system.yaml file (over-engineering)

**[2026-01-08 11:15]:** Investigation completed
- Status: Complete
- Key outcome: Clear 3-phase implementation plan with ~5-7 hours estimated work
