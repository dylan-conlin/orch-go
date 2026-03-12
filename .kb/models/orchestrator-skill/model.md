---

# Orchestrator Skill Model

## What This Model Captures

The orchestrator skill is the system's primary soft harness for shaping orchestrator agent behavior. This model captures the skill's nature, failure modes, design tensions, and the architectural principles that emerged from 6 investigations spanning Jan-Mar 2026.

## Core Nature

The orchestrator skill is a **probability-shaping document** — not a grammar, not a rulebook. It shifts behavioral distributions but provides 0% formal guarantee of compliance. It operates in a hostile signal environment where Claude Code's system prompt actively promotes competing behaviors (Task tool, Edit/Write, direct investigation) at a 17:1 signal advantage.

The skill has two jobs with incompatible scaling properties:
- **Knowledge transfer** (~50 item budget): routing tables, vocabulary, intent distinctions, tool ecosystem, Dylan interface protocols. These survive prompt dilution and show measurable behavioral lift over bare Claude.
- **Behavioral shaping** (~4 norm budget): delegation, filtering, act-by-default, answer-the-question-asked. These fail under dilution at 5+ co-resident constraints and reach bare parity at 10+.

**Caveat:** The behavioral budget (~2-4) is based on N=3 unreplicated experiments. The knowledge budget (~50) has definitional ambiguity — counting table rows yields ~84 items in the current skill. Both thresholds are directional hypotheses, not established findings.

## The Two-Layer Architecture

The critical design insight (Feb 24, 2026): behavioral enforcement requires infrastructure, not prompts.

| Layer | What It Does | Mechanism | Scaling |
|-------|-------------|-----------|---------|
| **Prompt** (skill content) | Knowledge transfer: routing tables, vocabulary, protocols, intent distinctions | SKILL.md loaded at session start | ~50 items before dilution degrades |
| **Infrastructure** (hooks) | Behavioral enforcement: tool restrictions, code access, spawn ceremony, completion gates | 6 PreToolUse hooks in ~/.claude/settings.json | Deterministic, no dilution |

The skill tells agents what to DO (knowledge). Hooks prevent what they SHOULDN'T (behavior). This resolves the competing-instruction-hierarchy problem by moving enforcement out of the signal-ratio competition entirely.

### Hook Coverage (6 active, 1 missing)

| Hook | Enforces | Tests |
|------|----------|-------|
| gate-bd-close | Only orch complete closes issues | 37 |
| gate-orchestrator-bash-write | No Edit/Write for orchestrators | 308 |
| gate-orchestrator-git-remote | No git push for workers | 64 |
| gate-spawn-context-validation | --issue and --intent required | 68 |
| nudge-orchestrator-investigation-drift | Coaching on code reads | 38 |
| nudge-orchestrator-spawn-context | kb context before spawn | 40 |
| code-access gate (designed, NOT registered) | Block code file reads | 0 |

## Critical Invariants

1. **Behavioral constraint budget is ~4.** Adding a 5th behavioral norm requires removing one. The budget is at capacity.
2. **Knowledge framing, not prohibition framing.** "The system works like Y" not "NEVER do X." Prohibition framing triggers MUST fatigue and competes with system prompt at 17:1 disadvantage.
3. **Infrastructure enforces, prompts describe.** Any behavioral boundary that matters enough to enforce must have a hook. Prompt-only constraints are probabilistic guidelines, not enforcement.
4. **Hook-enforced constraints are removed from the skill.** If a hook enforces it, the skill doesn't repeat it. This prevents accretion of redundant prohibition text.
5. **The skill undergoes accretion-crisis cycles.** New features create legitimate pressure to add content. Without active measurement (bare-parity testing), the skill will regrow to crisis levels in ~2-3 months.

## Why This Fails (12 Failure Modes)

### Layer A: Prompt-Level Failures

**A1. Frame Collapse** — Orchestrator drops into worker-level implementation. Triggered by agent failure ("let me just fix it"). Defense-in-depth signals exist but unreliable for self-diagnosis. Status: **Open/fundamental.**

**A2. Competing Instruction Hierarchy** — Correct identity ("I'm an orchestrator") but wrong tools (Task tool instead of orch spawn). Claude Code system prompt has 17:1 signal advantage. Status: **Partially mitigated** by hooks + action-identity fusion.

**A3. Behavioral Constraint Dilution** — Constraints cancel each other at 5+ co-resident. Production skill had 50+ constraints. Status: **Resolved** by 82% size reduction + ≤4 norm budget.

**A4. MUST Fatigue** — Excessive emphasis language (20+ NEVER directives) creates "cry wolf" effect. Status: **Resolved** by knowledge framing shift.

### Layer B: Infrastructure-Level Failures

**B1. Skill Injection Failure** — Cross-project orchestrators receive no skill content. Two-bug chain in session detection. Status: **Open** — fix designed, not implemented.

**B2. Content Staleness** — CLI evolves faster than skill updates. 13 stale references found (7 harmful). Init-time caching means skillc deploy requires restart. Status: **Partially mitigated** — Mar 5 delta applied, caching bug remains.

**B3. Self-Termination** — Spawned orchestrators try to self-terminate instead of waiting. Status: **Resolved** by template fix (Jan 2026).

### Layer C: Structural Failures

**C1. Cascaded Intent Displacement** — Human intent reshaped at each translation layer. "Evaluate" becomes "audit." No routing path for experiential work. Status: **Open/fundamental** — intent clarification added but multi-layer frame attenuation is inherent.

