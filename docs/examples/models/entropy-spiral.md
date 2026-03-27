# Model: Entropy Spiral

<!-- ABOUT MODELS
    Models are synthesized understanding of a domain, built from
    evidence across multiple investigations and sessions.

    Key metadata:
      - Domain: What area of knowledge this model covers
      - Last Updated: When the model was last revised
      - Validation Status: How confident we are in the claims
          WORKING HYPOTHESIS = supported but not externally validated
          CONFIRMED = reproduced or externally validated
          OVERCLAIMED = claims exceed evidence (honest downgrade)
      - Synthesized From: The evidence base (investigations, probes,
          post-mortems, external sources) that built this model

    Models are never "done" — they evolve as probes test claims
    and new evidence accumulates.
-->

**Domain:** Agentic Systems / Failure Modes / Control Theory
**Last Updated:** 2026-03-06
**Validation Status:** WORKING HYPOTHESIS — observed in one system over 3 months. No external validation.
**Synthesized From:** 3 investigations, 3 post-mortems, git-verified evidence from 3 spirals totaling 1,625 lost commits.

---

## Summary (30 seconds)

<!-- Every model opens with a 30-second summary.
     A reader should know the core claim and its evidence
     without reading further. -->

An entropy spiral is a feedback loop where an agentic system degrades while reporting success. Locally-correct changes compose into globally-incoherent systems when velocity exceeds verification bandwidth. The mechanism requires three conditions: (1) agents can modify the infrastructure that governs them (mutable control plane), (2) autonomous velocity exceeds human verification bandwidth, (3) the system's self-reporting masks degradation. The fix is control plane immutability: gates, metrics, and circuit breakers must be architecturally unreachable by the agents they constrain.

---

## Core Mechanism

### The Feedback Loop

```
Agent makes locally-correct change
    |
Change alters ground truth for next agent
    |
Next agent makes locally-correct change against new ground truth
    |
System drifts from coherent state
    |
Drift detected -> agents spawn fixes
    |
Fixes alter ground truth further
    |
Spiral accelerates
```

**Key property:** Every individual commit is correct. The agents aren't broken. The composition of correct pieces is what fails — and it only fails when changes outpace verification.

### Three Enabling Conditions (all required)

<!-- Claims in models should be explicit about their conditions.
     "X happens" is weaker than "X happens when A, B, and C are true." -->

1. **Mutable control plane** — Agents can modify the dashboard, status logic, spawn system, verification gates, and completion pipeline. Each fix to infrastructure changes the rules for the next agent. This is the structural vulnerability.

2. **Velocity exceeding verification bandwidth** — The system produces changes faster than a human can verify behavior. At 45 commits/day, nobody can verify. Unverified velocity has negative value (0.96:1 fix-to-feature ratio means each feature produces nearly one bug).

3. **Self-referential reporting** — Every signal saying "it's working" comes from inside the system. Commit messages say "fix:", synthesis files say "success", the daemon continues spawning. No external verification exists to contradict the self-report.

### Escalation Pattern (empirically observed)

| Spiral | Duration | Commits | Trigger | Recovery |
|--------|----------|---------|---------|----------|
| First | 24 hours | 115 | Agents fixing agent infra | Rollback |
| Second | 6 days | 347 | Same root causes | Rollback |
| Third | 26 days | 1,163 | Same root causes | Rollback |

Each spiral was longer, produced more damage, and repeated the exact same root causes — because mitigations were documented but never implemented as hard gates.

---

## Why This Fails

<!-- "Why This Fails" sections name specific failure modes
     with concrete evidence, not abstract concerns. -->

### Failure Mode 1: Self-Modifying Infrastructure

Agents modify the code that tracks, spawns, and verifies agents. In the third spiral: 175 command files churned, 84 daemon files deleted, an entire subsystem accidentally deleted by a sync operation.

**The recursive problem:** Defenses against the spiral (gates, metrics, coaching) live inside the system. If agents modify the system, they modify the defenses. The pattern that causes the spiral is the same pattern that erodes protection against it.

### Failure Mode 2: Agent Self-Stabilization

Empirically proven to fail. A stabilization attempt by agents ("abandon contaminated agents, strip triage labels") was followed by 150 more commits in 3 days. Zombie processes, 8.4GB memory, crashes. Agents cannot halt a spiral they're part of.

### Failure Mode 3: Documentation Without Implementation

Three post-mortems. Five specific mitigations per post-mortem. All documented. None implemented. The system learned intellectually (investigations, principles) but not structurally (circuit breakers, hard gates). Documented lessons are data-plane artifacts — agents can read them, ignore them, or modify them.

---

## Constraints

<!-- Constraints are the prescriptive output of the model.
     They state what the system must do to prevent the failure
     modes described above. -->

- **Verification Bottleneck** — The system cannot change faster than a human can verify behavior. This is a real constraint, not a suggestion.
- **Control plane immutability** — Gates, metrics, circuit breakers, and verification infrastructure must be unreachable by agents. The human evolves the control plane; agents operate within it.
- **Local correctness != global correctness** — Correct pieces don't compose into a working system when changes outpace verification.
- **Agent self-stabilization fails** — Proven empirically across 3 spirals. The system cannot stabilize itself.

---

## The Seven Implications

1. **Control/data plane separation is not optional.** Agents modify application code (data plane) but not lifecycle infrastructure (control plane).
2. **Local correctness is fundamentally different from global correctness.** Traditional compositionality breaks under high-velocity multi-agent modification.
3. **Meta-oversight resolves via immutability, not hierarchy.** Adding watchers doesn't help — making the watching infrastructure unmodifiable does.
4. **Unverified velocity has negative value.** 0.96:1 fix-to-feature ratio means net contribution is negative after accounting for churn.
5. **Documentation doesn't prevent recurrence; immutable infrastructure does.** Mitigations as docs = data plane. Mitigations as gates = control plane.
6. **Pain-as-signal needs control-plane circuit breakers.** Agent-level friction detection exists; system-level halt does not.
7. **Verification bandwidth is a control-plane constraint.** Rate limits, commit caps, cooldowns — these pace velocity to verification bandwidth.

---

## References

<!-- Models track their evidence via merged probes and primary evidence.
     Probes are targeted experiments that test specific model claims.
     When a probe is merged, its findings are incorporated into the
     model text above, and the probe is listed here for provenance. -->

### Merged Probes

| Probe | Date | Key Finding |
|-------|------|-------------|
| Self-stabilization current gates | 2026-03-01 | All 48 gates across 4 layers remain agent-modifiable; mutable control plane gap CONFIRMED |
| Fix-feat ratio gate effectiveness | 2026-03-01 | Fix:feat ratio 0.88:1 (target 0.3:1); 42% of fixes are infrastructure fixes (gates fixing gates); gates convert catastrophic to gradual degradation |

### Key Statistics

- 1,625 commits lost across 3 rollbacks
- 5.4M LOC churn in third spiral (3.5M added, 1.8M deleted)
- 5,244 files created then deleted (33% of all created)
- 0.96:1 fix-to-feature ratio
- 0 human commits in 26 days
- 3 post-mortems with identical root causes, 0 mitigations implemented between them
