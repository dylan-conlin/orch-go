# Design: Eliminate Recurring Config Drift Between Claude Code Config Directories

**Phase:** Complete
**Date:** 2026-02-27
**Trigger:** Task tool guard hook added to `~/.claude/settings.json` didn't appear in `~/.claude-personal/settings.json`, causing orchestrator sessions launched via `cc personal` to have no Task tool blocking.

## Design Question

How do we prevent recurring configuration drift between `~/.claude/` (work) and `~/.claude-personal/` (personal) Claude Code config directories?

## Problem Framing

### Context

Dylan uses two Claude Code config directories for separate accounts:
- `~/.claude/` — work account (default, used when `CLAUDE_CONFIG_DIR` is unset)
- `~/.claude-personal/` — personal account (set via `cc personal` function)

The `cc` function in `~/.zshrc` switches accounts by setting `CLAUDE_CONFIG_DIR`:
```bash
cc() {
  case "$account" in
    personal) export CLAUDE_CONFIG_DIR=~/.claude-personal ;;
    work)     unset CLAUDE_CONFIG_DIR ;;
  esac
  # ...
}
```

`CLAUDE_CONFIG_DIR` redirects ALL user-level files: `settings.json`, `CLAUDE.md`, `skills/`, `hooks/`, `projects/`, `keybindings.json`. This means every config file must exist in both directories.

### Current Drift Audit

| File/Dir | `~/.claude/` (work) | `~/.claude-personal/` (personal) | Impact |
|----------|---------------------|----------------------------------|--------|
| settings.json | Full hooks config (including `agreements-check-hook.py`) | Missing `agreements-check-hook.py` | **Task tool guard missing on personal** |
| settings.json | Missing `pyright-lsp` plugin | Has it enabled | Minor (accidental) |
| CLAUDE.md | 12KB global instructions | **MISSING** | **Personal sessions get NO global instructions** |
| skills/ | Present | **MISSING** | **Personal sessions get NO user-level skills** |
| hooks/ | Script files present | **MISSING** | Hook commands use `$HOME/.claude/hooks/` so still work, but fragile |
| statusline.sh | Present | **MISSING** | Status line may not render on personal |
| projects/ | Present | Different set | Correctly separate (per-account project memory) |
| .claude.json | N/A | OAuth account info | Correctly separate (per-account auth) |
| history.jsonl | 112MB | 5.3MB | Correctly separate (per-account history) |

### Success Criteria

1. **Zero drift for shared config** — hooks, CLAUDE.md, skills, statusline must always be identical between accounts
2. **Zero maintenance** — adding a new hook to `settings.json` should automatically appear in both accounts
3. **Per-account runtime data preserved** — auth, history, todos, projects stay separate
4. **Simple** — no custom tooling, config generation scripts, or merge logic

### Constraints

- Cannot change the `cc` function (out of scope)
- Cannot change Claude Code internals
- Must work with current `CLAUDE_CONFIG_DIR` behavior
- Auth (`.claude.json`) must remain per-account

### Scope

- **In:** Shared config files (settings.json, CLAUDE.md, skills/, hooks/, statusline.sh)
- **Out:** Runtime data (history, todos, projects, .claude.json), cc function changes, Claude Code internals

## Exploration

### Fork 1: How to Unify Shared Configuration

**Options:**

- **A: Symlink individual config files** — `ln -sf ~/.claude/settings.json ~/.claude-personal/settings.json` (and same for CLAUDE.md, skills/, hooks/, statusline.sh)
- **B: Config generation script** — Read shared base + per-account overrides, generate both settings.json files
- **C: Symlink entire config directory** — `ln -sf ~/.claude ~/.claude-personal`

**Substrate says:**
- Principle: **Coherence Over Patches** — duplicated settings.json is two copies of the same state. Adding a hook to one without the other is exactly the "locally correct, globally incoherent" pattern.
- Principle: **Gate Over Remind** — "remember to update both configs" is a reminder that will fail under cognitive load. Structural elimination (symlinks) makes drift impossible.
- Model: **Architectural Enforcement / Silent Toolchain Failures** — agents spawned via personal account silently run without hooks. Same defect class as stale skills.

**Analysis:**

| Option | Zero drift? | Zero maintenance? | Per-account data? | Simple? |
|--------|-------------|-------------------|-------------------|---------|
| A: Symlink files | Yes | Yes | Yes | Yes |
| B: Config generation | Yes if run | No (must remember to run) | Yes | No |
| C: Symlink entire dir | Yes | Yes | **No** (auth shared) | Too simple |

Option C fails because `.claude.json` (with OAuth account info), `history.jsonl`, `projects/`, and other runtime data must be per-account.

Option B violates Gate Over Remind — the generation script is a reminder to run, not a gate.

**Recommendation: Option A (symlink individual config files)**

### Fork 2: Which Files to Symlink

**Config files (shared, should symlink):**
```
settings.json    — hooks, permissions, env, sandbox, model
CLAUDE.md        — global user instructions
skills/          — user-level skill definitions
hooks/           — hook script files
statusline.sh    — status line command
```

**Runtime files (per-account, do NOT symlink):**
```
.claude.json     — OAuth account, startup counter, tips
history.jsonl    — session history
todos/           — todo lists
session-env/     — session environment
debug/           — debug logs
file-history/    — file edit history
backups/         — file backups
projects/        — project auto-memory
stats-cache.json — usage statistics
paste-cache/     — clipboard cache
plugins/         — plugin state
telemetry/       — telemetry
tasks/           — task lists
```

**Note on hooks/:** Hook commands in `settings.json` already use absolute paths (`$HOME/.claude/hooks/...`), so the scripts execute correctly regardless of config dir. But symlinking the `hooks/` directory ensures any future hook script discovery (if Claude Code ever looks in `$CLAUDE_CONFIG_DIR/hooks/` directly) works correctly.

