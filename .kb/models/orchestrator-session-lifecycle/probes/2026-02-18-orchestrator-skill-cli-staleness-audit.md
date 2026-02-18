# Probe: Orchestrator Skill CLI Staleness Audit

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Does the orchestrator skill (source: `orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template`) accurately represent the current CLI commands, flags, and default behaviors of `orch`, `bd`, and `kb`?

The model claims orchestrators use specific commands/flags for spawning, monitoring, and completing work. This probe tests those claims against actual `--help` output.

---

## What I Tested

Ran `--help` on every command referenced in the orchestrator skill's SKILL.md.template (the authoritative source) and cross-referenced each CLI reference.

```bash
# Commands tested
orch --help                    # Full command list
orch spawn --help              # Spawn flags, modes, backends
orch complete --help           # Completion flags, gates
orch status --help             # Status flags
orch review --help             # Review flags
orch monitor --help            # Monitor flags
orch wait --help               # Wait flags
orch send --help               # Send flags
orch resume --help             # Resume flags
orch abandon --help            # Abandon flags
orch daemon --help             # Daemon subcommands
orch frontier --help           # DOES NOT EXIST (exit 1)
orch focus --help              # Focus flags
orch drift --help              # Drift flags
orch next --help               # Next flags
orch rework --help             # DOES NOT EXIST (exit 1)
orch reflect --help            # DOES NOT EXIST (exit 1)
orch clean --help              # Clean flags
orch doctor --help             # Doctor flags
orch work --help               # Work flags
orch kb --help                 # KB subcommands
orch kb archive-old --help     # DOES NOT EXIST (kb has: ask, extract)
orch hotspot --help            # Hotspot flags
orch session --help            # Session subcommands
orch serve --help              # Serve flags
bd --help                      # BD command list
bd label --help                # Label subcommands
bd label add --help            # Correct label syntax
bd create --help               # Create flags
kb --help                      # KB command list

# Model default verification
cat pkg/model/model.go         # DefaultModel = sonnet, Aliases map
```

---

## What I Observed

### SEVERITY 1: ACTIVELY HARMFUL (teaches wrong flags/commands that cause errors)

#### 1. `--opus` flag does not exist
**Location in template:** Lines 231-234 (Model Selection table), line 236
**Stale text:**
```
| **--opus** | opus + tmux | `orch spawn --opus architect "task"` |
**When to use --opus:** systematic-debugging, investigation, architect...
```
**Reality:** No `--opus` flag exists. The correct flag is `--model opus`.
**Proposed replacement:**
```
| **--model opus** | opus (backend-dependent) | `orch spawn --bypass-triage --model opus architect "task"` |
**When to use --model opus:** systematic-debugging, investigation, architect...
```

#### 2. `orch frontier` does not exist
**Location in template:** Line 320 (Proactive Hygiene Checkpoint), Line 621 (Tools & Commands)
**Stale text:**
```
Session start/end: run `orch frontier` and `bd list --status=in_progress`
**Lifecycle:** `orch spawn SKILL "task"` | `orch frontier` | `orch complete <id>` | `orch review`
```
**Reality:** `orch frontier` is an unknown command (exit 1). No similar subcommand exists.
**Proposed replacement:** Remove `orch frontier` from both locations. Use `orch review` + `bd ready` for session hygiene.

#### 3. `orch rework` does not exist
**Location in template:** Line 625 (Tools & Commands)
**Stale text:**
```
**Strategic/post-completion:** `orch focus` | `orch drift` | `orch next` | `orch rework` | `orch reflect` | `orch kb archive-old`
```
**Reality:** `orch rework` is an unknown command (exit 1). CLI suggests `orch work` as closest match.
**Proposed replacement:** Remove `orch rework` from the command list.

#### 4. `orch reflect` does not exist
**Location in template:** Line 625 (Tools & Commands)
**Stale text:** Same line as #3 above.
**Reality:** `orch reflect` is an unknown command (exit 1). Note: `orch daemon reflect` exists (runs kb reflect analysis), and `kb reflect` exists as a standalone command.
**Proposed replacement:** Remove `orch reflect`. If the intent is kb reflection, reference `kb reflect` directly.

#### 5. `orch kb archive-old` does not exist
**Location in template:** Line 625 (Tools & Commands)
**Stale text:** Same line as #3 above.
**Reality:** `orch kb` only has subcommands `ask` and `extract`. No `archive-old` subcommand. Note: `kb archive` exists as a standalone kb command.
**Proposed replacement:** Remove `orch kb archive-old`. If the intent is kb archival, reference `kb archive` directly.

