### Redundancy is Load-Bearing

Apparent redundancy in LLM skill documents serves a structural function. Each instance of a constraint fires at a different cognitive phase of the generation process.

**The test:** "Before removing a 'redundant' statement, can I identify which cognitive phase it covers that no other instance covers?"

**What this means:**

- LLM generation has distinct phases: situation recognition, approach planning, tool selection, response generation, self-monitoring
- A constraint stated in a table fires during planning. The same constraint in an anti-pattern list fires during self-monitoring. These are not redundant — they cover different phases.
- Compressing five instances to one clean statement removes coverage at four phases. The agent encounters the constraint once, at one decision point, and may have already committed to the wrong approach at an earlier phase.

**What this rejects:**

- "This is stated five times, that's poor writing" (it's multi-phase coverage)
- "DRY applies to skill documents" (DRY is for code maintainability, not attention allocation)
- "One clear statement is better than five scattered ones" (one clear statement fires at one phase)

**The failure mode:** The orchestrator skill stated "never read code" in five places: a tool action table, a bold rule, a checklist, a fast-path trigger, and an anti-pattern entry. A redesign compressed these to one statement. The agent read code on its first real task — the single statement fired too late, after the agent had already decided to "quickly check one file."

**Why distinct from Infrastructure Over Instruction:** IoI says enforce via tools, not documents. This principle says *when you must use documents*, redundancy is the mechanism that makes them work. IoI is about choosing the right layer. This is about designing effectively within the document layer.

**Boundary condition — dilution at scale:** Redundancy is load-bearing *per constraint* (multiple instances of the same rule across phases). But *across constraints*, dilution is load-bearing too. At ≤4 co-resident behavioral norms, each gets adequate attention. At 5+, dilution begins. At 10+, all constraints approach bare-parity (zero behavioral return). The orchestrator skill at 50+ constraints scored 22/56 vs bare 17/56 — the 5-point lift came entirely from knowledge items, not from any of the redundantly-stated behavioral constraints.

**The crossover:** Redundancy helps when you have few constraints stated multiple times. Redundancy hurts when you have many constraints, because each additional constraint dilutes all others — including the redundant ones. The ≤4 norm threshold is the crossover point.

**Evidence:** Feb 6 2026 skill compression (2,185 → 619 lines) immediately degraded delegation compliance. Mar 1 behavioral testing: compressed and original versions scored nearly identically against bare (39% vs 38%), but the original's redundant constraint statements showed measurably higher compliance in multi-turn interactive sessions where different phases activated different instances. Mar 4 2026: skill at 50+ constraints showed 5/7 scenarios at bare parity — dilution overwhelmed phase-coverage benefit at scale.

**Provenance:** Grammar Design Discipline synthesis (Mar 1 2026), corroborated by behavioral testing baseline and grammar recovery validation. Boundary condition from orchestrator-skill model synthesis (Mar 5 2026).
