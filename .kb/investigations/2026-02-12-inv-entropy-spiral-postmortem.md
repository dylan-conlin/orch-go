<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Entropy spiral was caused by autonomous agents operating without human review gates, producing 1163 commits (5.4M LOC churn) with zero human intervention over 26 days.

**Evidence:** Git history shows 1162/1163 commits by "Test User" (agent identity), 5244 files created then deleted, zero human commits in period.

**Knowledge:** Missing circuit breakers: no human-in-the-loop review, no churn metrics, no LOC gates, no "stop and ask human" triggers.

**Next:** Implement mandatory human review gates, churn monitoring, and automatic halt on runaway metrics.

**Authority:** strategic - This involves irreversible architectural decisions about human oversight

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

# Investigation: Entropy Spiral Postmortem

**Question:** What patterns led to 1163 agent commits churning 5.4M LOC with zero human oversight, and what mitigations would prevent recurrence?

**Started:** 2026-02-12
**Updated:** 2026-02-12
**Owner:** Investigation Worker
**Phase:** Investigating
**Next Step:** Deep analysis of commit patterns
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Massive Scale - 1163 commits, 5.4M LOC churn, 26 days

**Evidence:**
- Date range: Jan 18 2026 to Feb 12 2026 (26 days)
- Total commits: 1163
- Authors: 1162 "Test User" (agent), 1 "Claude"
- Total churn: 3,568,921 lines added, 1,845,816 lines deleted = 5,414,737 LOC touched
- Net change: +1,723,643 lines (not -16K as initially stated)
- Files created: 15,432
- Files deleted: 5,957
- Files created then deleted: 5,244 (33% of created files were abandoned)

**Source:** 
```bash
git log --numstat 0bca3dec..entropy-spiral-feb2026
git shortlog -sn 0bca3dec..entropy-spiral-feb2026
```

**Significance:** Zero human commits in 26 days indicates complete automation with no review gates. The 5.4M LOC churn (vs 1.7M net) shows massive rework/reversal of work.

---

### Finding 2: Churned Artifact Categories

**Evidence:**
Top churn by category:
1. `.beads/issues.jsonl` - 66,118 LOC churn (issue tracking metadata)
2. Session transcripts (`session-ses_*.md`) - 20K-18K LOC each
3. Workspace ACTIVITY.json files - 10K-19K LOC each (agent activity logs)
4. `cmd/orch/spawn_cmd.go` - 9,880 LOC churn
5. `pkg/daemon/daemon.go` - 7,218 LOC churn

Churned files by directory:
- `.kb/investigations` - 407 files created then deleted
- `.kb/archive/investigations` - 185 files
- `cmd/orch` - 175 files created then deleted
- `.kb/decisions` - 79 files
- `pkg/daemon` - 46 files (more deleted than created: 77 created, 84 deleted)

**Source:** `git log --numstat` analysis

**Significance:** Agents were creating investigations, making decisions, then abandoning them. Core code (`cmd/orch`, `pkg/daemon`) had more files deleted than created - indicating work was being built then torn down.

---

### Finding 3: Commit Type Distribution Reveals Patterns

**Evidence:**
- 168 feat commits (new features)
- 161 fix commits (fixing breakage)
- 133 bd sync commits (automated issue tracking)
- 95 investigation commits
- 91 chore commits
- 60 architect commits
- 45 inv commits
- 32 refactor commits
- 29 test commits

Fix:Feat ratio = 0.96:1 (nearly equal fixes to features)

**Source:** `git log --format="%s" | sed 's/:.*//' | sort | uniq -c`

**Significance:** High fix:feat ratio suggests features were being introduced without adequate testing, requiring immediate fixes. The 133 `bd sync` commits (11% of total) indicate significant overhead from automated tracking.

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
