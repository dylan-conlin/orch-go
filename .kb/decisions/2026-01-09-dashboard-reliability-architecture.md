---
status: active
blocks:
  - keywords:
      - dashboard architecture redesign
      - replace dashboard infrastructure
---

# Decision: Dashboard Reliability Architecture

**Date:** 2026-01-09
**Status:** Proposed
**Context:** Chronic dashboard instability requiring constant manual restarts

## Problem

Dashboard is fundamentally unreliable:
- Constantly down or showing stale data
- Requires frequent manual restarts
- Hard refresh "fixes" never actually fix anything
- 186 investigations mention "restart"
- Daily friction not captured in artifacts

## Root Causes (5 Systemic Patterns)

### 1. No Observability
**Symptom:** Can't tell when dashboard is broken until you look at it.
- No health checks on launchd services
- No alerts when services crash
- No visibility into stale cache state
- Dashboard shows data but it's untrustworthy

### 2. Multiple Sources of Truth
**Symptom:** Same service running in multiple places causes conflicts.
- Vite in launchd (persistent) AND tmuxinator (manual)
- Server running old binary after rebuild
- Cache shows stale data with no indication

**Evidence:**
- Jan 3: Vite orphan pileup from launchd restarts
- Jan 7, Dec 24: Server running old binary after `make build`

### 3. Rebuild/Restart Not Atomic
**Symptom:** Many manual steps to deploy a change.
- `make build` ≠ running new binary
- Restart vite ≠ browser gets new bundle
- No single "deploy" command

**Evidence:**
- Jan 9: Model badges need hard refresh (never works)
- Dec 24: Phase badges not showing until server restart

### 4. No Self-Healing
**Symptom:** Failures accumulate, nothing recovers automatically.
- Orphaned vite processes pile up
- Stale caches stay stale
- Services die and stay dead

**Evidence:**
- Jan 3: Vite orphan processes (fixed with AbandonProcessGroup)
- Jan 3: CPU spike from 20+ bd processes (fixed with caching)
- Daemon constantly not running

### 5. Cache Invalidation is Manual
**Symptom:** No way to know when cached data is stale.
- Browser cache vs vite bundle
- Dashboard API cache vs beads state
- No cache busting strategy

**Evidence:**
- Jan 9: Model badges "browser cache issue" never fixed by hard refresh

## Decision: Reliability-First Architecture

### 1. Health Check System

**Command:**
```bash
orch doctor          # One-time health check
orch doctor --watch  # Continuous monitoring with alerts
orch doctor --fix    # Auto-fix common issues
```

**Checks:**
- [ ] launchd services alive (com.orch-go.serve, com.orch-go.web, com.opencode.serve)
- [ ] Ports responding (3348, 5188, 4096)
- [ ] Binary versions match (running vs built)
- [ ] No orphaned processes (vite, bd)
- [ ] Cache freshness (API response timestamps)

**Alerts:**
- Desktop notification on failure
- Log to `~/.orch/health.log`

### 2. Atomic Deploy

**Command:**
```bash
orch deploy  # Rebuild and restart everything atomically
```

**Steps:**
1. Build binary (`make build`)
2. Kill orphaned processes
3. Restart launchd services (kickstart -k)
4. Wait for health checks to pass
5. Display deployment status

**Output:**
```
Building orch binary...        ✓
Killing orphaned processes...  ✓
Restarting serve API...        ✓
Restarting web UI...           ✓
Health checks...               ✓

Dashboard available at http://localhost:5188
```

### 3. Single Source of Truth

**Changes:**
- [x] Remove vite from tmuxinator workers-orch-go (done Jan 3)
- [ ] launchd owns ALL persistent services
- [ ] Registry is canonical agent state
- [ ] Dashboard pulls from registry API only

**Anti-pattern:** Never run same service from multiple sources.

### 4. Self-Healing Daemon

**Command:**
```bash
orch doctor --daemon  # Run as background process
```

