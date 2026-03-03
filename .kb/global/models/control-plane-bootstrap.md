# Model: Control-Plane Bootstrap

**Created:** 2026-02-15
**Status:** Active
**Context:** Discovered while implementing verifiability-first enforcement for orch-go daemon. The enforcement mechanism was auto-closed by the very system it was supposed to enforce against.

---

## What This Is

A mental model for bringing enforcement infrastructure online in self-modifying agentic systems. The core problem: you can't deploy a control-plane mechanism through the data-plane pipeline it governs, because the pipeline will process it like any other work — bypassing the very gates being built.

**The core insight:** The first deployment of any enforcement mechanism must happen outside the system being enforced. The daemon must be off while you build the brake.

---

## The Bootstrap Paradox

```
You want:    Daemon checks IsPaused() before spawning
To build:    Agent writes IsPaused() code, daemon spawns the agent
Problem:     Daemon auto-completes the agent without checking IsPaused()
             (because IsPaused() doesn't exist yet)
Result:      The enforcement mechanism is "complete" but was never enforced
```

This is not a hypothetical. On Feb 15, 2026:
- orch-go-ydzu (VerificationTracker wiring) was auto-closed by the daemon without human review
- orch-go-lyp3 (checkpoint infrastructure) was auto-closed by the daemon without human review
- Both were P1 control-plane issues implementing the verification-first decision
- The irony: the infrastructure designed to prevent autonomous closure was itself autonomously closed

---

## How This Works

### The Bootstrap Sequence (Required)

```
1. HALT the governed system
   └── Daemon off. No autonomous spawning.
   └── Manual orch spawn only.

2. BUILD the enforcement mechanism
   └── Agent writes the code (spawned manually)
   └── Agent commits locally
   └── Agent claims "Phase: Complete"

3. VERIFY behaviorally (human observes, not reads)
   └── Rebuild binary
   └── Run the governed system in test mode
   └── Observe the gate firing (log line, refusal, file write)
   └── Observe the gate blocking (system refuses to proceed)
   └── Both positive and negative paths verified

4. ACTIVATE the governed system
   └── Daemon on — now operating under the new constraint
   └── First real cycle should show enforcement active
```

**The critical property:** Step 3 cannot be delegated. If an agent verifies the enforcement, you have an agent verifying agent work — the self-referential loop the enforcement exists to break.

### Claims

**Claim 1: Control-plane components cannot be deployed through the data-plane pipeline they govern.**

Evidence: orch-go-ydzu and orch-go-lyp3 were auto-closed by the daemon. The daemon processed them as normal work items. The verification checkpoint file and IsPaused() wiring existed as code but were never tested against the running daemon.

**Claim 2: Every control-plane gate needs an observable signal that proves it fired.**

Evidence: IsPaused() was wired into the daemon (commit c9f50b73) but produced no log output. Three debugging agents "fixed" it with passing tests while production showed nothing. Only when explicit log lines were added (`Verification check: N/M unverified completions`) could a human observe whether the gate was actually executing. Without observable signals, "wired" and "not wired" are indistinguishable from outside.

**Claim 3: State that governs enforcement must persist in files, not process memory.**

Evidence: The VerificationTracker counter reset to 0 on daemon restart. 63 unverified completions in the backlog didn't trigger the pause because the counter only tracked within-session completions. Fix: SeedFromBacklog() reads the checkpoint file on startup and diffs against completed issues. The counter now reflects reality across restarts.

**Claim 4: The first deployment of enforcement must be human-verified end-to-end before the governed system resumes.**

Evidence: The verification sequence for daemon pause required 6 steps, each building on the previous. Steps 1-3 (checkpoint write) worked. Steps 4-6 (daemon reads checkpoint) failed three times — three debugging agents claimed success while production was unchanged. Only human behavioral verification (creating a test issue, observing the daemon spawn it without pause, observing the daemon refuse after threshold) proved the chain worked.

**Claim 5: Multiple code paths that should share behavior must share implementation.**

Evidence: The daemon had 4 independent Config construction sites (runDaemonLoop, runDaemonOnce, runDaemonDryRun, runDaemonPreview). Each debugging agent fixed one path. New fields set in DefaultConfig() silently failed in production because production didn't use DefaultConfig(). Fix: single daemonConfigFromFlags() function all paths use. Structural elimination of the class of bug.

---

## Where This Applies

### Strong Fit

| Domain | Why |
|--------|-----|
| **Verification gates** | Gates that check whether work was verified before proceeding |
| **Rate limiters** | Throttles on autonomous operation speed |
| **Circuit breakers** | Halting mechanisms that stop runaway processes |
| **Permission systems** | Access controls on who can modify what |
| **Monitoring infrastructure** | Systems that observe the systems they monitor |

### The Pattern Generalizes

Any system where Component A governs Component B's behavior, and Component B is the mechanism that deploys Component A, has the bootstrap paradox. The resolution is always the same: deploy A through a channel B doesn't control, verify A works, then let B operate under A's governance.

---

## Constraints

### What This Model Enables

- Predictable deployment of enforcement mechanisms
- Confidence that gates actually work (not just exist as code)
- Clear protocol for "turning on" new constraints

### What This Model Constrains

- Cannot deploy control-plane changes while the daemon is running
- Requires human time for behavioral verification (not delegatable)
- Slows down meta-level changes (enforcement changes are slower than feature changes)

---

## The Anti-Pattern: Enforcement Theater

When enforcement mechanisms exist as code, pass tests, and are "complete" but have never been observed working in production. Symptoms:
- Agent claims "Phase: Complete" with passing tests
- Code review shows the gate exists
- But no human has observed the gate firing against the real system
- The governed system continues operating as if the gate doesn't exist

The entropy spiral postmortem found the same pattern at documentation level: post-mortems documented mitigations, none were implemented. Enforcement theater is the code-level equivalent: implementation exists, enforcement doesn't.

---

## Integration with Existing Models

### Verifiability-First Development

This model operationalizes verifiability-first for the special case of enforcement infrastructure. "Specify behavior and verify outcomes" applied to the verification system itself.

### Verification Bottleneck Principle

The bootstrap sequence is the extreme case of verification bottleneck: the human must personally verify the enforcement before the system can operate at all. No delegation possible.

### Infrastructure Over Instruction

The bootstrap paradox IS the infrastructure-over-instruction principle applied recursively: even the infrastructure that enforces behavior needs infrastructure (the bootstrap protocol) to ensure it works.

---

## Evolution

| Date | Change | Trigger |
|------|--------|---------|
| 2026-02-15 | Created | Observed daemon auto-closing its own pause mechanism during verifiability-first implementation. Three debugging agents failed to wire IsPaused() into production. Bootstrap sequence emerged from the recovery process. |

---

## See Also

- `.kb/decisions/2026-02-14-verifiability-first-hard-constraint.md` — The decision that triggered the bootstrap
- `.kb/investigations/2026-02-14-inv-entropy-spiral-deep-analysis.md` — Why enforcement exists
- `.kb/decisions/2026-02-15-daemon-unified-config-construction.md` — The config divergence that caused 3 failed fixes
- `~/orch-knowledge/kb/models/verifiability-first-development.md` — Parent model
