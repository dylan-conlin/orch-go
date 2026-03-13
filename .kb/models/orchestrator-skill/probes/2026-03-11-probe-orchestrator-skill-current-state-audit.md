# Probe: Orchestrator Skill Current State Audit

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-11
**Status:** Complete

---

## Question

Does the current orchestrator skill match the design decisions from the 6 investigations (Jan 18 - Mar 5, 2026)? What's implemented, what drifted, what's pending?

---

## What I Tested

Compared investigation recommendations against actual deployed artifacts:

```bash
# Line counts
wc -l ~/.claude/skills/meta/orchestrator/SKILL.md  # 512 lines (deployed)
wc -l skills/src/meta/orchestrator/.skillc/SKILL.md.template  # 486 lines (source)
wc -l .kb/investigations/evidence/2026-02-28-orchestrator-intent-spiral/orchestrator-skill-snapshot.md  # 493 lines (Feb 28 snapshot)

# Hook registration
cat ~/.claude/settings.json | python3 -c "import json,sys; print(json.dumps(json.load(sys.stdin).get('hooks',{}), indent=2))"

# Hook implementations
ls -la ~/.orch/hooks/

# Token trajectory from stats.json
# Dec 2025: 12,390 → Jan 29 peak: 23,908 → Mar 1 peak: 27,200 → Mar 4 v4: 4,830 → Mar 11 current: 5,995

# Deployed skill structure verification
grep '^## ' ~/.claude/skills/meta/orchestrator/SKILL.md  # 16 sections
grep -c 'NEVER' ~/.claude/skills/meta/orchestrator/SKILL.md  # Checking prohibition language removal
```

Verified each investigation recommendation against current skill content, hook registrations, and skillc infrastructure.

---

## What I Observed

### Token Trajectory (from stats.json)
| Date | Tokens | Event |
|------|--------|-------|
| Dec 22, 2025 | 12,390 | Initial |
| Jan 29, 2026 | 23,908 | Pre-restructure peak |
| Feb 6 | 6,571 | Major trim |
| Feb 28 | 6,376 | Snapshot (investigation 6) |
| Mar 1 | 27,200 | 2,368-line monstrosity (accretion peak) |
| Mar 4 | 4,830 | v4 simplification (investigation 4) |
| Mar 11 | 5,995 | Current (grew +1,165 from inv 5 additions) |

### Recommendation Implementation Rates
- **Investigation 2 (behavioral compliance):** 6/7 recommendations implemented
- **Investigation 3 (testing infrastructure):** 4/5 tools built, testing blocked
- **Investigation 4 (simplification):** 5/6 criteria met, behavioral gate pending
- **Investigation 5 (72-commit delta):** 7/7 changesets applied

### Hook Coverage
6 of 7 hooks from investigation 4 are registered and working. The 7th (code-access gate) is partially covered by the investigation-drift nudge on Read matcher.

### Structural Drift from Feb 28 Snapshot
Feb 28 snapshot used 7 numbered sections (Identity & Action Space → Hard Constraints & Reference). Current uses 16 named sections organized by domain (Role → Workspace & Tier Architecture). Complete structural overhaul.

Content type shift: Feb 28 had constraint-heavy text (8-checkbox Pre-Response Checks, "Inviolable Constraints" section, Tool Action Space "You CANNOT" table). Current is knowledge-transfer dominant (routing tables, vocabulary definitions, 4-norm behavioral section with explicit note that hooks handle enforcement).

### Key Pending Items
1. **Behavioral validation of v4:** `skillc test` bare-parity regression blocked — can't run from spawned agent (CLAUDECODE env var)
2. **Graduated hook response:** Hooks are binary (nudge/block), not graduated as inv 2 recommended
3. **A/B compliance measurement:** No before/after compliance data exists

---

## Model Impact

- [x] **Confirms** invariant: The orchestrator skill lifecycle follows a pattern of accretion → crisis → simplification → gradual regrowth. Token trajectory shows this clearly (12K → 27K → 5K → 6K).
- [x] **Confirms** invariant: Infrastructure enforcement (hooks) replaces prompt-level constraints. 6 hooks now enforce what 350+ lines of prohibition text used to attempt.
- [x] **Extends** model with: The behavioral validation gate from investigation 4 remains open — v4 was deployed without passing the `skillc test` bare-parity check. This means the 82% token reduction is validated structurally but not behaviorally. The skill could theoretically be worse than bare Claude on some scenarios.
- [x] **Extends** model with: Post-simplification regrowth is already visible — 5,995 tokens (Mar 11) vs 4,830 (Mar 4), a 24% increase in 7 days from investigation 5's factual additions. This matches the accretion pattern the model predicts.

---

## Notes

Full audit details (recommendation tracking, hook coverage matrix, drift analysis, pending items) written to SYNTHESIS.md in workspace: `.orch/workspace/og-inv-task-orchestrator-skill-11mar-8e8d/SYNTHESIS.md`