**Recommendation:** Symlink all 5 config items listed above. Leave all runtime files as-is.

### Fork 3: How to Prevent Future Drift for New Config Files

**Options:**

- **A: Document only** — Record the symlink convention, rely on memory
- **B: Validation script** — A script that checks symlinks are intact, run manually or via hook
- **C: Modified cc function** — Check/repair symlinks on account switch

**Substrate says:**
- Principle: **Gate Over Remind** — Documentation alone (A) is a reminder. We need structural validation.
- Constraint: Cannot modify cc function (C is out of scope)

**Recommendation: Option B** — A lightweight validation script that can be run manually or integrated into an existing hook. It checks that all expected symlinks exist and point to the correct targets.

## Synthesis

### ⭐ RECOMMENDED: Selective Symlinks + Validation Check

**Implementation (one-time setup):**

```bash
# Back up current personal settings
cp ~/.claude-personal/settings.json ~/.claude-personal/settings.json.bak

# Create symlinks for shared config
ln -sf ~/.claude/settings.json ~/.claude-personal/settings.json
ln -sf ~/.claude/CLAUDE.md ~/.claude-personal/CLAUDE.md
ln -sf ~/.claude/skills ~/.claude-personal/skills
ln -sf ~/.claude/hooks ~/.claude-personal/hooks
ln -sf ~/.claude/statusline.sh ~/.claude-personal/statusline.sh
```

**Validation script** (`~/.orch/hooks/check-config-symlinks.sh`):

```bash
#!/bin/bash
# Validates Claude Code config symlinks are intact
# Run manually: check-config-symlinks.sh
# Or integrate into existing SessionStart hook

EXPECTED_SYMLINKS=(
  "settings.json"
  "CLAUDE.md"
  "skills"
  "hooks"
  "statusline.sh"
)

errors=0
for file in "${EXPECTED_SYMLINKS[@]}"; do
  target="$HOME/.claude/$file"
  link="$HOME/.claude-personal/$file"

  if [[ ! -e "$target" ]]; then
    continue  # Source doesn't exist, nothing to symlink
  fi

  if [[ ! -L "$link" ]]; then
    echo "⚠️ CONFIG DRIFT: $link is not a symlink (expected → $target)"
    errors=$((errors + 1))
  elif [[ "$(readlink "$link")" != "$HOME/.claude/$file" && "$(readlink "$link")" != "$target" ]]; then
    echo "⚠️ CONFIG DRIFT: $link points to $(readlink "$link"), expected $target"
    errors=$((errors + 1))
  fi
done

if [[ $errors -gt 0 ]]; then
  echo "Fix: ln -sf ~/.claude/<file> ~/.claude-personal/<file>"
fi
```

**Why this is right:**

1. **Zero drift by construction** — symlinks make both configs the same file, not copies
2. **Zero maintenance** — new hooks added to `settings.json` automatically appear for both accounts
3. **Per-account data preserved** — auth, history, projects stay separate
4. **Validation catches breakage** — if a Claude Code update overwrites a symlink with a real file, the check catches it
5. **Reversible** — if per-account settings are ever needed, just replace the symlink with a real file

**Trade-off accepted:** Cannot have per-account settings differences. Currently there's no legitimate need for this — the only difference (pyright plugin) was accidental drift.

**When this would change:** If Claude Code adds account-specific settings that should differ between work/personal (e.g., different model defaults per account). At that point, switch to a generation approach.

### Alternative: Config Generation Script

- **Pros:** Supports per-account differences
- **Cons:** Must remember to run, adds complexity, violates Gate Over Remind
- **When to choose:** If legitimate per-account settings differences emerge

## Recommendations

### Implementation Plan

**Phase 1: One-time setup (5 minutes)**
1. Back up `~/.claude-personal/settings.json`
2. Create 5 symlinks (settings.json, CLAUDE.md, skills/, hooks/, statusline.sh)
3. Verify personal sessions now have hooks, instructions, and skills

**Phase 2: Validation (optional, 15 minutes)**
1. Create `~/.orch/hooks/check-config-symlinks.sh`
2. Integrate into an existing SessionStart hook or run periodically

### File Targets

| Action | File |
|--------|------|
| Create symlink | `~/.claude-personal/settings.json → ~/.claude/settings.json` |
| Create symlink | `~/.claude-personal/CLAUDE.md → ~/.claude/CLAUDE.md` |
| Create symlink | `~/.claude-personal/skills → ~/.claude/skills` |
| Create symlink | `~/.claude-personal/hooks → ~/.claude/hooks` |
| Create symlink | `~/.claude-personal/statusline.sh → ~/.claude/statusline.sh` |
| Create script | `~/.orch/hooks/check-config-symlinks.sh` |

### Acceptance Criteria

1. `ls -la ~/.claude-personal/settings.json` shows symlink to `~/.claude/settings.json`
2. `diff ~/.claude/settings.json ~/.claude-personal/settings.json` produces no output
3. `cc personal` sessions have access to global CLAUDE.md instructions
4. `cc personal` sessions have access to all user-level skills
5. Adding a hook to `~/.claude/settings.json` automatically appears for personal account

### Out of Scope

- Modifying the `cc` function
- Per-account settings differences
- Automating the symlink creation (one-time manual step)
- Changing how Claude Code resolves `CLAUDE_CONFIG_DIR`

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This resolves a recurring config drift pattern (will recur with every new hook/skill/config)

**Suggested blocks keywords:**
- "claude config drift"
- "CLAUDE_CONFIG_DIR"
- "personal account settings"
- "cc personal hooks missing"
