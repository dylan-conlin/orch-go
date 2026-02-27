# Session Synthesis

**Agent:** og-debug-fix-ghostty-workers-25feb-ad27
**Issue:** orch-go-1253
**Duration:** 2026-02-25 18:33 → 2026-02-25 18:50
**Outcome:** success

---

## Plain-Language Summary

The Ghostty workers window wasn't following orchestrator window switches because of two independent bugs. First, the tmux hook runs the sync script in the background (`run-shell -b`), and when that script calls `tmux display-message -p '#{session_name}'` to detect which session it's in, tmux picks an arbitrary client — sometimes the workers client instead of the orchestrator. This made the script think it wasn't in the orchestrator and exit early. Second, for the toolshed project, the directory name (`scs-special-projects`) doesn't match the workers session name (`workers-toolshed`), so even when the session check worked, the name lookup failed.

The fix passes all tmux context (session name, CWD, PID, client TTY) as arguments from the hook definition — where tmux expands format strings in the correct context — instead of querying from inside the background script. For the name mismatch, the script now reads `claude.tmux_session` from `.orch/config.yaml`, with fallback to basename convention and a reverse-lookup for unambiguous cases.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key verification is Dylan switching orchestrator windows between projects and confirming the workers Ghostty follows.

---

## Delta (What Changed)

### Files Modified
- `~/.local/bin/sync-workers-session.sh` — Rewrote to accept context as arguments, added config-based session name resolution with 3-tier priority (config → basename → reverse lookup)
- `~/.tmux.conf.local` (line 61) — Updated hook to pass `#{session_name} #{pane_current_path} #{pane_pid} #{client_tty}` as arguments
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/.orch/config.yaml` — Added `claude.tmux_session: workers-toolshed` for explicit mapping

### Files Created
- `.kb/models/follow-orchestrator-mechanism/probes/2026-02-25-probe-run-shell-background-context-loss.md` — Probe documenting the context loss bug and model impact

---

## Evidence (What Was Observed)

### Root Cause 1: run-shell -b context loss
- `tmux display-message -p '#{session_name}'` from inside a `run-shell -b` script returned `workers-toolshed` instead of `orchestrator`
- Same format expanded in the hook's command string returned `orchestrator` correctly
- `run-shell -b` executes asynchronously with no inherent client context; `display-message` picks an arbitrary client

### Root Cause 2: Directory basename ≠ workers session name
- Orchestrator window "to" has CWD `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects`
- Script derived `workers-scs-special-projects` from basename — session doesn't exist
- Correct session is `workers-toolshed` (subdirectory-level project)

### Tests Run
```bash
# Simulated orchestrator switching to orch-go
bash ~/.local/bin/sync-workers-session.sh orchestrator /Users/dylanconlin/Documents/personal/orch-go 43854 /dev/ttys000
# Result: workers client switched from workers-toolshed to workers-orch-go ✓

# Simulated orchestrator switching to toolshed
bash ~/.local/bin/sync-workers-session.sh orchestrator /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects 51041 /dev/ttys000
# Result: workers client switched from workers-orch-go to workers-toolshed ✓

# Empty CWD fallback (Claude Code scenario)
bash ~/.local/bin/sync-workers-session.sh orchestrator "" 43854 /dev/ttys000
# Result: lsof fallback resolved to orch-go, switch successful ✓
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `run-shell -b` in tmux hooks loses client context — background shell processes have no inherent client association. All format variables must be expanded in the hook definition, not queried from within the script.
- Monorepo projects where parent has `.orch/` but workers sessions correspond to subdirectories require explicit `tmux_session` config.

### Decisions Made
- Use 3-tier session name resolution: explicit config → basename convention → unique reverse lookup. Ambiguous reverse lookups (2+ matches) require explicit config rather than guessing.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (script fixed, hook updated, config added)
- [x] Tests passing (manual simulation of both switch directions verified)
- [x] Probe documented
- [x] Ready for `orch complete orch-go-1253`

**Note:** Full end-to-end verification requires Dylan to switch orchestrator windows from the orchestrator Ghostty terminal. The worker agent cannot trigger hooks with the correct client_tty from its terminal.

---

## Unexplored Questions

- **Global hook firing for all sessions**: The `-g` hook fires for every session's window selection. This means the script runs (and exits quickly) even for workers session window selections. Not a problem (guard clause exits), but could be changed to a session-specific hook: `set-hook -t orchestrator after-select-window ...`
- **Follow-orchestrator model staleness**: The model was last updated 2026-01-15 and needs update with the two new failure modes documented in the probe.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-fix-ghostty-workers-25feb-ad27/`
**Probe:** `.kb/models/follow-orchestrator-mechanism/probes/2026-02-25-probe-run-shell-background-context-loss.md`
**Beads:** `bd show orch-go-1253`
