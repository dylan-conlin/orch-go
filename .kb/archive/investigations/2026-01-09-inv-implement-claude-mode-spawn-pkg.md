<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented Claude mode spawning in `pkg/spawn/claude.go` and integrated it into the `spawn` command.

**Evidence:** `SpawnClaude`, `MonitorClaude`, `SendClaude`, and `AbandonClaude` functions created and project compiles successfully (`go build ./...`).

**Knowledge:** Claude mode uses a file-based context approach (`claude --file SPAWN_CONTEXT.md`) and leverages tmux for process management.

**Next:** Close issue and mark complete.

**Promote to Decision:** recommend-no (tactical implementation of existing architecture)

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

# Investigation: Implement Claude Mode Spawn Pkg

**Question:** How to implement Claude mode spawning using tmux and the Claude CLI?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** opencode
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Claude Mode Spawn Architecture
**Evidence:** The task requires launching `claude --file SPAWN_CONTEXT.md` in a tmux window.
**Source:** `SPAWN_CONTEXT.md` and `SESSION_HANDOFF.md`.
**Significance:** Claude mode uses a file-based approach for initial prompt context, different from OpenCode's API-based approach.

### Finding 2: Tmux Integration
**Evidence:** `pkg/tmux` provides functions for window creation, sending keys, and capturing pane content.
**Source:** `pkg/tmux/tmux.go`.
**Significance:** `SpawnClaude`, `MonitorClaude`, `SendClaude`, and `AbandonClaude` can be implemented by wrapping these existing tmux utilities.

### Finding 3: Spawn Mode Configuration
**Evidence:** Project configuration now includes `spawn_mode: claude`.
**Source:** `pkg/config/config.go`.
**Significance:** The `spawn` command needs to detect this mode and route to the appropriate implementation.

---

## Synthesis

**Key Insights:**
1. **Dual Spawn Backend** - The system now supports two backends: OpenCode (HTTP API or tmux) and Claude (tmux + CLI).
2. **Tmux as Universal Interface** - Both backends can use tmux for visible, interactive sessions.
3. **File-based Context** - `claude --file` is a robust way to provide large context (SPAWN_CONTEXT.md) to the agent.

**Answer to Investigation Question:**
Claude mode spawning is implemented in `pkg/spawn/claude.go` using `tmux.CreateWindow` to launch a new window and `tmux.SendKeys` to run `claude --file SPAWN_CONTEXT.md`. Monitoring, sending messages, and abandoning agents are handled via tmux pane capture, send-keys, and kill-window respectively.

---

## Structured Uncertainty

**What's tested:**
- ✅ Compilation of `pkg/spawn/claude.go` (verified: `go build ./pkg/spawn/...` passed)
- ✅ Integration into `cmd/orch/spawn_cmd.go` (verified: `go build ./cmd/orch/...` passed)
- ✅ Project-wide compilation (verified: `go build ./...` passed)

**What's untested:**
- ⚠️ Actual execution of `claude --file` (not possible to test without `node` and `claude` CLI in this environment)
- ⚠️ End-to-end integration with `orch spawn` (requires live tmux and `claude` CLI)

---

## Implementation Recommendations

### Recommended Approach ⭐
**Tmux-wrapped Claude CLI** - Use `pkg/spawn/claude.go` to manage the lifecycle of Claude agents via tmux windows.

---


## Findings

### Finding 1: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

### Finding 2: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

### Finding 3: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

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
