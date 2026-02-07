# Decision: Beads Daemon Rapid Restart Prevention

**Date:** 2026-01-22
**Status:** Recommended (needs beads implementation)
**Decision Maker:** Dylan + Claude

## Context

Beads SQLite database corruption has occurred 3+ times in 2 days (Jan 21-22, 2026). Root cause analysis identified a common pattern across all incidents: daemon enters rapid restart loop (any failure causes restart within seconds), and high-frequency database open/close cycles corrupt the WAL file.

**Evidence:**
- Jan 21: 57 daemon restarts in one day (sandbox chmod failures)
- Jan 22: 9+ restarts in 3 minutes (legacy fingerprint validation)
- All incidents show 0-byte WAL file = incomplete checkpoint operation

**Impact:** ~3+ hours lost to recovery, recurring daily.

## Question

How should beads prevent rapid daemon restart loops that cause database corruption?

## Decision

**Implement multi-layer protection against rapid restarts:**

### Layer 1: Restart Backoff (Critical)

Add minimum interval between daemon start attempts:

```go
// In daemon startup
const minRestartInterval = 30 * time.Second

func canStartDaemon() bool {
    lastStart := readLastStartTime()
    if time.Since(lastStart) < minRestartInterval {
        log.Warn("Daemon restart too soon, backing off")
        return false
    }
    writeLastStartTime(time.Now())
    return true
}
```

### Layer 2: Pre-flight Validation (Important)

Check prerequisites BEFORE database open:

```go
func validatePrerequisites() error {
    // Sandbox detection (before any DB operation)
    if inSandbox() {
        return fmt.Errorf("daemon cannot run in sandbox environment")
    }

    // Fingerprint check (without opening DB)
    if legacyDatabaseExists() {
        return fmt.Errorf("legacy database needs migration first")
    }

    return nil
}
```

### Layer 3: Graceful Degradation (Recommended)

Fall back to direct mode when daemon cannot start:

```go
func ensureDaemon() error {
    if err := validatePrerequisites(); err != nil {
        log.Warn("Daemon unavailable, using direct mode", "reason", err)
        setDirectMode(true)
        return nil  // Don't fail, just degrade
    }
    return startDaemon()
}
```

### Layer 4: Health Monitoring (Future)

Add `orch doctor` checks for daemon stability:
- Restart frequency (< 1/minute)
- WAL file state (no orphaned WAL without main DB)
- Daemon uptime (> 5 minutes = healthy)

## Alternatives Considered

### Option B: Disable WAL Mode

- **Pros:** Simpler, no checkpoint race conditions
- **Cons:** Performance regression, doesn't fix rapid restart issue
- **Rejected because:** Root cause is rapid restarts, not WAL itself

### Option C: Fix Each Individual Failure Cause

- **Pros:** Targeted fixes for sandbox, fingerprint, etc.
- **Cons:** New failure modes will appear, same pattern will recur
- **Rejected because:** Pattern is "rapid restarts cause corruption" regardless of cause

### Option D: Remove Daemon Auto-Start

- **Pros:** Eliminates rapid restart cycle entirely
- **Cons:** Major UX change, users must manually start daemon
- **Rejected because:** Too disruptive, backoff achieves same protection

## Consequences

**Positive:**
- Rapid restart loops become impossible
- Corruption incidents should drop to near-zero
- Daemon failures degrade gracefully to direct mode
- `orch doctor` can detect unhealthy daemon state

**Negative:**
- 30s backoff means slower daemon recovery after legitimate crashes
- Pre-flight checks add small overhead to daemon start
- Requires beads codebase changes (not just orch-go)

## Implementation Status

| Layer | Status | Owner |
|-------|--------|-------|
| Restart Backoff | Recommended | beads |
| Pre-flight Validation | Recommended | beads |
| Graceful Degradation | Recommended | beads |
| Health Monitoring | Future | orch-go |

**Workaround until implemented:**

```bash
# Force direct mode for agents
export BD_DIRECT_MODE=1

# Or disable daemon in sandbox SPAWN_CONTEXT
```

## Evidence

- **Investigation:** `.kb/investigations/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md`
- **Model:** `.kb/models/beads-database-corruption.md`
- **Daemon logs:** `.beads/daemon.log` (57 restarts on Jan 21)
- **Corruption backups:** `.beads/backup-corrupted-2026-01-2{1,2}/`
