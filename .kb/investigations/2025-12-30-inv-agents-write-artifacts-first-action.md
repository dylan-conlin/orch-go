<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Skill templates tell agents to create artifacts first but don't require immediate content - agents can die leaving only empty templates.

**Evidence:** Agent ses_48f9cc3d5ffeJKzTJGDEhGBxxa hit API error leaving empty template at .kb/investigations/; workflow.md and investigation phase docs lack "immediate write" requirement.

**Knowledge:** Three-point artifact writing (on-start, on-error, on-completion) would ensure every spawn leaves an audit trail even when agents fail catastrophically.

**Next:** Implement changes to investigation skill workflow.md and feature-impl investigation phase to require immediate artifact population.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Agents Write Artifacts First Action

**Question:** How can we ensure agents leave an audit trail even when they fail before completing their task?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Investigation skill workflow lacks "immediate write" requirement

**Evidence:** workflow.md lines 1-9 show:
```
1. Create investigation file: `kb create investigation {slug}`
2. Fill in your question
3. Try things, observe what happens
4. Run a test to validate your hypothesis
5. Fill conclusion only if you tested
6. Commit
```
There's no explicit requirement to write meaningful content IMMEDIATELY after creating the file. Step 2 says "fill in your question" but doesn't require it as a blocking action before proceeding.

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/workflow.md:1-9`

**Significance:** If an agent creates the template, then encounters a fatal error (like the 100 PDF pages API limit), the artifact contains only empty template placeholders with no record of what was attempted.

---

### Finding 2: Feature-impl investigation phase has similar gap

**Evidence:** The investigation phase guide (lines 17-30) says:
```markdown
### 1. Create Investigation Template (Before Exploring)
**Critical:** Create template at START, not at end. Forces progressive documentation.
```
And then step 2 (lines 65-70) says:
```markdown
### 2. Fill Question and Metadata
Edit investigation file with precise question from SPAWN_CONTEXT:
- **Question:** Specific, answerable question
- **Started:** Today's date
- **Status:** In Progress
```
But again, there's no COMMIT or forced checkpoint between template creation and exploration. An agent could create the template, start exploring, hit an API error, and leave nothing useful.

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc/phases/investigation.md:17-70`

**Significance:** Same problem affects feature-impl agents doing investigation work.

---

### Finding 3: No error-handling artifact writing exists

**Evidence:** Searched both skill templates for "error" handling guidance. Neither skill mentions writing to the artifact on failure:
- investigation skill: No error-handling section
- feature-impl skill: No guidance on what to write if exploration fails

When agent ses_48f9cc3d5ffeJKzTJGDEhGBxxa hit "A maximum of 100 PDF pages may be provided", it died without updating the artifact.

**Source:** All files in `.skillc/` directories for both skills

**Significance:** Fatal errors leave no trace. We lose visibility into what was attempted and why it failed.

---

## Synthesis

**Key Insights:**

1. **Create-then-populate gap** - Both investigation skill and feature-impl have a workflow where template creation is separate from content population. This gap is where agents can die silently.

2. **Three-point artifact writing pattern** - To ensure audit trails, artifacts should be written at three points:
   - **On start:** Immediately write "Investigating: [question]" + "First approach: [what I'm trying]" and COMMIT
   - **On error:** Before dying, write "Error encountered: [error]" to Findings section and COMMIT
   - **On completion:** Full synthesis (current behavior)

3. **Progressive documentation isn't enough** - The skills already say "document progressively" but don't make the FIRST write a hard requirement with immediate commit.

**Answer to Investigation Question:**

To ensure agents leave an audit trail even when they fail:

1. Add an **"Immediate Checkpoint"** step to workflow.md (investigation skill) requiring agents to write their question AND first approach, then COMMIT before any exploration
2. Add the same checkpoint to feature-impl's investigation phase
3. Add **"Error Recovery"** guidance telling agents to write errors to their artifact before session ends

The key change is making the first artifact write a blocking checkpoint (with commit) rather than optional metadata filling.

---

## Structured Uncertainty

**What's tested:**

