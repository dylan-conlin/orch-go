# Session Synthesis

**Agent:** og-inv-trace-account-distribution-26feb-6f06
**Issue:** orch-go-1111
**Outcome:** success

---

## Plain-Language Summary

The automatic account distribution system is **fully wired** end-to-end through the spawn flow. When you run `orch spawn` or `orch work`, the system checks capacity on your "work" (primary, 20x) account first. If it's below 20% remaining on either the 5-hour or 7-day limit, it automatically switches to the "personal" (spillover, 5x) account. This works by injecting `CLAUDE_CONFIG_DIR=~/.claude-personal` into the Claude CLI launch command so the agent runs under the alternate account's credentials. The prior probe from Feb 24 found zero integration — every gap it identified has since been closed. This probe's own SPAWN_CONTEXT.md demonstrates the system working live: account resolved to "personal" via spillover heuristic because work's 5h usage was at 95%.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key outcomes:
- 22 tests pass (10 account resolution + 12 launch command including configDir injection)
- Complete call chain traced from CLI entry through resolver through env var injection
- Prior probe's gap analysis table (6 gaps) all verified closed

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestration-cost-economics/probes/2026-02-26-probe-account-distribution-wiring-trace.md` - Probe confirming full account distribution wiring

### Files Modified
- None (investigation-only, no code changes)

---

## Evidence (What Was Observed)

- `resolveAccount()` exists at `pkg/spawn/resolve.go:429-505` with 3-tier precedence (CLI → heuristic → default)
- `buildCapacityFetcher()` at `cmd/orch/shared.go:425-464` creates process-level CapacityCache with 5-min TTL
- `BuildClaudeLaunchCommand()` at `pkg/spawn/claude.go:59-88` injects `CLAUDE_CONFIG_DIR` and unsets `CLAUDE_CODE_OAUTH_TOKEN`
- `SpawnClaude()` at `pkg/spawn/claude.go:130` passes `cfg.AccountConfigDir` to the launch command builder
- `accounts.yaml` has work(primary, 20x, ~/.claude) and personal(spillover, 5x, ~/.claude-personal)
- OpenCode backend correctly does NOT use account distribution (different auth mechanism)
- Daemon spawns via `orch work` which calls `runSpawnWithSkillInternal()` → heuristic runs inside subprocess

### Tests Run
```bash
go test ./pkg/spawn/ -run "Account|BuildClaudeLaunchCommand" -v
# 22 tests pass: 10 account resolution + 12 launch command (0.008s)
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `~/.claude` (work account's configDir) is the default, so it's never injected via env var — only `~/.claude-personal` gets explicit injection. This is correct behavior, not a gap.
- `buildCapacityFetcher()` returns nil with <2 accounts or no roles → zero overhead for single-account setups

### Externalized via `kn`
- No new knowledge to externalize. Probe confirms existing model claims. Leave it Better: Straightforward investigation, probe confirms what the model says.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (22/22)
- [x] Probe file has Status: Complete
- [x] Ready for `orch complete orch-go-1111`

---

## Unexplored Questions

Straightforward session, no unexplored territory.

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-trace-account-distribution-26feb-6f06/`
**Probe:** `.kb/models/orchestration-cost-economics/probes/2026-02-26-probe-account-distribution-wiring-trace.md`
**Beads:** `bd show orch-go-1111`
