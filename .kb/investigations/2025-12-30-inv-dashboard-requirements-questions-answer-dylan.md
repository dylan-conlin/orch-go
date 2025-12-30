<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [To be filled at end]

**Evidence:** [To be filled at end]

**Knowledge:** [To be filled at end]

**Next:** [To be filled at end]

---

# Investigation: Dashboard Requirements - What Questions Should It Answer for Dylan?

**Question:** Given Dylan's asymmetric access (orchestrator has CLI, Dylan only has dashboard + emacs with friction), what 3-5 questions does Dylan most frequently need answered, and how should the dashboard answer them?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** og-inv-dashboard-requirements-questions-30dec
**Phase:** Investigating
**Next Step:** Catalog CLI capabilities, derive key questions
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A  
**Superseded-By:** N/A

---

## Context

**The Problem:** Dylan has asymmetric access to orchestration information:
- **Orchestrator (Claude):** Full CLI access - can run `orch status`, `bd ready`, `kb context`, etc.
- **Dylan:** Limited to dashboard (browser) + emacs (with context-switching friction)

The dashboard should be Dylan's CLI equivalent - surfacing the same rich info orchestrator has, curated and navigable.

**Prior Failed Attempts:**
- "Needs Attention" section - solution before understanding the problem
- "Recent Wins" section - solution before understanding the problem
- "Ops/History split" - two modes before knowing what to show in each
- 43 prior dashboard investigations exist but all tactical, none strategic

---

## Findings

### Finding 1: CLI Capabilities Dylan Can't Easily Access Today

**Evidence:** Catalog of CLI commands and their dashboard equivalence:

| CLI Command | What It Shows | Dashboard Status |
|-------------|---------------|------------------|
| `orch status` | Active agents, phase, runtime, processing state | ✅ Active Agents section |
| `orch status --all` | Include phantom agents | ⚠️ Partial (stale filter at 30min) |
| `bd ready` | Issues ready to work on (no blockers) | ✅ Ready Queue section |
| `bd list` | All issues with filtering | ❌ NOT AVAILABLE |
| `bd show <id>` | Full issue details, comments, dependencies | ❌ NOT AVAILABLE |
| `bd blocked` | Blocked issues with blocker details | ⚠️ Count only in stats |
| `bd stats` | Issue statistics | ✅ Stats bar |
| `kb context "<topic>"` | Prior knowledge on a topic | ❌ NOT AVAILABLE |
| `kb search` | Search investigations/decisions | ❌ NOT AVAILABLE |
| `orch complete <id>` | Verify and close agent | ❌ Requires CLI |
| `orch spawn` | Start new agent | ❌ Requires CLI |
| `orch review` | Batch pending completions | ⚠️ Pending Reviews section (partial) |
| `orch tail <id>` | Recent agent output | ❌ NOT AVAILABLE |
| `orch focus` | Set/check current focus | ⚠️ Focus indicator (read-only) |
| `orch usage` | Claude Max usage stats | ✅ Shown in tooltips |
| `orch daemon status` | Daemon health, capacity | ✅ Daemon indicator |
| `orch session status` | Current orchestrator session | ❌ NOT AVAILABLE |
| `kn decide/tried/constrain` | Quick knowledge capture | ❌ Requires CLI |

**Source:** Analysis of cmd/orch/*.go, web/src/routes/+page.svelte

**Significance:** Key gaps are:
1. No way to see full issue details (bd show)
2. No way to search/browse knowledge base (kb context)
3. No agent output visibility (orch tail)
4. All actions require CLI - dashboard is read-only

### Finding 2: Dashboard Currently Has Two Modes with Unclear Purpose

**Evidence:** From +page.svelte analysis:
- **Operational Mode:** Up Next, Active Agents, Needs Attention, Recent Wins, Ready Queue
- **Historical Mode:** Full archive with filters, SSE stream, event panels

The mode distinction exists but the boundaries are unclear:
- "Needs Attention" appears in Ops mode (consolidated errors, pending reviews, blocked)
- "Recent Wins" (completed in 24h) appears in Ops mode
- Ready Queue appears in BOTH modes
- Up Next appears in BOTH modes

**Source:** web/src/routes/+page.svelte lines 581-1031

**Significance:** The Ops/History split was implemented before understanding what questions each mode should answer. The modes share many components, suggesting the split may not be the right abstraction.

### Finding 3: Dashboard API Provides Rich Data But UI Underutilizes It

**Evidence:** The serve.go API endpoints provide:
- `/api/agents` - Full agent details including synthesis, tokens, gap_analysis, last_activity
- `/api/beads/ready` - Ready issues with priority, type, labels
- `/api/beads/blocked` - Blocked issues with blocker details, action needed
- `/api/errors` - Error pattern analysis (recurring patterns, recent errors)
- `/api/patterns` - Behavioral patterns (repeated failures)
- `/api/pending-reviews` - Synthesis recommendations to review
- `/api/reflect` - kb reflect suggestions (stale, promote, synthesis)

But the UI doesn't expose much of this:
- Synthesis detail requires clicking into agent detail panel
- Blocked issues only show count, not details
- Error patterns not surfaced prominently
- Patterns endpoint exists but not visible in main UI
- Reflect suggestions not visible

**Source:** cmd/orch/serve.go lines 239-330

**Significance:** The data infrastructure exists to answer Dylan's questions - the problem is presentation, not data availability.

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