**Behaviors:**
- Monitor launchd services every 30s
- Kill orphaned processes (PPID=1 vite, long-running bd)
- Restart crashed services
- Log all interventions to `~/.orch/doctor.log`

**Integration:** Add to launchd as `com.orch.doctor.plist`.

### 5. Cache with Invalidation

**API Changes:**
```go
// Add version header to all API responses
w.Header().Set("X-Orch-Version", version)
w.Header().Set("X-Cache-Time", time.Now().Format(time.RFC3339))
```

**Dashboard Changes:**
```typescript
// Check version on every API response
if (response.headers['x-orch-version'] !== currentVersion) {
  showReloadBanner("New version available - reload to update");
}

// Show staleness warning
const cacheTime = response.headers['x-cache-time'];
if (Date.now() - Date.parse(cacheTime) > 60000) {
  showStaleWarning("Data may be out of date");
}
```

**Browser cache busting:**
- Add `?v=<git-hash>` to all asset URLs
- Update on vite restart

## Implementation Sequence

### Phase 1: Observability (P0)
1. Implement `orch doctor` command
2. Add health checks for all services
3. Add desktop notifications on failure

**Success:** Can see when dashboard is broken without looking at it.

### Phase 2: Atomic Deploy (P0)
1. Implement `orch deploy` command
2. Test full restart cycle
3. Document in CLAUDE.md

**Success:** One command deploys changes end-to-end.

### Phase 3: Self-Healing (P1)
1. Implement `orch doctor --daemon`
2. Add to launchd
3. Monitor for 1 week

**Success:** Services recover automatically without manual intervention.

### Phase 4: Cache Invalidation (P1)
1. Add version headers to API
2. Add staleness detection to dashboard
3. Implement cache busting for assets

**Success:** Dashboard shows accurate state or explicit "stale" warning.

## Success Criteria

**Incremental, testable improvements (not magic fixes):**

- [ ] `orch doctor` catches at least 3 of the 5 common failure modes (launchd dead, port not responding, orphaned processes, stale binary, stale cache)
- [ ] `orch deploy` works end-to-end without manual intervention (rebuild → restart → health check pass)
- [ ] Desktop notification fires when dashboard breaks (tested by killing a service)
- [ ] Dashboard shows explicit "STALE DATA" warning when cache is >60s old
- [ ] Can answer "is the dashboard actually working?" by running one command (`orch doctor`)

**Long-term goal (aspirational, not guaranteed):**
- Reduce "did you restart?" questions by 50% over 2 weeks (baseline: ~daily)
- Trust restored enough to use dashboard as primary view instead of tmux

**What this WON'T fix:**
- Bugs in dashboard UI logic (Svelte issues, etc.)
- Browser-specific caching weirdness
- Race conditions we haven't discovered yet
- Human error (forgetting to run `orch deploy`)

## Tradeoffs

**Cons:**
- More code to maintain (health checks, self-healing)
- Slower deploys (wait for health checks)
- Alert fatigue risk (if too sensitive)

**Pros:**
- Trust in dashboard restored
- Reduced daily friction
- Failures visible before they impact work
- Clear path to production-readiness

## Alternatives Considered

**Option A: Rebuild dashboard from scratch**
- **Rejected:** Too expensive, doesn't fix infrastructure issues

**Option B: Accept unreliability, document workarounds**
- **Rejected:** Already tried this (186 investigations), doesn't scale

**Option C: Use only tmux-based agents (no dashboard)**
- **Rejected:** Loses visibility into parallel work

## Related

- Investigation: `.kb/investigations/2026-01-03-inv-server-management-architecture-confusion-tmuxinator.md`
- Investigation: `.kb/investigations/2026-01-03-inv-orch-serve-causes-cpu-spike.md`
- Investigation: `.kb/investigations/2025-12-22-debug-dashboard-shows-0-agents-despite-api-returning-209.md`
- Investigation: `.kb/investigations/2025-12-24-inv-dashboard-phase-badges-not-showing.md`
- Constraint: 186 investigations mention "restart"
