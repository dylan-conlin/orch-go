# Session Synthesis

**Agent:** og-inv-tmux-readiness-probe-18feb-541a
**Issue:** orch-go-1039
**Outcome:** success

---

## Plain-Language Summary

The tmux spawn readiness timeout (orch-go-1034, P0) is NOT caused by slow TUI startup or a broken readiness detection function. The `IsOpenCodeReady` function works correctly and detects the TUI in ~3 seconds. The actual root cause is that `BuildOpencodeAttachCommand` adds a `--model` flag to the `opencode attach` command, but `opencode attach` doesn't support `--model`. This causes OpenCode to display help text instead of starting the TUI, so the readiness probe polls an empty window for 15 seconds and times out. Since a model is always resolved (default: sonnet), every tmux spawn fails.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for exact commands and expected outcomes.

---

## Delta (What Changed)

### Files Modified
- `.kb/models/spawn-architecture/probes/2026-02-18-tmux-readiness-timeout.md` - Completed probe with root cause analysis

### Commits
- (pending)

---

## Evidence (What Was Observed)

- `IsOpenCodeReady` returns true for real OpenCode TUI content in <3ms (tmux.go:551-563)
- Fresh `opencode attach` (without `--model`) reaches ready state in 3.06-3.56s — well within 15s timeout
- `opencode attach --help` shows NO `--model` flag (only: --dir, --session, --password)
- `opencode run --help` shows `--model` IS supported, but `run` requires a message
- Live price-watch window (workers-price-watch:2) showed help text output, confirming the failure mode
- `BuildOpencodeAttachCommand` (tmux.go:276-287) unconditionally adds `--model` when `cfg.Model != ""`
- `BuildSpawnConfig` (extraction.go:724) always sets `Model: ctx.ResolvedModel.Format()`
- Default model: `anthropic/claude-sonnet-4-5-20250929` (model.go:19-22)

### Tests Run
```bash
# IsOpenCodeReady against real and synthetic content
go run /tmp/test_readiness.go
# All edge cases PASS, real TUI content correctly detected

# Timing test without --model (works)
go run /tmp/test_spawn_timing2.go
# ✅ READY detected after 8 polls (latency: 3.55s)

# Timing test with --model via opencode attach (fails)
# Observed in workers-price-watch:2 — help text displayed, no TUI

# opencode run --attach --model (fails differently)
go run /tmp/test_run_attach.go
# Error: You must provide a message or a command
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/spawn-architecture/probes/2026-02-18-tmux-readiness-timeout.md` - Root cause probe

### Constraints Discovered
- `opencode attach` does not support `--model` flag — model must be set via different mechanism
- `opencode run --attach` supports `--model` but requires a message argument
- `~/.bun/bin` is not in PATH for new tmux windows — `OPENCODE_BIN` env var is required

### Externalized via `kb`
- (see below — kb quick entries)

---

## Next (What Should Happen)

**Recommendation:** close (probe complete, findings feed into orch-go-1034 fix)

### Discovered Work
- **orch-go-1034** (existing P0): Root cause now identified — fix should either remove `--model` from `BuildOpencodeAttachCommand` or add `--model` support to `opencode attach` in the fork

---

## Unexplored Questions

- Should the socket awareness gap (only `SessionExists` uses `tmuxCommand()`) be a separate issue? Affects daemon tmux spawns from overmind context.
- When did `--model` flag support diverge between `opencode attach` and `opencode run --attach`? Was it ever supported?

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-tmux-readiness-probe-18feb-541a/`
**Probe:** `.kb/models/spawn-architecture/probes/2026-02-18-tmux-readiness-timeout.md`
**Beads:** `bd show orch-go-1039`
