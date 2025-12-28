<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created beads epic (orch-go-6uli) with 5 child issues representing a 6-month phased consolidation of the orch ecosystem from 8 repos to 4 functional units.

**Evidence:** Epic structure verified via `bd show orch-go-6uli` showing 5 children with proper dependency chain; references prior investigation 2025-12-24-inv-full-ecosystem-audit-scope-simplify.md for analysis.

**Knowledge:** Epics with parallel component work require a final integration child issue (constraint from prior knowledge); Phase 5 added for integration verification and documentation.

**Next:** Orchestrator should review epic structure, prioritize phases, and begin spawning work for Phase 1 (kb absorbs kn) when ready.

---

# Investigation: Epic Orch Ecosystem Consolidation Month

**Question:** How should we structure the beads epic for the 6-month orch ecosystem consolidation?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None - epic created and ready for orchestrator review
**Status:** Complete

**Extracted-From:** .kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md

---

## Findings

### Finding 1: Epic Created with 5 Phases

**Evidence:** 
```
orch-go-6uli: Epic: Orch Ecosystem Consolidation (6-month)
Children (5):
  ↳ orch-go-6uli.1: Phase 1: kb absorbs kn (Month 1)
  ↳ orch-go-6uli.2: Phase 2: orch-go Python parity (Month 2-3)
  ↳ orch-go-6uli.3: Phase 3: beads abstraction layer (Month 4)
  ↳ orch-go-6uli.4: Phase 4: Final cleanup (Month 5-6)
  ↳ orch-go-6uli.5: Phase 5: Integration verification and documentation
```

**Source:** `bd show orch-go-6uli`

**Significance:** All 4 requested phases plus mandatory integration phase created. Dependencies correctly chained.

---

### Finding 2: Dependency Chain Correctly Configured

**Evidence:** Each phase depends on the previous:
- Phase 2 depends on Phase 1 (kb/kn merge before Python parity)
- Phase 3 depends on Phase 2 (beads abstraction after orch-go parity)
- Phase 4 depends on Phase 3 (cleanup after abstraction)
- Phase 5 depends on Phase 4 (integration after all phases)

**Source:** `--deps` flags used in bd create commands

**Significance:** Ensures sequential execution and prevents starting later phases before dependencies complete.

---

### Finding 3: Integration Issue Added Per Constraint

**Evidence:** Constraint from prior knowledge: "Epics with parallel component work must include a final integration child issue"

Phase 5 (orch-go-6uli.5) includes:
- E2E workflow testing with unified tooling
- Skill reference validation
- Documentation updates
- Migration guide creation

**Source:** SPAWN_CONTEXT.md prior knowledge section

**Significance:** Prevents the "swarm agents build components but nothing wires them together" failure mode.

---

## Synthesis

**Key Insights:**

1. **Sequential dependency is correct** - Each phase builds on prior work (kb/kn merge enables cleaner Python parity, which enables beads abstraction, etc.)

2. **Integration phase is critical** - Without explicit integration verification, consolidated tools might not work together in real workflows

3. **All phases have triage:review label** - Orchestrator can review and relabel to triage:ready when prioritizing

**Answer to Investigation Question:**

The epic is structured with 5 sequentially-dependent child issues covering: kb/kn merge, Python parity, beads abstraction, cleanup, and integration verification. This matches the 6-month timeline from the source investigation while adding the mandatory integration issue per documented constraints.

---

## Structured Uncertainty

**What's tested:**

- ✅ Epic created successfully (verified: `bd show orch-go-6uli` shows structure)
- ✅ Dependencies set correctly (verified: each phase has `--deps` to previous)
- ✅ All phases have triage:review label (verified: `--label` flag used)

**What's untested:**

- ⚠️ Timeline estimates are from prior investigation (not re-validated)
- ⚠️ Feature parity list for Phase 2 may have gaps (from Python orch-cli --help comparison)
- ⚠️ agentlog decision criteria not fully specified (Phase 4)

**What would change this:**

- If Python orch-cli has features not listed, Phase 2 scope would expand
- If beads API changes significantly, Phase 3 scope would need revision
- If skillc integration becomes urgent, Phase 4 scope would expand

---

## Implementation Recommendations

### Recommended Approach: Sequential Phase Execution

**Why this approach:**
- Each phase has clear deliverables
- Dependencies prevent out-of-order work
- Orchestrator can review after each phase

**Trade-offs accepted:**
- Slower than parallel work
- 6-month timeline may stretch if blockers found

**Implementation sequence:**
1. Phase 1 (Month 1): kb absorbs kn - lowest risk, immediate value
2. Phase 2 (Month 2-3): Python parity - blocks deprecation
3. Phase 3 (Month 4): beads abstraction - testability improvement
4. Phase 4 (Month 5-6): Final cleanup - depends on all prior work
5. Phase 5: Integration verification - validates entire consolidation

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md` - Source investigation with consolidation analysis

**Commands Run:**
```bash
# Create epic
bd create "Epic: Orch Ecosystem Consolidation (6-month)" --type epic ...

# Create child issues with dependencies
bd create "Phase 1: kb absorbs kn (Month 1)" --parent orch-go-6uli ...
bd create "Phase 2: orch-go Python parity (Month 2-3)" --deps orch-go-6uli.1 ...
# ... (5 total phases)

# Verify structure
bd show orch-go-6uli
bd epic status orch-go-6uli
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md` - Source analysis
- **Epic:** `orch-go-6uli` - The created beads epic
- **Workspace:** `.orch/workspace/og-feat-epic-orch-ecosystem-27dec/` - Spawn workspace

---

## Investigation History

**2025-12-27 18:05:** Investigation started
- Initial question: How should we structure the beads epic for ecosystem consolidation?
- Context: Spawned from orchestrator to create actionable work items from prior investigation

**2025-12-27 18:08:** Epic and all child issues created
- Epic orch-go-6uli created with 5 phases
- All dependencies and labels configured

**2025-12-27 18:10:** Investigation completed
- Status: Complete
- Key outcome: Beads epic ready for orchestrator review and prioritization
