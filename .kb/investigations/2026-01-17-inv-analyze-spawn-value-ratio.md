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

# Investigation: Analyze Spawn Value Ratio

**Question:** What percentage of spawned agents produce lasting value, and what does this ratio tell us about system health?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-feat-analyze-spawn-value-17jan-2be7
**Phase:** Synthesizing
**Next Step:** Complete analysis and produce recommendations
**Status:** In Progress

---

## Findings

### Finding 1: High Completion Rate (86.7%)

**Evidence:**
- Total spawns tracked: 2,150 (via `jq -r '.type' ~/.orch/events.jsonl | grep "session.spawned" | wc -l`)
- Completed agents: 1,865 (via `jq -r '.type' ~/.orch/events.jsonl | grep "agent.completed" | wc -l`)
- Abandoned agents: 254 (via `jq -r '.type' ~/.orch/events.jsonl | grep "agent.abandoned" | wc -l`)
- Completion rate: 1,865 / 2,150 = 86.7%
- Abandonment rate: 254 / 2,150 = 11.8%

**Source:**
- `~/.orch/events.jsonl` - Event telemetry log
- Commands: `jq -r '.type' ~/.orch/events.jsonl | sort | uniq -c | sort -rn`

**Significance:**
86.7% completion rate suggests the spawn granularity is appropriate - agents are scoped such that they can usually complete their work. The 11.8% abandonment rate is reasonable friction indicating agents do get stuck occasionally but it's not systemic. This counters the "spawn 50, get 10" hypothesis - we're getting value from most spawns at the completion level.

---

### Finding 2: Beads Issue Closure Rate is 98.1%

**Evidence:**
- Total beads issues: 2,098 (via `bd stats`)
- Closed issues: 2,059 (via `bd stats`)
- Open issues: 36
- In Progress: 3
- Closure rate: 2,059 / 2,098 = 98.1%

**Source:**
- `.beads/` database
- Command: `bd stats`

**Significance:**
98.1% closure rate indicates that once issues are created, they almost always get resolved. Combined with the high completion rate, this suggests the system is effective at converting spawned work into closed issues. However, this doesn't tell us if the closed issues produced lasting value (commits, knowledge artifacts) or were just closed administratively.

---

### Finding 3: Artifact Production is Strong

**Evidence:**
- Total investigation files: 947 (via `find .kb/investigations -name "*.md" | wc -l`)
- Investigations with "Status: Complete": 601 (via `grep -l "Status: Complete" .kb/investigations/*.md | wc -l`)
- Investigations with synthesis sections: 769 (via `grep -l "## Synthesis" .kb/investigations/*.md | wc -l`)
- SYNTHESIS.md artifacts in workspaces: 322 (via `find .orch/workspace -name "SYNTHESIS.md" | wc -l`)
- Completion rate for investigations: 601 / 947 = 63.5%
- Synthesis rate: 769 / 947 = 81.2%

**Source:**
- `.kb/investigations/` directory
- `.orch/workspace/` directory
- Commands: `find .kb/investigations -name "*.md" -type f`, `find .orch/workspace -name "SYNTHESIS.md"`

**Significance:**
81.2% of investigations have synthesis sections, indicating agents are externalizing their findings, not just completing and abandoning. The 322 SYNTHESIS.md files show that full-tier spawns are producing the required artifacts. However, 63.5% completion rate for investigations is lower than the 86.7% agent completion rate, suggesting some investigations are started but not finished (possibly abandoned).

---

### Finding 4: Git Commit Linkage Shows Real Value

**Evidence:**
- Total commits since Dec 1, 2025: 2,187 (via `git log --all --oneline --since="2025-12-01" | wc -l`)
- Commits referencing beads issues (orch-go-*): 488 (via `git log --all --grep="orch-go-" --oneline | wc -l`)
- Beads-linked commit ratio: 488 / 2,187 = 22.3%

**Source:**
- Git history
- Commands: `git log --all --oneline --since="2025-12-01"`, `git log --all --grep="orch-go-" --oneline`

**Significance:**
22.3% of commits reference beads issues, indicating that roughly 1 in 5 commits are tied to tracked work. This is a proxy for "lasting value" - code changes that persist in the repository. Given 2,059 closed issues and 488 commits referencing beads, the ratio is 488 / 2,059 = 23.7% of closed issues have associated commits. This suggests ~76% of closed issues may be investigations, decisions, or administrative tasks without code changes.

---

### Finding 5: Abandonment Increased Sharply in January 2026

**Evidence:**
- December 2025 spawns: 1,194 (via temporal analysis of events.jsonl)
- December 2025 abandonments: 53
- January 2026 spawns: 956
- January 2026 abandonments: 201
- Dec abandonment rate: 53 / 1,194 = 4.4%
- Jan abandonment rate: 201 / 956 = 21.0%

**Source:**
- `~/.orch/events.jsonl`
- Commands: `jq -r 'select(.type == "session.spawned") | .timestamp' ~/.orch/events.jsonl | while read ts; do date -r "$ts" "+%Y-%m"; done | sort | uniq -c`

