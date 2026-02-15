# Probe: Daemon Warn-and-Continue Anti-Pattern Audit

**Date:** 2026-02-15  
**Context:** Post-Feb 14 duplicate spawn incident (orch-go-2ma)  
**Model:** Daemon Autonomous Operation  
**Claim tested:** The duplicate spawn incident was an isolated case vs. a systematic pattern in daemon code

---

## Question

The Feb 14 2026 duplicate spawn incident was caused by a warn-and-continue pattern in the spawn flow (UpdateBeadsStatus failing silently, logging warning, proceeding with spawn). Does this pattern exist elsewhere in daemon code?

---

## Method

1. Audited all daemon code files for warn-and-continue patterns:
   - `pkg/daemon/daemon.go` (1319 lines)
   - `pkg/daemon/spawn_tracker.go` (151 lines)
   - `pkg/daemon/spawn_failure_tracker.go` (97 lines)
   - `pkg/daemon/completion_processing.go` (345 lines)
   - `pkg/daemon/completion.go` (309 lines)
   - `cmd/orch/daemon.go` (894 lines)

2. Categorized findings into:
   - **Critical**: Spawn prerequisites that should be hard gates
   - **Acceptable**: Non-critical operations (logging, monitoring)

---

## Findings

### Critical Warn-and-Continue Patterns (Should Be Fail-Fast)

#### 1. Dependency Check Failure (`pkg/daemon/daemon.go:366-371`)

```go
blockers, err := beads.CheckBlockingDependencies(issue.ID)
if err != nil {
    if d.Config.Verbose {
        fmt.Printf("  DEBUG: Warning: could not check dependencies for %s: %v\n", issue.ID, err)
    }
    // Continue checking - don't skip issue just because we can't check dependencies
}
```

**Pattern:** Logs warning when dependency check fails, continues processing issue  
**Risk:** Could spawn work that's actually blocked by dependencies, wasting agent slots  
**Violation:** Spawn prerequisite fail-fast constraint  
**Issue:** orch-go-nff

#### 2. Epic Children List Failure (`pkg/daemon/daemon.go:426-431`)

```go
children, err := listChildren(epicID)
if err != nil {
    if d.Config.Verbose {
        fmt.Printf("  DEBUG: Warning: could not list children of epic %s: %v\n", epicID, err)
    }
    continue
}
```

**Pattern:** Logs warning when epic child listing fails, skips to next epic  
**Risk:** Silently drops spawnable work from that epic  
**Violation:** Spawn prerequisite fail-fast constraint  
**Issue:** orch-go-j26

#### 3. Extraction Setup Failure (`pkg/daemon/daemon.go:792-795`)

```go
if err != nil {
    if d.Config.Verbose {
        fmt.Printf("  Warning: extraction setup failed for %s: %v (proceeding with normal spawn)\n", issue.ID, err)
    }
    // Fall through to normal spawn on extraction setup failure
}
```

**Pattern:** Logs warning when auto-extraction setup fails, proceeds with normal spawn  
**Risk:** Spawns work on critical hotspot (>1500 lines) without extraction, violates extraction gate  
**Violation:** Auto-extraction gate + spawn prerequisite fail-fast constraint  
**Issue:** orch-go-r9t

#### 4. Rollback Failures After Spawn Failure (`pkg/daemon/daemon.go:885-889, 1025-1028`)

```go
if rollbackErr := UpdateBeadsStatus(issue.ID, "open"); rollbackErr != nil {
    if d.Config.Verbose {
        fmt.Printf("  Warning: failed to rollback status for %s: %v\n", issue.ID, rollbackErr)
    }
}
```

**Pattern:** Logs warning when rollback fails after spawn failure, continues  
**Risk:** Issue left in inconsistent state (marked in_progress but spawn failed), orphaned issue  
**Violation:** Spawn prerequisite fail-fast constraint  
**Issue:** orch-go-a3s

