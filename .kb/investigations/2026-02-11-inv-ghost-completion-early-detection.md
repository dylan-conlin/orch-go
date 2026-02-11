<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Ghost completion detection needs three early warning points: orch phase command check, orch status display flag, and spawn context agent self-check instruction.

**Evidence:** Found commit counting logic in complete_gates.go:515, phase command in phase_cmd.go:66, status display in status_display.go, and spawn template in context_template.go:434.

**Knowledge:** The COMMIT_EVIDENCE gate catches ghost completion during orch complete, but we need earlier signals when agents report Phase: Complete.

**Next:** Implement three detection points with warnings (non-blocking early signals, blocking gate remains at orch complete).

**Authority:** implementation - Adding early detection to existing verification infrastructure, no new architectural patterns.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Ghost Completion Early Detection

**Question:** Where should ghost completion detection warnings be added to provide early signal before the final COMMIT_EVIDENCE gate?

**Started:** 2026-02-11
**Updated:** 2026-02-11
**Owner:** Claude (spawned agent)
**Phase:** Complete
**Next Step:** Implement findings
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: orch phase command writes phase but doesn't validate commits

**Evidence:** The `runPhaseWithDB` function in phase_cmd.go:96 writes phase directly to SQLite and optionally adds a bd comment, but has no commit validation logic.

**Source:** cmd/orch/phase_cmd.go:66-194

**Significance:** When an agent runs `orch phase <id> Complete`, the system records it without checking if any work was actually committed. This is the first point where we can add early detection.

---

### Finding 2: Existing commit counting logic in complete_gates.go

**Evidence:** The commit evidence gate uses `git rev-list --count <merge-base>..<branch>` to count commits ahead of baseline. This logic exists at cmd/orch/complete_gates.go:504-532.

**Source:** cmd/orch/complete_gates.go:515

**Significance:** We can reuse this exact logic in the orch phase command to check for commits when Phase: Complete is set. The pattern is already established and tested.

---

### Finding 3: orch status display shows agent phase but no commit warning

**Evidence:** The status display in status_display.go shows agent status and phase but has no logic to flag suspicious "Phase: Complete + 0 commits" combinations.

**Source:** cmd/orch/status_display.go:39-150

**Significance:** This is the second detection point - when viewing agent status, we should visually flag agents claiming completion without commits.

---

### Finding 4: Spawn context template includes completion protocol but no self-check

**Evidence:** The SpawnContextTemplate in context_template.go:10-426 includes session complete protocol and checklists, but doesn't instruct agents to verify they have commits before declaring Phase: Complete.

**Source:** pkg/spawn/context_template.go:208-226

**Significance:** This is the third detection point - we can add explicit instruction for agents to self-check for commits before reporting completion.

---

## Synthesis

**Key Insights:**

1. **Three-layer detection strategy** - Ghost completion can be caught at three points: (1) when agent reports it via orch phase, (2) when orchestrator views status, (3) before agent declares completion via spawn context instruction. Each layer provides progressively earlier signal.

2. **Reuse existing commit counting** - The commit counting logic from complete_gates.go can be extracted and reused. No need to invent new git logic.

3. **Warning vs blocking** - The early detection points should emit warnings, not block. The COMMIT_EVIDENCE gate at orch complete is the blocking safety net. Early warnings give agents and orchestrators a chance to catch the issue before final verification.

**Answer to Investigation Question:**

The three detection points should be:
1. **orch phase command** (finding 1 + 2): When setting "Phase: Complete", check worktree for commits and emit stderr warning if 0 commits found
2. **orch status display** (finding 3): Flag agents with Phase: Complete + 0 commits with a visual indicator (e.g., "⚠️ 0 commits" or similar)
3. **spawn context template** (finding 4): Add instruction in session complete protocol to self-check for commits before declaring Phase: Complete

All three are advisory/warning signals. The blocking gate remains at orch complete (COMMIT_EVIDENCE).

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

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Three-layer early warning system** - Add non-blocking warnings at three detection points before the final COMMIT_EVIDENCE gate.

**Why this approach:**
- Reuses existing commit counting logic from complete_gates.go (no new git logic)
- Non-blocking warnings preserve agent autonomy while providing visibility
- Multi-layer defense catches ghost completion at progressively earlier points
- Final blocking gate at orch complete remains the safety net

**Trade-offs accepted:**
- Warnings can be ignored by agents (acceptable - final gate still blocks)
- Requires git operations which may fail (acceptable - gracefully degrade to no warning)
- Some duplication of commit checking logic (acceptable - small code, high value)

**Implementation sequence:**
1. Add commit check helper function (reuse from complete_gates.go pattern)
2. Integrate into orch phase command when Phase: Complete is set
3. Add warning flag to orch status display for Phase: Complete + 0 commits
4. Update spawn context template with self-check instruction

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
