# Probe: Layered Constraint Enforcement Design

**Model:** architectural-enforcement
**Date:** 2026-03-02
**Status:** Complete

---

## Question

The architectural enforcement model asserts that "Gates must be infrastructure-enforced, not instruction-reliant" (Invariant 1) and that "Instruction-Based Enforcement Under Pressure" is a systemic failure mode. Investigation orch-go-89at quantified the constraint dilution threshold: behavioral constraints regress to bare parity at 10 competing constraints, while the production orchestrator skill has 50+.

**Claim under test:** The model's multi-layer enforcement architecture (spawn gates, completion gates, coaching, declarative boundaries) can be extended to cover skill-level behavioral constraints, not just accretion/hotspot enforcement. Specifically: can the existing hook infrastructure (PreToolUse, PostToolUse, Stop, SessionStart) absorb the ~31 hard-enforceable behavioral constraints currently expressed only in prompt?

**Specific question:** What is the mapping from behavioral constraint types to enforcement mechanisms, and does the existing hook infrastructure have sufficient coverage, or are new mechanism types needed?

---

## What I Tested

**Audit approach:**
1. Read complete orchestrator skill (2,368 lines) and classified all 151 constraints (87 behavioral, 64 knowledge)
2. Inventoried all 15 existing hooks in ~/.orch/hooks/ and ~/.claude/hooks/
3. Mapped each behavioral constraint to its enforcement mechanism (existing, new hook, coaching nudge, or prompt-only)
4. Analyzed the Claude Code hook API capabilities (PreToolUse deny/allow/coaching, PostToolUse observe, Stop block/allow, SessionStart inject)

**Existing enforcement inventory:**

| Hook | Type | Behavioral Constraint | Mechanism |
|------|------|----------------------|-----------|
| `--disallowedTools` (spawn-time) | Hard gate | No Task/Edit/Write/NotebookEdit | Tool removal |
| `gate-bd-close.py` | Hard gate | No `bd close` on agent-worked issues | Bash command intercept |
| `gate-orchestrator-task-tool.py` | Hard gate | No Task tool for orchestrators | PreToolUse deny |
| `gate-orchestrator-code-access.py` | Coaching | Code file reads → delegation consideration | PreToolUse allow+context |
| `enforce-phase-complete.py` | Hard gate | Must report Phase: Complete before exit | Stop block |
| `orchestrator-session-kn-gate.py` | Hard gate | Must capture knowledge before session end | SessionEnd gate |
| `pre-commit-knowledge-gate.py` | Hard gate | Must capture knowledge before commits | PreToolUse gate |

**Constraint classification results:**

| Category | Count | Example | Enforcement |
|----------|-------|---------|-------------|
| Hard behavioral (tool-level) | 31 | "Don't use Edit tool" | Infrastructure: hooks + --disallowedTools |
| Soft behavioral (pattern-level) | 28 | "Coaching nudge for code reads" | Infrastructure: coaching hooks |
| Judgment behavioral (reasoning) | 28 | "Don't present >1 option without filtering" | Prompt only (budgeted ≤4/section) |
| Knowledge (additive) | 64 | "Daemon auto-spawns triage:ready issues" | Prompt (survives dilution at 10+) |

---

## What I Observed

1. **The existing hook infrastructure covers ~7 of 31 hard-enforceable behavioral constraints.** The remaining 24 need new hooks or extensions to existing hooks. Most gaps are in bash command filtering (filesystem writes, git remote operations) and spawn validation (context completeness, orientation frame).

2. **The hook API is sufficient — no new mechanism types needed.** PreToolUse (deny/allow/coaching) + Stop (block/allow) + SessionStart (inject) covers all enforcement needs. The key insight: deny for hard gates, allow+context for coaching nudges, and block for exit gates are the three enforcement patterns, and they map cleanly to constraint types.

3. **Coaching nudges are the correct enforcement for ~28 constraints.** The gate-orchestrator-code-access.py history shows that blocking (deny) caused fabrication and wrong diagnoses. Coaching (allow + contextual message) is the right calibration for constraints where blocking would cause worse behavior. This pattern extends to investigation drift detection, spawn context validation, and session boundary checks.

4. **~28 judgment constraints CANNOT move to infrastructure.** These are pure reasoning constraints (option filtering, question framing, response targeting, autonomy decisions). They must stay in prompt. Applying the dilution budget (≤4 per section, 2 sections), only ~8 can survive in prompt. The rest must be either: (a) reformulated as knowledge constraints, or (b) accepted as best-effort.

5. **~20 behavioral constraints can be reformulated as knowledge.** Example: "Don't ask 'want me to complete them?'" (behavioral) → "Orchestrators auto-complete agents at Phase: Complete" (knowledge about the norm). Knowledge constraints survive dilution at 10+ competitors, so reformulation effectively immunizes them.

---

## Model Impact

- [x] **Confirms** Invariant 1: "Gates must be infrastructure-enforced, not instruction-reliant." The dilution evidence quantifies WHY — behavioral constraints in prompt fail at 10+ competitors, and the orchestrator has 87 behavioral constraints. Infrastructure enforcement is not a preference but a necessity.

- [x] **Extends** the model with: A constraint taxonomy that maps enforcement mechanisms to constraint types:
  - **Hard behavioral** → Infrastructure gates (deny)
  - **Soft behavioral** → Infrastructure coaching (allow + context)
  - **Judgment behavioral** → Prompt (budgeted ≤4/section) OR reformulated as knowledge
  - **Knowledge** → Prompt (survives dilution, no budget limit needed until ~50+)

- [x] **Extends** the model with: The "behavioral → knowledge reformulation" technique. Some constraints phrased as prohibitions ("don't do X") can be reformulated as norms ("the pattern is Y"), moving them from the behavioral bucket (4-constraint budget) to the knowledge bucket (10+ constraint tolerance).

- [x] **Confirms** "Why This Fails" §3: "Instruction-Based Enforcement Under Pressure." The 87 behavioral constraints in the orchestrator skill are a concrete instance of this failure mode. The dilution evidence (bare parity at 10) means ~83 of them (87 - 4 budgeted) are non-functional in prompt.

- [x] **Extends** the model with: Implementation priority ordering for new hooks based on enforcement impact and existing infrastructure gaps. The model's four layers (spawn, completion, coaching, declarative) map to concrete hook implementations with a phased rollout.

---

## Notes

- Full design document: .kb/investigations/2026-03-02-design-layered-constraint-enforcement-architecture.md
- Evidence base: .kb/investigations/2026-03-01-inv-test-constraint-dilution-threshold.md (orch-go-89at)
- Prior decision: .kb/decisions/2026-02-26-two-layer-action-compliance.md (established the two-layer pattern)
- The 151-constraint audit was performed by reading the complete orchestrator skill (2,368 lines)
