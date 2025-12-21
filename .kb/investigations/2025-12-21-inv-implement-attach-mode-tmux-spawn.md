<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Confidence:** [Level] ([Percentage]) - [Key limitation in one phrase]

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Implement Attach Mode for Tmux Spawn

**Question:** How to implement an "attach" mode for `orch spawn --tmux` that automatically attaches the user's terminal to the newly created tmux window?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** opencode
**Phase:** Investigating
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Current Tmux Spawn Behavior
**Evidence:** `runSpawnTmux` in `cmd/orch/main.go` creates a new tmux window, sends the `opencode` command, waits for it to be ready, sends the prompt, and then calls `tmux select-window -t windowTarget`.

**Source:** `cmd/orch/main.go:837-922`

**Significance:** The window is already focused within the tmux session, but the user's terminal is not attached to that session/window if they are outside of it.

---

### Finding 2: Attaching to Tmux
**Evidence:** Standard tmux commands for attaching/switching are `attach-session` and `switch-client`.
- Outside tmux: `tmux attach-session -t target`
- Inside tmux: `tmux switch-client -t target`

**Source:** Tmux documentation and common usage.

**Significance:** We can use these commands to implement the "attach" behavior. We need to detect if we are already inside a tmux session using the `TMUX` environment variable.

---

### Finding 3: Successful Implementation and Validation
**Evidence:** Added `Attach` function to `pkg/tmux` and `--attach` flag to `orch spawn`. Running `./build/orch spawn --attach ...` successfully switched the current tmux window to the newly created one.

**Source:** Manual validation and unit tests.

**Significance:** The feature is fully implemented and verified.

---

## Synthesis

**Key Insights:**

1. **Attach vs Switch** - The implementation correctly distinguishes between being inside or outside of tmux to use the correct command (`switch-client` vs `attach-session`).

2. **Integration Point** - The attachment logic is integrated at the end of `runSpawnTmux`, ensuring the agent is fully spawned and the prompt sent before the user is attached.

**Answer to Investigation Question:**
Attach mode was implemented by adding a `pkg/tmux.Attach` function that uses `switch-client` or `attach-session` based on the `TMUX` environment variable. A new `--attach` flag was added to `orch spawn` which triggers this logic after a successful tmux spawn.

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add --attach flag and use tmux attach-session/switch-client** - This approach was chosen and implemented.

**Why this approach:**
- It provides a seamless experience for users who want to immediately interact with the spawned agent.
- It handles both inside-tmux and outside-tmux scenarios.

**Implementation sequence:**
1. Added `Attach(windowTarget string) error` to `pkg/tmux/tmux.go`.
2. Added `spawnAttach` flag to `cmd/orch/main.go`.
3. Updated `runSpawnWithSkill` and `runSpawnTmux` to handle the new flag.
4. Called `tmux.Attach(windowTarget)` at the end of `runSpawnTmux`.

---

## Summary (D.E.K.N.)

**Delta:** Implemented `--attach` mode for `orch spawn` to automatically attach/switch to the new tmux window.

**Evidence:** Manual validation showed successful window switching; unit tests verified command construction.

**Knowledge:** Tmux attachment requires different commands (`attach-session` vs `switch-client`) depending on whether the caller is already inside tmux.

**Next:** None - implementation complete.

**Confidence:** Very High (95%) - verified in real tmux environment.



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
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
