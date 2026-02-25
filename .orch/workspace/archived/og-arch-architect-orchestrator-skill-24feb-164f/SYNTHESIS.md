# Session Synthesis

**Agent:** og-arch-architect-orchestrator-skill-24feb-164f
**Issue:** orch-go-1178
**Duration:** 2026-02-24
**Outcome:** success

---

## Plain-Language Summary

Orchestrator agents load the skill, can describe their role ("I'm your orchestrator"), but at action time revert to Claude Code defaults — using the Task tool instead of `orch spawn`, and `bd close` instead of `orch complete`. This investigation found the root cause: identity declarations are additive (they don't conflict with anything) while action constraints are subtractive (they fight the Claude Code system prompt, which actively promotes the Task tool with a 17:1 signal advantage). The "action space restriction" claimed by the skill is just a markdown table, not actual infrastructure enforcement — violating the system's own "Infrastructure Over Instruction" principle. The fix requires two layers: restructure the skill to fuse action constraints with identity at the top (leveraging the identity compliance that already works), and add tool-layer enforcement via Claude Code hooks that detect and flag prohibited tool usage in orchestrator sessions.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root. Key outcomes:
- Investigation artifact produced at `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md`
- Probe produced at `.kb/models/orchestrator-session-lifecycle/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md`
- 6 findings with evidence, concrete structural recommendations for skill rewrite

---

## TLDR

Diagnosed why orchestrator agents comply with identity but not action constraints: the Claude Code system prompt has structural priority and 17:1 signal advantage over skill-level action constraints. Recommended two-layer fix: restructure skill with action-identity fusion at top + add tool-layer enforcement via hooks.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` - Full investigation with diagnosis and recommendations
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md` - Probe confirming/extending model claims

### Files Modified
- None (investigation-only session)

### Commits
- (pending at session close)

---

## Evidence (What Was Observed)

- Signal ratio: ~500 words in system prompt promoting Task tool vs ~30 words in skill constraining it (17:1)
- Action constraints position: first at line 68 (inside 9-item checklist), full detail at line 594 (88% of 640-line skill)
- Tool Action Space table is a markdown description, not infrastructure enforcement — all tools remain available
- Identity declarations appear 5+ times in first 108 lines; action constraints appear 2 times before line 100
- Prior probes confirm: skill organized around identity not needs (Feb 16), 5 injection paths with caching bugs (Feb 17), 13 stale CLI references (Feb 18)
- Academic research (ICLR 2025, AgentSpec ICSE 2026, PCAS) confirms prompt-level constraints insufficient for action enforcement

### Tests Run
```bash
# Structural analysis of skill positioning
# Read ~/.claude/skills/meta/orchestrator/SKILL.md - 640 lines
# Action constraints at lines: 68, 75, 594-604
# Identity declarations at lines: 33, 35, 37-39, 96-108
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` - Full diagnosis and structural recommendations
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md` - Model extensions: action space described not enforced, competing instruction hierarchy failure mode

### Decisions Made
- Action constraints must be fused with identity declarations (not separate section) because identity compliance already works
- Skill needs two-layer fix: prompt restructuring for salience + infrastructure enforcement for durability
- "Restricting action space via guidelines" is an oxymoron — needs infrastructure enforcement

### Constraints Discovered
- Claude Code system prompt has structural priority over user-level content (system > user in instruction hierarchy)
- Prompt-level action constraints face 17:1 signal disadvantage against system prompt tool promotion
- Identity and action compliance are mechanistically different — testing one doesn't validate the other

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue 1:** Restructure orchestrator skill with action-identity fusion (Layer 1)
**Skill:** feature-impl
**Context:**
```
Restructure the orchestrator skill template at ~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template.
Key changes: (1) Fuse action constraints with identity at top of skill, (2) Add affordance replacement at all spawn/completion decision points, (3) Reduce from 640 to <450 lines. See investigation .kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md for full spec.
```

**Issue 2:** Investigate Claude Code hook feasibility for tool-layer enforcement (Layer 2)
**Skill:** investigation
**Context:**
```
Spike: Can Claude Code hooks intercept tool calls and inject feedback? Need to confirm hook API supports PreToolCall or PostToolCall patterns. If feasible, design an orchestrator-guard hook that detects ORCH_ORCHESTRATOR=1 and warns on Task tool / Edit / Write usage. See investigation for architectural context.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Can Claude Code hooks actually intercept individual tool calls? The hook API may only support session-level or message-level triggers.
- What's the actual compliance rate of current orchestrator sessions? We have anecdotal evidence (the Toolshed incident) but no systematic measurement.
- Does the `ORCH_WORKER=1` env var pattern (used to skip orchestrator skill for workers) provide a clean model for `ORCH_ORCHESTRATOR=1` detection?

**Areas worth exploring further:**
- A/B testing skill restructuring: run 10 orchestrator sessions with old skill vs 10 with restructured skill, measure action compliance
- Whether OpenCode's `experimental.chat.system.transform` hook could inject action constraints at system level (bypassing the hierarchy problem)

**What remains unclear:**
- Whether the 17:1 signal ratio is the dominant factor or just one of several equally important factors
- How much compliance improvement Layer 1 alone would achieve (estimated 60-70%, needs testing)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-architect-orchestrator-skill-24feb-164f/`
**Investigation:** `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md`
**Beads:** `bd show orch-go-1178`
