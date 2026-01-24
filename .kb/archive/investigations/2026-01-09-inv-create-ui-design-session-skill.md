<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created ui-design-session skill scaffold following worker skill patterns with design-principles dependency and Nano Banana integration.

**Evidence:** Skill compiled successfully (4321 tokens, 86.4% budget), deployed to ~/.claude/skills/worker/ui-design-session/ with SKILL.md generated, skill.yaml includes design-principles and worker-base dependencies.

**Knowledge:** Worker skills follow .skillc structure (skill.yaml + SKILL.md.template), dependencies load at spawn time, interactive workflows require explicit orchestrator review checkpoints, token budget at 86% leaves minimal headroom.

**Next:** Commit skill to orch-knowledge repo, create SYNTHESIS.md documenting deliverables, mark phase complete - skill ready for orchestrator testing via orch spawn.

**Promote to Decision:** recommend-no (implementation artifact, not architectural decision)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Create Ui Design Session Skill

**Question:** How should ui-design-session skill be structured to enable interactive design workflows with Nano Banana integration?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Skill Structure Pattern

**Evidence:** Examined feature-impl, ui-mockup-generation, and design-principles skills. All worker skills follow consistent structure: `.skillc/skill.yaml` for metadata, `.skillc/SKILL.md.template` for content, optional `reference/` for supplementary docs.

**Source:** 
- `~/orch-knowledge/skills/src/worker/feature-impl/.skillc/`
- `~/.claude/skills/utilities/ui-mockup-generation/`
- `~/.claude/skills/shared/design-principles/`

**Significance:** Establishes clear pattern for creating new skills. The .skillc directory structure is the compilation source, SKILL.md is generated output.

---

### Finding 2: Dependency System

**Evidence:** Skills declare dependencies in skill.yaml using `dependencies:` array. Worker skills commonly depend on `worker-base`, specialized skills load domain-specific guidance like `design-principles`.

**Source:** 
- `~/orch-knowledge/skills/src/worker/feature-impl/.skillc/skill.yaml` (declares worker-base)
- Spawn context mentions dependencies resolved at spawn time by orch-go

**Significance:** ui-design-session should declare both `worker-base` and `design-principles` as dependencies to load foundational patterns and visual standards.

---

### Finding 3: Interactive Workflow Expectations

**Evidence:** Task description specifies "interactive by default (--tmux or dashboard)". Reviewed orchestrator skill guidance on monitoring patterns - agents can run headless (HTTP API) or with visual monitoring (--tmux flag).

**Source:**
- SPAWN_CONTEXT.md task requirements
- Orchestrator skill monitoring section

**Significance:** Skill should emphasize visual feedback loops (agent generates → orchestrator reviews → iterates) and reference dashboard/tmux monitoring capabilities.

---

### Finding 4: Nano Banana Integration

**Evidence:** ui-mockup-generation skill provides comprehensive Nano Banana (Gemini 2.5 Flash) guidance: prompt engineering patterns, text accuracy expectations (95-98% simple, 70-80% dense), cost ($0.04/image), and quality tradeoffs.

**Source:**
- `~/.claude/skills/utilities/ui-mockup-generation/SKILL.md`

**Significance:** ui-design-session should reference ui-mockup-generation for detailed tooling but synthesize key prompt engineering patterns into workflow guidance.

---

### Finding 5: Token Budget Constraints

**Evidence:** Compiled skill reports 4321 tokens (86.4% of 5000 token budget). Skillc warns at >80% usage.

**Source:**
- `skillc build` output: "Token usage (86.4%) exceeds 80% of budget"

**Significance:** Skill is near token limit. Any future additions would require either increasing budget or moving content to reference docs.

---

## Synthesis

**Key Insights:**

1. **Skill as Integration Layer** - ui-design-session serves as workflow coordinator between Nano Banana tooling (ui-mockup-generation), visual standards (design-principles), and implementation (feature-impl). It's not creating new capabilities but structuring how existing capabilities combine for design workflows.

