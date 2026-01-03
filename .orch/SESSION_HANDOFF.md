# Session Handoff - Jan 3, 2026

## What Happened This Session

System spiraled Dec 27 - Jan 2. 347 commits, 40 "fixes," dashboard state machine grew from 5 to 7 states with time thresholds. Everything reported "working" while nothing actually worked. User lost trust.

**Actions taken:**
1. Rolled back to Dec 27 (`fb0af37f`)
2. Wrote post-mortem: `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md`
3. Implemented Dev/Ops mode protection system
4. Cherry-picked TTY detection fix
5. Manually added daemon skill inference (labels, title patterns)
6. Spawned investigation to analyze what's recoverable

## Dev/Ops Mode (NEW)

Structural protection to prevent agents modifying agent infrastructure during operations.

```bash
orch mode          # Show current mode (ops by default)
orch mode dev "reason"   # Enable infra changes
orch mode ops      # Protect infra
```

- Pre-commit hook blocks infra changes unless `.dev-mode` exists
- `orch status` shows warning when in dev mode
- Bypass: `ORCH_INFRA_BYPASS="reason" git commit` (logged)

**Protected paths:**
- cmd/orch/serve.go, main.go, status.go
- pkg/state/, pkg/opencode/
- web/src/lib/stores/agents.ts, daemon.ts
- web/src/lib/components/agent-card/

## Recovery Status

Investigation completed: `.kb/investigations/2026-01-03-inv-analyze-commits-between-fb0af37f-dec.md`

**Already recovered:**
- `4304b7dd` - TTY detection fix
- `75b0f389` - daemon skill inference (manually added)

**Priority 1 (Critical) - Still need:**
| Commit | Description | Notes |
|--------|-------------|-------|
| `10cc03ca` | headless spawn honors --model flag | Conflicts in main.go |
| `8b42ddd3` | headless spawn lifecycle cleanup | Conflicts in main.go |
| `735ac6a2` | full skill inference in spawn paths | Conflicts in main.go |
| `fb1bc009` | move triage:ready removal to complete | Conflicts in main.go |
| `b2b19b4a` | daemon skips failing issues | May be cleaner |
| `bbc95b5e` | daemon rate limiting | New feature |

**Priority 2 (High Value):**
- New CLI commands: reconcile, changelog, sessions, servers
- Verification gates: git diff, build verification
- Beads deduplication

**Approach:** Manual extraction recommended over cherry-pick due to conflicts in main.go

## Current State

```bash
orch mode         # ops (protected)
orch status       # Should show 3 idle agents from this session
orch doctor       # Services healthy
```

**Git status:**
- On master, ahead of origin by ~12 commits
- Clean working tree
- Tests pass

## Next Session Focus

Recover Priority 1 commits using manual extraction:
1. Read each commit's diff
2. Extract the relevant function/logic
3. Apply to current codebase
4. Test after each

Start with `b2b19b4a` (daemon skips failing issues) - likely cleaner since it's in pkg/daemon/ not main.go.

## Key Files

- Post-mortem: `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md`
- Recovery investigation: `.kb/investigations/2026-01-03-inv-analyze-commits-between-fb0af37f-dec.md`
- Mode history: `.orch/mode-history.jsonl`
