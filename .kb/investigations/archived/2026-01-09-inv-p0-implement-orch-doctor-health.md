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

# Investigation: P0 Implement Orch Doctor Health

**Question:** What functionality is missing from the existing `orch doctor` implementation to meet Phase 1 requirements from the Dashboard Reliability Architecture decision?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** Agent og-feat-p0-implement-orch-09jan-2a90
**Phase:** Investigating
**Next Step:** Implement missing Phase 1 features
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Existing doctor.go has significant functionality already implemented

**Evidence:** 
- OpenCode server health check (port 4096) - lines 218-241
- orch serve health check (port 3348 with HTTPS) - lines 243-309
- Beads daemon check - lines 311-336
- Binary staleness check - lines 427-469
- Stalled session detection - lines 510-606
- `--fix` flag for auto-starting services - lines 177-209
- `--sessions`, `--config`, `--docs` flags for advanced checks - lines 104-116

**Source:** cmd/orch/doctor.go:1-1180, cmd/orch/doctor_test.go:1-485

**Significance:** Core infrastructure is in place; only need to add missing Phase 1 features rather than building from scratch.

---

### Finding 2: Missing Phase 1 Requirements from Decision Document

**Evidence:**
Phase 1 requirements (.kb/decisions/2026-01-09-dashboard-reliability-architecture.md:166-171):
- ✗ `--watch` mode with desktop notifications - NOT IMPLEMENTED
- ✗ Check launchd services (com.orch-go.serve, com.orch-go.web, com.opencode.serve) - NOT CHECKED
- ✗ Port 5188 (web UI) - NOT CHECKED (only checks 4096 and 3348)
- ✗ Orphaned vite processes - NOT CHECKED (only checks bd daemon presence)
- ✗ Cache freshness (API response timestamps) - NOT IMPLEMENTED

**Source:** 
- cmd/orch/doctor.go:30-57 (doctorCmd definition)
- .kb/decisions/2026-01-09-dashboard-reliability-architecture.md:76-82

**Significance:** Need to implement 5 additional features to complete Phase 1.

---

### Finding 3: Desktop notification infrastructure already exists

**Evidence:**
- OpenCode completion service uses desktop notifications
- `opencode.NewCompletionService()` handles notifications in cmd/orch/main.go:154-171
- Notification sending exists in pkg/opencode/completion.go (inferred from usage)

**Source:** cmd/orch/main.go:154-171

**Significance:** Can reuse existing notification infrastructure for `--watch` mode rather than implementing from scratch.

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
