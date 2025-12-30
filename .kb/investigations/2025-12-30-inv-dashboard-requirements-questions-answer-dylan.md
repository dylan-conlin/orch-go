<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dylan's 5 questions collapse to one: "What needs my attention?" - the dashboard should be reorganized around attention needs, not operational vs historical time.

**Evidence:** CLI capability analysis shows gaps in actionable visibility; current Ops/History modes share many components and solve wrong problem; attention hierarchy shows Dylan needs binary classification (needs attention vs swarm OK).

**Knowledge:** Dashboard is an attention router, not information portal. The Ops/History split should be killed. Attention items (pending reviews, errors, blocked-needs-action) should be consolidated into single prominent panel.

**Next:** Implement attention-first redesign: consolidate Attention Panel, kill mode toggle, demote informational sections (Ready Queue, event streams).

---

# Investigation: Dashboard Requirements - What Questions Should It Answer for Dylan?

**Question:** Given Dylan's asymmetric access (orchestrator has CLI, Dylan only has dashboard + emacs with friction), what 3-5 questions does Dylan most frequently need answered, and how should the dashboard answer them?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** og-inv-dashboard-requirements-questions-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

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

### Finding 4: Dylan's Workflow Patterns Reveal Key Questions

**Evidence:** Analyzing the orchestration workflow from CLAUDE.md and skill files:

**Dylan's Interaction Patterns:**
1. **Glance checks:** Quick looks at dashboard between deep work (coding, reading)
2. **Intervention points:** Decides when to engage orchestrator for action
3. **End-of-day review:** What happened today? What needs follow-up?
4. **Morning check:** What completed overnight? What's blocked?

**Key Questions Dylan Needs Answered:**

| Question | Frequency | Current Answer | Pain Point |
|----------|-----------|----------------|------------|
| **Q1: "What needs my attention right now?"** | Very High (every glance) | Scattered - errors, pending reviews, blocked issues in different places | No single "attention required" view |
| **Q2: "What's the swarm doing?"** | High (during active work) | Active Agents section works well | Good! But no quick "health" indicator |
| **Q3: "Did anything complete that I need to review?"** | High (morning + end-of-day) | Pending Reviews section exists | Works, but competing with other "attention" items |
| **Q4: "What went wrong?"** | Medium (when problems surface) | Errors indicator + Needs Attention | No way to see error details or patterns |
| **Q5: "What's next?"** | Medium (planning moments) | Ready Queue + Up Next | Exists but not prominently featured |

**Source:** Analysis of CLAUDE.md orchestration workflow, dashboard usage patterns, prior decisions about dashboard design

**Significance:** The core need is **attention management** - Dylan needs to know when to engage vs when to stay focused on other work. The dashboard's job is to minimize unnecessary interruptions while surfacing genuine needs.

### Finding 5: The Attention Hierarchy Pattern

**Evidence:** Examining what actually requires Dylan's attention vs orchestrator handling:

| Signal | Requires Dylan? | Why? |
|--------|-----------------|------|
| Agent completed with synthesis | **YES** | Dylan needs to review changes, especially UI |
| Agent error/failure | **YES** | May need to diagnose, decide retry strategy |
| Agent blocked on question | **YES** | Only Dylan can provide answer |
| Agent spawned | No | Orchestrator manages spawning |
| Agent progressing normally | No | Orchestrator monitors |
| Issue ready to work on | No | Daemon/orchestrator handles |
| Issue blocked by open issue | No | Workflow handles dependencies |
| Usage warning (>80%) | **YES** | Dylan decides account switch |
| Focus drift detected | Maybe | Dylan decides if intentional |

**Key Insight:** Dylan needs a **binary classification** at glance time:
1. **🔴 Needs Dylan:** Something requires human judgment/action
2. **🟢 Swarm OK:** Orchestrator handling everything, Dylan can focus elsewhere

**Source:** Analysis of prior decisions, workflow patterns, attention triggers