**C2. Error-Correction Feedback Loop** — Corrections make orchestrator more anxious → over-corrects → drifts further. Skill checklists become amplifiers. Status: **Open/fundamental** — may be an LLM behavioral pattern.

**C3. Orientation-Identity Mismatch** — Skill organized for orchestrator self-description, not Dylan's needs. Status: **Partially mitigated** — v4 reorganized around Dylan's four orientation moments, but role section still leads.

### Layer D: Temporal Failures

**D1. Temporal Attention Decay** — Skill constraints lose salience over session duration while system prompt remains persistent. Status: **Open/fundamental** — no mitigation exists.

**D2. Knowledge Surfacing Gap** — Interactive session debriefs evaporate when session closes. Next orchestrator lacks comprehension context. Status: **Partially mitigated** by orch debrief command.

### Interaction Effects

| Interaction | Mechanism |
|-------------|-----------|
| A3 amplifies A2 | More constraints = weaker per-constraint signal = less resistance to system prompt |
| C2 amplifies C1 | Corrections drive orchestrator deeper into wrong methodology |
| D1 amplifies A2 | System prompt constant, skill signal decays → widening ratio over time |
| B1 makes A1-A4 moot | No skill injection = agent operates as generic assistant |
| B2 creates false C1 | Stale commands cause confusion cascades |

### Evolution Drivers

Three failure modes drove the major evolutionary changes:
- **A3 (Dilution)** → 82% size reduction (Mar 4)
- **A2 (Competing Hierarchy)** → Hook-based enforcement strategy (Feb-Mar)
- **C1 (Intent Displacement)** → Intent clarification additions (Feb 28+)

## Design Tensions (9 Tensions, 3 Categories)

### Fundamental (require management, not resolution)

**T1. Knowledge-Transfer vs Behavioral-Constraint** — Both needed, incompatible scaling (~50 vs ~4). Managed via two-layer architecture.

**T2. Grammar vs Probability-Shaper** — Authors write rules (NEVER/MUST); runtime treats them as probability adjustments. The gap between authoring intuition and operational reality drives over-specification. Managed via knowledge framing.

**T3. Simplicity vs Completeness** — New features create legitimate accretion pressure. Without measurement, growth is the default. Managed via constraint budgets + reference doc offloading.

**Key insight:** Treating fundamental tensions as solvable caused the accretion cycle (640→2,368 lines). They require management strategies (budgets, measurement, infrastructure offloading), not resolution attempts.

### Resolved

**T4. Prompt vs Infrastructure Enforcement** — Infrastructure wins for behavioral boundaries. 6 hooks replace ~350 lines of prohibition text.

**T5. Accretion vs Simplification** — Constraint budgets + hook offloading. Subject to regression pressure from T3.

**T6. Identity vs Action Compliance** — Action-identity fusion + hooks. Identity is additive (no conflict); actions are subtractive (compete with system prompt).

**T7. Orchestrator-Centric vs Dylan-Centric** — Partially resolved. v4 reorganized around Dylan's orientation moments. Role section still leads. (Contested: Worker 1 says "partially mitigated," Worker 2 says "resolved.")

### Live (need ongoing work)

**T8. Testing Feasibility vs Measurement Need** — skillc test blocked by CLAUDECODE env var. Constraint budgets are untested hypotheses. The behavioral validation gate for v4 was NEVER completed.

**T9. Soft-Preference Compliance** — Some preferences (orch spawn vs bd create, kb context before spawn) can't be hooked because both options are legitimate. Prompt-level compliance ceiling estimated at ~70-80%.

## The Accretion-Crisis Cycle

Token trajectory shows clear cyclical pattern:
- Dec 2025: 12K → 20K (accretion)
- Jan 2026: 20K → 12K → 16K (crisis + regrowth)
- Feb 2026: 16K → 6.5K → 9K → 27.2K (two trims + rapid accretion to PEAK)
- Mar 2026: 27.2K → 4.8K → 6.0K (v4 simplification + regrowth starting)

Current regrowth: +24% in 7 days (4,830→5,995 tokens). Projected: ~10K tokens by early April at current rate.

**Anti-accretion mechanisms:**
- Constraint budgets (≤4 behavioral, ~50 knowledge)
- Reference doc offloading (non-essential content to .skillc/reference/)
- Bare-parity testing (measures whether additions are earning their keep)
- Token monitoring threshold (trigger review at 7K)

## Current State (Mar 11, 2026)

- **Size:** 486 lines template / 512 lines deployed / 5,995 tokens
- **Structure:** 16 named sections (domain-organized, not lifecycle-ordered)
- **Behavioral norms:** Exactly 4 (delegation, filter, act-by-default, answer-the-question)
- **Hook coverage:** 6/7 registered, 555+ tests
- **Implementation rate:** 22/25 recommendations from investigations 1-5
- **CRITICAL PENDING:** Behavioral validation via skillc test never completed

## Open Questions

1. Is the ~2-4 behavioral norm budget the right number? (N=3, unreplicated)
2. What counts as a "knowledge item" for budget purposes? (50 conceptual units vs 84 table rows)
3. Does v4 actually outperform bare Claude? (No behavioral data exists)
4. Can the error-correction feedback loop (C2) be addressed in the skill, or is it fundamental to LLMs?
5. Would decomposing the skill into multiple single-purpose documents resolve T3?
