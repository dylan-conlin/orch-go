<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Backend selection is well-tested at unit level but intentionally lacks end-to-end spawning tests to prevent recursive incidents.

**Evidence:** 22 unit test cases cover backend selection priority chain; constraints prohibit end-to-end spawning tests; all existing tests pass.

**Knowledge:** Testing follows "unit tests and code review" approach rather than actual spawning; infrastructure warnings are tested to prevent agent deaths during server restarts.

**Next:** Close investigation - testing approach is documented and constraints are understood.

**Promote to Decision:** recommend-no (this documents existing testing approach, doesn't establish new pattern)

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

# Investigation: Test Backend Selection

**Question:** How does the orchestration system select backends/models for spawning agents, and what testing exists for this functionality?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Backend selection logic is well-tested with unit tests

**Evidence:** Found comprehensive unit tests in `backend_test.go` covering all priority levels: explicit flags, project config, global config, and defaults. All 22 test cases pass successfully.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/backend_test.go` - TestResolveBackend, TestValidateBackendModelCompatibility, TestResolveBackendPriorityChain

**Significance:** The core backend selection logic has good test coverage for the priority chain: 1) --backend flag, 2) --opus flag, 3) project config, 4) global config, 5) default opencode.

---

### Finding 2: Backend-model compatibility validation is tested

**Evidence:** Tests validate that opencode backend + opus model combinations produce warnings (known auth issues), while claude backend + opus is valid. Test cases cover various model strings.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/backend_test.go` - TestValidateBackendModelCompatibility function

**Significance:** Prevents users from accidentally using incompatible backend-model combinations that would fail at runtime.

---

### Finding 3: Infrastructure work detection and warnings are tested

**Evidence:** Tests verify that critical infrastructure work (opencode server, serve.go, pkg/opencode) triggers warnings when using opencode backend, suggesting --backend claude --tmux instead.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/backend_test.go` and `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd_test.go` - TestIsCriticalInfrastructureWork

**Significance:** Prevents agents from dying when OpenCode server restarts during infrastructure work by warning users to use claude backend.

---

## Synthesis

**Key Insights:**

1. **Backend selection is well-tested at unit level but not end-to-end** - The system has comprehensive unit tests for the backend selection logic, but intentionally avoids end-to-end spawning tests to prevent recursive spawn incidents.

2. **Testing follows a "unit tests and code review" approach** - Based on constraints discovered, worker agents test spawn functionality via unit tests and code review rather than actual spawning, preventing runaway iteration loops.

3. **Infrastructure warnings are tested** - The system tests that critical infrastructure work triggers appropriate warnings about backend selection to prevent agents from dying during server restarts.

**Answer to Investigation Question:**

The orchestration system selects backends/models through a clear priority chain (flags → project config → global config → default) that is well-tested with unit tests. However, there are no end-to-end tests that actually spawn agents with different backends - this is intentional to prevent recursive spawn testing incidents. The testing approach focuses on unit tests for selection logic and code review rather than actual spawning.

---

## Structured Uncertainty

**What's tested:**

- ✅ Backend selection priority chain (flags → config → defaults) - verified by 22 unit test cases
- ✅ Backend-model compatibility validation - verified by 6 test cases
- ✅ Infrastructure work detection and warnings - verified by 14 test cases
- ✅ Command building for spawning - verified by TestBuildSpawnCommand
- ✅ Model auto-selection logic - verified by 5 test cases

**What's untested:**

- ⚠️ Actual end-to-end spawning with different backends - intentionally avoided per constraint
- ⚠️ Real-world OpenCode server interactions with different models
- ⚠️ Claude CLI backend execution in tmux mode

**What would change this:**

- If end-to-end spawn tests were added (would violate current constraint)
- If actual backend execution tests showed different behavior than unit tests predict
- If infrastructure warnings didn't match real server restart behavior

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