**Significance:** The current dashboard mixes "informational" and "actionable" items. The redesign should prioritize actionable items (attention required) and make informational items secondary (available on demand but not prominent).

---

## Synthesis

**Key Insights:**

1. **Dashboard is an Attention Router, Not an Information Portal** - Dylan's primary need is to know when to engage (Finding 4, 5). The dashboard should be binary: "attention needed" vs "swarm OK". Informational items (stats, history, SSE streams) are secondary and should not compete for attention.

2. **The Ops/History Split Solves the Wrong Problem** - The distinction should be "attention required vs. information available", not "current vs. historical" (Finding 2). A completed agent needing review is "attention required" even though it's historical. An active agent progressing normally doesn't need attention even though it's current.

3. **API Data Infrastructure is Sufficient** - The `/api/errors`, `/api/pending-reviews`, `/api/patterns`, `/api/reflect` endpoints already provide what's needed (Finding 3). The problem is UI presentation, not data availability.

4. **Five Questions Map to Two Primitives** - Dylan's questions (Finding 4) collapse to:
   - "What needs my attention?" → **Attention Panel** (errors, pending reviews, blocked-needs-action)
   - "What's the status?" → **Status Panel** (active agents, swarm health, stats)

**Answer to Investigation Question:**

**What 3-5 questions does Dylan most frequently need answered?**

1. **"What needs my attention right now?"** - The primary question, asked at every glance
2. **"What's the swarm doing?"** - Secondary, for situational awareness  
3. **"Did anything complete that I need to review?"** - Subset of Q1, but high frequency
4. **"What went wrong?"** - Subset of Q1, triggered by errors
5. **"What's ready for next?"** - Planning question, lower frequency

**How should the dashboard answer them?**

The dashboard should have a single **Attention Panel** that consolidates all items requiring Dylan's judgment. Everything else is informational and should be available but not prominent. The Ops/History distinction should be **killed** in favor of attention-based organization.

---

## Structured Uncertainty

**What's tested:**

