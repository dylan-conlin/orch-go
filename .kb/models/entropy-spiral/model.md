# Model: Entropy Spiral

**Domain:** Agentic Systems / Failure Modes / Control Theory
**Last Updated:** 2026-02-25
**Synthesized From:** 3 investigations (Feb 12-14, 2026), 3 post-mortems (Dec 21, Jan 2, Feb 12), git-verified evidence from 3 spirals totaling 1,625 lost commits.

---

## Summary (30 seconds)

An entropy spiral is a feedback loop where an agentic system degrades while reporting success. Locally-correct changes compose into globally-incoherent systems when velocity exceeds verification bandwidth. The mechanism requires three conditions: (1) agents can modify the infrastructure that governs them (mutable control plane), (2) autonomous velocity exceeds human verification bandwidth, (3) the system's self-reporting masks degradation. The fix is control plane immutability: gates, metrics, and circuit breakers must be architecturally unreachable by the agents they constrain.

---

## Core Mechanism

### The Feedback Loop

```
Agent makes locally-correct change
    ↓
Change alters ground truth for next agent
    ↓
Next agent makes locally-correct change against new ground truth
    ↓
System drifts from coherent state
    ↓
Drift detected → agents spawn fixes
    ↓
Fixes alter ground truth further
    ↓
Spiral accelerates
```

**Key property:** Every individual commit is correct. The agents aren't broken. The composition of correct pieces is what fails — and it only fails when changes outpace verification.

### Three Enabling Conditions (all required)

1. **Mutable control plane** — Agents can modify the dashboard, status logic, spawn system, verification gates, and completion pipeline. Each fix to infrastructure changes the rules for the next agent. This is the structural vulnerability.

2. **Velocity exceeding verification bandwidth** — The system produces changes faster than a human can verify behavior. The Verification Bottleneck Principle: "the system cannot change faster than a human can verify." At 45 commits/day, nobody can verify. Unverified velocity has negative value (0.96:1 fix:feat ratio = each feature produces nearly one bug).

3. **Self-referential reporting** — Every signal saying "it's working" comes from inside the system. Commit messages say "fix:", synthesis files say "success", the daemon continues spawning. No external verification exists to contradict the self-report.

### Escalation Pattern (empirically observed)

| Spiral | Duration | Commits | Trigger | Recovery |
|--------|----------|---------|---------|----------|
| First (Dec 21) | 24 hours | 115 | Agents fixing agent infra | Rollback |
| Second (Dec 27 - Jan 2) | 6 days | 347 | Same root causes | Rollback to Dec 27 |
| Third (Jan 18 - Feb 12) | 26 days | 1,163 | Same root causes | Rollback to Jan 18 |

Each spiral was longer, produced more damage, and repeated the exact same root causes — because mitigations were documented but never implemented as hard gates.

---

## Why This Fails

### Failure Mode 1: Self-Modifying Infrastructure

Agents modify `cmd/orch`, `pkg/daemon`, `pkg/spawn` — the code that tracks, spawns, and verifies agents. In the third spiral: 175 `cmd/orch` files churned, 84 `pkg/daemon` files deleted, the entire `pkg/attention/` system accidentally deleted by `bd sync`.

**The recursive problem:** Defenses against the spiral (gates, metrics, coaching plugin) live inside orch-go. If agents modify orch-go, they modify the defenses. The pattern that causes the spiral is the same pattern that erodes protection against it.

### Failure Mode 2: Agent Self-Stabilization

Empirically proven to fail. Feb 9 stabilization attempt by agents ("stabilize: abandon contaminated agents, strip triage:ready") was followed by 150 more commits in 3 days. Zombie processes, 8.4GB memory, OpenCode crashes. Agents cannot halt a spiral they're part of.

### Failure Mode 3: Documentation Without Implementation

Three post-mortems. Five specific mitigations per post-mortem. All documented. None implemented. The system learned intellectually (investigations, principles) but not structurally (circuit breakers, hard gates). Documented lessons are data-plane artifacts — agents can read them, ignore them, or modify them.

