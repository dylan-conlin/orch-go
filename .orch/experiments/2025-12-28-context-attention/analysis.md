# Experiment: Context Attention

**Status:** Complete  
**Created:** 2025-12-28

## Hypothesis

Constraint position and framing affect whether agents recognize pre-existing answers.

## Background

An agent spawned to investigate "OpenCode redirect loop" had the answer literally in their spawn context at line 85-86:

> OpenCode has no /health endpoint - use /session to check server status.
> OpenCode returns 500 'redirected too many times' for any invalid route.

Despite this, the agent spent ~30 minutes re-investigating and reached the same conclusion.

This was the 4th investigation on the same topic. Prior investigations existed, a kn entry existed, and it was surfaced in spawn context. The knowledge surfacing worked. The agent attention/recognition failed.

## Key Finding

**The problem isn't position or framing - it's task semantics.**

Information ("KNOWN ANSWER: X") is treated as context to verify.
Instruction ("DO NOT INVESTIGATE") changes what the agent does.

The investigation skill's mandate to "test before concluding" overrides any pre-provided answers unless explicitly countermanded.

## Results

### Run A-1: "KNOWN ANSWER" framing (top position)

| Metric | Value |
|--------|-------|
| Tool Calls Before Recognition | 0 (never recognized) |
| Recognized Existing Answer | No |
| Time to Recognition | 283s (full investigation) |
| Outcome | reinvestigated_fully |

**Notes:** Agent had KNOWN ANSWER at line 1, still did full investigation (~5min). Did NOT cite the known answer. Wrote "Related Artifacts: None (this is a new investigation)" despite having answer upfront.

### Run A-2: "DO NOT INVESTIGATE" framing

| Metric | Value |
|--------|-------|
| Tool Calls Before Recognition | 1 |
| Recognized Existing Answer | Yes |
| Time to Recognition | 330s |
| Outcome | cited_existing |

**Notes:** Agent CITED the known answer, did NOT re-run tests, explicitly noted "Not re-verified in this session (intentionally - answer already known)". This framing worked.

## Analysis

### Position Effect
Not tested in isolation - but A-1 showed that even top position with "KNOWN ANSWER" label failed.

### Framing Effect
Critical finding: **Information framing fails, instruction framing works.**

| Framing | Behavior |
|---------|----------|
| "KNOWN ANSWER: X" | Treated as hypothesis to verify |
| "DO NOT INVESTIGATE. Document that X is solved." | Followed as instruction |

### Root Cause

The investigation skill says "you cannot conclude without testing." This instruction conflicts with pre-provided answers. Agents resolve the conflict by treating KNOWN ANSWER as a claim requiring verification.

## Conclusions

1. **Surfacing ≠ Compliance** - Knowledge can be perfectly surfaced and still ignored if task framing conflicts.

2. **Skills define behavior more than context** - The skill's core instruction ("test before concluding") overrode the KNOWN ANSWER.

3. **Task framing is the strongest lever** - Not position, not labeling, but what the agent is told to DO.

## Implications

### System Changes Implemented

**Pre-spawn gate added to orchestrator skill:**

When `kb context` returns an answer that addresses the investigation question, orchestrator now presents options:
- [C] Cite - Document without re-investigation
- [V] Verify - Confirm still true
- [I] Investigate anyway
- [S] Skip - Don't spawn

Each option has specific task framing that changes agent behavior.

### Future Experiments

This infrastructure enables:
- Model comparison (which models respect constraints best?)
- Skill testing (do skills produce expected behaviors?)
- Prompt engineering with evidence (A/B test framings)
- Context optimization (minimum viable context?)

## References

- Run A-1: `.orch/experiments/2025-12-28-context-attention/runs/A-1.json`
- Run A-2: `.orch/experiments/2025-12-28-context-attention/runs/A-2.json`
- Investigation A-1: `.kb/investigations/2025-12-28-inv-known-answer-opencode-no-health.md`
- Investigation A-2: `.kb/investigations/2025-12-28-inv-stop-already-solved-not-investigate.md`
- Orchestrator skill update: `~/.claude/skills/meta/orchestrator/SKILL.md`