- ✅ CLI capabilities catalog verified by reading cmd/orch/*.go source
- ✅ Dashboard sections analyzed by reading web/src/routes/+page.svelte
- ✅ API endpoints verified by reading cmd/orch/serve.go

**What's untested:**

- ⚠️ Whether attention-based organization actually improves Dylan's workflow (hypothesis)
- ⚠️ Whether the binary attention/info split is sufficient or needs more granularity
- ⚠️ Whether Dylan actually wants the historical archive at all (might be orchestrator-only)

**What would change this:**

- User feedback from Dylan after using attention-based prototype
- Discovery that Dylan frequently needs the historical archive for reference
- Discovery that attention items need priority levels (P0/P1/P2)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Attention-First Dashboard** - Reorganize around Dylan's attention needs, not operational vs historical time.

**Why this approach:**
- Directly addresses Finding 4: Dylan's primary question is "what needs my attention?"
- Aligns with Finding 5: Attention items need prominence, informational items are secondary
- Leverages Finding 3: API data already exists, just needs better UI presentation

**Trade-offs accepted:**
- Kills the Ops/History mode toggle that was recently implemented
- May need to re-educate orchestrator about new dashboard structure
- Loses the "full archive" view for exploratory browsing

**Implementation sequence:**
1. **Consolidate Attention Panel** - Merge Needs Attention, Pending Reviews, and blocked-needs-action into single "🔴 Attention Required" section
2. **Simplify main view** - Active Agents + Attention Panel only
3. **Make Info on-demand** - Ready Queue, history, SSE stream become collapsed/secondary

### Alternative Approaches Considered

**Option B: Keep Ops/History but Improve Ops Mode**
- **Pros:** Less disruptive, incremental improvement
- **Cons:** Doesn't solve the core problem (Finding 2) - the split is wrong abstraction
- **When to use instead:** If Dylan actually uses History mode frequently for reference

**Option C: Three-tier: Attention / Active / Archive**
- **Pros:** Clear separation of concerns
- **Cons:** More complexity, may over-engineer the solution
- **When to use instead:** If binary attention/info split proves insufficient

**Rationale for recommendation:** The Ops/History split was implemented before understanding the problem (per task context). The investigation reveals the real need is attention-based, not time-based.

---

### Section Verdicts (Keep/Kill/Merge)

| Section | Verdict | Reason |
|---------|---------|--------|
| **Stats Bar** | ✅ KEEP | Quick glance health indicator |
| **Mode Toggle (Ops/History)** | ❌ KILL | Wrong abstraction per Finding 2 |
| **Up Next** | ⚠️ MERGE → Attention Panel | Priority issues need attention |
| **Active Agents** | ✅ KEEP | Core status visibility |
| **Needs Attention** | ⚠️ MERGE → Attention Panel | Consolidate attention items |
| **Recent Wins** | ❌ KILL or DEMOTE | Informational, not actionable |
| **Ready Queue** | ⚠️ DEMOTE | Informational, orchestrator manages |
| **Pending Reviews** | ⚠️ MERGE → Attention Panel | Requires Dylan's attention |
| **SSE Stream** | ⚠️ DEMOTE | Developer tool, not operational |
| **Agent Lifecycle** | ⚠️ DEMOTE | Developer tool, not operational |
| **Filter Bar** | ⚠️ KEEP for search only | Archive browsing less important |

### Implementation Details

**What to implement first:**
- Create unified "Attention Required" section that consolidates:
  - Pending synthesis reviews (from /api/pending-reviews)
  - Agent errors (from /api/errors)
  - Blocked issues needing action (from /api/beads/blocked where needs_action=true)
  - Agents asking questions (BLOCKED status)
  - Usage warnings (>80%)
- Remove Mode Toggle
- Move Ready Queue, event panels to collapsed/secondary position

**Things to watch out for:**
- ⚠️ Constraint: "Dashboard must be fully usable at 666px width" - attention panel must fit
- ⚠️ Constraint: "Dylan doesn't interact with dashboard directly" - orchestrator uses Glass
- ⚠️ Prior decision: "24-hour threshold for Recent vs Archive" - may not apply if archive demoted

**Areas needing further investigation:**
- How to handle high-volume attention items (>10 at once)?
- Should attention items have priority levels (P0 = red, P1 = yellow)?
- Does Dylan ever use the archive for anything important?

**Success criteria:**
- ✅ Dylan can answer "do I need to engage?" with a single glance at top of dashboard
- ✅ Attention Panel is empty when swarm is healthy
- ✅ No repeated scanning of multiple sections needed for attention check

---

## References

**Files Examined:**
- `cmd/orch/main.go` - CLI command structure and spawn options
- `cmd/orch/serve.go` - API endpoints for dashboard
- `web/src/routes/+page.svelte` - Dashboard UI structure
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestration workflow patterns

**Commands Run:**
```bash
# CLI capabilities discovery
bd --help
kb --help
```

**External Documentation:**
- SPAWN_CONTEXT.md prior constraints - Dashboard design constraints

**Related Artifacts:**
- **Prior Decision:** "Dashboard progressive disclosure" - May conflict with attention-based redesign
- **Prior Decision:** "Ops/History split" - To be killed per this investigation
- **Prior Decision:** "24-hour threshold" - May not apply after redesign

---

## Investigation History

**2025-12-30 ~15:00:** Investigation started
- Initial question: What questions should the dashboard answer for Dylan?
- Context: 43 prior dashboard investigations were tactical, none strategic

**2025-12-30 ~15:30:** Key insight discovered
- Attention hierarchy pattern: Dylan needs binary attention/info split
- The Ops/History split solved the wrong problem

**2025-12-30 ~16:00:** Investigation completing
- Status: Complete
- Key outcome: Dashboard should be reorganized around attention needs, not time