- ✅ Workflow.md lacks immediate write requirement (verified: read file, no "commit" or checkpoint after step 2)
- ✅ Feature-impl investigation phase has same gap (verified: read file, no commit requirement after template creation)
- ✅ No error-handling guidance exists in either skill (verified: searched all .skillc files)

**What's untested:**

- ⚠️ Whether agents will follow the new "immediate checkpoint" guidance (requires real spawn test)
- ⚠️ Whether error recovery guidance can be followed when agent is crashing (depends on error type)

**What would change this:**

- If Claude agents have a way to automatically write to artifacts on crash (OS-level), our skill guidance would be unnecessary
- If `kb create` already populates question from context, the gap might be smaller than assumed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add "Immediate Checkpoint" pattern to skill workflows** - Require agents to write question + first approach AND COMMIT before any exploration.

**Why this approach:**
- Ensures every spawn leaves at minimum: question being investigated, first approach being tried
- Works with existing `kb create` workflow (just adds a hard commit requirement)
- Small change with high impact on audit trail visibility

**Trade-offs accepted:**
- Extra commit in the middle of work (but artifacts already recommend progressive commits)
- Can't capture errors that happen BEFORE first approach (but that's rare)

**Implementation sequence:**
1. Update `investigation/.skillc/workflow.md` - Add "Immediate Checkpoint" step after file creation
2. Update `feature-impl/.skillc/phases/investigation.md` - Same pattern
3. Optionally add "Error Recovery" section to both skills

### Alternative Approaches Considered

**Option B: OS-level crash handler**
- **Pros:** Would capture all errors automatically
- **Cons:** Not possible - agents run in Claude's environment, no OS hooks available
- **When to use instead:** Never (not feasible)

**Option C: Post-spawn verification by orchestrator**
- **Pros:** Orchestrator could check artifact has content before proceeding
- **Cons:** Doesn't solve the root cause - artifact still empty if agent dies
- **When to use instead:** Could be a complementary detection mechanism

**Rationale for recommendation:** Skill guidance is the lever we can pull. Agents follow skill instructions, so making immediate checkpoint mandatory will change behavior.

---

### Implementation Details

**What to implement first:**
- Investigation skill workflow.md (most commonly used)
- Feature-impl investigation phase (extends coverage)

**Things to watch out for:**
- ⚠️ Skill files are in `~/orch-knowledge/skills/src/worker/{skill}/.skillc/` 
- ⚠️ After editing, need to run `skillc deploy` to regenerate SKILL.md files
- ⚠️ Keep changes minimal - just add the checkpoint requirement

**Areas needing further investigation:**
- Whether other skills (systematic-debugging, architect) need similar changes
- How to handle errors that occur mid-exploration (currently no guidance)

**Success criteria:**
- ✅ Next agent spawn with investigation skill writes to artifact and commits within first 3 tool calls
- ✅ If agent dies after checkpoint, artifact shows what was being attempted
- ✅ Spawns no longer leave empty template files

---

## References

**Files Examined:**
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/workflow.md` - Investigation skill main workflow
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/intro.md` - Investigation skill intro  
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/template.md` - Investigation skill template guidance
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/completion.md` - Investigation skill completion
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/self-review.md` - Investigation skill self-review
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc/phases/investigation.md` - Feature-impl investigation phase

**Commands Run:**
```bash
# Find skill files
glob **/*.md in ~/orch-knowledge/skills/src/worker/investigation/.skillc
glob **/*.md in ~/orch-knowledge/skills/src/worker/feature-impl/.skillc
```

**External Documentation:**
- SPAWN_CONTEXT evidence: Agent ses_48f9cc3d5ffeJKzTJGDEhGBxxa failed with "100 PDF pages max" error

**Related Artifacts:**
- N/A (new investigation, no prior related work found)

---

## Investigation History

**2025-12-30:** Investigation started
- Initial question: How can agents leave audit trails even when they fail?
- Context: Agent died on API error, left only empty template in .kb/investigations/

**2025-12-30:** Found the gap
- Workflow.md has no immediate commit requirement after template creation
- Feature-impl investigation phase has same gap

**2025-12-30:** Investigation completed
- Status: Complete
- Key outcome: Need "Immediate Checkpoint" pattern requiring commit after filling question + first approach
