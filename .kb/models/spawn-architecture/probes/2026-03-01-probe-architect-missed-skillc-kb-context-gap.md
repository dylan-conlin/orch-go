# Probe: Architect Missed skillc as Correct Home for Skill Tools — KB Context Query Gap

**Model:** spawn-architecture
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The architect design (orch-go-dlw9) placed skill lint/test/compare tools in orch-go instead of skillc. Was this caused by a gap in the spawn context — specifically, did the kb context query fail to surface the decision "skillc and orch build skills are complementary, not competing"?

Secondary questions:
1. Did the architect consider tool placement as a design question at all?
2. What gate or context injection would have caught this?
3. Is this a pattern — does orch attract tools that belong elsewhere?

---

## What I Tested

### Test 1: Was the skillc/orch boundary decision in the architect's spawn context?

```bash
# Searched the archived spawn context for orch-go-dlw9
grep -c -i "skillc and orch build skills are complementary" \
  .orch/workspace/archived/og-arch-design-infrastructure-systematic-01mar-1da9/SPAWN_CONTEXT.md
# Result: 0

# What query was used?
grep "Query:" SPAWN_CONTEXT.md
# Result: **Query:** "design infrastructure orchestrator"
```

### Test 2: Does the architect's query surface the skillc decision?

```bash
kb context "design infrastructure orchestrator"
# Result: 11 constraints, 7 decisions surfaced
# NONE mention skillc as a tool or skillc/orch boundaries
# "skillc" appears only in one decision about SESSION_HANDOFF.md
```

### Test 3: What query WOULD surface the skillc boundary decision?

```bash
kb context "skillc skill lint test authoring"
# Result: Surfaces "skillc and orch build skills are complementary, not competing"
# Also surfaces: "skillc cannot compile SKILL.md templates without template expansion"
# Also surfaces: "skillc deploy does not signal OpenCode server to reload"
```

### Test 4: Did the architect consider tool placement?

```bash
# Searched architect output for tool placement considerations
grep -n -i "skillc\|orch-knowledge\|skill compiler\|domain boundary\|correct repo\|belongs in\|tool placement" \
  .kb/investigations/2026-03-01-design-infrastructure-systematic-orchestrator-skill.md
# Result: "skillc" appears only in 2 lines — both as a noun (a deploy tool), never as a placement target
# Line 246: "SKILL.md  # Active skill (deployed via skillc)"
# Line 258: "orch-knowledge/skills/src/meta/orchestrator/.skillc/"
# No mention of tool placement as a design question
```

### Test 5: What was in the architect's task framing?

```bash
# The orchestrator's frame for orch-go-dlw9 asked:
# "Should there be a skill linter that checks for known anti-patterns?"
# This was framed as one of 5 DESIGN QUESTIONS
#
# The task description says:
# "Design infrastructure for making controlled, systematic edits to the orchestrator skill"
#
# Both frame the work within orch-go — no cross-repo question was asked
```

---

## What I Observed

### Finding 1: The kb context query "design infrastructure orchestrator" does NOT surface skillc-related decisions

The query derived from the task title extracts keywords: "design", "infrastructure", "orchestrator". These match orch-go's infrastructure and orchestrator session decisions but contain zero signal about skill authoring toolchain boundaries.

The critical decision — "skillc and orch build skills are complementary, not competing" — is indexed under "skillc", "skill", "lint", "authoring" but NOT "design", "infrastructure", or "orchestrator".

**Root cause: keyword extraction from task title produces a query in the wrong semantic domain.** The task was about "skill testing infrastructure" but the keywords that matter for tool placement ("skillc", "skill authoring", "skill compiler") aren't in the title.

### Finding 2: The architect never considered tool placement as a design question

The architect investigation has 6 forks (design decision points):
1. What do you actually measure?
2. What's the testing protocol?
3. What does a scenario look like?
4. How does `--bare` mode integrate?
5. Should there be a skill linter?
6. What's the iteration loop?

**Fork 0 was never asked: "Where should these tools live?"** The architect assumed orch-go because:
- The spawn ran in orch-go
- The task framing ("should there be a skill linter?") assumes implementation, not placement
- File targets were pre-specified: `cmd/orch/skill_lint_cmd.go`, `pkg/skill/lint.go`