#### 5. Completion Processing Error (`cmd/orch/daemon.go:425-427`)

```go
completionResult, err := d.CompletionOnce(completionConfig)
if err != nil && daemonVerbose {
    fmt.Fprintf(os.Stderr, "[%s] Completion processing error: %v\n", timestamp, err)
}
```

**Pattern:** Logs error when completion processing fails, continues daemon loop  
**Risk:** Completed agents never marked ready-for-review, orphaned work accumulates, verification pause never triggers  
**Violation:** Spawn prerequisite fail-fast constraint (completion is part of spawn lifecycle)  
**Issue:** orch-go-mpu

---

### Acceptable Warn-and-Continue (Non-Critical Operations)

The following patterns are acceptable because they involve non-critical operations:

1. **Logging failures** (multiple locations) - acceptable since logging is for observability, not correctness
2. **Status file write failures** (`cmd/orch/daemon.go:508`) - acceptable since status file is monitoring only
3. **Resume signal check failures** (`cmd/orch/daemon.go:302`) - acceptable since it's a convenience feature
4. **Event logging failures** (multiple locations) - acceptable since events are for audit trail, not correctness

---

## Evidence

**Claim:** The Feb 14 incident was an isolated case  
**Finding:** **CONTRADICTS** - Found 5 critical warn-and-continue patterns in daemon code

**Pattern:** Warn-and-continue in spawn prerequisites is systematic, not isolated  
**Root cause:** No architectural constraint enforcing fail-fast for spawn prerequisites

---

## Constraint Established

Created `kb-035b64`: **Spawn prerequisites are hard gates, not soft warnings**

> The Feb 14 2026 duplicate spawn incident (orch-go-2ma) was caused by UpdateBeadsStatus failing silently - the code logged a warning and continued spawning, creating 10 duplicate agents when SpawnedIssueTracker TTL expired. Spawn flow prerequisite checks (beads status update, dependency checks, epic expansion, extraction gates) MUST fail-fast rather than warn-and-continue. Failing without marking the issue prevents duplicate spawns and surfaces the real problem immediately. Pattern: if a spawn prerequisite fails, return error or skip the issue - never log warning and spawn anyway.

---

## Recommendations

1. **Immediate:** Fix the 5 critical patterns (issues created: orch-go-nff, orch-go-j26, orch-go-r9t, orch-go-a3s, orch-go-mpu)
2. **Architectural:** Add linting rule to detect `fmt.Printf.*Warning.*continue` pattern in daemon spawn flow
3. **Structural:** Make fail-fast the default - warn-and-continue should require explicit justification (comment explaining why it's safe)

---

## Impact on Model

**Model claim:** "Spawn prerequisites are validated before spawning"  
**Probe finding:** TRUE for primary path (UpdateBeadsStatus now fails fast after Feb 14 fix), FALSE for secondary paths (dependency checks, epic expansion, extraction gate, rollback)

**Model update needed:** Add section on prerequisite validation patterns distinguishing between:
- **Primary dedup** (beads status update) - NOW fail-fast ✓
- **Secondary prerequisites** (dependencies, extraction) - STILL warn-and-continue ✗
- **Tertiary monitoring** (logging, status files) - ACCEPTABLE to continue ✓

---

## Testing Notes

**What was tested:**
- Read and analyzed all daemon code files for error handling patterns
- Categorized patterns by criticality (spawn prerequisites vs. monitoring)
- Cross-referenced with Feb 14 incident to identify similar patterns

**What was NOT tested:**
- Actual execution of these code paths (static analysis only)
- Frequency of these errors in production (would need log analysis)
- Impact of fixing these patterns (requires implementation and testing)

---

## Status

**Verified:** 2026-02-15  
**Updated model:** Pending (will update after fixes are implemented)  
**Issues created:** 5 (orch-go-nff, orch-go-j26, orch-go-r9t, orch-go-a3s, orch-go-mpu)  
**Constraint created:** kb-035b64
