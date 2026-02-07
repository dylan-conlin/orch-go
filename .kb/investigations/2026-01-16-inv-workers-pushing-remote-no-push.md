<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The no-push rule exists in orchestrator skill but is missing from worker SPAWN_CONTEXT template, causing workers to push to remote when they shouldn't.

**Evidence:** Found "Worker rule: Commit your work, call /exit. Don't push." in orchestrator skill at ~/.claude/skills/, but SpawnContextTemplate in pkg/spawn/context.go:38 has no such guidance.

**Knowledge:** Worker agents only see SPAWN_CONTEXT.md during spawn - they don't load the orchestrator skill where this rule lives. The template must include worker-specific git guidance.

**Next:** Add explicit "NEVER push to remote" section to SpawnContextTemplate in pkg/spawn/context.go.

**Promote to Decision:** Actioned - constraint documented in CLAUDE.md (Push Requires User Approval)

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

# Investigation: Workers Pushing Remote No Push

**Question:** Why are workers pushing to remote repositories when they should only commit locally and call /exit?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** og-arch-workers-pushing-remote-16jan-4bec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Orchestrator skill contains worker no-push rule

**Evidence:** Found "**Worker rule:** Commit your work, call `/exit`. Don't push." in multiple orchestrator skill files with the reasoning: "Pushes can trigger deploys that disrupt production systems (e.g., collection runs). The user decides when the system is ready for deployment, not the orchestrator."

**Source:** 
- ~/.claude/skills/SKILL.md
- ~/.claude/skills/meta/orchestrator/SKILL.md
- ~/.claude/skills/src/meta/orchestrator/SKILL.md

**Significance:** The rule exists and has clear rationale, but workers don't load the orchestrator skill - they only see SPAWN_CONTEXT.md.

---

### Finding 2: SPAWN_CONTEXT template lacks git operation guidance

**Evidence:** The SpawnContextTemplate in pkg/spawn/context.go:38-286 contains SESSION COMPLETE PROTOCOL that says "After your final commit, BEFORE typing anything else" but provides no guidance about whether to push or not push to remote.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:38-286

**Significance:** Without explicit guidance in the spawn context, workers may assume they should push (following general developer habits or AGENTS.md guidance which is for orchestrators).

---

### Finding 3: AGENTS.md has conflicting guidance

**Evidence:** AGENTS.md line 24 says "PUSH TO REMOTE - This is MANDATORY" with rule "Work is NOT complete until `git push` succeeds". This contradicts the worker-specific no-push rule.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/AGENTS.md:24-39

**Significance:** AGENTS.md is for orchestrators (who DO push), but if workers see it, they'll get conflicting signals. Workers only see SPAWN_CONTEXT.md, so fixing the template is sufficient.

---

## Synthesis

**Key Insights:**

1. **Workers don't load orchestrator skill** - Worker agents only see SPAWN_CONTEXT.md during spawn. The orchestrator skill (where the no-push rule lives) is loaded via ORCH_WORKER=0 for orchestrators only.

2. **Template is the single source of truth for workers** - The SpawnContextTemplate in pkg/spawn/context.go is generated for every worker spawn and is the only guidance they receive about git operations.

3. **Multiple completion protocol locations** - The template has three places where SESSION COMPLETE PROTOCOL appears (no-track, tracked, and final step). The no-push guidance needs to appear in all three to be effective.

**Answer to Investigation Question:**

Workers push to remote because the SPAWN_CONTEXT template lacks explicit "don't push" guidance. The rule exists in the orchestrator skill (lines found in ~/.claude/skills/), but workers never load that skill. The fix is to add the no-push guidance with rationale to all SESSION COMPLETE PROTOCOL sections in pkg/spawn/context.go:38-286, ensuring workers see it regardless of spawn mode.

---

## Structured Uncertainty

**What's tested:**

- ✅ Orchestrator skill contains no-push rule (verified: found in 3 skill file locations with grep)
- ✅ SPAWN_CONTEXT template lacked git guidance (verified: read template, no push/no-push guidance existed)
- ✅ Added guidance appears in generated context (verified: wrote test TestGenerateContext_NoPushGuidance, passes)
- ✅ All existing tests still pass (verified: ran go test ./pkg/spawn/, all pass)

**What's untested:**

- ⚠️ Real worker will see and follow the guidance (will be tested when next worker is spawned)
- ⚠️ Guidance is prominent enough to prevent pushes (depends on worker attention)

**What would change this:**

- Finding would be wrong if workers still push after this fix (would indicate guidance isn't prominent enough or is being ignored)
- Finding would be wrong if template rendering fails (tests would catch this)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add explicit no-push guidance to SPAWN_CONTEXT template** - Insert prominent "NEVER run git push" section with rationale in all three SESSION COMPLETE PROTOCOL locations in pkg/spawn/context.go.

**Why this approach:**
- Workers only see SPAWN_CONTEXT.md - this is the only place to reach them
- Matches orchestrator skill wording for consistency
- Provides both rule and rationale (why pushes are dangerous)
- Placed immediately before completion steps for maximum visibility

**Trade-offs accepted:**
- Adds ~4 lines to spawn context (acceptable, critical safety guidance)
- Workers must read and follow (can't technically prevent git push, only guide against it)

**Implementation sequence:**
1. Add no-push guidance block to all 3 SESSION COMPLETE PROTOCOL sections (completed)
2. Write test to verify guidance appears in generated context (completed)
3. Verify all tests pass (completed)
4. Commit and verify with real worker spawn (in progress)

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
- /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go - SPAWN_CONTEXT template definition
- ~/.claude/skills/SKILL.md - Orchestrator skill with worker no-push rule
- ~/.claude/skills/meta/orchestrator/SKILL.md - Meta-orchestrator skill (same rule)
- /Users/dylanconlin/Documents/personal/orch-go/AGENTS.md - General agent guidance (orchestrator-focused)

**Commands Run:**
```bash
# Search for no-push rule in orchestrator skills
rg -A 3 -B 3 "Don't push" ~/.claude/skills/

# Search for SPAWN_CONTEXT usage across codebase
grep -r "SPAWN_CONTEXT" --include="*.go" /Users/dylanconlin/Documents/personal/orch-go/

# Run tests after adding no-push guidance
go test ./pkg/spawn/ -run TestGenerateContext_NoPushGuidance -v
go test ./pkg/spawn/ -v
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Issue:** orch-go-f16wc - Bug report that triggered this investigation
- **Workspace:** .orch/workspace/og-arch-workers-pushing-remote-16jan-4bec - This agent's workspace

---

## Investigation History

**2026-01-16 (start):** Investigation started
- Initial question: Why are workers pushing to remote repositories when they should only commit locally?
- Context: Bug discovered in price-watch project where agent pw-feat-add-scheduledjobexecution-model-16jan-ee43 pushed to remote

**2026-01-16 (findings):** Located root cause
- Found no-push rule in orchestrator skill but not in SPAWN_CONTEXT template
- Workers don't load orchestrator skill (ORCH_WORKER=1), only see spawn context

**2026-01-16 (implementation):** Added no-push guidance
- Added prominent "NEVER run git push" guidance to all 3 SESSION COMPLETE PROTOCOL locations
- Includes rationale about production deploy risks
- Wrote test TestGenerateContext_NoPushGuidance to verify

**2026-01-16 (completion):** Investigation completed
- Status: Complete
- Key outcome: Added explicit no-push guidance to worker spawn context template, verified with tests
