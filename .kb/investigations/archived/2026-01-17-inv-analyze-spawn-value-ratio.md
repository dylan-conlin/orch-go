<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The spawn-to-value ratio is 86.7% at completion level and 81.2% for artifact production, contradicting the "spawn 50, get 10" hypothesis; however, abandonment jumped from 4.4% to 21.0% in January 2026 (4.8x increase), signaling a system health concern.

**Evidence:** Analyzed 2,150 spawns via events.jsonl (86.7% completion rate), 947 investigation files (81.2% have synthesis), 2,098 beads issues (98.1% closed), and 2,187 git commits (22.3% reference beads); temporal analysis shows Dec 2025 had 4.4% abandonment vs Jan 2026 at 21.0%.

**Knowledge:** "Lasting value" has three levels - completion (86.7%), artifact production (81.2%), and code changes (23.7%); most value is knowledge work rather than implementation; sharp abandonment increases indicate system degradation requiring investigation, not just higher difficulty.

**Next:** Investigate January abandonment spike (spawn context quality, model changes, or prompt degradation); add telemetry dashboard for real-time spawn health monitoring; improve daemon adoption from 30% through better triage labeling.

**Promote to Decision:** recommend-no (operational analysis informing system improvements, not architectural constraint)

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
**Phase:** Complete
**Next Step:** Close investigation and create follow-up issue for January abandonment spike investigation
**Status:** Complete

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

- ✅ Completion rate of 86.7% verified via events.jsonl analysis (jq queries on 2,150 spawn events and 1,865 completion events)
- ✅ Artifact production rate of 81.2% verified via filesystem search (grep -l "## Synthesis" across 947 investigation files)
- ✅ Beads closure rate of 98.1% verified via bd stats command (2,059 closed / 2,098 total)
- ✅ Git commit linkage of 22.3% verified via git log analysis (488 commits with "orch-go-" pattern / 2,187 total since Dec 1)
- ✅ Abandonment increase verified via temporal jq analysis (53 Dec abandonments / 1,194 Dec spawns vs 201 Jan abandonments / 956 Jan spawns)

**What's untested:**

- ⚠️ Root cause of January abandonment spike (hypothesized as context quality, model changes, or prompt degradation - not verified)
- ⚠️ Whether archived investigations (11.8%) correlate with abandoned agents (causation not tested, only rates match)
- ⚠️ Impact of daemon vs manual spawning on completion rate (not stratified by spawn source)
- ⚠️ Definition of "lasting value" beyond quantitative metrics (qualitative assessment not performed)
- ⚠️ Whether the 23.7% code commit ratio is appropriate for this system's purpose (no baseline comparison)

**What would change this:**

- January abandonment hypothesis would be falsified if analysis of abandoned agent logs shows different failure patterns (e.g., all timeout-related)
- Archive-abandonment correlation would be disproven if timestamps show archives precede abandonment tracking (Dec 20 start date)
- Daemon spawn performance would be proven different if stratified analysis shows significant completion rate variance by spawn source
- "Lasting value" definition would shift if orchestrator feedback indicates knowledge artifacts have low reuse rate
- Code commit ratio would be concerning if similar systems show 50%+ commit rates (need external benchmarks)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Abandonment Spike Investigation First** - Create investigation to diagnose January abandonment spike before implementing systemic changes.

