### Strategic-First Orchestration

In areas with patterns (hotspots, persistent failures, infrastructure), strategic approach is the default. Tactical requires justification.

**The test:** "Is this a hotspot (5+ bugs), persistent failure (2+ abandons), or infrastructure work? If yes, architect required."

**What this means:**

- HOTSPOT areas (5+ bugs) → refuse tactical debugging, require architect first
- Persistent failure (2+ abandons) → auto-spawn architect to investigate pattern
- Infrastructure work (orchestration system itself) → auto-apply escape hatch (--backend claude)
- In patterned areas, tactical path requires explicit justification

**The insight:**

We consistently face choices between tactical (fix symptom) and strategic (understand pattern). The strategic path almost always:
- Costs less total time (1 architect vs 3+ debugging attempts)
- Fixes the real problem (design flaw vs surface bug)
- Prevents future bugs (coherent system vs patches)

Yet we keep defaulting to tactical. This principle makes strategic the default where patterns exist.

**What this rejects:**

- "Consider architect" as optional suggestion (make it required)
- Warnings the user can ignore (make them blocking gates)
- Asking permission to apply principles (orchestrator applies them)
- Treating tactical and strategic as equivalent options (strategic is better in patterned areas)

**Current behavior vs strategic-first behavior:**

| Situation | Current (Suggestion) | Strategic-First (Requirement) |
|-----------|---------------------|-------------------------------|
| HOTSPOT warning (5+ bugs) | "Consider architect" → User decides | Refuse to spawn debugging → Require architect first |
| Persistent failure (2+ abandons) | "Try reliability-testing" → Suggestion | Auto-spawn architect → Investigate pattern |
| Infrastructure work | Warning about circular dependency | Auto-apply --backend claude |
