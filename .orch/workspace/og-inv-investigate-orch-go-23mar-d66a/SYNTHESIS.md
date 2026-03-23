# Session Synthesis

**Agent:** og-inv-investigate-orch-go-23mar-d66a
**Issue:** orch-go-ad3vx
**Duration:** 2026-03-23
**Outcome:** success

---

## Plain-Language Summary

Orch-go's 329-trial coordination evidence reveals that the industry's "multi-agent doesn't work" consensus is both right and wrong. It's right that agent-to-agent communication fails (0-30% success across 120 trials). It's wrong that multi-agent is universally unviable — structural coordination (assigning agents to non-overlapping regions instead of having them talk) achieves 100% success at N=2. The investigation identifies three product archetypes that use this insight: constraint-first parallelism (proven, like autoresearch), region-routed concurrent work (proven at small scale), and contract-mediated composition (highest value but requires solving the composition-verification gap — the problem where individually correct components produce a broken whole). The critical finding is that the barrier to high-value multi-agent products isn't coordination anymore — it's composition verification.

## TLDR

Orch-go's coordination evidence supports three multi-agent product patterns beyond single-transform, but the highest-value pattern (contract-mediated composition) requires solving the composition-verification gap — where individually correct agent outputs fail to compose into a correct whole. The industry's failure with multi-agent is specifically a communication-based-coordination failure, not an inherent multi-agent limitation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-23-inv-investigate-coordination-product-applicability.md` — Full investigation with 5 findings, 4 problem categories, 3 product archetypes, evidence gaps analysis
- `.orch/workspace/og-inv-investigate-orch-go-23mar-d66a/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-investigate-orch-go-23mar-d66a/VERIFICATION_SPEC.yaml` — Verification specification

### Files Modified
- None (pure investigation, no code changes)

---

## Evidence (What Was Observed)

- Coordination model covers 329 trials across 8 experiments with explicit evidence quality annotations on every claim
- Reddit poster's failed products used exactly the frameworks the model predicts fail (CrewAI, LangGraph — gate-based coordination)
- Reddit poster's successful products are all Category A (parallelizable-independent) — no coordination needed
- Modification tasks self-coordinate (40/40 SUCCESS) — the coordination problem is specific to additive tasks with gravitational convergence
- Compositional correctness gap appears identically in 3 domains: SE (daemon.go +892 lines), sheet metal DFM (SCS AI Part Builder 0% recall), LED routing (valid geometry, disconnected channels)
- Scaling from N=2 to N=4,6 degrades pairwise success from 100% to 67-70% — products must account for this ceiling
- Automated attractor discovery works (7/7 SUCCESS from 2 observed collisions) — structural constraints can be self-discovered

### Tests Run
```bash
# No code tests — this is a pure analysis investigation
# Evidence reviewed: coordination model, 10+ probe files, 5 thread files, 2 model files
```

---

## Architectural Choices

No architectural choices — task was analysis/investigation within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-23-inv-investigate-coordination-product-applicability.md` — Product applicability analysis of coordination findings

### Constraints Discovered
- Composition verification is domain-specific and cannot be solved with a general-purpose framework
- Structural placement degrades at N>2 with limited insertion points — products must either ensure sufficient insertion points or limit concurrency
- Cross-domain transfer of coordination principles is plausible but entirely unvalidated

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification criteria. Key outcomes:
- Investigation file complete with D.E.K.N., 5 findings, prior work table, structured uncertainty
- Three product archetypes identified with evidence quality ratings
- Composition-verification gap identified as critical barrier
- Evidence gaps explicitly catalogued (domain, model, scale, task type, integration mechanism)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-ad3vx`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Can the composition-verification gap be closed with an automated composition gate in the completion pipeline? The open-loop thread suggests "file-overlap detection between concurrent agents" as a cheap first version.
- Would the "coordination tax inversion" insight (best coordination = eliminated coordination) resonate as a blog post thesis? It's counter-narrative to the agent-swarm hype cycle.
- Does the Anthropic production pattern (lead agent defines work regions) actually implement Archetype 3, or is it closer to Archetype 2? The model classifies it as "attractor-dominant" but doesn't detail the composition layer.

**What remains unclear:**
- Whether any product team has built a successful multi-agent product using structural coordination (as opposed to communication) — we have orch-go production evidence but no external product evidence
- Whether composition verification can be made domain-agnostic at all, or if it's inherently domain-specific (integration testing for code, connectivity for physical design, coherence for content)

---

## Friction

No friction — smooth session. Read-heavy investigation with no tool or process blockers.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-orch-go-23mar-d66a/`
**Investigation:** `.kb/investigations/2026-03-23-inv-investigate-coordination-product-applicability.md`
**Beads:** `bd show orch-go-ad3vx`
