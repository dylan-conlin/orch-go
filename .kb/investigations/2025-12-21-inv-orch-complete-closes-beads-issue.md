<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The runComplete function closes beads issues before updating registry status, with three silent failure modes that prevent registry updates.

**Evidence:** Code review of cmd/orch/main.go:1162-1174 shows registry.New(), reg.Find(), and reg.Complete() failures are silently ignored while beads issue is closed first (line 1155).

**Knowledge:** Registry update must happen BEFORE beads close to maintain consistency; silent failures prevent debugging and create state inconsistencies.

**Next:** Implement fix to update registry first with explicit error handling, add test to prevent regression.

**Confidence:** High (85%) - Code path is clear, but haven't tested all failure modes in practice.

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

# Investigation: orch complete registry status update

**Question:** Why does `orch complete` close beads issues but fail to update registry status to completed?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Agent og-debug-orch-complete-closes-21dec
**Phase:** Investigating
**Next Step:** Write test to reproduce, then implement fix
**Status:** In Progress
**Confidence:** High (85%)

---

## Findings

### Finding 1: Silent failure when registry creation fails

**Evidence:** In cmd/orch/main.go:1162-1164, the code creates a registry but silently continues if it fails:

```go
reg, err := registry.New("")
if err == nil {  // Only proceeds if NO error
    // ... registry update code
}
```

If `registry.New("")` returns an error, no error message is printed and the registry update is skipped entirely.

**Source:** cmd/orch/main.go:1162-1174

**Significance:** If the registry file is locked, corrupted, or has permission issues, `orch complete` will close the beads issue but fail to update the registry status. The agent remains in "active" state instead of being marked "completed".

---

### Finding 2: Silent failure when agent not found in registry

**Evidence:** In cmd/orch/main.go:1165-1166, if the agent isn't found, the code silently continues:

```go
agent := reg.Find(beadsID)
if agent != nil {  // Only proceeds if agent is found
    // ... complete and save
}
```

**Source:** cmd/orch/main.go:1165-1173

**Significance:** If the agent was never registered (e.g., registry was cleared, or spawn didn't register properly), the beads issue gets closed but the registry is never updated. No error is shown to the user.

---

### Finding 3: Silent failure when Complete() returns false

**Evidence:** In cmd/orch/main.go:1168-1172, the registry is only saved if Complete() returns true:

```go
if reg.Complete(agent.ID) {
    if err := reg.Save(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to save registry: %v\n", err)
    }
}
```

The Complete() method (registry.go:463-479) only returns true if the agent status is StateActive. If the agent is already completed, abandoned, or deleted, Complete() returns false and Save() is never called.

**Source:** cmd/orch/main.go:1168-1172, pkg/registry/registry.go:463-479

**Significance:** Edge case where running `orch complete` twice on the same agent would close the beads issue both times but fail silently the second time.

---

## Synthesis

**Key Insights:**

1. **The complete command has a critical ordering issue** - The beads issue is closed BEFORE the registry update is attempted, and multiple silent failures can prevent the registry update while still closing the issue. This creates an inconsistent state where beads shows the issue as closed but the registry still shows the agent as active.

2. **Error handling follows "warn but continue" pattern** - The code treats registry operations as optional/best-effort rather than critical. This is likely wrong since the registry is the source of truth for agent lifecycle state used by `orch clean`, `orch status`, and `orch review`.

3. **Silent failures prevent debugging** - Without error messages, users have no visibility into why their registry state is inconsistent. They only discover the problem when `orch clean` doesn't clean completed agents or `orch review` doesn't show them.

**Answer to Investigation Question:**

The runComplete function (cmd/orch/main.go:1091-1192) closes the beads issue first (line 1155), then attempts to update the registry (lines 1162-1174). Three silent failure modes prevent the registry update: (1) registry creation failure, (2) agent not found, (3) Complete() returning false. In all cases, the beads issue is already closed, creating an inconsistent state where beads shows "closed" but registry shows "active".

---

## Confidence Assessment

**Current Confidence:** [Level] ([Percentage])

**Why this level?**

[Explanation of why you chose this confidence level - what evidence supports it, what's strong vs uncertain]

**What's certain:**

- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]

**What's uncertain:**

- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]

**What would increase confidence to [next level]:**

- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]

**Confidence levels guide:**

- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Update registry BEFORE closing beads issue, with explicit error handling** - Change runComplete to mark agent as completed in registry first, only closing the beads issue if registry update succeeds.

**Why this approach:**

- Prevents inconsistent state by ensuring registry is updated before beads issue is closed
- Makes errors visible with explicit error messages instead of silent failures
- Maintains transactional integrity (if registry update fails, beads issue remains open as a signal)
- Matches the expected behavior shown in TestCompleteMarksForClean test

**Trade-offs accepted:**

- If beads close fails after registry update, we'll have opposite inconsistency (registry shows completed, beads shows open)
- Mitigated by: retrying beads close, or accepting that registry is source of truth for agent state

**Implementation sequence:**

1. Add explicit error handling for registry.New() - return error if registry can't be opened
2. Add explicit error for agent not found - return error if reg.Find() returns nil
3. Move registry update BEFORE beads close - ensures registry is updated first
4. Add warning if Complete() returns false - indicates agent already completed/abandoned
5. Only close beads issue after successful registry update and save

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

- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
