# Probe: run-shell -b loses tmux client context in hooks

**Model:** follow-orchestrator-mechanism
**Date:** 2026-02-25
**Status:** Complete

---

## Question

The model documents the sync script using `tmux display-message -p '#{session_name}'` to detect the orchestrator session. Does this work correctly when invoked from `run-shell -b` in a global `after-select-window` hook?

---

## What I Tested

1. Set hook to log `display-message` results from within the script:
```bash
# Inside script called from run-shell -b:
CURRENT_SESSION=$(tmux display-message -p '#{session_name}')
# → Returns "workers-toolshed" (WRONG, should be "orchestrator")
```

2. Set hook to expand formats in the hook definition itself:
```bash
set-hook -g after-select-window 'run-shell -b "echo #{session_name} >> /tmp/log"'
# → Expands to "orchestrator" (CORRECT)
```

3. Compared: tmux expands `#{...}` formats in the hook's command string BEFORE spawning the background shell. But inside the shell script, `tmux display-message -p` makes a NEW client connection that resolves to an arbitrary client.

---

## What I Observed

- **`tmux display-message -p` from inside `run-shell -b`**: Returns wrong session (whichever client tmux considers "most recently active"). Non-deterministic — sometimes returns orchestrator, sometimes workers.
- **`#{session_name}` expanded in hook command string**: Always returns correct session (the one where the window was selected).
- **Root cause**: `run-shell -b` executes asynchronously. The background process has no inherent client context. When the script calls `tmux display-message -p`, tmux must pick a client arbitrarily.

**Second bug found**: Directory basename `scs-special-projects` doesn't match workers session name `workers-toolshed` (monorepo with nested `.orch/` directories at both parent and child levels).

---

## Model Impact

- [x] **Contradicts** invariant: The model documents the sync script using `tmux display-message -p '#{session_name}'` as reliable detection. It is NOT reliable from `run-shell -b` — the client context is racy.
- [x] **Extends** model with: Two-bug fix pattern:
  1. Pass context via tmux format expansion in the hook definition (expanded before shell spawns)
  2. Add config-based session name resolution (`claude.tmux_session` in `.orch/config.yaml`) for projects where directory basename ≠ workers session name

---

## Notes

- The model's Failure Mode 1 (empty pane_current_path) and Failure Mode 2 (wrong tmux socket) are still valid.
- New Failure Mode 6: `run-shell -b` context loss — `display-message` inside background scripts resolves to arbitrary client.
- New Failure Mode 7: Directory basename ≠ workers session name (monorepo/nested project case).
- Fix requires passing `#{session_name} #{pane_current_path} #{pane_pid} #{client_tty}` as args from the hook, and reading `tmux_session:` from `.orch/config.yaml`.
