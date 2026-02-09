# System Reliability Model (Feb 2026 Crisis → Recovery)

**Summary:** The orch ecosystem suffered chronic instability from a single root cause — unbounded resource consumption without lifecycle management — manifesting across 5 components. A coordinated intervention (Feb 7) deployed 4 fixes that reduced CPU from 75% to ~12% and eliminated an entire process from the runtime. Phase 3 stability measurement is now active. This model captures the failure taxonomy, intervention chain, and remaining exposure.

**Synthesized from:** 6 investigations, 2 decisions, direct orchestrator engagement.

---

## Core Mechanism: The Unbounded Resource Pattern

Every major reliability failure in the orch ecosystem shares the same DNA:

```
Resource created → No lifecycle boundary → Accumulates → System pressure → Crash/degradation
```

This shipped 5 times across different components because nothing in the development process checked for it — not agent prompts, not code review, not linting, not investigation synthesis.

### The Five Failure Modes (Same Root Cause)

| # | Component | Resource | Accumulation | Kill Mechanism | Status |
|---|-----------|----------|-------------|----------------|--------|
| 1 | OpenCode server | Instance cache (per-directory) | Unbounded Map<string, Context> | macOS jetsam at 8.4GB RSS | Fixed: LRU/TTL eviction |
| 2 | Dashboard (bun) | Dev server processes | Orphaned on crash/restart (PPID=1) | CPU saturation (75%+) | **Eliminated:** Static build |
| 3 | bd subprocesses | Shell-out processes | Dashboard → bd comments per agent, no cap | CPU spikes, system freeze | Fixed: 12-max semaphore, 10s timeout |
| 4 | Beads SQLite | WAL file | Daemon rapid-restart race conditions | Database corruption | Previously fixed: JSONL migration |
| 5 | bd sync | Memory allocation | JSONL import loads everything in-memory | OOM kill | Mitigated: bd-sync-safe.sh |

### Why It Repeated 5 Times

The 779 investigations in ~2 months were symptom-level, not pattern-level. Each investigation found a bug and recommended a fix, but no synthesis connected them as instances of the same defect class. The constraint decision (2026-02-07) establishes structural prevention.

---

## The Intervention Chain (Feb 7, 2026)

### Phase 0: Emergency Stabilization
- Killed 13 orphaned bun processes
- CPU: 75% → 38%

### Phase 1: Targeted Fixes
| Fix | What | Impact |
|-----|------|--------|
| 1a: OpenCode Instance Eviction | LRU max 20, 30min TTL for idle instances | Prevents 8.4GB growth |
| 1b: Bun Zombie Prevention | Wrapper script kills stale bun before restart | Superseded by Phase 2 |
| 1c: bd Subprocess Hardening | 12-max semaphore, 10s timeout, singleflight dedup | Prevents CPU stampedes |

### Phase 2: Structural Simplification
- Eliminated bun dev server entirely
- Go binary serves pre-built Svelte assets from `web/build/`
- **4 processes → 3 processes** (api + daemon + opencode)
- Dashboard URL: `localhost:5188` → `localhost:3348`
- Entire zombie bun failure category removed

### Results

| Metric | Before | After |
|--------|--------|-------|
| Bun processes | 15 (13 zombies) | 1 (managed) |
| Runtime processes | 4 | 3 |
| bd subprocess cap | None (stampedes of 20+) | 12 max, 10s timeout |
| OpenCode memory governance | None (8.4GB before jetsam) | LRU max 20, 30min TTL |
| CPU (orch ecosystem) | 75% | ~12% |

---

## Why OpenCode Dies (Detailed)

The deep investigation chain (3 investigations) established:

1. **Kill mechanism:** macOS jetsam (kernel memory pressure subsystem), not JS errors, not overmind, not file descriptors
2. **Growth mechanism:** Unbounded per-directory Instance cache → each workspace triggers bootstrap (10 subscriptions per instance) → 120 directories = +50MB RSS without any model calls
3. **Scale context:** OpenCode (8.4GB) + 26 agent processes (14.3GB) = 22.7GB on 36GB system → 0.3GB free → jetsam
4. **SSE cleanup gap:** Client-abort path cleans up, but server-initiated close path skips cleanup (connected=40/disconnected=0 in stress test)
5. **Self-healing was disabled:** Overmind's `--can-die opencode` meant killed processes stayed dead. `--auto-restart` exists but was unused.

**Fix applied:** LRU/TTL eviction (max 20 live, 30min idle TTL) + SSE shared teardown path. Deployed in opencode fork.

