## Summary (D.E.K.N.)

**Delta:** Implemented targeted --skip-{gate} flags for orch complete, replacing blanket --force with individual bypass flags.

**Evidence:** All 23 new tests pass. Manual validation confirms --skip-reason validation (requires 10+ chars) and verification.bypassed events are logged.

**Knowledge:** Targeted skip flags provide accountability - each bypass is logged with gate, reason, and beads_id for audit.

**Next:** Close - all deliverables complete.

**Promote to Decision:** recommend-no (implementation, not architectural choice)

---

# Investigation: Implement Targeted Skip Gate Flags

**Question:** How to replace blanket --force with targeted --skip-{gate} flags that require reasons and log bypasses?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Gate Constants in pkg/verify/check.go

**Evidence:**
```go
const (
    GatePhaseComplete     = "phase_complete"
    GateSynthesis         = "synthesis"
    GateTestEvidence      = "test_evidence"
    GateVisualVerify      = "visual_verification"
    GateGitDiff           = "git_diff"
    GateBuild             = "build"
    GateConstraint        = "constraint"
    GatePhaseGate         = "phase_gate"
    GateSkillOutput       = "skill_output"
    GateDecisionPatchLimit = "decision_patch_limit"
)
```

**Source:** pkg/verify/check.go:12-22

**Significance:** These constants define the gate names to use for skip flags and event logging.

### Finding 2: VerifyCompletionFull Returns GatesFailed Array

**Evidence:** The verification result includes `GatesFailed []string` which maps to the gate constants.

**Source:** pkg/verify/check.go (VerificationResult struct)

**Significance:** This enables filtering out skipped gates from the failure list.

### Finding 3: Events Package Has Pattern for Typed Events

**Evidence:** EventTypeVerificationFailed and LogVerificationFailed already exist as patterns.

**Source:** pkg/events/logger.go:165-197

**Significance:** Following this pattern for EventTypeVerificationBypassed maintains consistency.

---

## Synthesis

**Key Insights:**

1. **Targeted skips require mapping** - Each --skip-* flag maps to a specific Gate* constant for consistent naming
2. **Reason requirement adds accountability** - 10-char minimum ensures non-trivial explanations
3. **Event logging enables audit** - verification.bypassed events capture who bypassed what and why

**Answer to Investigation Question:**

Implementation required:
1. Add SkipConfig struct to hold all skip flags
2. Add validateSkipFlags() to enforce --skip-reason requirement
3. Modify verification logic to filter out skipped gates
4. Log verification.bypassed events for each skipped gate
5. Add deprecation warning to --force

---

## References

**Files Examined:**
- `pkg/verify/check.go` - Gate constants and verification logic
- `cmd/orch/complete_cmd.go` - Complete command implementation
- `pkg/events/logger.go` - Event logging patterns

**Commands Run:**
```bash
go build ./...  # Verify compilation
go test ./cmd/orch/... ./pkg/events/... -short  # Run tests
orch complete --help  # Verify CLI output
```