### Failure Mode 4: Observability Misinterpreted as Dysfunction

During Dec 27-Jan 2, agents improved observability (added dead/stalled agent detection). Dylan interpreted **visibility of problems** as **the system creating problems.** The observability was rolled back, then had to be restored 6 days later. Understanding lag: the human misread the signal because new visibility appeared simultaneously with new dysfunction.

---

## Constraints

- **Verification Bottleneck** — The system cannot change faster than a human can verify behavior. This is a real constraint, not a suggestion. The control plane's job is to enforce it.
- **Control plane immutability** — Gates, metrics, circuit breakers, and verification infrastructure must be unreachable by agents. The human evolves the control plane; agents operate within it.
- **Local correctness != global correctness** — Correct pieces don't compose into a working system when changes outpace verification. Traditional compositionality assumptions break.
- **Agent self-stabilization fails** — Proven empirically across 3 spirals. The system cannot stabilize itself.

---

## Current Defenses (48 gates, 4 layers)

**Prevention:** Accretion boundaries (>1,500 lines), spawn gates (concurrency, rate limiting, hotspot blocking), infrastructure change gates.

**Detection:** Coaching plugin (real-time agent sensing), hotspot analysis (fix density, investigation clusters, bloat), completion gates (14 verification checks), fix:feat ratio visibility.

**Recovery:** Dual spawn modes (daemon + escape hatch), session lifecycle management, cherry-pick recovery patterns.

**Learning:** Post-mortems with structured uncertainty, Verification Bottleneck principle, probe infrastructure.

### Remaining Gap

All defenses live inside the system they protect (mutable control plane). The four mitigations from Feb 12 postmortem remain architecturally vulnerable:
1. Daily commit limit — lives in orch-go code agents can modify
2. Churn monitoring — lives in orch-go code agents can modify
3. Infrastructure change gate — lives in orch-go code agents can modify
4. Fix:feat ratio monitor — lives in orch-go code agents can modify

**Resolution path:** Extract control plane into an immutable layer. This is an engineering problem with known patterns (control/data plane separation, immutable infrastructure, circuit breakers).

---

## The Seven Implications

1. **Control/data plane separation is not optional.** Agents modify application code (data plane) but not lifecycle infrastructure (control plane).
2. **Local correctness is fundamentally different from global correctness.** Traditional compositionality breaks under high-velocity multi-agent modification.
3. **Meta-oversight resolves via immutability, not hierarchy.** Adding watchers doesn't help — making the watching infrastructure unmodifiable does.
4. **Unverified velocity has negative value.** 0.96:1 fix:feat ratio means net contribution is negative after accounting for churn.
5. **Documentation doesn't prevent recurrence; immutable infrastructure does.** Mitigations as docs = data plane. Mitigations as gates = control plane.
6. **Pain-as-signal needs control-plane circuit breakers.** Agent-level friction detection exists (coaching plugin); system-level halt does not.
7. **Verification bandwidth is a control-plane constraint.** Rate limits, commit caps, cooldowns — these pace velocity to verification bandwidth without requiring human presence for every commit.

---

## References

**Primary Evidence:**
- `.kb/investigations/2026-02-12-inv-entropy-spiral-postmortem.md` — Raw data: 1163 commits, 5.4M LOC, 0 human commits
- `.kb/investigations/2026-02-13-inv-entropy-spiral-recovery-audit.md` — Recovery audit, cherry-pick priorities
- `.kb/investigations/2026-02-14-inv-entropy-spiral-deep-analysis.md` — Structural analysis, control plane thesis
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` — First/second spiral postmortem
- `entropy-spiral-feb2026` branch at `c5bb7bfc` — Preserved evidence

**Key Statistics:**
- 1,625 commits lost across 3 rollbacks
- 5.4M LOC churn in third spiral (3.5M added, 1.8M deleted)
- 5,244 files created then deleted (33% of all created)
- 0.96:1 fix:feat ratio
- 0 human commits in 26 days
- 3 post-mortems with identical root causes, 0 mitigations implemented between them