**Remaining exposure:**
- Heap-object-level attribution untested (Bun heap inspector not attachable)
- ACP session manager has no eviction policy
- Multi-day production growth curve not yet validated post-fix

---

## Stability Measurement (Phase 3)

### Design
- Dedicated `~/.orch/stability.jsonl` with periodic health snapshots + intervention detection
- Doctor daemon hooks: snapshots every 5min, manual-recovery detection via health state transitions
- Streak computation: time since last infrastructure intervention

### Intervention Categories

| Source | Resets Streak? | Rationale |
|--------|---------------|-----------|
| `manual_recovery` | Yes | Infrastructure failure requiring human fix |
| `doctor_fix` | Yes | `orch doctor --fix` manual invocation |
| `agent_abandoned` | **No** | Hygiene operation, not infrastructure failure |

The crash-free streak false positive investigation found that `orch abandon` was incorrectly resetting the streak. Fix: filter `agent_abandoned` from streak calculation (aligns with observation/intervention decision).

### Process Census False Positives
- `isOrchRelatedProcess()` was too broad — flagged launchd-managed processes (overmind, tmux) as orphans
- Fix: whitelist legitimate PPID=1 processes before keyword matching

### Success Criterion
> One week of sessions without manual recovery intervention.

If stable → reliability focus lifts, feature work resumes.
If not → investigate which failure mode persists, deploy targeted fix, reset the clock.

---

## Structural Prevention (Decision: Unbounded Resource Constraints)

Five permanent constraints adopted to make this defect class structurally detectable:

| Constraint | Enforcement | Status |
|-----------|-------------|--------|
| C1: Every goroutine/subprocess/cache has bounded lifetime | golangci-lint custom rule | ⏳ Needs implementation within 1 week |
| C2: Process creation requires process cleanup | Pre-commit hook | ⏳ Needs implementation within 1 week |
| C3: Spawn prompts include resource audit scope | SPAWN_CONTEXT template | ⏳ Needs template update |
| C4: Caches require max-size at construction | API design (constructor) | ⏳ Design-level |
| C5: Weekly resource-class investigation synthesis | Manual (Fridays) | Active |

**Risk:** C1 and C2 must ship as automated checks within 1 week or they're decorative. Constraints without enforcement atrophy.

---

## Remaining Exposure (What Could Still Break)

| Risk | Severity | Mitigation |
|------|----------|------------|
| OpenCode memory growth post-fix not validated | Medium | Monitor RSS over 48h+ with new eviction |
| ACP session manager unbounded | Low | Not exercised in current workflows |
| bd-sync-safe.sh is workaround, not fix | Low | Acceptable for current usage patterns |
| C1/C2 enforcement not yet automated | High | Must ship within 1 week per decision |
| Doctor daemon itself could fail | Low | Launchd manages it; self-healing design |

---

## References

### Investigations (Provenance Chain)
- `2026-02-07-inv-system-reliability-crisis-diagnosis-and-fix.md` — Meta-diagnosis, 4 fix deployment
- `2026-02-07-inv-actually-kills-opencode-server-process.md` — Jetsam kill mechanism discovery
- `2026-02-07-inv-opencode-server-memory-leak-4gb.md` — Instance cache retention root cause
- `2026-02-07-inv-crash-free-streak-false-positive-agent-abandoned.md` — Streak metric accuracy
- `2026-02-07-inv-process-census-false-positives.md` — Orphan detection accuracy
- `2026-02-07-design-automatic-stability-measurement.md` — Phase 3 measurement design

### Decisions
- `2026-02-07-unbounded-resource-consumption-constraints.md` — Structural prevention
- `2026-02-07-static-dashboard-eliminate-bun-dev-server.md` — Process elimination
- `2026-01-28-two-tier-disk-cleanup-infrastructure.md` — Threshold-triggered recovery for disk pressure
- `2026-01-21-colima-over-docker-desktop.md` — Runtime stability decision for local Docker workflows
- `2026-01-14-understanding-lag-pattern.md` — Interpret new observability as visibility, not automatic degradation
- `2026-01-14-trust-calibration-assert-knowledge.md` — Human expertise assertion as reliability guardrail

### Prior Art
- `2026-01-22-inv-strategic-audit-daemon-reliability-multiple.md`
- `2026-01-26-inv-analyze-local-share-opencode-crash.md`
- `2026-01-23-inv-opencode-server-crashes-under-load.md`
