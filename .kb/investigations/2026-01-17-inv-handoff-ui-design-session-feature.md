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

# Investigation: Handoff Ui Design Session Feature

**Question:** How do we enable clean handoff from ui-design-session to feature-impl with approved mockup path, design prompt, and design notes?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** feature-impl agent
**Phase:** Investigating
**Next Step:** Implement Config struct changes and spawn command flag
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Spawn context is generated via pkg/spawn/context.go

**Evidence:** The `WriteContext` function (line 500) generates SPAWN_CONTEXT.md files using a template (SpawnContextTemplate, line 54). The template is populated with data from a Config struct.

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:500-585`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:54-318`

**Significance:** To add design handoff info to spawn context, we need to: (1) Add fields to the Config struct, (2) Modify the template to include design reference section, (3) Pass design workspace info from spawn command.

---

### Finding 2: Config struct holds all spawn context data

**Evidence:** The Config struct (pkg/spawn/config.go) holds fields like Task, SkillName, ProjectDir, WorkspaceName, etc. These are populated from spawn command flags and passed to GenerateContext.

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/config.go` (need to read this file)
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1158-1183` (where Config is built)

**Significance:** We need to add design handoff fields to Config struct (DesignWorkspace, DesignMockupPath, DesignPromptPath, DesignNotes).

---

### Finding 3: Spawn command has many flags for customization

**Evidence:** The spawn command has flags like --issue, --phases, --mode, --validation, --workdir, etc. (spawn_cmd.go:168-193)

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:168-193`

**Significance:** We can add a --design-workspace flag to reference a prior design session workspace. The flag value would be used to locate design artifacts and populate the Config fields.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

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

**Add Design Handoff Support via --design-workspace Flag** - Enable feature-impl agents to receive design context from completed ui-design-session workspaces.

**Why this approach:**
- Uses existing spawn flag pattern (consistent with --workdir, --issue, etc.)
- Leverages existing workspace structure (.orch/workspace/{name}/screenshots/)
- No new concepts - just reads artifacts from prior workspace
- Aligns with existing handoff mechanisms (beads issues, session handoff)

**Trade-offs accepted:**
- Requires orchestrator to manually specify design workspace (not auto-discovered)
- Design workspace must still exist (can't handoff from archived workspaces without extra logic)
- Simple string passing - no validation that design workspace actually exists

**Implementation sequence:**
1. Add Config fields for design handoff data (foundational data structure)
2. Add --design-workspace flag to spawn command (user interface)
3. Implement design artifact reading logic (populate Config from workspace)
4. Update spawn context template to include design reference section (agent-facing output)
5. Test the handoff flow manually

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
