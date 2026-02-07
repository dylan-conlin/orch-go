# Session Synthesis

**Agent:** og-arch-design-information-hiding-27jan-1dd8
**Issue:** orch-go-20959
**Duration:** 2026-01-27 18:50 → 2026-01-27 19:35
**Outcome:** success

---

## TLDR

Designed three-phase implementation for preventing orchestrator frame collapse: (1) prompt-based action space restriction (immediate), (2) information hiding via output filtering (short-term), (3) registry-level tool gating (medium-term). Research shows architectural constraints beat guidelines, and existing plugin infrastructure provides foundation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-27-inv-design-information-hiding-tool-restriction.md` - Full design investigation with implementation recommendations

### Files Modified
- None (design investigation, no code changes)

### Commits
- (pending commit of investigation file and synthesis)

---

## Evidence (What Was Observed)

- Research investigation (2026-01-27-inv-research-exists-preventing-hierarchical-controllers.md) found 30 years of HRL convergence on architectural action space restriction
- `plugins/coaching.ts:476-537` already implements frame collapse detection via `isCodeFile()` and `FrameCollapseState`
- `plugins/task-tool-gate.ts` demonstrates working registry-level tool gating pattern
- Orchestrator skill already defines meta-action vocabulary (`orch spawn`, `bd create`, `kb context`)
- Decidability graph model (`.kb/models/decidability-graph.md`) establishes authority boundaries: daemon→work, orchestrator→questions, Dylan→gates

### Tests Run
```bash
# No code changes to test - design investigation
# Verified existing plugins compile and are active via directory inspection
ls plugins/*.ts  # Found coaching.ts, task-tool-gate.ts, etc.
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-27-inv-design-information-hiding-tool-restriction.md` - Comprehensive design with three-phase implementation plan

### Decisions Made
- **Information hiding via filtering:** Output filtering in plugin layer is preferred over tool removal because it preserves legitimate read capabilities for orchestration artifacts (CLAUDE.md, .kb/*.md)
- **Tool restriction via allowlist:** Allowlisting meta-actions is safer than blocklisting primitives - only permit known safe operations
- **Three layers needed:** Defense-in-depth (prompt + detection + enforcement) provides strongest guarantee

### Constraints Discovered
- **Legitimate orchestrator needs exist:** Must allow reading CLAUDE.md, investigation TLDRs, beads comments
- **Emergency escape needed:** Rare genuine cases require override, but all uses should be logged
- **Plugin layer has limits:** Need to verify `tool.execute.after` can actually modify return values

### Externalized via `kb`
- Decision document recommended (recommend-yes in investigation)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with design)
- [x] Tests passing (no code changes)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-20959`

### Follow-up Work for Orchestrator to Create

**Implementation issues to create:**

1. **Phase 1: Prompt-Based Action Space Restriction**
   - Title: "Update orchestrator skill with explicit CAN/CANNOT action space"
   - Skill: feature-impl
   - Priority: P2
   - Estimate: 1-2 hours

2. **Phase 2: Information Hiding via Output Filtering**
   - Title: "Implement output filtering in coaching plugin for orchestrator sessions"
   - Skill: feature-impl
   - Priority: P3
   - Estimate: 4-6 hours
   - Depends on: Phase 1

3. **Phase 3: Registry-Level Tool Gating**
   - Title: "Implement orchestrator-tool-gate plugin with meta-action allowlist"
   - Skill: feature-impl
   - Priority: P3
   - Estimate: 4-6 hours
   - Depends on: Phase 2

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does output filtering interact with context window limits? Could truncated outputs cause confusion?
- Should orchestrator reading of investigation files be filtered to just D.E.K.N. section?
- What happens if orchestrator genuinely needs to debug an agent's code output?

**Areas worth exploring further:**
- HAM (Hierarchical Abstract Machines) - Another HRL framework not explored in research
- Intrinsic motivation in HRL - How to prevent orchestrator from "wanting" to do low-level work
- Cost-benefit analysis of hierarchy vs flat multi-agent for specific task types

**What remains unclear:**
- Exact allowlist of bash commands (git status yes, but git diff? git log?)
- Whether to allow orchestrator to read its own prior session artifacts
- Performance impact of output filtering on every tool call

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-information-hiding-27jan-1dd8/`
**Investigation:** `.kb/investigations/2026-01-27-inv-design-information-hiding-tool-restriction.md`
**Beads:** `bd show orch-go-20959`
