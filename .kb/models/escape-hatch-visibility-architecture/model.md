# Model: Backend Independence & Visibility Architecture

**Purpose:** Documents how the Claude CLI backend provides infrastructure independence via tmux, and how dual-window Ghostty setup maximizes visibility for all tmux-based agent work.

**Created:** 2026-01-13
**Last Updated:** 2026-03-18 (decay probe — reframed from "escape hatch" to "backend independence")
**Status:** Active
**Related Guides:** resilient-infrastructure-patterns.md

---

## Problem Statement

When working on critical infrastructure (OpenCode server, orch daemon, dashboard services), you need a backend that:
1. **Doesn't depend on what might fail** (independence)
2. **Shows you what's happening** (visibility)
3. **Can complete the work** (capability)

Since Feb 19, 2026 (Anthropic OAuth ban), the Claude CLI backend — which satisfies all three criteria — is the **default backend** for all Anthropic model work. What was once an opt-in "escape hatch" is now the primary operating mode.

The visibility requirement creates an architectural dependency: **tmux-based spawning benefits from dual-window Ghostty setup** for real-time observation.

---

## Historical Context

| Date | State | Primary Path | Escape Hatch |
|------|-------|-------------|--------------|
| Jan 2026 | Original | OpenCode API (headless) | Claude CLI (tmux) |
| Feb 19, 2026 | OAuth ban | Claude CLI (tmux) | N/A — independence is default |
| Mar 2026 | Current | Claude CLI (tmux) for Anthropic; OpenCode API for non-Anthropic | Auto-detected for infrastructure work |

The "escape hatch" terminology is historical. The principle (backend independence) remains valid; the opt-in framing does not.

---

## Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│             Default Spawn (Anthropic models)                 │
│  orch spawn feature-impl "task" --issue ID                  │
│  (spawn_mode: claude via .orch/config.yaml)                 │
└─────────────────────────────────────────────────────────────┘
                              ↓
        ┌─────────────────────┼─────────────────────┐
        ↓                     ↓                     ↓
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  Claude CLI      │  │  Tmux window     │  │  Opus model      │
│  backend         │  │  (automatic)     │  │  (default)       │
│                  │  │                  │  │                  │
│  Independence:   │  │  Visibility:     │  │  Capability:     │
│  - Uses Claude   │  │  - See progress  │  │  - Best model    │
│    CLI not API   │  │  - Intervene     │  │  - Can handle    │
│  - Independent   │  │  - Debug         │  │    hard tasks    │
│    of OpenCode   │  │                  │  │                  │
│    server        │  │  (no --tmux flag │  │                  │
│  - Survives      │  │   needed; claude │  │                  │
│    crashes       │  │   backend always │  │                  │
│                  │  │   uses tmux)     │  │                  │
└──────────────────┘  └──────────────────┘  └──────────────────┘
                              ↓
                   ENHANCED BY (for visibility)
                              ↓
┌─────────────────────────────────────────────────────────────┐
│           Dual-Window Ghostty Setup                         │
│                                                             │
│  ┌───────────────────┐        ┌───────────────────┐        │
│  │ Left Ghostty      │        │ Right Ghostty     │        │
│  │                   │        │                   │        │
│  │ orchestrator      │        │ workers-{project} │        │
│  │ tmux session      │        │ tmux session      │        │
│  │                   │        │                   │        │
│  │ - Delegation      │ ─────► │ - Agent windows   │        │
│  │ - Monitoring      │ auto-  │ - Servers window  │        │
│  │ - orch commands   │ switch │ - Visual progress │        │
│  └───────────────────┘        └───────────────────┘        │
└─────────────────────────────────────────────────────────────┘
                              ↓
                          ENABLES
                              ↓
┌─────────────────────────────────────────────────────────────┐
│              Intervention Capability                        │
│                                                             │
│  - Observe agent progress in real-time                      │
│  - Send messages mid-execution (orch send)                  │
│  - Attach to session to see full context                    │
│  - Kill if stuck (tmux kill-window)                         │
│  - Survives OpenCode crashes (tmux persists)                │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Relationships

