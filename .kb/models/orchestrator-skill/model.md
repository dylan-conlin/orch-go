# Model: Orchestrator Skill

**Created:** 2026-03-09
**Status:** Active
**Source:** Synthesized from 5 investigation(s)

## What This Is

[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]

---

## Core Claims (Testable)

### Claim 1: [Concise claim statement]

[Explanation of the claim. What would you observe if it's true? What would falsify it?]

**Test:** [How to test this claim]

**Status:** Hypothesis

### Claim 2: [Concise claim statement]

[Explanation of the claim.]

**Test:** [How to test this claim]

**Status:** Hypothesis

---

## Implications

[What follows from these claims? How should this model change behavior, design, or decision-making?]

---

## Boundaries

**What this model covers:**
- [Scope item 1]

**What this model does NOT cover:**
- [Exclusion 1]

---

## Evidence

| Date | Source | Finding |
|------|--------|---------|
| 2026-03-09 | Model creation | Initial synthesis from source investigations |

---

## Open Questions

- [Question that further investigation could answer]
- [Question about model boundaries or edge cases]

## Source Investigations

### 2026-01-18-inv-update-orchestrator-skill-add-frustration.md

**Delta:** Added Frustration Trigger Protocol to orchestrator skill - a mode shift triggered when Dylan voices frustration.
**Evidence:** Skill updated at `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template` and deployed to `~/.claude/skills/meta/orchestrator/SKILL.md`.
**Knowledge:** This is a gate Dylan controls (not a reminder) - voices frustration to trigger mode shift from tactical to probing.
**Next:** None - implementation complete.

---

### 2026-02-24-design-orchestrator-skill-behavioral-compliance.md

**Delta:** Orchestrator agents comply with identity declarations but not action constraints because identity is additive (no competing instructions) while actions are subtractive (directly conflicts with Claude Code system prompt that promotes Task tool and bd close).
**Evidence:** Signal ratio analysis shows 17:1 competing instruction disadvantage — system prompt has ~500 words promoting Task tool, skill has ~30 words constraining it. Action constraints appear at 88% depth in 640-line skill. System prompt has structural priority in Claude's instruction hierarchy (system > user content).
**Knowledge:** The problem is not skill content quality but instruction hierarchy position. Prompt-level action restrictions operate as guidelines, not enforcement. Recent research (AgentSpec ICSE 2026, PCAS) confirms: prompts describe desired behavior, infrastructure enforces it. A two-layer fix is needed: restructure skill for salience AND add tool-layer enforcement.
**Next:** Implement the recommended two-layer approach: (1) restructure skill with action-identity fusion at top, (2) add Claude Code hook or plugin that intercepts prohibited tool usage for orchestrator sessions.

---

### 2026-03-05-inv-design-orchestrator-skill-update-incorporating.md

**Delta:** 10 infrastructure changes map to 13 specific edits in SKILL.md.template; 4 changes require no skill edits (already present or worker-only).
**Evidence:** Audited all 451 lines of SKILL.md.template against CLI help output, daemon source, spawn source, and review_tier.go. Verified `orch frontier` is removed, `--no-track` creates lightweight issues, `--dry-run` flag exists, review tiers are live with 4 levels.
**Knowledge:** 6 changes are high-confidence factual updates (stale refs, new commands). 3 are behavioral additions (daemon, review tiers, plans). 1 is already complete (synthesis-as-comprehension). Net line change: +3 (within "near zero" target).
**Next:** Implement edits to SKILL.md.template and reference/tools-and-commands.md, rebuild with `skillc build`.