#### 6. `orch clean --stale` does not exist
**Location in template:** Line 649 (Error Recovery)
**Stale text:**
```
**Nuclear options:** `orch clean --stale` | `orch clean --all` | `rm ~/.orch/registry.lock`
```
**Reality:** `orch clean` has flags: `--all`, `--dry-run`, `--workspaces`, `--sessions`, `--preserve-orchestrator`, `--workspace-days`, `--session-days`. No `--stale` flag.
**Proposed replacement:** `orch clean --workspaces` | `orch clean --all` | `rm ~/.orch/registry.lock`

#### 7. `orch clean --untracked --stale` does not exist
**Location in template:** Line 326 (Backlog Triage)
**Stale text:**
```
Use `orch complete` for tracked work, `orch clean --untracked --stale` for stale untracked backlog.
```
**Reality:** Neither `--untracked` nor `--stale` are valid flags for `orch clean`.
**Proposed replacement:** `orch clean --workspaces` for stale workspace cleanup. Review `orch review --stale` for stale/untracked agents.

### SEVERITY 2: MISLEADING (teaches wrong mental model of behavior)

#### 8. Model Selection table: default is NOT "sonnet + headless"
**Location in template:** Lines 231-234 (Model Selection table)
**Stale text:**
```
| **(none)** | sonnet + headless | `orch spawn architect "task"` |
```
**Reality:** Default model IS sonnet (`pkg/model/model.go` line 19-22: `DefaultModel = anthropic/claude-sonnet-4-5-20250929`). But default backend is `claude` (NOT `opencode`/headless). From `orch spawn --help`:
```
Backend Modes (--backend):
  claude:   Uses Claude Code CLI in tmux (Max subscription, unlimited Opus) (default)
  opencode: Uses OpenCode HTTP API
```
So `(none)` actually gives **sonnet + claude backend (tmux)**, not headless.
**Proposed replacement:**
```
| **(none)** | sonnet + claude backend (tmux) | `orch spawn --bypass-triage architect "task"` |
| **--model opus** | opus + claude backend (tmux) | `orch spawn --bypass-triage --model opus architect "task"` |
| **--backend opencode** | sonnet + headless (HTTP API) | `orch spawn --bypass-triage --backend opencode architect "task"` |
```

#### 9. Spawn modes description conflates backend with spawn mode
**Location in template:** Line 227, Line 544
**Stale text:**
```
**Spawn modes:** Default (headless via HTTP API) | `--tmux` (visual monitoring) | `--inline` (blocking, for debugging)
```
and:
```
Spawn modes: headless default, `--tmux` opt-in; policy/orchestrator skills auto-default to tmux.
```
**Reality:** The default backend is `claude` (tmux-based), not `opencode` (headless). The `--tmux` flag is meaningful for the `opencode` backend. With `--backend claude` (the default), spawns are always in tmux.
**Proposed replacement:**
```
**Backends:** `--backend claude` (default, Claude CLI in tmux) | `--backend opencode` (headless HTTP API, use `--tmux` for tmux)
**Modes:** `--inline` (blocking TUI) | `--tmux` (opencode backend: tmux window) | default (backend-dependent)
```

#### 10. `bd label <id> triage:ready` syntax is incomplete
**Location in template:** Lines 132, 578 (and compiled SKILL.md lines 69, 107, 183)
**Stale text:**
```
Core commands: `bd ready` | `bd label <id> triage:ready` | `bd label <id> triage:review`
```
**Reality:** `bd label` requires a subcommand. Correct syntax is `bd label add <id> triage:ready`.
**Proposed replacement:**
```
Core commands: `bd ready` | `bd label add <id> triage:ready` | `bd label add <id> triage:review`
```

#### 11. Missing `--bypass-triage` in spawn examples
**Location in template:** Lines 223-225 (Spawning Methods), Line 233 (Model Selection)
**Stale text:**
```
2. `orch spawn --bypass-triage SKILL "task" --issue <id>` — From beads issue
3. `orch spawn --bypass-triage SKILL "task"` — Auto-creates beads issue
4. `orch spawn --bypass-triage SKILL "task" --no-track` — Throwaway experiments only

...but in Model Selection table:
| **(none)** | sonnet + headless | `orch spawn architect "task"` |
| **--opus** | opus + tmux | `orch spawn --opus architect "task"` |
```
**Reality:** The Spawning Methods section correctly uses `--bypass-triage`, but the Model Selection examples on lines 233-234 omit it. Manual spawn REQUIRES `--bypass-triage`.
**Proposed replacement:** Add `--bypass-triage` to all `orch spawn` examples, including the Model Selection table.

### SEVERITY 3: COSMETIC / MINOR