**Significance:**
Abandonment rate jumped from 4.4% to 21.0% in January, a 4.8x increase. This could indicate: (1) spawning agents with insufficient context, (2) harder problems being tackled, (3) model degradation, or (4) system changes that made agent work more difficult. The first abandonment event was Dec 20, 2025, suggesting abandonment tracking was added mid-December. This sharp increase is a red flag requiring investigation.

---

### Finding 6: Investigation Archive Rate is 11.8%

**Evidence:**
- Total investigations: 947
- Archived investigations: 112 (via `find .kb/investigations/archived -name "*.md" -type f | wc -l`)
- Active investigations: 835 (via `find .kb/investigations -name "*.md" -type f ! -path "*/archived/*" | wc -l`)
- Archive rate: 112 / 947 = 11.8%

**Source:**
- `.kb/investigations/` and `.kb/investigations/archived/` directories
- Commands: `find .kb/investigations/archived -name "*.md"`, `find .kb/investigations ! -path "*/archived/*"`

**Significance:**
Only 11.8% of investigations are archived, meaning 88.2% remain active/accessible. This suggests most investigations retain their value over time and are kept available for future reference. The archive rate matches the abandonment rate (11.8%), possibly indicating abandoned investigations get archived.

---

### Finding 7: Daemon Spawns are 30% of Total

**Evidence:**
- Total daemon spawn events: 646 (via `jq -r '.type' ~/.orch/events.jsonl | grep "daemon.spawn" | wc -l`)
- Triage-bypassed spawns (manual): 586 (via `jq -r '.type' ~/.orch/events.jsonl | grep "spawn.triage_bypassed" | wc -l`)
- Daemon spawn rate: 646 / 2,150 = 30.0%
- Manual spawn rate: 586 / 2,150 = 27.3%

**Source:**
- `~/.orch/events.jsonl`
- Commands: Event type frequency analysis

**Significance:**
Only 30% of spawns come from the daemon (autonomous operation), while 27.3% are explicitly manual (triage bypassed). The remaining ~43% are likely OpenCode interactive spawns or other mechanisms. This indicates the system is still heavily orchestrator-driven rather than autonomous. Higher daemon adoption could improve throughput but requires better triage labeling discipline.

---

## Synthesis

**Key Insights:**

1. **Completion is High, but Value Definition is Layered** - 86.7% of spawned agents complete (Finding 1), and 98.1% of beads issues close (Finding 2), but lasting value must be measured at multiple levels: (1) Completion - agent finishes work, (2) Artifact Production - knowledge externalized (81.2% have synthesis, Finding 3), (3) Code Changes - commits in git (22.3% of commits link to beads, Finding 4). The "spawn-to-value" ratio depends on which level you measure.

2. **System Health Appears Good with One Red Flag** - The system shows healthy signs: high completion rates (86.7%), strong artifact production (81.2% synthesis rate), and reasonable issue closure (98.1%). However, abandonment jumped from 4.4% to 21.0% in January 2026 (Finding 5), a 4.8x increase. This sharp change is a system smell requiring investigation - possible causes include context quality degradation, harder problems, or model changes.

3. **Most Value is Knowledge, Not Code** - Only 23.7% of closed issues (488 / 2,059) have associated git commits (Finding 4), indicating ~76% of completed work produces knowledge artifacts (investigations, decisions, models) rather than code changes. This is not necessarily bad - the orchestration system's value includes understanding and coordination, not just implementation. However, it suggests the system is heavily oriented toward exploration and learning rather than execution.

4. **Daemon Adoption is Low (30%)** - Only 30% of spawns come from the daemon (Finding 7), indicating the system is still heavily orchestrator-driven. Higher daemon adoption could improve throughput but requires: (1) better triage labeling discipline, (2) trust in autonomous spawning, (3) clearer issue typing for skill inference.

**Answer to Investigation Question:**

**The spawn-to-value ratio is 86.7% at completion level, but only ~24% produce code commits, while ~81% produce knowledge artifacts.** The "system smell" hypothesis (spawn 50, get 10) is **not supported** by the data - the system is effective at converting spawns to completions. However, the definition of "lasting value" is critical:

- **If value = completion:** 86.7% success rate (1,865 / 2,150)
- **If value = artifact with synthesis:** ~81% success rate (769 / 947 investigations have synthesis)
- **If value = code changes:** ~24% success rate (488 commits / 2,059 closed issues)

The system is healthy for knowledge work but shows lower execution-to-code ratios. The sharp abandonment increase in January (4.4% → 21.0%) is the primary concern requiring investigation. Possible causes: context gap issues, model changes, harder problems, or spawn prompt degradation.

**Limitations:**
- Telemetry for model performance (orch-go-x67lc) was just added Jan 17, so outcome tracking is incomplete
- "Lasting value" is subjective - some investigations inform decisions without producing commits
- Abandonment tracking started Dec 20, so pre-December data is unavailable
- Cannot distinguish between "investigation that found nothing" vs "abandoned investigation"

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
