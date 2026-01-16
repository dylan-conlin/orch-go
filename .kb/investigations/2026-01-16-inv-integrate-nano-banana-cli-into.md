<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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

# Investigation: Integrate Nano Banana Cli Into

**Question:** How should the Nano Banana CLI be integrated into the ui-design-session skill to enable reliable mockup generation with iteration support?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Worker agent (orch-go-gy1o4.3.2)
**Phase:** Investigating
**Next Step:** Document findings and create implementation plan
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Working CLI exists at ~/.claude/tools/nano-banana/

**Evidence:** The generate_mockup.py CLI is fully functional with:
- Support for markdown prompt files or direct text via --prompt flag
- Gemini 2.5 Flash Image integration
- Cost feedback (~$0.04/image)
- Time feedback (~10 seconds)
- Multiple output format support with automatic indexing

**Source:** /Users/dylanconlin/.claude/tools/nano-banana/generate_mockup.py (lines 1-163)

**Significance:** We have a working foundation. No need to build CLI from scratch, just need to integrate it properly into the skill workflow.

---

### Finding 2: ui-design-session skill has weak CLI integration

**Evidence:** Current skill guidance (lines 206-217 in template) says:
```bash
# If tooling exists at ~/.claude/tools/nano-banana/
cd ~/.claude/tools/nano-banana
uv run generate_mockup.py prompt.md --output mockup.png
```
Then adds: "If tooling not available: Document prompts in workspace and report to orchestrator for generation."

This creates ambiguity - agents don't know if tooling is available or how to verify it.

**Source:** ~/orch-knowledge/skills/src/worker/ui-design-session/.skillc/SKILL.md.template (lines 206-217)

**Significance:** Weak integration causes agents to skip CLI and ask orchestrator to generate mockups manually, defeating the purpose of the automation.

---

### Finding 3: No iteration workflow defined

**Evidence:** Skill template mentions iteration conceptually (lines 249-256) but doesn't specify:
- File naming convention for versions (v1, v2, v3)
- How to track which version is current
- Where to store prompts for regeneration
- How to handle multiple mockup variations

**Source:** ~/orch-knowledge/skills/src/worker/ui-design-session/.skillc/SKILL.md.template (lines 249-256)

**Significance:** Without explicit iteration guidance, agents will create ad-hoc naming schemes, making it hard to track design evolution.

---

### Finding 4: Skills built with skillc from source

**Evidence:** The ui-design-session skill is built from:
- Source: ~/orch-knowledge/skills/src/worker/ui-design-session/.skillc/SKILL.md.template
- Config: ~/orch-knowledge/skills/src/worker/ui-design-session/.skillc/skill.yaml
- Deployed to: ~/.claude/skills/worker/ui-design-session/SKILL.md via `skillc deploy`

**Source:** skill.yaml metadata and SKILL.md.template AUTO-GENERATED comment headers

**Significance:** Changes must be made to the source template, then deployed via skillc. Cannot edit deployed SKILL.md directly.

---

## Synthesis

**Key Insights:**

1. **CLI is ready, skill guidance is not** - The technical infrastructure exists and works well. The problem is purely in the skill documentation - agents need explicit, confident guidance on how to invoke the CLI.

2. **Iteration needs structure** - Design iteration is inherently messy, but the skill should impose structure: prompt files in prompts/ directory, mockups in mockups/ with v1/v2/v3 naming, and a clear "current" marker.

3. **Feedback loop needs visibility** - Cost/time feedback already exists in CLI output, but skill should emphasize displaying this to user via bd comment to create transparency around resource usage.

**Answer to Investigation Question:**

The CLI integration should be strengthened by: (1) Removing "if available" conditional language and asserting CLI location exists, (2) Adding explicit workflow for prompt file creation → CLI invocation → mockup storage, (3) Defining iteration naming convention (mockup-v1.png, mockup-v2.png), (4) Requiring cost/time feedback via bd comment after generation. The CLI itself needs no changes - it already supports all required features.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Update ui-design-session skill template with explicit CLI workflow** - Replace conditional "if available" language with confident step-by-step CLI invocation guidance including iteration naming and cost feedback.

**Why this approach:**
- Addresses Finding 2: Removes ambiguity about CLI availability
- Addresses Finding 3: Adds explicit iteration workflow with naming convention
- Addresses Finding 1: Leverages existing working CLI without modification
- Simplest path: Only skill template needs updating, no CLI changes

**Trade-offs accepted:**
- Assumes ~/.claude/tools/nano-banana/ exists (reasonable - it's part of setup)
- Requires manual smoke test after deployment (acceptable per constraints)
- No automated verification CLI is working (can add later if needed)

**Implementation sequence:**
1. Update SKILL.md.template "Generate Mockups" section (lines 206-228) with explicit workflow
2. Add "Iteration Workflow" subsection with v1/v2/v3 naming convention
3. Add "Cost/Time Feedback" requirement to report generation stats via bd comment
4. Deploy via `skillc deploy` to ~/.claude/skills/
5. Manual smoke test: spawn ui-design-session agent and verify CLI invocation works

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- Update "Generate Mockups" section (lines 206-228) - foundational workflow
- Add explicit directory structure guidance (prompts/, mockups/)
- Add iteration naming convention subsection

**Things to watch out for:**
- ⚠️ GEMINI_API_KEY environment variable must be set - add troubleshooting note
- ⚠️ `uv` must be installed - add setup verification step
- ⚠️ Workspace path resolution - ensure agents use correct workspace directory
- ⚠️ File naming collisions - iteration naming should prevent overwrites

**Areas needing further investigation:**
- Future: Should CLI support batch generation (multiple variations in one call)?
- Future: Should there be a helper script to wrap CLI with workspace path logic?
- Future: Integration with glass for visual verification of generated mockups?

**Success criteria:**
- ✅ Agent can invoke CLI without orchestrator intervention
- ✅ Iteration naming is consistent (v1, v2, v3)
- ✅ Cost/time feedback appears in bd comments
- ✅ Generated mockups are properly committed to workspace
- ✅ Manual smoke test passes (spawn agent, generate mockup, verify files)

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