#### 12. `bd comment` is deprecated, should be `bd comments add`
**Location in template:** Line 106
**Stale text:**
```
In beads: `bd comments add <id> "FRAME: <why Dylan cares>"`
```
**Reality:** This reference is actually correct in the template (uses `bd comments add`). However, the compiled SKILL.md still shows the old form in some places inherited from worker-base dependency. The template itself is fine.
**Status:** No change needed in orchestrator skill source; worker-base may need update.

#### 13. Tools & Commands reference file path is stale
**Location in template:** Line 631
**Stale text:**
```
**Reference:** `.skillc/reference/tools-and-commands.md`
```
**Reality:** The actual path is `skills/src/meta/orchestrator/.skillc/reference/tools-and-commands.md` (relative to orch-knowledge root). The `.skillc/` relative path may be ambiguous depending on working directory context.
**Status:** Minor — the reference is contextually correct within the skill compilation system.

#### 14. `--type gate` in bd create
**Location in template:** Line 252
**Stale text:**
```
Gate checkpoints: `bd create "Gate: description" --type gate -l triage:review -l area:X`
```
**Reality:** `bd create --help` does list `gate` as a valid type. This is correct.
**Status:** No change needed.

---

## Summary Table

| # | Stale Reference | Severity | Location (template line) |
|---|----------------|----------|--------------------------|
| 1 | `--opus` flag | HARMFUL | 231-234, 236 |
| 2 | `orch frontier` | HARMFUL | 320, 621 |
| 3 | `orch rework` | HARMFUL | 625 |
| 4 | `orch reflect` | HARMFUL | 625 |
| 5 | `orch kb archive-old` | HARMFUL | 625 |
| 6 | `orch clean --stale` | HARMFUL | 649 |
| 7 | `orch clean --untracked --stale` | HARMFUL | 326 |
| 8 | Default = "sonnet + headless" | MISLEADING | 231-233 |
| 9 | "Spawn modes: Default (headless)" | MISLEADING | 227, 544 |
| 10 | `bd label <id>` (missing subcommand) | MISLEADING | 132, 578 |
| 11 | Missing --bypass-triage in examples | MISLEADING | 233-234 |
| 12 | bd comment deprecation | COSMETIC | worker-base (not this template) |
| 13 | Reference file path | COSMETIC | 631 |

**Total: 7 HARMFUL + 4 MISLEADING + 2 COSMETIC = 13 stale references**

---

## Model Impact

- [x] **Contradicts** invariant: The orchestrator skill's Tools & Commands section claims commands exist that do not (`frontier`, `rework`, `reflect`, `kb archive-old`). The Model Selection guidance teaches a non-existent `--opus` flag. The spawn mode description gives a wrong default (headless vs claude/tmux).
- [x] **Extends** model with: The skill does not document the `--backend` flag dimension at all. Backend selection (claude vs opencode) is a critical spawn decision that the skill doesn't address. The skill also doesn't reflect that `--bypass-triage` is now mandatory for manual spawns (it's in the Spawning Methods section but not in the examples that orchestrators would copy-paste).

---

## Notes

### Root Cause Analysis

The staleness concentrates in two areas:
1. **Section 2 (Spawning):** The Model Selection table and spawn mode descriptions haven't been updated since the `--backend` flag was introduced and the default changed from opencode to claude. The `--opus` convenience flag was apparently planned but never implemented — the actual flag is `--model opus`.
2. **Section 7 (Reference/Tools & Commands):** The command list on line 621-627 was written when more commands existed or were planned. `frontier`, `rework`, `reflect`, and `kb archive-old` were either removed, renamed, or never implemented.

### Impact Assessment

The `--opus` flag staleness is the highest-impact issue. This is the most-loaded skill (~37k tokens in every orchestrator session). Every orchestrator that tries `--opus` will get an error and need to figure out the correct flag. Given that the skill explicitly recommends `--opus` for 6 skill types (systematic-debugging, investigation, architect, codebase-audit, research, design-session), this error fires frequently.

The `orch frontier` reference in the Proactive Hygiene Checkpoint (line 320) is the second-highest impact because it's a mandatory session-boundary action that can't be performed.

### Commands That DO Exist But Aren't Referenced

The skill doesn't mention several commands that exist and may be useful for orchestrators:
- `orch hotspot` (referenced in CLAUDE.md but not in the skill's Tools & Commands)
- `orch learn` (learning loop for gap suggestions)
- `orch deploy` (atomic rebuild/restart)
- `orch reconcile` (zombie issue detection)
- `orch changelog` (aggregated changelog)
- `orch patterns` (behavioral patterns)

These are informational, not harmful — the skill could benefit from documenting them but their absence doesn't cause errors.