### 1. Backend Selection Logic

The `DetermineSpawnBackend()` function in `pkg/orch/spawn_backend.go` resolves backend via this priority:

1. Explicit `--backend` flag (highest)
2. Explicit `--model` flag or user `default_model` config
3. Project config `spawn_mode` (`.orch/config.yaml`)
4. User config `backend` (`~/.orch/user.yaml`)
5. Infrastructure work auto-detection → forces `claude`
6. Default: `opencode`

For orch-go, step 3 resolves to `claude` via `spawn_mode: claude` in `.orch/config.yaml`, so all Anthropic spawns use Claude CLI + tmux by default.

### 2. Claude Backend → Tmux (Automatic)

When `spawn_mode: claude`, ALL spawns create tmux windows automatically via `runSpawnClaude()` in `pkg/orch/spawn_modes.go`. No `--tmux` flag is needed. The `--tmux` flag is only relevant for the `opencode` backend path.

### 3. Dual-Window Setup → Zero-Step Observation

The dual-window architecture enables real-time intervention:

```
Left Ghostty (orchestrator)          Right Ghostty (workers)
─────────────────────────            ────────────────────────
$ orch spawn ...             ───►   [Agent window appears]
$ orch status                        [Watch progress live]
$ orch send <id> "status?"   ───►   [Agent receives message]
```

**Without dual-window:**
```
$ orch spawn ...
$ orch status                        [Says "running" but can't see it]
$ tmux list-sessions                 [Find the session name]
$ tmux attach -t workers-X           [Finally see the window]
  [Now in worker session, can't run orch commands]
```

Single-window adds ~5 steps to check progress. Dual-window makes it zero steps.

---

## Architectural Constraints

### Constraint 1: Dual-Window Maximizes Tmux Visibility

**Why:** Since Claude backend (tmux) is now the default, visibility matters for ALL work:
- Immediate visual confirmation agent started
- Continuous progress monitoring without context switching
- Fast intervention if agent gets stuck

Single-window setup still works but requires manual `tmux attach` for observation.

### Constraint 2: Dual-Window Setup REQUIRES Auto-Switch Hook

**Why:** Without the `after-select-window` hook in `.tmux.conf.local`:
- Right Ghostty stays on old project when you switch contexts
- Defeats auto-following behavior

**Implementation:** The tmux hook at `~/.tmux.conf.local:62`:
```bash
set-hook -g after-select-window 'run-shell -b "~/.local/bin/sync-workers-session.sh #{session_name} #{pane_current_path} #{pane_pid} #{client_tty}"'
```

**Historical note:** Hook was disabled 2026-01-08 due to unwanted switches with single-window setup. Re-enabled for dual-window usage.

### Constraint 3: Non-Anthropic Models Use Headless (No Tmux)

Non-Anthropic models (GPT, Gemini, DeepSeek) route through the OpenCode API backend:
- Headless (no tmux window)
- Monitoring via dashboard at http://localhost:5188
- Dual-window setup is irrelevant for these spawns

---

## Decision Points

### When do I benefit from dual-window setup?

```
Are you using Claude backend (tmux spawns)?
  ├─ YES → Dual-window gives zero-step observation
  │         (Recommended for all tmux-based work)
  └─ NO  → Using OpenCode backend (headless)
            → Dashboard monitoring, dual-window irrelevant
```

### Infrastructure auto-detection

The `isInfrastructureWork()` function in `pkg/orch/spawn_backend.go` auto-detects infrastructure tasks by keyword matching and forces `claude` backend. This is largely redundant when `spawn_mode: claude` is already configured, but serves as a safety net for projects without explicit config.

---

## Evolution

### Phase 1: Single-Window (Jan 2026)
- One Ghostty window
- Claude CLI was opt-in "escape hatch"
- Manual `tmux attach` to observe
- Hook disabled (caused unwanted switches)

