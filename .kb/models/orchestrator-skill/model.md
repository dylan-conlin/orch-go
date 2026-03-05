# Model: Orchestrator Skill Design

**Domain:** Skill authoring and behavioral compliance for orchestrator agents
**Last Updated:** 2026-03-05
**Synthesized From:** 6 investigations (Jan 18 - Mar 5, 2026)

---

## Summary (30 seconds)

The orchestrator skill is a knowledge transfer mechanism, not a behavior enforcement mechanism. Behavioral testing proves that knowledge items (routing tables, vocabulary, framing protocols) produce measurable lift over bare Claude, while prohibition-based constraints (NEVER/MUST statements) hit dilution at ~5 co-resident constraints and become inert at 10+. Hook infrastructure handles enforcement; the skill handles comprehension. The successful simplification from 2,368 to 422 lines (82% reduction) validated this split.

---

## Core Mechanism

The skill operates through **two independent channels** that were conflated for months before being separated:

1. **Knowledge transfer** (skill document) — Routing tables, vocabulary definitions, framing protocols, intent clarification patterns. These give the agent information it wouldn't otherwise have. Behavioral testing shows +5 point lift (22/56 vs 17/56 bare).

2. **Behavioral enforcement** (hook infrastructure) — 6+ active hooks intercept prohibited tool usage (bash-write gate, git-remote gate, bd-close gate, spawn-ceremony nudge, investigation-drift nudge, spawn-context validation). These prevent actions regardless of what the skill document says.

### Key Components

| Component | Channel | Sticks? | Evidence |
|-----------|---------|---------|----------|
| Routing tables (skill→intent mapping) | Knowledge | Yes (+4/8) | Behavioral testing Mar 1 |
| Framing vocabulary (Thread→Insight→Position) | Knowledge | Yes (+2/8) | Behavioral testing Mar 1 |
| Intent clarification (experiential vs production) | Knowledge | Yes (+2/8) | Intent spiral case study Feb 28 |
| Delegation prohibition ("never implement") | Enforcement | No (1/8 = bare) | Behavioral testing Mar 1 |
| Anti-sycophancy ("don't just validate") | Enforcement | No (3/8 = bare) | Behavioral testing Mar 1 |
| Reconnection framing (3-layer protocol) | Enforcement | No (0-1/8 = bare) | Behavioral testing Mar 1 |

### Critical Invariants

1. **Skill length ≤ 500 lines / 5,000 tokens** — Beyond this, constraint dilution makes additional content inert
2. **≤ 4 behavioral norms** — Research shows dilution begins at 5 co-resident constraints
3. **Knowledge framing, not prohibition** — "Here's how routing works" beats "NEVER route incorrectly"
4. **Hook-enforced behaviors must NOT appear in skill text** — Dual authority creates confusion about what's enforced vs advisory

---

## Why This Fails

### Failure Mode 1: Constraint Dilution
At 10+ MUST/NEVER statements, agents treat ALL constraints as advisory. The 2,368-line v3 skill had 50+ constraints — effectively equivalent to bare Claude on 5 of 7 test scenarios.

### Failure Mode 2: Instruction Hierarchy Inversion
Claude Code's system prompt has ~500 words promoting Task tool; a skill's ~30 words constraining it face a 17:1 signal disadvantage. System prompt > user prompt by design. Prompt-level constraints cannot reliably override system-level defaults.

### Failure Mode 3: Dual Authority
When both a hook AND skill text prohibit the same action, agents receive conflicting signal types (infrastructure block vs prose guidance). This creates ambiguity about enforcement level and degrades trust in the skill overall.

### Failure Mode 4: Mechanical Staleness
The skill drifts from infrastructure quickly. 72 commits in 3 days introduced 10 changes, 6 requiring skill edits. Without a sync mechanism, the skill describes a system that no longer exists.

---

## Constraints

### Why can't we enforce behavior through skill text alone?

**Constraint:** Instruction hierarchy (system > user) means user-level skill content is structurally subordinate to system prompt defaults.

**Implication:** Behavioral constraints in skills are probabilistic suggestions, not deterministic rules. Only infrastructure hooks provide deterministic enforcement.

**This enables:** Dramatically simpler skill documents focused on knowledge transfer
**This constrains:** All behavioral enforcement must be implemented as hooks, not prose

### Why ≤ 4 behavioral norms?

**Constraint:** Empirically validated dilution threshold. At 5+ co-resident behavioral constraints, compliance drops toward bare baseline.

**Implication:** Every behavioral norm competes with every other. Adding a 5th doesn't add — it degrades the existing 4.

**This enables:** Ruthless prioritization of which behaviors matter most
**This constrains:** Cannot add "just one more" constraint without removing one

### Why must behavioral testing use pattern-match scoring, not LLM-as-judge?

**Constraint:** LLM-as-judge creates a closed evaluation loop — same model family evaluating same model family. Behavioral proxies (what the agent actually does) are more honest than self-assessment.

**Implication:** Test scenarios must define observable action patterns (contains/doesn't-contain), not qualitative rubrics.

**This enables:** Reproducible measurement of skill impact across versions
**This constrains:** Some subtle behaviors (tone, framing quality) are harder to test

---

## Evolution

**2026-01-18:** Skill is evolvable — added frustration trigger protocol as mode-shift gate. Skill operates as monolith (identity + constraints + knowledge all interleaved).

**2026-02-24:** Discovered structural problem. Identity compliance is additive (layers on defaults), but action constraints are subtractive (fight defaults). The skill was trying to do both, succeeding at identity, failing at constraints. Recommended two-layer fix: prompt restructuring + infrastructure enforcement.

**2026-02-28:** Intent spiral case study revealed skill's routing table is its most valuable section — caught ambiguous "evaluate Playwright CLI vs MCP" intent that bare Claude would route incorrectly.

**2026-03-01:** Behavioral testing infrastructure built. Measured: v3 skill scores 22/56 vs bare 17/56. 5-point lift concentrated in knowledge transfer (routing, vocabulary, framing). 5 of 7 scenarios at bare parity = dead-weight constraint text.

**2026-03-04:** Simplified v4 deployed: 2,368→422 lines (82% reduction). Removed all hook-enforced constraint text. Kept ≤4 behavioral norms as knowledge framing. Behavioral gate pending.

**2026-03-05:** 72-commit infrastructure delta created 6 mechanical mismatches. Skill drifts faster than anticipated — sync mechanism needed.

---

## References

**Investigations:**
- `.kb/investigations/2026-01-18-inv-update-orchestrator-skill-add-frustration.md` - Added frustration trigger protocol
- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` - Discovered instruction hierarchy problem, 17:1 signal disadvantage
- `.kb/investigations/2026-03-01-design-infrastructure-systematic-orchestrator-skill.md` - Built behavioral testing infrastructure, 3-layer measurement
- `.kb/investigations/2026-03-04-design-simplify-orchestrator-skill.md` - Validated knowledge-only approach, deployed v4
- `.kb/investigations/2026-03-05-inv-design-orchestrator-skill-update-incorporating.md` - 72-commit delta sync, 6 surgical edits

**Decisions informed by this model:**
- `.kb/decisions/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` - Two-layer enforcement architecture

**Related models:**
- `.kb/models/architectural-enforcement/` - How hook infrastructure works
- `.kb/models/coaching-plugin/` - Agent behavioral coaching layer
