# Model: Orchestrator Skill

**Domain:** Orchestrator / Skill Design / Behavioral Constraint Architecture
**Created:** 2026-03-12
**Status:** Active — quantitative thresholds are directional hypotheses pending replication. Qualitative findings are robust.
**Synthesized From:** 7 investigations (Feb 24 – Mar 11, 2026) and 4 probes on behavioral compliance, constraint dilution, emphasis language, and skill simplification. Evidence base: 67 extracted claims — 39 measured, 26 analytical, 2 assumed. 14 high-confidence (multi-source verified).

---

## Summary (30 seconds)

The orchestrator skill is a probability-shaping document that fuses three content types — knowledge, stance, and behavioral constraints — into a single artifact injected into orchestrator sessions. Knowledge transfers reliably. Stance transfers indirectly but powerfully. Behavioral constraints compete with the Claude Code system prompt at a ~17:1 signal disadvantage and dilute as constraint count increases. The design response is two-layer enforcement: keep knowledge and stance in the skill document; move behavioral constraints to infrastructure (hooks, tool interception). The skill follows a recurring accretion-crisis-simplification cycle (12K → 27K → 5K → 6K tokens), managed by a ~4-constraint behavioral budget. The specific budget number is hypothesized — the underlying dilution experiment did not replicate.

---

## Model Boundary

This model owns **how the orchestrator skill document shapes orchestrator behavior**: injection mechanisms, dilution dynamics, enforcement strategies, failure modes, and design tensions.

**Orchestrator-session-lifecycle** owns **how orchestrator sessions work**: session types, state derivation, checkpoint discipline, hierarchical completion, resume protocol. When a failure mode like frame collapse spans both models, session-lifecycle owns detection and prevention patterns; this model owns the root cause analysis (why framing overrides skill instructions).

**Behavioral-grammars** (`.kb/global/models/behavioral-grammars/`) owns general principles: constraints are probabilistic, redundancy provides phase coverage, situational pull overwhelms static reinforcement. This model applies those principles to the specific orchestrator skill context — do not duplicate general theory here.

---

## Core Claims

### Claim 1: The skill is a probability-shaping document, not a grammar

Skills loaded into LLM context have 0% formal enforcement guarantee. They shift output distributions but cannot mechanically prevent violations. Every agent framework surveyed (8/8) moved from prompt-level to infrastructure enforcement for critical behavioral constraints.

**Evidence quality:** Multi-source analytical (2 investigations). Supported by agent framework landscape survey (Mar 1, 2026).

### Claim 2: Knowledge transfers reliably; behavioral constraints don't

The fundamental asymmetry of skill content. Knowledge constraints (routing table structure, intent types, vocabulary) produce measurable lift over bare Claude (+4/8 on routing, +2/8 on framing, +2/8 on intent). Behavioral constraints (delegate, don't use Task tool, don't read code) show zero measurable lift on 5 of 7 tested scenarios — bare parity.

**Evidence quality:** Highest-confidence finding in the cluster. Confirmed across 4 independent sources: behavioral compliance probe (Feb 24), testing baseline (Mar 1), simplification investigation (Mar 4), grammar-first investigation (Mar 4).

### Claim 3: Identity compliance is not predictive of action compliance

Orchestrators reliably adopt identity declarations ("I'm an orchestrator") while failing action constraints ("use orch spawn, not Task tool"). These operate on different dimensions: identity is additive (no conflict with system prompt), action is subtractive (conflicts with built-in affordances). The 17:1 signal ratio (system prompt promoting Task tool vs skill constraining it) makes action compliance structurally unwinnable at prompt level alone.

**Evidence quality:** Multi-source measured (3 sources). The 17:1 ratio is a word-count measurement, not a model assumption.

### Claim 4: The behavioral constraint budget is HYPOTHESIZED at ~2-4 co-resident constraints

The dilution curve experiment (sonnet, N=3, Mar 1) found: 1-2 constraints achieve ceiling compliance; at 5+, variance returns; at 10, behavioral constraints regress to bare parity. However:

**The dilution curve did NOT replicate under clean isolation** (orch-go-zola, Mar 4). All specific threshold numbers (behavioral budget ~2-4, degradation at 5, bare parity at 10) are directional hypotheses, not established facts. The qualitative direction (more constraints = worse behavioral compliance) is supported by multiple sources. The specific numbers are artifacts of a single N=3 experiment that failed replication.