### Finding 3: The orchestrator's framing pre-decided the repo

The orchestrator's comment on orch-go-dlw9 includes:
```
5. Should there be a skill linter that checks for known anti-patterns (MUST fatigue, cosmetic redundancy, abstraction mismatch)?
```

This frames "skill linter" as an `orch` subcommand before the architect even starts. The architect then designed file targets (`cmd/orch/skill_lint_cmd.go`) consistent with this framing. The orchestrator pre-committed to orch-go by asking the question within the orch-go context.

### Finding 4: This IS a pattern — orch attracts cross-cutting tools

Evidence from kb context:
- **Template ownership split by domain**: "kb-cli owns knowledge artifacts; orch-go owns orchestration artifacts" — this decision exists but the architect had no trigger to apply it to skill authoring tools
- **skillc and orch build skills are complementary**: This decision explicitly delineates the boundary but was invisible to the architect
- **Skill output verification parses skill.yaml directly**: "skillc verify CLI doesn't exist, so we parse outputs.required from skill.yaml files in Go" — orch-go already contains skillc-domain functionality because skillc doesn't have it yet

### Finding 5: Multiple failure points, not a single root cause

The misplacement was a chain:
1. **Orchestrator framed the question inside orch-go** (pre-committed to repo)
2. **KB context query didn't surface the skillc/orch boundary decision** (wrong keywords)
3. **Architect skill has no "tool placement" fork** (doesn't ask "where should this live?")
4. **skillc's capabilities are invisible** at spawn time (no `skillc --help` or capability summary injected)
5. **The implementation agent (orch-go-12cf) executed in orch-go** without questioning placement

---

## Model Impact

- [x] **Extends** model with: KB context query derivation from task title creates a semantic blindspot when the critical decision lives in a different semantic domain than the task framing. The query "design infrastructure orchestrator" cannot surface "skillc is the correct home for skill authoring tools" because the keywords don't overlap. This is a structural limitation of keyword-based knowledge retrieval.

- [x] **Extends** model with: Spawn context for architect agents should include a "tool placement" question — "Where should these tools live? Consider: orch-go (runtime orchestrator), skillc (skill compiler), kb-cli (knowledge tools), orch-knowledge (skill sources)." Without this, architects default to wherever they were spawned.

- [x] **Extends** model with: The orchestrator itself can pre-commit to the wrong repo by framing the question within a specific project's context. The architect inherits this frame and never questions it. This is a "framing as authority" pattern — the question's frame carries implicit placement decisions.

---

## Notes

### Proposed Mitigations (in priority order)

1. **Architect skill: add "Fork 0: Where should this live?" gate** — Before designing implementation, architect should ask: "Is the spawning project the correct home for this tool?" This is cheap (one paragraph in architect skill) and catches the most common case.

2. **KB context: inject cross-repo boundary decisions for architect spawns** — When skill is `architect`, always inject decisions containing "complementary", "boundary", "owns", "split" regardless of query match. These are exactly the decisions that prevent misplacement.

3. **Orchestrator discipline: frame questions repo-agnostically** — Instead of "Should there be a skill linter [in orch]?", ask "Where should a skill linter live and what should it look like?" This prevents pre-committing to a repo.

4. **skillc capability summary in spawn context** — When spawning in orch-go, inject a brief summary of skillc's current capabilities so architects know what's available in the adjacent tool.

### Why orch-go-12cf explain-back already caught this

The explain-back comment on orch-go-12cf reads: "Lint rules implemented correctly in Go but in wrong repo — should be in skillc. Code is reference for skillc port." This means the human gate (explain-back) caught the misplacement after implementation. The question is whether it should have been caught earlier — at architect or orchestrator framing time.

### Connection to existing constraint

The constraint "Ask 'should we' before 'how do we' for strategic direction changes" is relevant but wasn't triggered here because the task didn't feel like a "strategic direction change" — it felt like straightforward infrastructure work within orch-go. The boundary question is more subtle than strategic direction.
