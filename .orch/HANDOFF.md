# Session Handoff — 2026-02-10 (Session 4: OpenCode)

## What Happened This Session

Fixed the recurring zombie bun process problem that caused 3+ forced reboots (WindowServer crash from RAM exhaustion). The root cause was that headless agent spawns (the default path) never write `.process_id` or ledger entries because OpenCode's server owns the bun process — orch never gets an `exec.Cmd` handle. Completed agents leave bun processes running indefinitely.

## What Was Fixed

### Commits (all pushed to remote)

1. **`de0a68be`** — `fix: add orch reap command and launchd safety net for zombie bun processes`
   - New `orch reap` standalone command (works without daemon)
   - `orch complete` now sweeps for orphaned processes after session deletion
   - Launchd plist runs `orch reap` every 5 minutes automatically
   - `ProcessStartTime()` for process age display

2. **`760757dc`** — `fix: tighten orphan detection to require --conditions=browser flag`
   - Prevents false matches against other bun projects with `src/index.ts` (chrome-devtools-mcp, clawdbot, anthropic-sdk, etc.)
   - `--conditions=browser` is OpenCode-specific

### Layered Defenses Now Active

1. **Launchd agent** (`com.orch.reap`) — runs every 5 min, kills orphans automatically. **Already installed and running.**
2. **`orch complete` sweep** — kills orphaned processes after session deletion
3. **`orch reap`** — manual standalone reaper (`--force` for post-crash, `--dry-run` to preview)
4. **Daemon reaper** — if daemon is running, reaps every 5 min
5. **`orch clean --processes`** — existing manual flag

### Remaining Limitation

When OpenCode API is down, `orch reap` (without `--force`) can't determine which processes are active, so it won't kill anything. The launchd agent has this same limitation. After a crash: use `orch reap --force` or `pkill -f 'bun.*src/index.ts'`.

### Not Fixed (Lower Priority)

- Process ledger still empty for headless spawns (`.process_id` not written). This is a defense-in-depth gap, not a blocking issue — the other layers compensate.
- Daemon still not running by default.

## System State at Handoff

- **Git:** 5 commits ahead of origin, all pushed. Unstaged: `.kb/quick/entries.jsonl`, `.orch/gate-skips.json`, `DYLANS_THOUGHTS.org` (operational artifacts, not code).
- **Stashes:** 4 (abandonment reason codes, claude config dir, circuit breaker removal, bd sync)
- **Build:** Clean, installed to `~/bin/orch`
- **Launchd:** `com.orch.reap` running (every 5 min). Logs: `~/.orch/logs/reap.log`
- **OpenCode:** Not running (post-reboot)
- **Dashboard:** Not running (post-reboot)
- **Daemon:** Not running
- **CLAUDE.md:** Updated with "Process Lifecycle & Zombie Prevention" section

## Resume Steps

1. **Start services:** `orch-dashboard start` (or `overmind start -D`)
2. **Verify reaper:** `orch reap --dry-run` (should query API now that OpenCode is up)
3. **Start daemon:** `orch daemon run` — issues waiting
4. **Sync beads:** `./scripts/bd-sync-safe.sh && git push`

## Key Files

- Reap command: `cmd/orch/reap_cmd.go`
- Orphan detection: `pkg/process/orphans.go` (`isOpenCodeAgentLine` — `--conditions=browser` filter)
- Complete sweep: `cmd/orch/complete_cleanup.go` (`sweepOrphanedProcessesAfterSessionDelete`)
- Process age: `pkg/process/starttime.go`
- Launchd plist: `scripts/com.orch.reap.plist`
- Install script: `scripts/install-reaper.sh`
- CLAUDE.md section: "Process Lifecycle & Zombie Prevention"
- Prior investigation: `.kb/investigations/2026-02-08-inv-design-process-lifecycle-cleanup-prevent.md`
