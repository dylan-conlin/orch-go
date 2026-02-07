# Synthesis: Define Success Spawn Telemetry Success

## Task Overview
Defined "success" for spawn telemetry to shape the future telemetry schema. Analyzed existing logging and completion logic to establish a robust definition.

## Key Findings
- **Clean Success Definition:** Success should be defined as `verification_passed && !forced`. This ensures the work met automated quality gates and did not require human override.
- **Yield Tiers:** Identified four stages of success: Claimed (Phase: Complete), Automated (Verify passed), Process (Auto-completed), and Final (Human verified).
- **Metric Gap:** Current logs show a low "Clean Success" rate (11% in last 100 completions), with 60% of tasks being "Forced", highlighting the need for this telemetry to identify friction points.

## Recommendations
- Expand `AgentCompletedData` in `pkg/events/logger.go` to include `Success`, `Escalation`, and `ClaimedSuccess`.
- Synchronize logging between manual and daemon-driven completions.

## Deliverables
- [Investigation File](/.kb/investigations/2026-01-09-inv-define-success-spawn-telemetry-success.md)
- [Decision Recorded](kb quick)
