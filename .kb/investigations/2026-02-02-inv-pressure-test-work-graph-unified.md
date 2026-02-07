<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The "unified attention model" framing is directionally correct but conflates three distinct concerns (observability, actionability, authority) that should be separated in the architecture while unified in the UI.

**Evidence:** Reviewed design against existing systems (daemon guide, completion verification model, orchestrator lifecycle, Work Graph Phase 1-3 designs). Found the proposed signal list is incomplete (missing knowledge signals, session signals, cross-project), and the "unified" label masks real architectural separation needed.

**Knowledge:** The attention reconciliation layer should be decomposed into: (1) signal collectors with defined interfaces, (2) a normalizer producing typed AttentionItems, (3) role-aware priority scoring, (4) deduplication. Work Graph consumes the output; daemon can optionally subscribe. This maintains separation of concerns while achieving unified surface.

**Next:** Accept the reframed architecture; implement as Phase 4 of Work Graph with explicit layer separation. Create issues for signal collectors (git-history, session-state, kb-reflect).

**Authority:** architectural - Cross-boundary design affecting Work Graph, daemon, and new reconciliation layer with multiple valid approaches

---

# Investigation: Pressure Test Work Graph Unified Attention Model

**Question:** Is "unified attention model" the right abstraction for Work Graph? What's missing? How should the attention reconciliation layer work?

**Started:** 2026-02-02
**Updated:** 2026-02-02
**Owner:** Claude (architect spawn)
**Phase:** Synthesizing
**Next Step:** Await Dylan's feedback before finalizing
**Status:** In Progress

<!-- Lineage -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: The "unified" framing conflates three distinct concerns

**Evidence:** The design document mixes:
- **Observability** (what exists, what state things are in)
- **Actionability** (what can be acted on right now)
- **Authority** (who should act - human, orchestrator, daemon)

The proposed signal table shows this mixing:

| Signal | Actual Concern |
|--------|----------------|
| "Issue open, no workspace, has commits" | Observability + implicit authority (needs human verification) |
| "Issue blocked, blocker closed" | Observability → triggers actionability |
| "Agent stuck >2h" | Observability + implicit authority (needs intervention) |
| "Daemon paused" | Observability about automation state |

