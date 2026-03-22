# Session Synthesis

**Agent:** og-inv-investigate-open-loop-22mar-79e9
**Issue:** orch-go-pn9au
**Outcome:** success

---

## Plain-Language Summary

The orch-go codebase has a systemic pattern where feedback infrastructure is built on both ends (events emitted, config defined, verdict processing ready) but the middle connection — the code that reads the signal and changes daemon behavior — is never written. I found 10 concrete instances of this, from the quality audit system (labels issues for audit but never spawns audit agents) to 11 periodic tasks registered in the scheduler but never called from the daemon's main loop. I named this "consumer-last construction": the system builds emitters during feature work, defers consumers indefinitely, and the feedback loop never closes. Production data confirms it: 1,285 completions, 513 accretion.delta events with no reader, 0 agent.rejected events, 0 audit verdicts, 0 reworks.

---

## TLDR

Systematic code audit found 10 open feedback loops in the completion/daemon pipeline. The pattern is "consumer-last construction" — events are emitted, config is defined, scheduler tasks are registered, but the daemon code that reads signals and changes behavior is never written. ~2,800 lines of structurally unreachable code exist as a result.

---

## Delta (What Changed)

### Files Created
- `.kb/models/completion-verification/probes/2026-03-22-probe-open-loop-infrastructure-code-audit.md` — Probe documenting all 10 open loops with code paths and production evidence

### Files Modified
- `.kb/models/completion-verification/model.md` — Updated §7 (reject exists but 0 events), added §8 (consumer-last construction pattern with 10 instances), added probe reference

### Commits
- Probe + model update + synthesis

---

## Evidence (What Was Observed)

### Production Event Data (Primary Evidence)
```bash
grep '"type":"agent.rejected"' ~/.orch/events.jsonl | wc -l         # 0
grep '"type":"agent.rejected"' ~/.orch/events-2026-03.jsonl | wc -l  # 0
grep -c "accretion.delta" ~/.orch/events.jsonl                       # 449
grep -c "accretion.delta" ~/.orch/events-2026-03.jsonl                # 64
grep '"type":"agent.completed"' ~/.orch/events.jsonl | wc -l         # 1188
grep '"type":"audit\.' ~/.orch/events.jsonl | wc -l                  # 0
```

### Code Audit (10 Open Loops)

1. **Quality audit**: periodic_audit.go labels issues → (nothing spawns audit agents) → audit_verdict.go processes results
2. **Accretion response**: 513 events emitted → daemon_loop.go:141 has blank wiring → no consumer
3. **Reject → learning**: reject_cmd.go emits events → learning.go aggregates → daemon never reads RejectedCount
4. **Comprehension queue**: labels added → ComprehensionQuerier never instantiated → gate fails open
5. **Verification metrics**: VerificationFailures/Bypasses aggregated → never read (dead code)
6. **Rework feedback**: ReworkCount aggregated → display-only in orient
7-10. **11 periodic tasks**: registered in scheduler.go → never called from daemon_periodic.go

### Scheduler Registration vs Invocation
- 24 tasks registered in `pkg/daemon/scheduler.go` (lines 125-148)
- 13 tasks called from `cmd/orch/daemon_periodic.go`
- 11 tasks are registered but structurally unreachable

---

## Architectural Choices

No architectural choices — this was a diagnostic investigation, not implementation work.

---

## Knowledge (What Was Learned)

### Pattern Named: Consumer-Last Construction

Every open loop follows the same 3-layer architecture:
1. **Emission/Collection** — Events emitted, labels added, metrics computed (ALWAYS built)
2. **Configuration/Registration** — Config fields, scheduler tasks, interfaces defined (ALWAYS built)
3. **Consumption/Action** — Daemon reads signal and changes behavior (MISSING in 10/10 cases)

This is NOT a "Phase 3 problem" (integration testing deferred). The consumer code was never written — there are no TODO comments, no stub functions, no placeholder implementations. The emitter is built during feature work; the consumer is a separate task that never gets created.

### Why It Keeps Happening

The codebase development pattern is: architect designs a system → workers implement components → each component is a beads issue → each issue is independently "complete" when its code compiles and tests pass. The emitter is one issue, the consumer would be a different issue, and the consumer issue either doesn't get created or is deprioritized. The system's own completion pipeline validates each piece in isolation, never testing the end-to-end loop.

### Constraints Discovered
- Only SuccessRate is consumed from the learning store (allocation.go:107-119, ±20% priority modulation)
- ComprehensionQuerier is never wired in the daemon — the comprehension gate is structurally inert
- 11 of 24 registered periodic tasks have no call site

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for evidence commands and expected outcomes.

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete (probe, model update, synthesis)
- [x] Investigation file has `Status: Complete`
- [x] Ready for `orch complete orch-go-pn9au`

### Suggested Follow-Up (Not In Scope)

The investigation maps the problem. Fixing it requires implementation decisions:

1. **Quick win**: Wire ComprehensionQuerier in daemon_loop.go (1 line of code, immediate behavioral impact)
2. **Quick win**: Add audit:deep-review → codebase-audit skill routing in InferSkillFromLabels()
3. **Medium**: Build accretion response consumer (read events, aggregate, create extraction issues)
4. **Medium**: Wire RejectedCount into daemon allocation scoring
5. **Structural**: Audit all 11 unlinked periodic tasks — delete dead ones, wire live ones

---

## Unexplored Questions

- **Why has `orch reject` never been used?** The command exists with low friction (2 args). Is it a discoverability problem (not shown in `orch review`), a cultural problem (rejecting agent work feels wasteful), or is completed work actually good enough?
- **Are the 11 unlinked periodic tasks dead features or future work?** Some (TaskReflect, TaskKnowledgeHealth) may have been superseded by other mechanisms.
- **Would closing these loops actually improve agent quality?** The system produces working code 100% of the time by every metric — but that may be because the metrics can't express failure.

---

## Friction

- Friction: none — smooth session. Subagent exploration was efficient for parallel codebase search.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-investigate-open-loop-22mar-79e9/`
**Probe:** `.kb/models/completion-verification/probes/2026-03-22-probe-open-loop-infrastructure-code-audit.md`
**Beads:** `bd show orch-go-pn9au`
