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

# Investigation: Workers Attempting Restart Orch Servers

**Question:** Why are workers attempting to restart orch servers via tmux, and do they need awareness of launchd-managed services?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Investigation agent
**Phase:** Investigating
**Next Step:** Clarify whether the symptom is about `orch serve` (dashboard) or `orch servers` (project dev servers)
**Status:** In Progress

---

## Findings

### Finding 1: SPAWN_CONTEXT.md includes `orch servers start/stop` instructions

**Evidence:** The feature-impl skill's phase-validation.md and phase-implementation-tdd.md files both reference:
```bash
orch servers stop <project>
orch servers start <project>
```

This instruction appears in SPAWN_CONTEXT.md via `GenerateServerContext()` which is called for UI-focused skills (feature-impl, systematic-debugging, reliability-testing).

**Source:** 
- `pkg/spawn/context.go:939-941` - Generates the instructions
- `pkg/spawn/config.go:52-56` - `SkillIncludesServers` mapping
- `~/.claude/skills/worker/feature-impl/reference/phase-validation.md:38-39`
- `~/.claude/skills/worker/feature-impl/reference/phase-implementation-tdd.md:113-114`

**Significance:** These are **project dev servers** (web frontends, APIs), NOT `orch serve` (the dashboard) or the daemon. This is the intended behavior for workers doing UI work - they need to restart their project's dev server after making changes.

---

### Finding 2: `orch servers start/stop` uses tmuxinator (legacy approach)

**Evidence:** The `orch servers start` command (servers.go:231-259) runs:
```go
cmd := exec.Command("tmuxinator", "start", sessionName)
```

This is documented as "Start servers via tmuxinator" and creates tmux sessions named `workers-{project}`.

**Source:**
- `cmd/orch/servers.go:54-65` - Command definition
- `cmd/orch/servers.go:231-259` - `runServersStart()` implementation
- Orchestrator skill mentions launchd as "preferred" but `orch servers` still uses tmuxinator

**Significance:** The orchestrator skill mentions launchd-based server management (`orch servers up/down`) as the preferred approach, but the actual implementation of `orch servers start/stop` still uses tmuxinator. This creates confusion about which approach workers should use.

---

### Finding 3: Two distinct server concepts being conflated

**Evidence:** The original investigation question mentions "orch servers" but there are actually two different things:
1. **`orch serve`** - The dashboard API server running on localhost:5188, managed via launchd
2. **`orch servers`** - Project development servers (web, api), managed via tmuxinator

Workers should NEVER need to restart `orch serve` (that's orchestrator infrastructure).
Workers MAY need to restart `orch servers` (their project's dev servers) when doing UI work.

**Source:**
- Orchestrator skill: "Dashboard at `http://localhost:5188` (`orch serve`) for real-time visibility"
- servers.go - Only manages project servers, not `orch serve`
- Daemon runs via launchd, not tmux

**Significance:** Need to clarify which "servers" are being referenced. If workers are trying to restart the dashboard or daemon, that's a problem. If they're just running `orch servers start/stop` for their project, that's expected behavior.

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
