## Summary (D.E.K.N.)

**Delta:** [To be filled after testing]

**Evidence:** [To be filled after testing]

**Knowledge:** [To be filled after testing]

**Next:** [To be filled after testing]

**Authority:** architectural - Finding affects skill design principles (constraint budget) across all skills

---

# Investigation: Constraint Dilution Threshold for 3-Form Structural Diversity

**Question:** Does the 3-form structural diversity pattern (table + checklist + examples) survive when multiple constraints compete in the same document? At what constraint count does compliance drop below ceiling (8/8)?

**Defect-Class:** configuration-drift

**Started:** 2026-03-01
**Updated:** 2026-03-01
**Owner:** og-inv-test-constraint-dilution-01mar-d0c9
**Phase:** Investigating
**Next Step:** Run skillc test for all variants
**Status:** In Progress

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-01-inv-test-hypothesis-redundancy-saturation-point.md | extends | yes — 3-form = [8,8,8] in isolation | pending — does it hold under competition? |
| .kb/investigations/2026-03-01-inv-test-hypothesis-constraint-violation-rate.md | extends | yes — full skill 0% delegation at high complexity | pending — is there a middle ground? |
| .kb/investigations/2026-03-01-investigation-orchestrator-skill-behavioral-testing-baseline.md | extends | yes — both variants at bare parity for 5/7 scenarios | pending — is dilution the explanation? |

---

## Test Design

**Approach:** Increasing constraint density variants, all using 3-form structural diversity.

| Variant | Constraint Count | Total Expressions | Estimated Tokens | Contains |
|---------|-----------------|-------------------|------------------|----------|
| Bare | 0 | 0 | 0 | Nothing |
| 1C-D | 1 (delegation) | 3 | ~200 | Delegation only |
| 1C-I | 1 (intent) | 3 | ~200 | Intent only |
| 2C | 2 | 6 | ~400 | Delegation + Intent |
| 5C | 5 | 15 | ~1000 | Both + 3 fillers |
| 10C | 10 | 30 | ~2000 | Both + 8 fillers |

**Measurement probes:** delegation-probe and intent-clarification-probe (same as aj58)

**Model:** sonnet (matching aj58 for comparability)

**Runs:** 3 per variant (variance measurement)

**Filler constraints** (realistic orchestrator behaviors, irrelevant to measurement probes):
1. Anti-sycophancy: Don't hedge or over-apologize
2. Phase reporting: Report phase transitions via bd comment
3. No bd close: Workers must never run bd close
4. Architect routing: Route hotspot work to architect
5. Session close protocol: Follow exact commit order
6. Beads tracking: Track progress via beads
7. Context loading: Load SPAWN_CONTEXT before acting
8. Tool restriction: Prefer dedicated tools over shell commands

---

## Findings

### Finding 1: Starting approach

**Evidence:** Setting up test infrastructure with 4 constraint density variants (1C, 2C, 5C, 10C). Using same scenarios and model as aj58 investigation for direct comparability.

**Source:** Test artifacts in workspace

**Significance:** Bridges the gap between aj58's isolated 3-form ceiling (8/8) and the full-skill failure (0% delegation). If dilution is gradual, prompt engineering can manage constraint budgets. If it's a threshold, we need to find the cliff.

---

## Investigation History

**2026-03-01 21:40:** Investigation started
- Initial question: Does 3-form structural diversity survive constraint competition?
- Context: aj58 found 3-form achieves ceiling in isolation, but full skill (50+ constraints) produces 0% delegation. Need to find the dilution threshold.
