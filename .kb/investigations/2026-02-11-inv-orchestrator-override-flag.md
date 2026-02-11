# Investigation: Add --orchestrator-override flag to orch complete

**Status:** Active  
**Date:** 2026-02-11  
**Task:** orch-go-o018r

## Context

When agent sessions die after committing work, orchestrator needs to bypass specific gates (like phase_complete) while still running other core gates (build/test/commit_evidence). Current options are inadequate:
- `--force`: Skips ALL gates (too broad)
- `--skip-*` flags: Blocked for core gates like phase_complete

Need a new `--orchestrator-override` flag that:
1. Accepts a gate name parameter
2. Requires explicit justification via --reason
3. Only skips the named gate
4. Logs override event to agentlog
5. Is distinct from --force

## Findings

### 1. Gate System Architecture

**Evidence:** Examined `pkg/verify/check.go:19-38` and `cmd/orch/complete_gates.go`

**Source:** Gate constants defined in `pkg/verify/check.go`

```go
const (
    GatePhaseComplete      = "phase_complete"
    GateSynthesis          = "synthesis"
    GateCommitEvidence     = "commit_evidence"
    GateTestEvidence       = "test_evidence"
    // ... more gates
)
```

**Significance:** Gates are string constants. The system has two tiers:
- **Tier 1 (Core)**: phase_complete, commit_evidence, synthesis, test_evidence, git_diff
- **Tier 2 (Quality)**: build, model_connection, visual_verification, etc.

Core gates currently cannot be skipped via --skip-* flags (enforced in `validateSkipFlags()`).

---

### 2. SkipConfig System

**Evidence:** Examined `cmd/orch/complete_verify.go:17-36`

**Source:** SkipConfig struct with boolean fields for each skip flag

```go
type SkipConfig struct {
    TestEvidence     bool
    ModelConnection  bool
    Visual           bool
    // ... one bool per gate
    PhaseComplete    bool
    CommitEvidence   bool
    Reason           string
    BatchMode        bool
}
```

**Significance:** Current design has one boolean field per gate. The `shouldSkipGate()` method checks these booleans. Core gate skips are validated in `validateSkipFlags()` and rejected.

---

### 3. Skip Flag Implementation Pattern

**Evidence:** Examined `cmd/orch/complete_cmd.go:34-66, 180-211`

**Source:** Flag declarations and getSkipConfig()

```go
var (
    completeSkipPhaseComplete bool
    completeSkipCommitEvidence bool
    // ... one var per skip flag
    completeSkipReason string
)

completeCmd.Flags().BoolVar(&completeSkipPhaseComplete, "skip-phase-complete", false, "...")
completeCmd.Flags().StringVar(&completeSkipReason, "skip-reason", "", "...")
```

**Significance:** Each skip flag is a separate boolean var. The `--skip-reason` is shared across all skip flags. The `validateSkipFlags()` function checks that reason is provided and that no core gates are being skipped.

---

### 4. Event Logging System

**Evidence:** Examined `cmd/orch/complete_verify.go:230-263` and `pkg/events/logger.go:38-39`

**Source:** logSkipEvents() function and EventTypeVerificationBypassed

```go
func logSkipEvents(skipConfig SkipConfig, beadsID, workspace, skill string) {
    logger := events.NewLogger(events.DefaultLogPath())
    for _, gate := range skipConfig.skippedGates() {
        event := events.Event{
            Type: events.EventTypeVerificationBypassed,
            Data: map[string]interface{}{
                "beads_id": beadsID,
                "gate": gate,
                "reason": skipConfig.Reason,
            },
        }
        logger.LogVerificationBypassed(...)
    }
}
```

**Significance:** Events are logged via `events.Logger` to `~/.orch/events.jsonl` (agentlog). Each gate bypass is logged individually with the reason. The infrastructure for logging overrides already exists.

---

### 5. Core Gate Protection

**Evidence:** Examined `cmd/orch/complete_verify.go:183-228`

**Source:** validateSkipFlags() and coreGateSkips()

```go
func validateSkipFlags(skipConfig SkipConfig) error {
    coreSkips := skipConfig.coreGateSkips()
    if len(coreSkips) > 0 {
        return fmt.Errorf("core gates cannot be skipped: %s (use --force to bypass all verification)", 
            strings.Join(coreSkips, ", "))
    }
    // ...
}
```

**Significance:** Core gates (phase_complete, commit_evidence, synthesis, test_evidence, git_diff) are explicitly protected from being skipped via --skip-* flags. This validation happens before any gates run. The error message suggests using --force, which skips everything.

---

### 6. Force Flag Behavior

**Evidence:** Examined `cmd/orch/complete_gates.go:44-65`

**Source:** verifyCompletion() function

```go
if completeForce {
    // --force: run verification to capture which gates would have failed, but don't block
    result, err := verify.VerifyCompletionFull(...)
    fmt.Println("Skipping all verification (--force) - DEPRECATED: use targeted --skip-* flags")
    return outcome, nil
}
```

**Significance:** --force skips ALL gates but still runs verification to capture what would have failed. It's deprecated and logs a warning. This is the only current way to bypass core gates.

---

## Synthesis

The current system has:
1. **Two-tier gate architecture**: Core (5 gates) + Quality (10 gates)
2. **Targeted skip flags**: One per gate, but core gates are protected
3. **Force flag**: Bypasses everything (deprecated, too broad)
4. **Event logging**: Infrastructure exists for logging bypasses

The gap: **No way to surgically bypass a single core gate with justification**. The orchestrator needs this when an agent dies after committing but before reporting Phase: Complete.

### Implementation Strategy

**Approach:** Add new --orchestrator-override flag that:
1. Accepts a gate name as parameter (not a boolean flag per gate)
2. Bypasses core gate protection (elevates orchestrator authority)
3. Requires --reason (reuse existing --skip-reason or add new --override-reason)
4. Logs to agentlog via existing events.EventTypeVerificationBypassed
5. Only skips the single named gate

**Key Decision:** Should this be:
- **Option A**: `--orchestrator-override <gate-name>` (accepts gate name)
- **Option B**: `--orchestrator-override` (boolean) + repurpose one of the existing --skip-* flags

**Recommendation:** Option A. More explicit, easier to understand, and makes the elevated privilege clear.

**Files to modify:**
1. `cmd/orch/complete_cmd.go`: Add flag declaration
2. `cmd/orch/complete_verify.go`: Extend SkipConfig, modify shouldSkipGate() to handle orchestrator override
3. `cmd/orch/complete_verify.go`: Modify validateSkipFlags() to allow core gate bypass when orchestrator-override is set
4. Event logging already works via existing logSkipEvents()

**Prior constraint to respect:** Registry updates must happen before beads close (this is already handled by the pipeline architecture).

---

## Next Steps

1. Design the flag API: `--orchestrator-override <gate-name> --reason "<justification>"`
2. Write failing tests for the new flag
3. Implement flag parsing and SkipConfig extension
4. Modify gate bypass logic
5. Add event logging (reuse existing infrastructure)
6. Verify registry update ordering constraint is maintained