### Phase 2: Dual-Window + Escape Hatch (Jan 2026)
- Two Ghostty windows (orchestrator + workers)
- Auto-switching via tmux hook
- Escape-hatch spawning for critical work
- Hook enabled (required for auto-switch)

### Phase 3: Claude Default (Feb 2026 - Current)
- Claude CLI became default backend (OAuth ban)
- ALL Anthropic spawns create tmux windows
- Dual-window benefits ALL work, not just "escape hatch"
- "Escape hatch" terminology became vestigial

---

## Verification Checklist

For optimal visibility with tmux-based spawns:

- [ ] Two Ghostty windows open (left: orchestrator, right: workers)
- [ ] tmux hook enabled: `tmux show-hooks -g | grep after-select-window`
- [ ] Sync script exists: `ls ~/.local/bin/sync-workers-session.sh`
- [ ] Auto-switch works: Switch orchestrator windows, right Ghostty follows

Without these, tmux windows are created but require manual attachment.

---

## Anti-Patterns

### Using tmux spawns without dual-window setup
**Problem:** Window created but invisible, must manually attach
**Fix:** Enable dual-window setup or use `orch attach` for ad-hoc observation

### Disabling tmux hook for convenience
**Problem:** Auto-switch breaks, right Ghostty doesn't follow context
**Fix:** Keep hook enabled when using dual-window workflow

---

## References

**Guides:**
- `.kb/guides/resilient-infrastructure-patterns.md` - Backend independence criteria (independence, visibility, capability)

**Code:**
- `pkg/orch/spawn_backend.go` - Backend selection logic (`DetermineSpawnBackend()`)
- `pkg/orch/spawn_modes.go` - Spawn mode routing (`DispatchSpawn()`, `runSpawnClaude()`)
- `cmd/orch/spawn_cmd.go` - Spawn command with `--backend` flag

**Config:**
- `~/.tmux.conf.local:62` - Auto-switch hook
- `~/.local/bin/sync-workers-session.sh` - Sync script for auto-switching
- `.orch/config.yaml` - Project-level `spawn_mode: claude`

**Constraints (kn):**
- kb-bf4f55: "Critical paths need independent escape hatches"

---

## Real-World Examples

### Example 1: Dashboard Reliability Crisis (Jan 10, 2026)

**Scenario:** OpenCode server crashing while building observability fixes

**With dual-window:**
```bash
orch spawn --backend claude --tmux feature-impl "fix crash"
# Right Ghostty immediately shows agent window
# Left Ghostty still available for orch commands
# Zero manual steps to observe progress
```

**Outcome:** 3 agents survived server crashes, completed fixes, because tmux independence + dual-window visibility enabled intervention.

### Example 2: Post-OAuth-Ban Default (Feb 2026+)

**Scenario:** All Anthropic spawns now use Claude CLI + tmux by default

```bash
orch spawn feature-impl "implement feature" --issue proj-123
# spawn_mode: claude in .orch/config.yaml
# Automatically creates tmux window — no --backend or --tmux flags needed
# Dual-window shows agent immediately in right Ghostty
```

**Key shift:** No explicit escape-hatch flags needed. The default path now provides independence + visibility.

---

## Summary

**Core insight:** Backend independence (Claude CLI + tmux) is now the default operating mode for Anthropic models. Dual-window Ghostty setup enhances visibility for ALL tmux-based work, providing zero-step observation and intervention capability.

```
Tmux-Based Spawns (default for Anthropic)
  → Independence: Claude CLI, no OpenCode dependency
  → Visibility: tmux windows (always created)
    → Enhanced by Dual-Window Setup
      → Requires Auto-Switch Hook (~/.tmux.conf.local:62)
```

---

## Probes

- 2026-03-18: Knowledge decay verification — reframed model from "escape hatch" to "backend independence." Claude CLI is now default, not opt-in. Fixed stale file references, updated decision trees, removed dead link to orch-knowledge docs.

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-14-inv-track-escape-hatch-spawn-usage.md
- .kb/investigations/archived/2026-01-14-inv-track-escape-hatch-stats-impl.md
