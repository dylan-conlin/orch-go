# Probe: Config Dir Drift — Scope and Prevention via Structural Elimination

**Model:** Architectural Enforcement
**Status:** Complete
**Date:** 2026-02-27

## Question

The Architectural Enforcement model documents gates for preventing code-level drift (hotspot enforcement, skill staleness). Does the model's framework extend to **configuration drift** between parallel Claude Code config directories? Specifically: is config drift a structural enforcement gap that needs a gate, or is it out of scope?

## What I Tested

### 1. Audited both config directories for current drift

```bash
diff <(jq --sort-keys . ~/.claude/settings.json) <(jq --sort-keys . ~/.claude-personal/settings.json)
```

**Drift in settings.json:**
| Setting | ~/.claude/ (work) | ~/.claude-personal/ (personal) |
|---------|-------------------|-------------------------------|
| `agreements-check-hook.py` in SessionStart | Present | **MISSING** |
| `pyright-lsp` plugin enabled | Missing | Present |

### 2. Tested CLAUDE_CONFIG_DIR scope

Confirmed via Claude Code documentation that `CLAUDE_CONFIG_DIR` redirects ALL user-level files:
- settings.json, CLAUDE.md, skills/, hooks/, projects/, keybindings.json

### 3. Audited both config directories for missing shared files

| File/Dir | ~/.claude/ (work) | ~/.claude-personal/ (personal) | Impact |
|----------|-------------------|-------------------------------|--------|
| settings.json | Full hooks config | Missing 1 hook | **Task tool guard missing** |
| CLAUDE.md | 12KB global instructions | **MISSING** | **No global instructions on personal** |
| skills/ | Present | **MISSING** | **No user-level skills on personal** |
| hooks/ | Script files | **MISSING** | Hook commands use `$HOME/.claude/hooks/` so still work |
| statusline.sh | Present | **MISSING** | Status line may not render |
| projects/ | Present | Different set | Correctly separate (per-account) |

### 4. Verified auth is per-directory

```bash
jq '.oauthAccount' ~/.claude-personal/.claude.json
# → personal OAuth account (user@example.com)
```

Auth stored in `.claude.json` within config dir — confirms dirs MUST remain separate for runtime data.

## What I Observed

1. **The drift is much larger than the triggering bug.** The Task tool guard was one symptom. Personal sessions are running without global CLAUDE.md, without skills, without the agreements hook.

2. **The config dir split is required for auth separation**, but nothing else needs to differ between accounts. All hook commands use absolute `$HOME/.claude/hooks/` paths — they don't depend on config dir resolution.

3. **This is a "defect class" problem**, not a single bug. Every time a hook, skill, or config file is added to `~/.claude/`, it silently doesn't appear in `~/.claude-personal/`. The failure is invisible until something breaks.

4. **The model's "Gate Over Remind" principle directly applies**: documenting "remember to update both configs" is a reminder that will fail. Structural elimination (symlinks) is the gate equivalent for config.

## Model Impact

**Extends** the Architectural Enforcement model:

- The model documents code-level enforcement (hotspot gates, skill staleness) but not config-level enforcement. Config drift is the same defect class — duplicated state that silently diverges.
- The model's "Silent Toolchain Failures" failure mode maps precisely: agents spawned via `cc personal` run with missing hooks, missing skills, missing instructions — silently. The failure is invisible to the operator.
- **New invariant candidate:** "Shared configuration must have a single source of truth. Parallel copies are a structural defect, not a maintenance task." This is the config-level equivalent of the code-level "instruction-based enforcement fails under pressure" invariant.
