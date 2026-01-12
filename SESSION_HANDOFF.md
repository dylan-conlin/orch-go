# Session Handoff - Strategic-First Orchestration

**Date:** 2026-01-11
**From Session:** 30h+ orchestrator session
**To Session:** Fresh focus on strategic orchestration

---

## Key Insight Discovered

**Strategic-First Orchestration Principle:**

We consistently face choices between tactical (fix symptom) and strategic (understand pattern) approaches. The strategic path almost always:
- Costs less total time (1 architect vs. 3+ debugging attempts)
- Fixes the real problem (design flaw vs. surface bug)
- Prevents future bugs (coherent system vs. patches)

**Yet we keep defaulting to tactical.**

---

## The Pattern

**Current behavior:**
- HOTSPOT warning (5+ bugs) → "Consider architect" → User decides → Often ignores
- Persistent failure (2+ abandons) → "Try reliability-testing" → Suggestion, not gate
- Infrastructure work → Warning about circular dependency → User proceeds anyway

**Strategic-first behavior:**
- HOTSPOT warning → **Refuse to spawn debugging** → Require architect first
- Persistent failure → **Auto-spawn architect** → Investigate pattern
- Infrastructure work → **Auto-apply --backend claude** → Infrastructure detection

**Principle:** In areas with patterns (hotspots, persistent failures, infrastructure), strategic approach is DEFAULT. Tactical requires justification.

---

## Work Done This Session

### Issues Created
1. **orch-go-vwjle** - Add stuck-agent detection and monitoring (P2)
2. **orch-go-6a0p1** - Bug: orch abandon doesn't remove agent from registry (P1)
3. **orch-go-ao6nf** - Add infrastructure work detection to triage/spawn (P2)

### Knowledge Captured
- **kb-4fddb6** - Constraint: Never spawn OpenCode infrastructure work without --backend claude --tmux

### Active Work
- **orch-go-rcah9** - Architect agent running in tmux (og-arch-review-design-coaching-11jan-f74a)
  - Investigating why coaching plugin has 8 bugs (hotspot)
  - Should produce decision document on architectural approach

### Technical Fixes
- Restarted OpenCode server (was down, port 4096)
- Fixed duplicate agent display bug (abandoned agents staying in registry)
- Identified stuck agent pattern (agents hang after 2min with no error)

---

## Next Session Focus

**Primary Goal:** Design and implement Strategic-First Orchestration

### Immediate Actions

1. **Let architect complete** (orch-go-rcah9)
   - Review its findings on coaching plugin design
   - Use as case study for strategic vs tactical

2. **Create principle document**
   - `.kb/principles.md` entry for "Strategic-First Orchestration"
   - Criteria for when strategic is required
   - How to override (and why you shouldn't)

3. **Make HOTSPOT a gate**
   - Change from warning to blocking error
   - Require `--force` to override (with justification)
   - Update spawn logic to refuse tactical approaches in hotspots

4. **Update orchestrator skill**
   - Change guidance from "consider architect" to "architect required"
   - Add infrastructure detection patterns
   - Elevate principles over preferences

### Design Questions to Answer

- **How do we detect infrastructure work?** (path patterns, keywords, manual flag)
- **What makes a good override justification?** (one-off, truly novel, post-strategic)
- **Should strategic-first apply to daemon auto-spawns?** (probably yes)
- **How do we measure success?** (fewer abandons in hotspots, faster time-to-resolution)

### Success Criteria

- HOTSPOT areas require architect first (enforced, not suggested)
- Infrastructure work auto-applies escape hatch
- Orchestrator applies principles without asking permission
- Tactical path requires explicit justification

---

## System State

**OpenCode server:** Running (port 4096)
**Dashboard:** Running (port 3348)
**Daemon:** Running (51 ready issues)

**Active agents:** 30 (6 running, 24 idle with 24 AT-RISK)
**Note:** Many idle agents need cleanup (orch clean)

**Branch:** master (pushed to origin/master)
**Uncommitted:** None (clean working tree)

---

## The Meta-Pattern

This insight connects to foundational principles:

- **Coherence Over Patches** - After 3rd fix in same area, examine design
- **Pressure Over Compensation** - Don't compensate for system failures, create pressure to fix
- **Premise Before Solution** - Understand the problem before implementing fixes

Strategic-first orchestration is the operational manifestation of these principles.

**The deeper realization:** Orchestrator's job is to **apply principles, not ask permission to apply them.**

---

## Context for Fresh Claude

You're starting a NEW session focused on implementing strategic-first orchestration. Previous session discovered the pattern but didn't implement it.

**What exists:**
- Principle insight (documented here)
- Concrete examples (coaching plugin hotspot, infrastructure work)
- Several related beads issues
- One kb constraint (infrastructure work)

**What doesn't exist yet:**
- Principle document in .kb/principles.md
- Gates in spawn logic (still warnings)
- Updated orchestrator skill guidance
- Infrastructure detection patterns

**Your job:** Design and implement the strategic-first model. This is foundational work that changes how the entire orchestration system operates.