**Downstream propagation failure:** Despite the replication caveat, 4 downstream artifacts (investigations #4, #5, #7, and the session-lifecycle model itself) cited these thresholds as established. This is a systemic risk in the knowledge system — caveats added to probes don't flow to citing artifacts.

**Evidence quality:** Measured but caveated (single-source, replication failed, N=3).

### Claim 5: Two-layer enforcement is required

Layer 1 (prompt): Knowledge and stance content in the skill document. These transfer reliably and don't require infrastructure enforcement. Layer 2 (infrastructure): Behavioral constraints via hooks and tool interception. 6 of 7 designed hooks are active (bash-write: 308 tests, git-remote: 64, bd-close: 37, investigation-drift: 38, spawn-ceremony: 40, spawn-context: 68). The 7th (code-access gate) is not registered.

**Evidence quality:** Highest-replication analytical finding (5 sources across 5 investigations).

### Claim 6: Emphasis language provides partial compliance lift at high constraint counts

CRITICAL/MUST/NEVER markers outperform neutral language (should/prefer/consider) at 5+ constraints. At 10C: emphasis produces 2/3 proposes-delegation vs neutral 0/3 (identical to bare). Combined across sessions: emphasis 10C = 33% compliance, neutral 10C = 0%.

**Tension with MUST fatigue:** The MUST-density linter (>3 per 100 words = warning) treats emphasis as an anti-pattern, while experiments show it helps. This contradiction is unresolved (see RC-2 in Design Tensions).

**Evidence quality:** Measured but noisy (N=3 per variant, single-turn --print mode, replication failure caveat inherited from dilution curve).

---

## Three Content Types

The skill fuses three artifact types with different transfer mechanisms and budgets. This taxonomy refines the knowledge/behavioral binary used in earlier work.

| Type | Transfer Mechanism | Budget | Examples |
|------|-------------------|--------|----------|
| **Knowledge** | Information addition. Reliable, survives high constraint counts. | ~50+ items | Routing table, intent types, vocabulary definitions, skill descriptions |
| **Stance** | Epistemic orientation shift. Transfers indirectly via attention allocation, not information. | 1-3 lines | "Test before concluding," "evidence hierarchy," "artifacts are claims, not evidence" |
| **Behavioral** | Action suppression. Competes with model priors and system prompt. Dilutes rapidly. | ~2-4 constraints (hypothesized) | Delegation rule, don't use Task tool, filter before presenting |

**Why stance is distinct:** Knowledge tells the agent what exists; stance orients how it approaches. "Evidence hierarchy" (knowledge) is different from "test before concluding" (stance). Stance produced larger discrimination on hard scenarios than knowledge in the skill-content-transfer trials (N=6: 0%→17%→83% on implicit contradictions). Different transfer mechanism means different design considerations.

**Source:** Behavioral-grammars model, Refinement section (Mar 6, 2026). See `.kb/global/models/behavioral-grammars/model.md` for general theory; `.kb/models/skill-content-transfer/` for specific evidence.

---

## Failure Modes (13)

Synthesized from 6 investigations, 7 probes, and 1 post-synthesis addition (Mar 2026). Organized by layer. Resolution status as of Mar 12, 2026.

### A. Prompt-Level (4) — Failures in skill content or instruction processing

| # | Mode | Root Cause | Status |
|---|------|-----------|--------|
| 1 | **Frame Collapse** — orchestrator drops to worker-level implementation | Vague goals → exploration → debugging. Framing cues override skill instructions | Open / fundamental |
| 3 | **Competing Instruction Hierarchy** — uses Task tool while maintaining orchestrator identity | 17:1 system prompt signal advantage. Identity ≠ action compliance | Partially mitigated (hooks) |
| 4 | **Behavioral Constraint Dilution** — constraints fail in full skill documents | Attention budget competition at 5+ co-resident behavioral constraints | Resolved by simplification (v4: 4 norms) |
| 10 | **MUST Fatigue** — emphasis saturation creates "cry wolf" effect | >3 MUST/NEVER/CRITICAL per 100 words; v3 had 20+ NEVER directives | Resolved by simplification (knowledge framing) |

### B. Infrastructure-Level (3) — Failures in injection, deployment, or enforcement

| # | Mode | Root Cause | Status |
|---|------|-----------|--------|
| 2 | **Self-Termination** — spawned orchestrator tries to /exit | Template contradicted hierarchical completion model | Resolved (template fix, Jan 2026) |
| 8 | **Content Staleness** — skill references nonexistent commands/flags | CLI evolves faster than skill updates; init-time caching persists stale versions | Partially mitigated (v4 + 72-commit audit) |
| 9 | **Cross-Project Injection Failure** — no skill content in non-orch-go projects | Two-bug chain: `is_spawned_agent()` conflation + `.orch/` directory gate on project-independent skill | Open (fix designed, not implemented) |

### C. Structural (3) — Failures in skill architecture or design patterns

| # | Mode | Root Cause | Status |
|---|------|-----------|--------|
| 5 | **State Derivation Disagreement** — different sources disagree on agent status | Four independent state sources with no coordination protocol | Mitigated (priority cascade: beads > Phase > SYNTHESIS > session) |
| 6 | **Cascaded Intent Displacement** — "evaluate" becomes "audit" across translation layers | Routing table lacks experiential paths; heavy skills override weak spawn prompts | Open / fundamental |
| 7 | **Error-Correction Feedback Loop** — corrections amplify anxiety and ceremony | Pre-spawn checklists become amplifiers; optimizes for "don't get corrected" over "understand intent" | Open / fundamental |

### D. Temporal (2) — Failures that emerge over time or across sessions

| # | Mode | Root Cause | Status |
|---|------|-----------|--------|
| 11 | **Temporal Attention Decay** — skill salience decreases over session duration | Skill injected once; system prompt reinforced every turn | Open / fundamental (checkpoint discipline is indirect mitigation) |
| 12 | **Knowledge Surfacing Gap** — interactive session comprehension lost at close | SESSION_HANDOFF.md is for spawned orchestrators only; interactive sessions had no equivalent | Partially mitigated (`orch debrief`) |

### E. Knowledge-Feedback (1) — Cross-cutting, added Mar 2026

| # | Mode | Root Cause | Status |
|---|------|-----------|--------|
| 13 | **Architect Design Bypass** — issue framing overrides prior architect design | 5-layer failure chain: issue framing > kb pointer > no injection mechanism > no skill checkpoint > rushed planning | Unmitigated |

### Failure Mode Interactions

| Interaction | Mechanism |
|-------------|-----------|
| Dilution (#4) amplifies Competing Hierarchy (#3) | More constraints = weaker signal per constraint = less resistance to system prompt |
| Error-Correction (#7) amplifies Intent Displacement (#6) | Corrections drive deeper into wrong methodology |
| Temporal Decay (#11) amplifies Competing Hierarchy (#3) | System prompt constant while skill signal decays |
| Injection Failure (#9) makes all prompt failures (#1-4) irrelevant | No skill = all prompt-level failures are moot |
| Architect Bypass (#13) amplifies Surfacing Gap (#12) | Designs committed to .kb/ but surfaced as low-salience pointers |

### Primary Evolutionary Drivers (Jan→Mar 2026)

Three failure modes drove the majority of the skill's evolution:
1. **Dilution (#4)** → 2,368→448 line simplification (82% reduction)
2. **Competing Hierarchy (#3)** → hook-based enforcement strategy (6 hooks, 555+ tests)
3. **Intent Displacement (#6)** → routing table extension with experiential/production/comparative intent types

---

## Design Tensions (9)

### Fundamental (3) — Never fully resolvable

**T-F1: Knowledge Transfer vs Behavioral Constraint.** The skill must teach (knowledge) and restrain (behavioral). These have different budgets, different transfer mechanisms, and compete for the same context space. More knowledge = more value; more behavioral constraints = less value per constraint.

**T-F2: Skill as Grammar vs Skill as Probability Shaper.** Intuition says skills define valid actions (grammar). Reality says skills shift output distributions (probability). Designing for grammar expectations (completeness, formal consistency) produces bloat. Designing for probability (targeted salience, minimal constraints) requires accepting non-determinism.

**T-F3: Simplicity vs Completeness.** Simpler skills are more effective per-constraint but miss edge cases. Complete skills cover edge cases but dilute core constraints. The accretion cycle is this tension oscillating.

### Managed (4) — Addressed but requiring ongoing discipline

**T-M1: Prompt vs Infrastructure Enforcement.** *Addressed by two-layer architecture:* knowledge/stance in prompt, behavioral in hooks. 6/7 hooks active. Residual: soft preferences (filter-before-presenting, act-by-default) are unhookable — they require judgment, not interception.

**T-M2: Accretion vs Simplification.** *Managed by constraint budget, not resolved.* Token trajectory: 12K → 27K → 5K → 6K. The v4 simplification (Mar 4) cut 82% while preserving knowledge value. But within 7 days, the skill regrew 24% from investigation-driven additions. The accretion forces are structural — each investigation produces "add this to the skill" recommendations. Budget discipline slows the cycle but doesn't stop it.

**T-M3: Structural Identity-Action Gap.** *Addressed by hook infrastructure.* The gap between "I'm an orchestrator" (identity) and "I'll use orch spawn" (action) is bridged by hooks that intercept prohibited tool calls. But hooks only cover the hookable surface — `bash-write`, `git-remote`, `bd-close`, `investigation-drift`, `spawn-ceremony`, `spawn-context`. The Task tool is not yet hooked.

**T-M4: Orchestrator-Centric vs Dylan-Centric Organization.** *Resolved by Feb 2026 restructure.* Skill reorganized around Dylan's 4 orientation moments (spawn, collaborate, status, reconnect) rather than the orchestrator's internal architecture. This is a completed design change, not an ongoing tension.

### Live (2) — Currently unresolved

**T-L1: Testing Feasibility.** `skillc test` is blocked from spawned agent sessions by the `CLAUDECODE` environment variable. This means the v4 skill was deployed without completing its behavioral validation gate. The entire design feedback loop (measure → redesign → validate → deploy) is open. Until this is fixed, skill design changes cannot be validated in the agent context that matters.

**T-L2: Residual Soft-Preference Compliance.** Some behavioral norms (filter-before-presenting, act-by-default) can't be enforced by hooks because they require judgment about what constitutes "presenting unfiltered" or "asking instead of acting." These remain prompt-level only, subject to all the dilution and competing-hierarchy dynamics. Anti-sycophancy shows zero signal across all experiments (3/8 = bare parity).

---

## Structured Uncertainty

### Validated (high confidence)

- Knowledge transfers stick, behavioral constraints don't (4 sources, measured)
- Identity ≠ action compliance — mechanistically different dimensions (3 sources)
- System prompt has ~17:1 signal advantage over skill action constraints (3 sources, measured)
- Two-layer enforcement is necessary — prompt for knowledge, infrastructure for behavioral (5 sources)
- Skill accretion follows crisis-simplification cycle (measured token trajectory)
- 82% token reduction preserved knowledge-transfer value (measured)
- Bare-parity testing works as behavioral validation method (measured, 30+ test runs)

### Directional Hypotheses (evidence exists but unreplicated)

- Behavioral budget ~2-4 constraints (dilution curve did NOT replicate under clean isolation)
- Knowledge budget ~50+ constraints (derived from unreplicated experiment)
- Emphasis language > neutral at high constraint counts (N=3, inherits replication caveat)
- U-curve for process over-application at 5-10 constraints (single source, N=10)
- Emphasis effect larger at higher constraint counts (N=3, directional only)

### Assumed (stated without direct evidence)

- LLM-as-judge is a closed loop per provenance principle (stated as principle, untested)
- Interactive session compliance may differ from --print mode measurements (reasonable assumption, never tested)

### Known Unknowns

- **`CLAUDECODE` blocker:** `skillc test` returns 0/0 scores from spawned agent sessions due to environment variable detection. This is the single largest technical blocker for the behavioral design feedback loop. Until resolved, no skill behavioral claims can be validated in production context.
- **--print vs interactive gap:** All behavioral measurements used single-turn `--print` mode. Real orchestrator sessions have multi-turn context, tools, and hooks. The transfer from lab to production is unknown.
- **Cross-model generalization:** Most experiments ran on sonnet. Opus may have different sensitivity to emphasis, constraint count, and dilution dynamics. The bare baseline shifted 3/8→6/8 between sessions (model/sampling drift).

---

## Accretion Lifecycle

The skill follows a documented cycle:

```
Stable state → feature additions ("add X to the skill") → accretion
→ crisis (dilution, staleness, bloat) → simplification audit
→ 80%+ reduction → regrowth begins immediately → stable state
```

**Token trajectory:** 12,390 (Dec 2025) → 23,908 (Jan 29) → 27,200 (Mar 1 peak) → 4,830 (Mar 4 v4) → 5,995 (Mar 11, +24% in 7 days).

**What drives accretion:** Each investigation produces recommendations framed as "add this knowledge/constraint to the skill." The force is structural — the skill is the only injection point for orchestrator context, so all improvements funnel through it.

**What breaks the cycle:** Behavioral testing gate (measure before/after simplification). Currently broken by CLAUDECODE env var blocker.

---

## Actionable Fix Designs (from probes, not yet implemented)

1. **Cross-project injection fix** (3 changes): Add `ORCH_SPAWNED=1` to spawn command, fix `is_spawned_agent()` to check `ORCH_SPAWNED` not `CLAUDE_CONTEXT`, restructure `main()` to decouple skill loading from `.orch/` gate.
2. **Task tool interception**: PreToolUse hook to intercept Task tool calls when `CLAUDE_CONTEXT=orchestrator` is set.
3. **Architect design checkpoint**: Feature-impl skill gate to verify architect alignment before implementing when `--architect-ref` is provided.
4. **Graduated hook response**: Replace binary nudge/block with graduated severity based on violation frequency.

---

## Open Questions

1. **Does the knowledge-vs-constraint asymmetry hold for other skills** (worker, architect), or is it specific to the orchestrator skill's relationship with the Claude Code system prompt?
2. **What is the actual constraint budget?** The qualitative direction is supported but the specific number 4 is arbitrary. A validated experiment with N≥10 runs under clean isolation would resolve this.
3. **What is the compliance rate in real interactive sessions?** All measurements are from `--print` mode. The gap between lab measurement and production behavior is unknown.
4. **Do emphasis effects hold for opus?** Only tested on sonnet. Cross-model generalization of specific thresholds is untested.
5. **Can the accretion cycle be structurally broken?** Or is it inherent to a system where one document is the sole injection point for orchestrator context?

---

## Evidence Base

### Investigations (7)
- Feb 24: Behavioral compliance — identity vs action gap, 17:1 signal ratio
- Mar 1: Testing infrastructure — bare-parity methodology, behavioral scenarios, DSL linting
- Mar 1: Constraint dilution threshold — dilution curve, replication failure caveat
- Mar 4: Simplification — 27K→5K tokens, 4-norm budget, hook verification
- Mar 4: Grammar-first design — U-curve, matched-pair principle, slot allocation
- Mar 5: 72-commit delta — 13 stale edits, infrastructure changes audit
- Mar 11: Design tension mapping — 9 tensions categorized, resolution status

### Probes (migrated from orchestrator-session-lifecycle)
- `2026-02-24`: Identity vs action compliance gap — core signal ratio finding
- `2026-02-25`: Cross-project injection failure — two-bug chain analysis with fix design
- `2026-03-02`: Emphasis language compliance — emphasis > neutral at high density
- `2026-03-11`: Failure mode taxonomy — 12→13 modes across 4 layers
- `2026-03-11`: Current state audit — token trajectory, hook coverage, pending items

### Synthesis Probes (created during model construction)
- `2026-03-12`: Evidence inventory — 67 claims classified by evidence quality
- `2026-03-12`: Contradiction analysis — 4 direct contradictions, 5 tensions, 3 recommendation conflicts
- `2026-03-12`: Gap analysis — boundary definition, 5 gaps identified, probe migration plan
- `2026-03-12`: Model construction — this model created from 3-worker parallel synthesis

---

## Related

- **Parent model:** `.kb/models/orchestrator-session-lifecycle/model.md` — session mechanics that the skill shapes
- **General theory:** `.kb/global/models/behavioral-grammars/model.md` — probabilistic constraint principles
- **Content types:** `.kb/models/skill-content-transfer/` — knowledge/stance/behavioral taxonomy with N=90 evidence
- **Defect classes:** `.kb/models/defect-class-taxonomy/model.md` — Class 3 (Stale Artifact Accumulation) applies to accretion cycle

## Auto-Linked Investigations

- .kb/investigations/archived/2025-12-23-inv-fix-skill-constraint-scoping-currently.md