**Source:** Design doc lines 56-66 (signal table), compared with `.kb/guides/daemon.md` (daemon's concern is mechanical execution), `.kb/models/completion-verification.md` (verification is gate-based, not attention-based)

**Significance:** Conflating these concerns will create architectural confusion. The UI can be unified while the backend maintains separation. The attention reconciliation layer needs to produce **typed attention items** that explicitly carry their concern type.

---

### Finding 2: The signal list is incomplete - missing knowledge, session, and cross-project signals

**Evidence:** The design lists 7 attention signals. From reviewing existing systems, at least these are missing:

**Knowledge signals (from kb system):**
- Investigation clusters needing synthesis (kb reflect `synthesis` type)
- Stale decisions with low citations (kb reflect `stale` type)
- Open actions (kb reflect `open` type) - explicit Next: items
- Pending question subtypes (`subtype:judgment`, `subtype:framing`) needing escalation

**Session signals (from orchestrator lifecycle):**
- Orchestrator checkpoint threshold exceeded (2h/3h/4h)
- Orchestrator SYNTHESIS.md not produced but session idle
- Frame collapse indicators (Edit tool on code files in orchestrator session)

**Cross-project signals:**
- Issues in secondary projects ready but no daemon watching
- Cross-project dependencies (issue in repo A blocked by issue in repo B)
- Multi-repo hydration showing untracked work

**External signals (acknowledged in design but not enumerated):**
- CI failure on main branch
- PR awaiting review
- Deploy blocked/pending

**Source:** `.kb/guides/daemon.md:357-399` (kb reflect integration), `.kb/models/orchestrator-session-lifecycle.md:98-114` (checkpoint discipline), Design doc line 81 (acknowledges external signals without enumerating)

**Significance:** The signal enumeration needs to be comprehensive before building the reconciliation layer. Incomplete signals mean incomplete attention model.

---

### Finding 3: The daemon's role is execution, not attention - they're complementary

**Evidence:** The daemon already computes "what to spawn next" via:
- `bd ready` polling (observability)
- `triage:ready` filtering (authority marker)
- Skill inference from type (actionability)
- Capacity management (execution constraint)

The design asks "Does this subsume the daemon's job?" (line 129-136). The answer is **no** - they serve different roles:

| Component | Role | Computes |
|-----------|------|----------|
| Work Graph | Observability surface | "Here's what needs attention, by whom" |
| Daemon | Execution engine | "I will spawn this specific issue now" |

The relationship is: Work Graph surfaces "daemon would spawn X but capacity full" as an attention signal. Daemon consumes `triage:ready` label to know what to execute. They don't overlap - they compose.

**Source:** `.kb/guides/daemon.md:10-19` (daemon is for batch/overnight work), Design doc line 131-136 (subsumption question)

**Significance:** The framing should clarify that Work Graph is **observability infrastructure** and daemon is **execution infrastructure**. The attention model informs both humans and automation but doesn't replace either.

---

### Finding 4: The audience ambiguity affects priority design

**Evidence:** The design implicitly assumes a single priority hierarchy (line 100-106):
1. Failures
2. Stuck
3. Ready for action
4. Informational

But different roles need different prioritization:

| Role | Top Priority | Reason |
|------|-------------|--------|
| **Dylan (meta-orchestrator)** | Strategic blockers, pending decisions | Needs to unblock, make calls |
| **Spawned orchestrator** | Worker status, tactical completions | Needs to synthesize, spawn next |
| **Daemon** | triage:ready + capacity | Mechanical execution |

The proposed hierarchy is correct for Dylan but may not be for other consumers.

**Source:** `.kb/models/orchestrator-session-lifecycle.md:28-45` (three-tier hierarchy shows different concerns per level), Design doc line 98-106 (single priority model)

**Significance:** The attention reconciliation layer should support **role-aware priority scoring**, not a single global priority. The API could accept a `role` parameter to return appropriately prioritized items.

---

### Finding 5: The git commit analysis gap is real but has clear solution

**Evidence:** The design correctly identifies that commits mentioning issues aren't currently surfaced (line 88-89, 95-96). The triggering investigation proposes `pkg/reconcile/likely_done.go` as the solution.

The building blocks exist:
- `pkg/verify/git_diff.go:214-226` - already uses `git log --since`
- `pkg/verify/test_evidence.go:226-239` - already checks for commits in spawn timeframe

What's needed:
- Extend to run outside spawn context (no workspace required)
- Cache results with TTL to avoid repeated git queries
- Surface as `CommitEvidence` signal type

**Source:** `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md:144-161` (git log can find commits by message), `pkg/verify/git_diff.go` (existing patterns)

**Significance:** This is the highest-impact new signal to add. The pattern is well-understood; it's implementation work, not design work.

---

### Finding 6: View-only interaction is correct; action buttons increase coupling

**Evidence:** The design asks about interaction model (line 119-127). The original Work Graph design chose keyboard-first, vim-style navigation (`.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md:112-117`).

Action buttons would require:
- Bi-directional data flow (read + write)
- State management for pending actions
- Error handling for failed actions
- Permission checks for destructive actions

View-only with terminal actions provides:
- Clean separation (Work Graph reads, terminal writes)
- Keyboard shortcuts to spawn terminal commands
- No additional state management

**Source:** Design doc line 119-127 (interaction model options), Original Work Graph design line 164 (keyboard-first)

**Significance:** The recommended interaction model is **view-only with keyboard shortcuts to common commands**. E.g., pressing `c` on a "ready to complete" item could copy `orch complete <id>` to clipboard or open a terminal pane.

---

## Synthesis

**Key Insights:**

1. **Unified surface, separated backend** - The "unified attention model" framing is valuable for the UI but misleading for the architecture. The backend should have clearly separated concerns (collectors, normalizer, scorer) that compose into a unified API response. This enables independent evolution of signal sources.

2. **Role-aware priority is essential** - A single priority hierarchy serves Dylan's needs but not orchestrator or daemon needs. The reconciliation layer should compute priority per-role, with the default being human (Dylan) priority.

3. **Three classes of missing signals** - Knowledge signals (kb reflect), session signals (orchestrator lifecycle), and cross-project signals are all absent from the current design. These represent 40-50% of what Dylan actually looks at when deciding "what next."

4. **Daemon is complementary, not subsumed** - Work Graph provides the attention surface; daemon provides execution capacity. The relationship is informational (Work Graph → daemon visibility) not directional (Work Graph → daemon control).

5. **Git history is the key tactical win** - Among all the missing signals, commit-based "likely done" detection has the clearest ROI. It addresses the triggering problem directly and has a well-understood implementation path.

**Answer to Investigation Question:**

**Is "unified attention model" the right abstraction?**

Yes for the UI, no for the architecture. The abstraction should be:
- **Unified attention surface** (what the user sees)
- **Composable signal collectors** (how the backend works)

**What's missing?**

1. Knowledge signals (kb reflect types: synthesis, stale, open)
2. Session signals (checkpoint thresholds, frame collapse)
3. Cross-project signals (multi-repo dependencies, untracked work)
4. Role-aware priority model (Dylan vs orchestrator vs daemon)
5. Signal typing (observability vs actionability vs authority)

**How should the attention reconciliation layer work?**

See Recommended Approach below.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon doesn't compute attention - it executes on labels (verified: read daemon.go, skill_inference.go)
- ✅ Completion verification is gate-based, not attention-based (verified: read completion-verification.md model)
- ✅ Work Graph Phase 1-3 use view-only interaction (verified: read design docs)
- ✅ kb reflect produces typed signals (verified: read daemon.md reflect integration)

**What's untested:**

- ⚠️ Role-aware priority will meet all stakeholder needs (hypothesis, not validated)
- ⚠️ Signal collectors can be implemented independently (architecture claim, not proven)
- ⚠️ Git commit queries will perform at acceptable latency (not benchmarked)
- ⚠️ Cross-project attention is valuable enough to justify complexity (ROI unclear)

**What would change this:**

- If Dylan prefers a single priority model, role-aware scoring adds unnecessary complexity
- If signal collection proves too expensive to run on every request, batch/cache model needed
- If cross-project work is rare, that signal class can be deferred indefinitely

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Adopt decomposed architecture | architectural | Cross-boundary affecting Work Graph, daemon visibility, new pkg/attention |
| Implement git commit collector first | implementation | Within existing patterns, clear scope |
| Add role parameter to API | architectural | Changes API contract, affects multiple consumers |
| Defer cross-project signals | strategic | Resource/scope tradeoff, may not be needed |

### Recommended Approach ⭐

**Composable Signal Architecture** - Implement the attention reconciliation layer as discrete components that compose into a unified API response.

**Architecture:**

```
┌─────────────────────────────────────────────────────────────────┐
│                    pkg/attention/                                │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │ BeadsCollector│  │ GitCollector │  │ SessionCollector│        │
│  │ (issues,      │  │ (commits,    │  │ (orchestrator   │        │
│  │  comments,    │  │  likely_done)│  │  state, checkpts)│       │
│  │  deps)        │  │              │  │                 │        │
│  └──────┬───────┘  └──────┬───────┘  └──────┬─────────┘         │
│         │                 │                  │                   │
│         └─────────────────┼──────────────────┘                   │
│                           ▼                                      │
│                   ┌───────────────┐                              │
│                   │   Normalizer   │                              │
│                   │ → AttentionItem│                              │
│                   └───────┬───────┘                              │
│                           ▼                                      │
│                   ┌───────────────┐                              │
│                   │ PriorityScorer│                              │
│                   │ (role-aware)  │                              │
│                   └───────┬───────┘                              │
│                           ▼                                      │
│                   ┌───────────────┐                              │
│                   │  Deduplicator │                              │
│                   └───────┬───────┘                              │
│                           ▼                                      │
│                      API Response                                │
└─────────────────────────────────────────────────────────────────┘
```

**AttentionItem type:**

```go
type AttentionItem struct {
    ID           string          // Unique identifier
    Source       string          // "beads", "git", "session", "kb"
    Concern      ConcernType     // Observability, Actionability, Authority
    Signal       string          // Human-readable signal type
    Subject      string          // What needs attention (issue ID, session ID, etc.)
    Summary      string          // One-line description
    Priority     int             // Role-specific priority score
    Role         string          // "human", "orchestrator", "daemon"
    ActionHint   string          // Suggested action ("orch complete X", "review decision")
    CollectedAt  time.Time
    Metadata     map[string]any  // Signal-specific data
}

type ConcernType int
const (
    Observability ConcernType = iota  // State information
    Actionability                      // Can be acted on now
    Authority                          // Requires specific actor
)
```

**Why this approach:**
- Maintains separation of concerns (each collector is independent)
- Enables incremental implementation (add collectors one at a time)
- Supports role-aware priority without global coupling
- Provides clear extension points for future signals

**Trade-offs accepted:**
- More complex than monolithic "compute everything" approach
- Requires defining AttentionItem interface upfront
- May have redundant queries if collectors aren't carefully coordinated

**Implementation sequence:**
1. Define `AttentionItem` type and `Collector` interface in `pkg/attention/types.go`
2. Implement `BeadsCollector` (wrap existing graph API, add actionability signals)
3. Implement `GitCollector` (port from verification gap investigation)
4. Add `/api/attention` endpoint that composes collectors
5. Update Work Graph to consume new endpoint alongside existing graph endpoint
6. Add `SessionCollector` for orchestrator signals
7. Add `KBCollector` for knowledge signals (if kb-cli exposes API)

### Alternative Approaches Considered

**Option B: Monolithic Attention Computer**
- **Pros:** Simpler architecture, single point of query optimization
- **Cons:** Hard to extend, couples all signal sources, harder to test
- **When to use instead:** If signal sources are proven stable and won't evolve

**Option C: Event-Driven Attention Stream**
- **Pros:** Real-time updates, no polling, more reactive
- **Cons:** Significant infrastructure (message bus, subscriptions), overkill for current scale
- **When to use instead:** If latency becomes critical or signal volume is very high

**Rationale for recommendation:** Option A (composable collectors) provides the right balance of flexibility and simplicity. It matches the existing pattern (graph API is already composable) and enables incremental delivery.

---

### Implementation Details

**What to implement first:**
- `pkg/attention/types.go` - AttentionItem, ConcernType, Collector interface
- `pkg/attention/beads.go` - BeadsCollector wrapping existing graph API
- `pkg/attention/git.go` - GitCollector with commit-message search

**Things to watch out for:**
- ⚠️ Git queries can be slow on large repos - implement with TTL cache from start
- ⚠️ Deduplication is tricky when same issue appears from multiple collectors
- ⚠️ Role parameter needs sensible default (human) to avoid breaking existing consumers
- ⚠️ AttentionItem.Metadata should be typed per-signal, not generic map (for type safety)

**Areas needing further investigation:**
- How kb-cli exposes reflect data (API vs file read vs subprocess)
- Whether session registry has enough state for orchestrator signals
- Performance characteristics of git log on repos with 10k+ commits
- Whether daemon should consume attention API or stay with bd ready

**Success criteria:**
- ✅ `/api/attention` returns typed items with priority scores
- ✅ Work Graph shows "likely done" issues from git collector
- ✅ Different roles see different priority ordering (via role param or separate views)
- ✅ Latency < 500ms for combined attention query
- ✅ New collectors can be added without modifying existing code (interface compliance)

---

## Answers to the 6 Open Questions

### Q1: What are ALL the attention signals?

**Proposed complete list:**

| Category | Signal | Source | Concern |
|----------|--------|--------|---------|
| **Issue State** | Issue open, ready (no blockers) | beads | Actionability |
| | Issue blocked, blocker just closed | beads | Actionability |
| | Issue open, workspace exists, Phase: Complete | beads comments | Actionability |
| | Issue open, marked in_progress, no active agent | beads + registry | Authority |
| **Commit Evidence** | Issue open, commits reference it, no workspace | git | Observability → Actionability |
| | Investigation file modified recently, issue open | git | Observability |
| **Agent State** | Agent stuck >2h | registry + time | Authority |
| | Agent idle >30m | SSE + time | Observability |
| | Agent crashed (session ended without Phase: Complete) | registry | Authority |
| **Orchestrator State** | Checkpoint threshold exceeded (2h/3h/4h) | session registry | Authority |
| | SYNTHESIS.md not produced, session idle | workspace + session | Authority |
| | Frame collapse indicators | session + tool usage | Authority (meta) |
| **Knowledge State** | Investigation cluster ready for synthesis | kb reflect | Authority |
| | Stale decision (low citations) | kb reflect | Observability |
| | Open Next: action pending | kb reflect | Actionability |
| | Question needs escalation (subtype:framing) | beads | Authority |
| **External State** | CI failure on main | external API | Authority |
| | PR awaiting review | github API | Actionability |
| | Deploy pending/blocked | external API | Authority |
| **Automation State** | Daemon paused, ready issues exist | daemon status | Observability |
| | Daemon at capacity, more work queued | daemon status | Observability |

### Q2: Where do these signals live today?

| Signal Category | Current Source | API Exists? | Gap |
|-----------------|----------------|-------------|-----|
| Issue state | beads graph API | ✅ Yes | None |
| Commit evidence | git log | ❌ No | Need git collector |
| Agent state | registry + SSE | ✅ Yes | Need unified query |
| Orchestrator state | session registry | ⚠️ Partial | Need checkpoint query |
| Knowledge state | kb reflect CLI | ❌ No | Need kb API or file read |
| External state | github/CI APIs | ❌ No | Out of scope initially |
| Automation state | daemon status file | ✅ Yes | Need to surface |

### Q3: What's the attention priority model?

**Role-aware priority matrix:**

| Priority | Human (Dylan) | Orchestrator | Daemon |
|----------|---------------|--------------|--------|
| P0 | Failures, crashes, blocked epics | Worker failures, stuck agents | (not used) |
| P1 | Pending decisions, strategic questions | Ready to complete, synthesis needed | (not used) |
| P2 | Ready issues (P0/P1 business priority) | Ready to spawn | triage:ready |
| P3 | Knowledge maintenance, stale decisions | Informational | (not used) |
| P4 | Everything else | Everything else | (not used) |

**Should this be configurable?** Not initially. Start with hardcoded role-aware scoring. Add config if needed.

### Q4: How does this change the orchestrator's workflow?

**CLI commands remain for scripting; Work Graph is the human interface.**

| Current Command | Status After | Reason |
|-----------------|--------------|--------|
| `orch status` | Remains | Useful for scripts, quick terminal checks |
| `bd ready` | Remains | Daemon uses it, scripting |
| `orch review` | May deprecate | Attention surface replaces this |
| `orch complete` | Remains | Execution action, not observation |

**New workflow:**
1. Open Work Graph tab
2. See prioritized attention items
3. Use keyboard shortcut to copy command
4. Execute in terminal

### Q5: What's the interaction model?

**View-only with keyboard shortcuts.**

| Key | Action | Target |
|-----|--------|--------|
| `c` | Copy completion command | Ready-to-complete items |
| `s` | Copy spawn command | Ready-to-spawn items |
| `o` | Open issue in browser | Any issue |
| `enter` | Expand/collapse details | Any item |
| `j/k` | Navigate | List |

No action buttons. Clean separation: Work Graph reads, terminal writes.

### Q6: Does this subsume the daemon's job?

**No.** They're complementary:

| | Work Graph | Daemon |
|---|------------|--------|
| Role | Observability surface | Execution engine |
| Computes | "What needs attention" | "What to spawn now" |
| Audience | Human, orchestrator | Automation only |
| Authority | Informational | Executes on labels |

Work Graph **surfaces** daemon state as attention signal. Daemon **executes** on labels set by humans/orchestrators. They compose, don't conflict.

---

## What's Missing from the Design

1. **Decomposed architecture** - The design doesn't specify how signals compose
2. **Role-aware priority** - Assumes single priority hierarchy
3. **Knowledge signals** - No mention of kb reflect integration
4. **Session signals** - No mention of orchestrator lifecycle signals
5. **Cross-project signals** - Brief mention but no design
6. **Signal typing** - No distinction between observability/actionability/authority
7. **Collector interfaces** - No extension model for future signals
8. **Caching strategy** - Performance not addressed

---

## Proposed Attention Reconciliation Layer (Summary)

```
Signal Sources          Reconciliation Layer              Consumers
─────────────           ───────────────────               ─────────

  beads ──────┐
              │
  git ────────┼──→ Collectors ──→ Normalizer ──→ Scorer ──→ API ──→ Work Graph
              │         │                           │
  sessions ───┤         │                           │
              │         ▼                           ▼
  kb ─────────┤    AttentionItem               PriorityScore
              │    (typed struct)              (role-aware)
  daemon ─────┘

```

**Key design decisions:**
1. Collectors are independent (no coupling between signal sources)
2. Normalizer produces typed AttentionItems (not generic maps)
3. Scorer is role-parameterized (default: human)
4. API serves both Work Graph and future CLI consumption
5. Daemon continues using bd ready (doesn't consume attention API)

---

## References

**Files Examined:**
- `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - The design being pressure-tested
- `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md` - Triggering investigation
- `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md` - Original Work Graph design
- `.kb/models/completion-verification.md` - Three-layer verification model
- `.kb/models/orchestrator-session-lifecycle.md` - Orchestrator lifecycle, checkpoint discipline
- `.kb/guides/daemon.md` - Daemon architecture and kb reflect integration

**Commands Run:**
```bash
# Check open issues for context
bd list --status=open | head -20

# Create investigation file
kb create investigation pressure-test-work-graph-unified
```

**Related Artifacts:**
- **Design:** `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - Being reviewed
- **Investigation:** `.kb/investigations/2026-02-02-inv-work-graph-verification-gap-issues.md` - Triggered this design
- **Model:** `.kb/models/completion-verification.md` - Related verification architecture

---

## Investigation History

**2026-02-02 14:00:** Investigation started
- Initial question: Pressure-test the Work Graph Unified Attention Model design
- Context: Spawned as architect task to challenge framing and answer 6 open questions

**2026-02-02 14:30:** Context gathered
- Read design document, triggering investigation, Work Graph designs, daemon guide
- Read completion verification model, orchestrator lifecycle model
- Identified three key framing issues: conflation, missing signals, audience ambiguity

**2026-02-02 15:00:** Synthesis phase - awaiting Dylan feedback
- Status: In Progress (waiting for Dylan before marking Complete)
- Key finding: Unified surface is correct; architecture needs decomposition into collectors
