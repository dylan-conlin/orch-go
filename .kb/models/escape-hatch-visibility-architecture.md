# Model: Escape Hatch Visibility Architecture

**Purpose:** Documents how escape-hatch spawning requires dual-window Ghostty setup to satisfy the "visibility" criterion for critical infrastructure work.

**Created:** 2026-01-13
**Status:** Active
**Related Guides:** resilient-infrastructure-patterns.md, orchestration-window-setup.md

---

## Problem Statement

When working on critical infrastructure (OpenCode server, orch daemon, dashboard services), you need an escape hatch that:
1. **Doesn't depend on what might fail** (independence)
2. **Shows you what's happening** (visibility)
3. **Can complete the work** (capability)

The visibility requirement creates an architectural dependency: **escape-hatch spawning requires dual-window Ghostty setup.**

---

## Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│             Escape Hatch Spawn Command                      │
│  orch spawn --bypass-triage --backend claude --tmux \      │
│    --model opus feature-impl "fix crash" --issue ID         │
└─────────────────────────────────────────────────────────────┘
                              ↓
        ┌─────────────────────┼─────────────────────┐
        ↓                     ↓                     ↓
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  --backend       │  │  --tmux          │  │  --model opus    │
│  claude          │  │                  │  │                  │
│                  │  │  Creates visible │  │  Quality for     │
│  Independence:   │  │  tmux window     │  │  complex work    │
│  - Uses Claude   │  │                  │  │                  │
│    CLI not API   │  │  Visibility:     │  │  Capability:     │
│  - Independent   │  │  - See progress  │  │  - Best model    │
│    of OpenCode   │  │  - Intervene     │  │  - Can handle    │
│    server        │  │  - Debug         │  │    hard tasks    │
│  - Survives      │  │                  │  │                  │
│    crashes       │  │                  │  │                  │
└──────────────────┘  └──────────────────┘  └──────────────────┘
                              ↓
                      REQUIRES (for visibility)
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
│  ✓ Observe agent progress in real-time                     │
│  ✓ Send messages mid-execution (orch send)                 │
│  ✓ Attach to session to see full context                   │
│  ✓ Kill if stuck (tmux kill-window)                        │
│  ✓ Survives OpenCode crashes (tmux persists)               │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Relationships

### 1. Escape Hatch Criteria → Spawn Flags

| Criterion | Spawn Flag | Why |
|-----------|------------|-----|
| **Independence** | `--backend claude` | Claude CLI doesn't use OpenCode server (can't kill itself) |
| **Visibility** | `--tmux` | Creates tmux window you can observe |
| **Capability** | `--model opus` | Best reasoning quality for complex/critical work |

### 2. --tmux Flag → Dual-Window Setup

**The dependency:**
- `--tmux` creates a tmux window in the `workers-{project}` session
- Without dual-window setup, the window is created but **invisible** (buried in background session)
- You'd need to manually `tmux attach -t workers-{project}` to see it
- Defeats the "visibility" purpose of escape-hatch spawning

**With dual-window setup:**
- Right Ghostty is **already attached** to `workers-{project}` session
- Auto-switches to matching project when you spawn (via `after-select-window` hook)
- Agent window appears **immediately visible** in right Ghostty
- No manual switching needed

### 3. Dual-Window Setup → Intervention Capability

The dual-window architecture enables real-time intervention:

```
Left Ghostty (orchestrator)          Right Ghostty (workers)
─────────────────────────            ────────────────────────
$ orch spawn --tmux ...       ───►   [Agent window appears]
$ orch status                        [Watch progress live]
$ orch send <id> "status?"    ───►   [Agent receives message]
$ tmux attach -t workers-X    ───►   [Already there, see output]
```

**Without dual-window:**
```
$ orch spawn --tmux ...
$ orch status                        [Says "running" but can't see it]
$ tmux list-sessions                 [Find the session name]
$ tmux attach -t workers-X           [Finally see the window]
  [Now in worker session, can't run orch commands]
$ tmux detach
$ orch send ...                      [Back to orchestrator]
```

The single-window flow adds ~5 steps to check progress. Dual-window makes it **zero steps** (already visible).

---

## Architectural Constraints

### Constraint 1: Escape Hatch Spawning REQUIRES Dual-Window Setup

**Why:** The "visibility" criterion of escape-hatch architecture demands:
- Immediate visual confirmation agent started
- Continuous progress monitoring without context switching
- Fast intervention if agent gets stuck

Single-window setup violates these requirements (hidden windows, manual switching).

**Implication:** If you're doing critical infrastructure work (P0/P1, OpenCode itself, daemon), you **must** have dual-window setup enabled.

### Constraint 2: Dual-Window Setup REQUIRES Auto-Switch Hook

**Why:** Without the `after-select-window` hook in `.tmux.conf.local`:
- Right Ghostty stays on old project when you switch contexts
- Defeats auto-following behavior
- Agent window may not be visible (in wrong session)

**Implication:** The tmux hook at `~/.tmux.conf.local:61` must be enabled:
```bash
set-hook -g after-select-window 'run-shell -b "~/.local/bin/sync-workers-session.sh"'
```

**Historical note:** Hook was disabled 2026-01-08 due to unwanted switches with single-window setup. Must be re-enabled for dual-window usage.

