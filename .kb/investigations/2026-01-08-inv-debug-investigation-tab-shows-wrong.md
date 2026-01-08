<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation tab shows wrong files for cross-project agents because `discoverInvestigationPath` uses an incorrect `ProjectDir` when the workspace cache lookup fails.

**Evidence:** Code trace shows that `agents[i].ProjectDir` starts as `s.Directory` (orchestrator's cwd) and is only overwritten if `beadsProjectDirs[beadsID]` lookup succeeds; when it fails, `discoverInvestigationPath` searches the wrong project's `.kb/investigations/`.

**Knowledge:** Cross-project agents need a reliable fallback for `ProjectDir` - the session directory from OpenCode reflects the orchestrator's cwd due to `--attach` bug, not the target project.

**Next:** Fix the fallback logic - don't use session directory for investigation discovery when it's not the agent's target project.

**Promote to Decision:** recommend-no - tactical bug fix, not architectural

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

# Investigation: Debug Investigation Tab Shows Wrong

**Question:** Why does the Investigation tab show the wrong file for cross-project agents?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** og-inv-debug-investigation-tab-08jan-de93
**Phase:** Investigating
**Next Step:** Implement fix for ProjectDir fallback logic
**Status:** In Progress

---

## Findings

### Finding 1: ProjectDir initialization uses orchestrator's cwd

**Evidence:** In `serve_agents.go:466`, when creating the `AgentAPIResponse`, `ProjectDir` is initialized from `s.Directory` which is the OpenCode session directory. Due to the `--attach` bug in OpenCode, this is often the orchestrator's cwd, not the target project.

**Source:** `cmd/orch/serve_agents.go:458-467`

**Significance:** For cross-project agents spawned with `--workdir`, the initial `ProjectDir` value is wrong.

---

### Finding 2: ProjectDir overwrite only happens if workspace cache has the beads ID

**Evidence:** At line 780-782, `agents[i].ProjectDir` is overwritten from `beadsProjectDirs[agents[i].BeadsID]` - but only if the lookup succeeds. If the workspace cache doesn't have this beads ID (new agent, workspace not scanned yet, etc.), the incorrect initial value remains.

**Source:** `cmd/orch/serve_agents.go:780-782`

**Significance:** When workspace cache lookup fails, `ProjectDir` remains as the orchestrator's cwd, causing `discoverInvestigationPath` to search the wrong `.kb/investigations/` directory.

---

### Finding 3: discoverInvestigationPath searches based on ProjectDir

**Evidence:** At line 792, `discoverInvestigationPath(workspaceName, agents[i].BeadsID, agents[i].ProjectDir, invDirCache)` is called with the potentially incorrect `ProjectDir`. The function searches `projectDir/.kb/investigations/` for matching files.

**Source:** `cmd/orch/serve_agents.go:787-795` and `cmd/orch/serve_agents.go:144-268`

**Significance:** If `ProjectDir` is wrong (orchestrator's cwd instead of target project), the auto-discovery searches the wrong project's investigations and may find an unrelated file with similar keywords.

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

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

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