**Why this approach:**
- January abandonment jumped 4.8x (4.4% → 21.0%), indicating acute degradation not baseline inefficiency (Finding 5)
- High baseline completion rate (86.7%) proves system fundamentals work; recent change broke something specific
- Root cause analysis prevents implementing wrong solution (e.g., improving triage won't fix model degradation)

**Trade-offs accepted:**
- Defers broader system improvements (daemon adoption, telemetry dashboard) until root cause known
- Why acceptable: Fixing acute problem first prevents masking symptoms with dashboard improvements

**Implementation sequence:**
1. **Investigate abandonment spike** - Analyze abandoned agent logs from Jan 2026, compare to Dec 2025 baseline; identify common failure patterns (context gaps, model errors, timeout patterns)
2. **Fix root cause** - Implement targeted fix based on investigation (e.g., improve spawn context quality, adjust model selection, fix prompt degradation)
3. **Add telemetry dashboard** - Build real-time spawn health monitoring to catch future degradation early (abandonment rate, completion time trends, artifact production)
4. **Improve daemon adoption** - Focus on triage labeling discipline and skill inference quality to increase from 30% daemon spawns

### Alternative Approaches Considered

**Option B: Build telemetry dashboard first**
- **Pros:** Provides observability for all future work, catches degradation early
- **Cons:** Doesn't fix the acute January abandonment spike; dashboard shows problem but doesn't solve it (Finding 5)
- **When to use instead:** If abandonment spike investigation shows no actionable root cause (i.e., it's just harder problems)

**Option C: Focus on daemon adoption immediately**
- **Pros:** Could increase throughput from 30% to higher autonomous operation
- **Cons:** Doesn't address quality issue (abandonment spike); automating broken spawning process just produces more abandonments (Finding 7)
- **When to use instead:** After abandonment spike is resolved and spawn quality is stable

**Rationale for recommendation:** The 4.8x abandonment increase is an acute system health issue requiring immediate diagnosis, not a chronic optimization opportunity. The high baseline completion rate (86.7%) proves the system works; something specific broke in January. Fix that first, then optimize.

---

### Implementation Details

**What to implement first:**
- Abandonment log analysis tool: Parse abandoned agent workspace artifacts to extract failure patterns (context gaps, errors, blocks)
- Temporal comparison: Compare Dec 2025 vs Jan 2026 abandoned agent characteristics (skill type, spawn source, duration before abandonment)
- Context quality metrics: Measure SPAWN_CONTEXT.md completeness, kb context depth, prior knowledge availability for abandoned vs completed agents

**Things to watch out for:**
- ⚠️ Telemetry gaps: events.jsonl doesn't capture *why* agents were abandoned, only *that* they were (need workspace artifact analysis)
- ⚠️ Confounding variables: January might have harder problems, not worse spawns (need qualitative assessment of issue difficulty)
- ⚠️ Selection bias: Archived investigations (11.8%) might not correlate with abandonments if archiving happens for other reasons
- ⚠️ Definition drift: "Lasting value" means different things at different levels (completion, artifacts, commits); choose metric based on system purpose

**Areas needing further investigation:**
- Why does daemon adoption remain low at 30%? (triage labeling discipline, trust issues, skill inference quality)
- What's the reuse rate of knowledge artifacts (investigations, decisions)? Are 81.2% synthesis sections actually referenced later?
- Does spawn source (daemon vs manual) correlate with completion rate? Could inform automation strategy.
- What's the optimal code-to-knowledge ratio for this system? 23.7% commits seems low, but is that appropriate for orchestration-focused system?
- Model performance stratification: Do certain models have higher abandonment rates? (requires orch-go-x67lc telemetry data)

**Success criteria:**
- ✅ Abandonment rate returns to ≤5% baseline (Dec 2025 level)
- ✅ Root cause identified and validated (can reproduce abandonment pattern in controlled test)
- ✅ Telemetry dashboard shows real-time spawn health metrics (completion rate, abandonment rate, artifact production over time)
- ✅ Daemon adoption increases to ≥50% without increasing abandonment rate (proves triage quality improved)

---

## References

**Files Examined:**
- `~/.orch/events.jsonl` - Event telemetry log with 7,442 lines covering spawn/complete/abandon events
- `.beads/` database - Issue tracking data via bd stats command
- `.kb/investigations/` directory - 947 investigation files analyzed for completion and synthesis rates
- `.kb/investigations/archived/` directory - 112 archived investigations analyzed for archive patterns
- `.orch/workspace/` directory - 322 SYNTHESIS.md artifacts counted for full-tier completion verification
- Git history - 2,187 commits since Dec 1, 2025 analyzed for beads issue linkage

**Commands Run:**
```bash
# Count spawn events
jq -r '.type' ~/.orch/events.jsonl | grep "session.spawned" | wc -l
# Result: 2,150

# Count completion events
jq -r '.type' ~/.orch/events.jsonl | grep "agent.completed" | wc -l
# Result: 1,865

# Count abandonment events
jq -r '.type' ~/.orch/events.jsonl | grep "agent.abandoned" | wc -l
# Result: 254

# Event type frequency
jq -r '.type' ~/.orch/events.jsonl | sort | uniq -c | sort -rn

# Beads statistics
bd stats
# Result: 2,098 total, 2,059 closed, 36 open, 3 in progress

# Investigation file counts
find .kb/investigations -name "*.md" | wc -l
# Result: 947

# Investigations with synthesis
grep -l "## Synthesis" .kb/investigations/*.md | wc -l
# Result: 769

# Investigations with Status: Complete
grep -l "Status: Complete" .kb/investigations/*.md | wc -l
# Result: 601

# SYNTHESIS.md artifacts in workspaces
find .orch/workspace -name "SYNTHESIS.md" | wc -l
# Result: 322

# Git commits since Dec 1
git log --all --oneline --since="2025-12-01" | wc -l
# Result: 2,187

# Commits referencing beads
git log --all --grep="orch-go-" --oneline | wc -l
# Result: 488

# Archived investigations
find .kb/investigations/archived -name "*.md" -type f | wc -l
# Result: 112

# Daemon spawn events
jq -r '.type' ~/.orch/events.jsonl | grep "daemon.spawn" | wc -l
# Result: 646

# Manual (triage-bypassed) spawn events
jq -r '.type' ~/.orch/events.jsonl | grep "spawn.triage_bypassed" | wc -l
# Result: 586
```

**External Documentation:**
- N/A (internal telemetry analysis)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-17-inv-add-spawn-telemetry-model-performance.md` - Telemetry system design feeding this analysis
- **Issue:** `orch-go-x67lc` - Completed telemetry implementation providing events.jsonl data source
- **Issue:** `orch-go-4tven.4` - This investigation's tracking issue

---

## Investigation History

**2026-01-17 (earlier agent):** Investigation started
- Initial question: What's the ratio of spawns to actual lasting value?
- Context: Hypothesis that system might spawn 50 agents/week but only 10 produce lasting value, indicating wrong granularity, premature spawning, or model mismatch
- Previous agent collected findings 1-7 and synthesized key insights

**2026-01-17 (current agent):** Investigation synthesis phase
- Completed D.E.K.N. summary: Spawn-to-value ratio is 86.7% at completion, 81.2% for artifacts, contradicting "spawn 50, get 10" hypothesis
- Completed structured uncertainty section: Separated tested claims from untested hypotheses
- Completed implementation recommendations: Prioritized abandonment spike investigation over broader improvements
- Added detailed references and commands run

**2026-01-17 (current):** Investigation completed
- Status: Complete
- Key outcome: System shows healthy spawn-to-value ratios (86.7% completion, 81.2% artifacts) but January abandonment spike (4.4% → 21.0%) indicates acute system degradation requiring immediate investigation