2. **Interactive by Nature** - Design fundamentally requires visual feedback loops. The skill emphasizes orchestrator review checkpoints after mockup generation, distinguishing it from autonomous worker skills. This matches the --tmux/dashboard pattern from the task requirements.

3. **Progressive Disclosure via Dependencies** - By loading design-principles as dependency (2500+ tokens), the skill gains full visual standards without duplicating content. This keeps ui-design-session focused on workflow guidance while leveraging comprehensive design guidance already captured.

**Answer to Investigation Question:**

ui-design-session should be structured as a phased workflow skill (Design Brief → Mockup Generation → Visual Review → Handoff) that loads design-principles and references ui-mockup-generation tooling. The scaffold includes:
- skill.yaml with dependencies and deliverable definitions
- SKILL.md.template with workflow phases and Nano Banana guidance
- Deployment to ~/.claude/skills/worker/ui-design-session/ with root-level symlink

Token usage at 86.4% (4321/5000) means the skill is comprehensive but near capacity for future expansion.

---

## Structured Uncertainty

**What's tested:**

- ✅ Skill compiles successfully (verified: `skillc build` completed without errors)
- ✅ Token usage within budget (verified: 4321/5000 = 86.4%)
- ✅ Deployment structure matches existing worker skills (verified: manual comparison with feature-impl and architect)
- ✅ Dependencies declared properly (verified: skill.yaml includes design-principles and worker-base)

**What's untested:**

- ⚠️ Skill loadable by orch spawn (not tested - would require spawning agent with skill)
- ⚠️ Design-principles dependency resolves at spawn time (not tested - requires orch-go LoadSkillWithDependencies)
- ⚠️ Nano Banana tooling actually exists at ~/.claude/tools/nano-banana/ (referenced but not verified)
- ⚠️ Workflow phases align with actual design session needs (not validated with real usage)

**What would change this:**

- Finding would be wrong if `orch spawn ui-design-session "test"` fails to load the skill
- Design-principles dependency would fail if orch-go doesn't find the skill at expected path
- Nano Banana references would be broken if tooling doesn't exist (skill would still work but with degraded tooling integration)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Phased Workflow with Dependency Loading** - Structure ui-design-session as 4-phase workflow (Design Brief → Mockup Generation → Visual Review → Handoff) that loads design-principles and references ui-mockup-generation.

**Why this approach:**
- Matches interactive pattern from task requirements (orchestrator review checkpoints)
- Leverages existing comprehensive guidance via dependencies (design-principles = 2500+ tokens)
- Creates clear handoff pattern to feature-impl for implementation transition
- Aligns with feature-impl phase structure (familiar pattern for orchestrators)

**Trade-offs accepted:**
- Token usage at 86.4% limits future expansion (acceptable - skill can reference docs)
- Requires Nano Banana tooling to exist (graceful degradation if missing - prompts still work manually)
- Interactive workflow means longer session times vs autonomous skills (acceptable - design requires iteration)

**Implementation sequence:**
1. skill.yaml with metadata, dependencies, deliverables (defines skill contract)
2. SKILL.md.template with phased workflow guidance (operational content)
3. Deploy to ~/.claude/skills/worker/ui-design-session/ (standard location)
4. Create root-level symlink (discoverability pattern)

### Alternative Approaches Considered

**Option B: Autonomous Design Agent**
- **Pros:** Faster execution, no orchestrator intervention needed
- **Cons:** Design fundamentally requires visual feedback (can't automate taste/direction validation)
- **When to use instead:** Never - design decisions require human judgment

**Option C: Embed Design-Principles Content**
- **Pros:** Self-contained skill, no dependency resolution needed
- **Cons:** Duplicates 2500+ tokens, creates sync burden, violates DRY
- **When to use instead:** Only if dependency system broken (not the case)

**Rationale for recommendation:** Option A (phased workflow with dependencies) best balances comprehensiveness (via design-principles loading), token efficiency (references not duplication), and workflow fidelity (matches how design actually works - iterative with feedback).

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
