# Session Synthesis

**Agent:** og-arch-design-daemon-driven-20mar-ade7
**Issue:** orch-go-f2ynp
**Duration:** 2026-03-20T17:00 → 2026-03-20T17:35
**Outcome:** success

---

## Plain-Language Summary

The system has completed 1,113 agent tasks with zero negative quality feedback — not because work quality is perfect, but because the feedback loop is structurally broken. This design creates a daemon-driven random quality audit system that closes the loop: the daemon periodically selects random completed work (weighted toward auto-completed tasks that skip human review), spawns an audit agent to check if the work actually matches the issue intent, and automatically rejects low-quality work via `orch reject`. The critical discovery is that even after `orch reject` ships, the daemon's learning system (`learning.go`) won't see rejection signals because it doesn't handle `agent.rejected` events — this gap must be fixed first.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

Key outcomes:
- Investigation produced at `.kb/investigations/2026-03-20-inv-design-daemon-driven-random-quality-audit.md`
- Completion-verification model updated with probe reference
- 3 implementation issues created: orch-go-9t5nv (learning fix), orch-go-9g5bp (daemon selection), orch-go-ie81t (verdict pipeline)
- Dependencies between issues reported to orchestrator

## Delta

- Designed 3-layer structural audit pipeline: selection → review → rejection
- Identified `learning.go` `RejectedCount` gap as prerequisite fix
- Navigated "no agent-judgment gates" constraint — post-hoc audits are architecturally distinct from completion gates
- Weighted auto-complete oversampling (60/40) targets highest-risk completions

## Evidence

- Read and analyzed: `audit_cmd.go`, `reject_cmd.go`, `daemon_periodic.go`, `scheduler.go`, `learning.go`, `logger.go`
- Verified: `ComputeLearning()` switch statement has no `EventTypeAgentRejected` case (grep confirmed 0 matches)
- Cross-referenced: CV model §7, §"Why No Agent-Judgment Gates?", human feedback probe, judge verdict probe
- 5 design forks navigated with substrate consultation (principles, models, decisions)

## Architectural Choices

1. **Daemon periodic task over launchd**: structural > advisory hierarchy — advisory mechanisms have 0% action rate
2. **Post-hoc audit agent over mechanical-only checks**: only agents can assess intent match (the highest-value quality signal)
3. **LOW confidence gate on auto-rejection**: prevents audit false positives from creating noise — surfaces for orchestrator review instead
4. **Auto-complete oversampling (60/40)**: 37% of completions skip all human gates — they deserve disproportionate audit scrutiny

## Unexplored Questions

- Audit agent precision/recall — unknown until first 10 audits run
- Whether `SuggestDowngrades()` should also handle upgrades (tightening compliance) based on rejection rates
- Whether knowledge-producing skills (investigations, architect) should also be audited

## Created Issues

- orch-go-9t5nv: Add RejectedCount to learning.go (Phase 1, prerequisite)
- orch-go-9g5bp: Daemon periodic audit selection task (Phase 2)
- orch-go-ie81t: Verdict-to-reject pipeline (Phase 3, depends on Phase 1+2)
