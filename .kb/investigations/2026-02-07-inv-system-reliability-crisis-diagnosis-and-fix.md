## Summary (D.E.K.N.)

**Delta:** Diagnosed system-wide reliability crisis (5 failure modes sharing one root cause: unbounded resource consumption) and deployed 4 fixes in one session, reducing CPU from 75% to ~12% and eliminating an entire process from the runtime.

**Evidence:** 13 orphaned bun processes consuming 40%+ CPU; bd subprocess stampedes of 15+ concurrent processes; OpenCode growing to 8.4GB RSS before macOS jetsam kill; 779 investigations in 2 months (fire-fighting signal).

**Knowledge:** All failure modes share the same DNA — unbounded resource consumption without lifecycle management. Reactive patches (caching, dedup) address symptoms. Structural simplification (eliminating processes entirely) removes failure categories.

**Next:** Phase 3: measure stability for 1 week (sessions without manual recovery intervention). If stable, reliability focus lifts and feature work resumes.

---

# Investigation: System Reliability Crisis — Diagnosis and Fix

**Question:** Why does the orch ecosystem crash every other session, and how do we fix it?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** Orchestrator (strategic meta-evaluation)
**Phase:** Complete (deployed, measuring)
**Status:** Complete

---

## Diagnosis

### The Meta-Pattern

Every failure mode shares the same DNA: **unbounded resource consumption without lifecycle management.**

- OpenCode: unbounded Instance cache → 8.4GB → jetsam kill
- Dashboard: bun dev server instances accumulate → 15 processes → CPU saturation
- bd: subprocess stampede → dozens of concurrent `bd comments` → CPU spike
- Daemon: rapid restart loop → WAL corruption (already resolved via JSONL migration)
- bd sync: unbounded memory allocation → OOM kill

### Five Failure Modes

| # | Component | Symptom | Root Cause | Frequency |
|---|-----------|---------|-----------|-----------|
| 1 | OpenCode server | 8.4GB RSS, macOS kills it | Unbounded Instance cache by directory key | Every few hours under load |
| 2 | Dashboard (bun) | 13+ zombie processes, 40% CPU | bun dev server orphans on restart | Every restart |
| 3 | bd subprocesses | CPU spikes, system freezes | Dashboard shells out to `bd comments` per agent, no cap | During dashboard refreshes |
| 4 | Beads SQLite | Database corruption | Daemon rapid-restart WAL race conditions | Resolved (JSONL-only) |
| 5 | bd sync | OOM kill | JSONL import loads everything into memory | Periodic |

### Scale Context

- 95,023 lines of Go code
- 3,209 commits (all since Dec 2025)
- 779 investigations in ~2 months
- 82 decisions

The investigation count is not a sign of learning — it's a sign of fighting fires.

---

## What We Deployed

### Phase 0: Emergency Stabilization
- Killed 13 orphaned bun processes (PPID=1, going back 20+ hours)
- **Impact:** CPU dropped from 75% to 38%

### Phase 1a: OpenCode Instance Eviction
- **Repo:** opencode fork (`~/Documents/personal/opencode`)
- **Change:** LRU/TTL eviction on Instance cache (max 20 live, 30min TTL for idle)
- **Change:** SSE `/event` route cleanup — shared teardown for abort and close paths
- **Files:** `packages/opencode/src/project/instance.ts`, `packages/opencode/src/server/server.ts`

### Phase 1b: Bun Zombie Prevention
- **Change:** Wrapper script `scripts/dashboard-web-dev.sh` kills stale bun before starting, PID tracking
- **Note:** Superseded by Phase 2 (bun dev server eliminated entirely)

### Phase 1c: bd Subprocess Hardening
- **Change:** Hard cap of 3 concurrent bd subprocesses system-wide via semaphore
- **Change:** 10s timeout on all bd CLI calls
- **Change:** Singleflight dedup on `/api/agents` comments cache misses
- **Change:** Structured logging when cap is hit (`event=bd_subprocess_cap_hit`)
- **Files:** `cmd/orch/serve_bd_limiter.go`, `pkg/beads/client.go`, `cmd/orch/serve_agents_cache.go`
- **Validation:** Cap logging confirmed active immediately after deployment

### Phase 2: Static Dashboard Build
- **Change:** Eliminated `bun run dev` from Procfile entirely
- **Change:** Go binary serves pre-built Svelte assets from `web/build/`
- **Change:** SPA fallback (non-API routes serve `index.html`)
- **Change:** `orch-dashboard restart` auto-rebuilds web assets when source is newer
- **Impact:** 4 processes → 3 processes. Entire zombie bun failure category eliminated.
- **Decision:** `.kb/decisions/2026-02-07-static-dashboard-eliminate-bun-dev-server.md`
- **Dashboard URL changed:** `localhost:5188` → `localhost:3348`

---

## Results

| Metric | Before | After |
|--------|--------|-------|
| Bun processes (orch-go) | 15 (13 zombies) | 1 (managed) |
| Runtime processes | 4 (api + web + daemon + opencode) | 3 (api + daemon + opencode) |
| bd subprocess cap | None (stampedes of 20+) | 3 max, 10s timeout |
| OpenCode memory governance | None (grew to 8.4GB) | LRU max 20 instances, 30min TTL |
| CPU (orch ecosystem) | 75% | ~12% |

---

## Phase 3: Measurement (In Progress)

**Success metric:** Sessions without manual recovery intervention, sustained for 1 week.

**What "manual recovery" means:**
- Having to kill zombie processes
- Having to restart crashed services
- Having to rebuild corrupted databases
- Having to abandon agents killed by infrastructure failures

**If stable after 1 week:** Reliability focus lifts, feature work resumes.

**If not stable:** Investigate which failure mode persists, deploy targeted fix, reset the clock.

---

## References

- **Decision:** `.kb/decisions/2026-02-07-static-dashboard-eliminate-bun-dev-server.md`
- **Prior investigation:** `.kb/investigations/2026-02-07-inv-opencode-server-memory-leak-4gb.md`
- **Prior investigation:** `.kb/investigations/2026-01-22-inv-strategic-audit-daemon-reliability-multiple.md`
- **Prior investigation:** `.kb/investigations/2026-01-26-inv-analyze-local-share-opencode-crash.md`
- **Model:** `.kb/models/beads-database-corruption.md`
- **Guide:** `.kb/guides/resilient-infrastructure-patterns.md`