### Constraint 3: Primary Path (Daemon) Does NOT Require Dual-Window

**Why:** Headless spawning (daemon + OpenCode API) doesn't create tmux windows:
- Agents run via HTTP API sessions
- No visual windows to display
- Monitoring via dashboard at http://localhost:5188

**Implication:** Dual-window setup is optional for normal workflow, **mandatory** for escape-hatch workflow.

---

## Decision Points

### When do I need dual-window setup?

```
Are you doing critical infrastructure work?
  ├─ NO  → Single-window fine, use dashboard for monitoring
  └─ YES → Is the work on OpenCode/daemon/dashboard itself?
             ├─ NO  → Probably fine with headless (daemon)
             └─ YES → MUST use dual-window + escape-hatch spawning
```

### When do I use escape-hatch spawning?

```
What am I working on?
  ├─ Feature/bug in application code
  │    → Normal workflow (daemon + headless)
  │
  ├─ P0/P1 infrastructure (OpenCode server, daemon, dashboard)
  │    → Escape hatch: --backend claude --tmux --model opus
  │       (Requires dual-window setup for visibility)
  │
  └─ Building fixes for crashes/instability
       → Escape hatch: --backend claude --tmux --model opus
          (System may crash while fixing it)
```

---

## Evolution

### Phase 1: Single-Window (Original)
- One Ghostty window
- Spawns create background tmux windows
- Manual `tmux attach` to observe
- Hook disabled (caused unwanted switches)

### Phase 2: Dual-Window (Current)
- Two Ghostty windows (orchestrator + workers)
- Auto-switching via tmux hook
- Immediate visibility for escape-hatch spawns
- Hook enabled (required for auto-switch)

### Phase 3: Unified Visibility (Future?)
- Dashboard shows tmux window output via SSE streaming
- Dual-window still preferred for critical work (can intervene)
- Single-window viable for read-only monitoring

---

## Verification Checklist

Before using escape-hatch spawning, verify:

- [ ] Two Ghostty windows open (left: orchestrator, right: workers)
- [ ] tmux hook enabled: `tmux show-hooks -g | grep after-select-window`
- [ ] Sync script exists: `ls ~/.local/bin/sync-workers-session.sh`
- [ ] Auto-switch works: Switch orchestrator windows, right Ghostty follows

Without these, `--tmux` flag creates invisible windows (defeats visibility criterion).

---

## Anti-Patterns

### ❌ Using --tmux without dual-window setup
**Problem:** Window created but invisible, must manually attach
**Fix:** Enable dual-window setup or use dashboard for headless monitoring

### ❌ Disabling tmux hook for convenience
**Problem:** Auto-switch breaks, right Ghostty doesn't follow context
**Fix:** Keep hook enabled when using dual-window workflow

### ❌ Using escape-hatch for non-critical work
**Problem:** Manual lifecycle (can't close via `orch complete`), slower, costs more
**Fix:** Reserve escape-hatch for P0/P1 infrastructure work only

---

## References

**Guides:**
- `.kb/guides/resilient-infrastructure-patterns.md` - Escape hatch criteria (independence, visibility, capability)
- `~/orch-knowledge/.orch/docs/orchestration-window-setup.md` - Dual-window architecture and auto-switching

**Config:**
- `~/.tmux.conf.local:58-61` - Auto-switch hook (must be enabled)
- `~/.local/bin/sync-workers-session.sh` - Sync script for auto-switching

**Constraints (kn):**
- kb-bf4f55: "Critical paths need independent escape hatches"
- kb-d562c9: "Never spawn OpenCode infrastructure work without --backend claude --tmux"

**Related Models:**
- Model selection guide has similar "when to use which path" decision trees

---

## Real-World Examples

### Example 1: Dashboard Reliability Crisis (Jan 10, 2026)

**Scenario:** OpenCode server crashing while building observability fixes

**Without dual-window:**
```bash
orch spawn --backend claude --tmux feature-impl "fix crash"
# Agent starts but window invisible
tmux list-sessions  # Find it manually
tmux attach -t workers-orch-go  # Switch to see it
# Now can't run orch commands without detaching
```

**With dual-window:**
```bash
orch spawn --backend claude --tmux feature-impl "fix crash"
# Right Ghostty immediately shows agent window
# Left Ghostty still available for orch commands
# Zero manual steps to observe progress
```

**Outcome:** 3 agents survived server crashes, completed fixes, because dual-window enabled intervention.

### Example 2: Session Resume Hook Debugging (Jan 13, 2026)

**Scenario:** Testing session resume hook injection in Claude Code

**Choice:** Used primary path (OpenCode API, headless)
- Not critical infrastructure (testing hook, not modifying orch-go spawn system)
- Crashes wouldn't prevent testing (hook is external)
- Dashboard sufficient for monitoring

**No dual-window needed** - this wasn't escape-hatch work.

---

## Summary

**Core insight:** The architectural choice of dual-window Ghostty setup isn't just "nice to have" - it's a **required component** of escape-hatch spawning architecture.

```
Critical Infrastructure Work
  → Requires Escape Hatch (independence + visibility + capability)
    → Visibility Requires --tmux Flag
      → --tmux Requires Dual-Window Setup
        → Dual-Window Requires Auto-Switch Hook
```

Remove any link in this chain and the visibility criterion fails.
