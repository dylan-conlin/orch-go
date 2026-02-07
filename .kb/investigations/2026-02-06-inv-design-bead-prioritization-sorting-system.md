## Summary (D.E.K.N.)

**Delta:** Sort logic should live in the orch daemon (not bd CLI), using named presets backed by multi-dimension pipelines in Go code, with graceful degradation for missing metadata.

**Evidence:** Codebase shows daemon already sorts by priority in `daemon_queue.go:49`, `bd ready` has basic `--sort` (hybrid/priority/oldest) but lacks cross-system dimensions (leverage, session context, staleness), and attention system already computes role-aware priority scores separately. The 5 named strategies require orch-level data (active sessions, leverage graph, kb locality) that bd CLI cannot access.

**Knowledge:** Sort strategies are a daemon concern because they integrate multiple data sources (beads, frontier/leverage, tmux sessions, kb areas). Beads should remain a simple sort provider for its own dimensions; orch layers compose cross-system intelligence on top.

**Next:** Implement sort strategies in `pkg/daemon/sort/` as named presets with a `SortFunc` interface, starting with Unblock Mode (leverages existing frontier calculations) and Flow State Mode (leverages existing area labels).

**Authority:** architectural - Cross-component (daemon + beads + frontier + attention), establishes new sort subsystem pattern, affects how work selection flows across the system.

---

# Investigation: Design Bead Prioritization Sorting System

**Question:** How should the bead prioritization sorting system be designed - where does sort logic live, how are sort pipelines represented, and how do we handle sparse metadata?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Decidability Graph model | extends | Yes - frontier.go implements leverage | No |
| Attention model decision (2026-02-03) | extends | Yes - confirmed single priority model | No |
| Daemon Autonomous Operation model | extends | Yes - daemon_queue.go sort confirmed | No |

---

## Findings

### Finding 1: Daemon already sorts by priority, bd CLI has basic sort support

**Evidence:** 
- `daemon_queue.go:49` sorts issues by `Priority` (lower = higher priority) using `sort.Slice`
- `daemon_crossproject.go:58` does the same for cross-project issues
- `bd ready --sort` supports three policies: `hybrid` (default), `priority`, `oldest`
- `ReadyArgs.SortPolicy` field exists in `pkg/beads/types.go:125` but orch-go never populates it

**Source:** `pkg/daemon/daemon_queue.go:49`, `pkg/beads/types.go:125`, `bd ready --help`

**Significance:** The current sort is single-dimension (priority only). The infrastructure for passing sort policies from orch to beads exists (`SortPolicy` field) but is unused. More sophisticated sorting requires dimensions that exist outside beads (leverage, session context, area locality).

---

### Finding 2: Sort dimensions span multiple data sources

**Evidence:** The 5 named strategies reference dimensions that require different systems:

| Dimension | Data Source | Currently Available |
|-----------|------------|-------------------|
| `priority` | beads | Yes - `issue.Priority` |
| `dependency_leverage` | frontier package | Yes - `frontier.CalculateFrontier()` computes transitive unblocking |
| `authority_level` | beads labels | Partially - `subtype:X`, `authority:X` labels exist |
| `context_locality` (area) | beads labels | ~33% populated (19/58 open issues have area labels) |
| `verification_cost` | Not available | No - would need estimation or historical data |
| `staleness×touch_count` | beads | Partial - `created_at`, `updated_at` available; touch_count not tracked |
| `active_session_area` | OpenCode/tmux | Yes - can query active sessions to determine current working area |

**Source:** `bd list --json`, `pkg/frontier/frontier.go`, `pkg/attention/types.go`, `pkg/beads/types.go`

**Significance:** No single system has all sort dimensions. Beads has issue metadata. Frontier has leverage. OpenCode has session context. This means sort logic must live in orch (which bridges all three), not in bd CLI.

---

### Finding 3: Attention system already does multi-source priority computation

**Evidence:** 
- `pkg/attention/` has 10+ collectors, each computing priority from different sources
- `AttentionItem.Priority` is the unified score consumed by the Work Graph
- Decision `2026-02-03-attention-model-single-priority.md` established: daemon uses `bd ready` independently, attention serves Dylan
- Attention collectors already handle: beads priorities, stuck agents, unblocked issues, stale issues, competing issues, duplicate candidates

**Source:** `pkg/attention/types.go`, `pkg/attention/beads.go`, `pkg/attention/unblocked_collector.go`

**Significance:** The attention system is the existing pattern for multi-source priority composition, but it serves the human (dashboard/Work Graph). The daemon needs its own sort layer that operates on the same data sources but with different semantics (batch selection vs attention routing). Per the 2026-02-03 decision, these should remain separate.

---

### Finding 4: Area label coverage is sparse but sufficient for opt-in sorting

**Evidence:**
- 58 open issues total
- ~19 have area labels (~33% coverage): area:cli (7), area:kb (3), area:skill (3), area:governance (2), area:beads (2), area:spawn (1), area:dashboard (1), area:daemon (1)
- Labels like `triage:ready` (3), `triage:review` (11), `effort:small` (3) also exist
- Issues without area labels can still be sorted by other dimensions (priority, staleness, leverage)

**Source:** `bd list --json --limit 0 --status open | jq labels`

**Significance:** Context locality (Flow State Mode) will work for labeled issues but must degrade gracefully. This argues for pipelines that skip unavailable dimensions rather than requiring all metadata present.

---

### Finding 5: Leverage calculation already exists and is performant

**Evidence:**
- `frontier.CalculateFrontier()` computes transitive leverage (what would unblock if completed)
- `calculateLeverage()` in `pkg/frontier/frontier.go:206` does BFS through dependency graph
- Frontier is already called by `/api/frontier` endpoint and `orch frontier` CLI
- Blocked issues sorted by `TotalLeverage` descending (line 84)

**Source:** `pkg/frontier/frontier.go:59-93`, `pkg/frontier/frontier.go:204-258`

**Significance:** The most complex sort dimension (dependency leverage) already has a working implementation. Unblock Mode can reuse `calculateLeverage()` directly. This significantly reduces implementation cost for the highest-value strategy.

---

## Synthesis

**Key Insights:**

1. **Sort logic belongs in orch daemon, not bd CLI** — The 5 named strategies require cross-system data (beads + frontier + sessions). bd CLI only has beads data. orch daemon already bridges all three systems and makes the spawn decision.

2. **Named presets are the right abstraction, not arbitrary composability** — The 5 strategies are coherent modes reflecting different operational contexts (unblock bottlenecks, maintain flow, fight fires, reduce debt, harvest decisions). Composing them (`--sort unblock,flow`) would create conflicting priorities (do I maximize unblocking or locality?). Named presets give a clear mental model.

3. **Pipelines should be code, not config** — Each strategy's pipeline integrates multiple Go packages (frontier, beads, attention). Encoding these in YAML/config would require inventing a DSL for "call frontier.CalculateFrontier(), map issues by leverage, then fallback to priority." Go code is more expressive and testable. Config selects which pipeline; code defines it.

4. **Graceful degradation is essential** — With 33% area label coverage and no verification_cost data, sort pipelines must skip unavailable dimensions. An issue without area labels should still be sortable by priority + leverage + staleness.

5. **Bootstrap is require-going-forward, not backfill** — Enforcing area labels on issue creation is lower friction than backfilling 58 issues. Requiring labels for new issues labeled `triage:ready` creates natural pressure to populate metadata (aligns with Pressure Over Compensation principle).

**Answer to Investigation Question:**

Sort logic should live in the **orch daemon** as **named presets** implemented as **Go sort functions** in `pkg/daemon/sort/`. Each strategy is a `SortFunc([]Issue) []Issue` that reads from multiple data sources (beads metadata, frontier leverage, active sessions). Strategies degrade gracefully when metadata is missing. The daemon's `NextIssueExcluding()` calls the active `SortFunc` instead of the current `sort.Slice(issues, func(i, j int) bool { return issues[i].Priority < issues[j].Priority })`. The active strategy is selected via daemon config or `orch daemon --sort-mode unblock`.

---

## Decision Forks

### Fork 1: Where does sort logic live?

**Options:**
- A: bd CLI (`bd ready --sort custom-strategy`)
- B: orch daemon (`pkg/daemon/sort/`)
- C: Shared library consumed by both

**Substrate says:**
- Principle: Compose Over Monolith — small focused tools that combine
- Principle: Share Patterns Not Tools — share schema, not implementation
- Model: Daemon Autonomous Operation — daemon already makes spawn selection
- Decision: Attention model single priority — daemon uses bd ready independently

**RECOMMENDATION:** Option B — orch daemon. Sort strategies require cross-system data (leverage from frontier, session context from OpenCode, area labels from beads). bd CLI has only beads data. The daemon already bridges all three systems and is the consumer of sort output.

**Trade-off accepted:** bd CLI's `--sort` flag won't gain new strategies. Acceptable because bd CLI serves human browsing, while daemon serves automated spawning — different audiences, different needs.

**When this would change:** If bd CLI gained plugin architecture for external data sources, sort could move there. Unlikely given Local-First principle.

---

### Fork 2: Named presets vs composable sort pipes

**Options:**
- A: Named presets only (`--sort-mode unblock`)
- B: Composable flags (`--sort leverage,locality,staleness`)
- C: Named presets with optional overrides (`--sort-mode unblock --boost locality`)

**Substrate says:**
- Principle: Progressive Disclosure — simple first, details available
- Model: Daemon — daemon config is YAML with simple fields
- Observation: The 5 strategies have conflicting optimization targets

**RECOMMENDATION:** Option A — Named presets only. The strategies optimize for different things (unblocking throughput vs context locality vs risk reduction). Composing conflicting optimizations produces incoherent orderings. Named presets provide a clear mental model: "the system is in unblock mode."

**Trade-off accepted:** Less flexibility than composable pipes. Acceptable because the 5 strategies cover the identified operational modes comprehensively. If a new mode is needed, add a new preset.

**When this would change:** If users frequently want "unblock mode but with locality boost" — then Option C makes sense. Observe need before building.

---

### Fork 3: How to represent sort pipelines

**Options:**
- A: Go code (functions in `pkg/daemon/sort/`)
- B: Config (YAML/JSON DSL for pipeline stages)
- C: Go code with config-selectable parameters

**Substrate says:**
- Principle: Local-First — files over databases, plain text over proprietary
- Principle: Infrastructure Over Instruction — code enforces, config suggests
- Observation: Pipelines call Go packages (frontier, beads client)

**RECOMMENDATION:** Option C — Go code with config-selectable parameters. Each strategy is a Go function (testable, type-safe, can call any package). Config selects which strategy and provides tuning knobs (e.g., staleness threshold). The Go function IS the pipeline; config parameterizes it.

**Trade-off accepted:** Adding a new strategy requires code change + rebuild. Acceptable because strategies change rarely (operational modes are stable) and Go code is far more expressive than any YAML DSL we'd invent.

**When this would change:** If non-Go tools need to define sort strategies (e.g., Python orchestrator). Current system is Go-only.

---

### Fork 4: Handling missing metadata gracefully

**Options:**
- A: Require all dimensions — fail/warn if metadata missing
- B: Skip missing dimensions — sort by available data only
- C: Use defaults for missing dimensions — assign neutral values

**Substrate says:**
- Principle: Graceful Degradation — core works without optional layers
- Observation: 33% area label coverage, 0% verification_cost data
- Principle: Pressure Over Compensation — let gaps create pressure to improve

**RECOMMENDATION:** Option B with elements of C — Skip missing dimensions, but assign neutral score (middle of range) so items with missing data sort between high and low scoring items rather than clustering at one end.

**Trade-off accepted:** Unlabeled issues get mediocre sort positions instead of optimal ones. This creates gentle pressure to add labels (Pressure Over Compensation) without blocking work.

**When this would change:** If label coverage reaches >80%, Option A (require) becomes viable for triage:ready issues.

---

### Fork 5: Integration with decidability graph

**Options:**
- A: Sort calls frontier.CalculateFrontier() directly
- B: Sort consumes pre-computed leverage from a cache
- C: Sort computes its own leverage independently

**Substrate says:**
- Principle: Compose Over Monolith — reuse existing, focused tools
- Model: Frontier already computes leverage correctly
- Observation: Frontier calls `bd show` for each open issue (expensive)

**RECOMMENDATION:** Option B — Pre-computed leverage from cache. `frontier.CalculateFrontier()` is expensive (N bd show calls for N open issues). The daemon poll loop already runs on 60s interval. Compute frontier once per poll, cache result, sort function reads from cache.

**Trade-off accepted:** Leverage data can be up to 60s stale. Acceptable for batch-mode daemon where 60s latency is already acceptable.

**When this would change:** If sort needs real-time leverage (e.g., interactive CLI). Then Option A is better despite cost.

---

### Fork 6: Bootstrap strategy

**Options:**
- A: Backfill existing issues with labels/metadata
- B: Require-going-forward for new issues
- C: Hybrid — auto-infer where possible, require for new

**Substrate says:**
- Principle: Pressure Over Compensation — don't compensate for gaps
- Observation: 58 open issues, many will close naturally
- Observation: Sort degrades gracefully (Fork 4)

**RECOMMENDATION:** Option B — Require-going-forward. Don't backfill 58 issues. Instead: 1) Sort works with available data now (priority + leverage). 2) New `triage:ready` issues should have area labels (soft requirement via daemon preview warning). 3) Over time, labeled population increases naturally.

**Trade-off accepted:** Historical issues sort sub-optimally. They close eventually; new issues have better data.

**When this would change:** If backlog grows to 200+ issues and area-based sorting becomes critical for triage.

---

## Structured Uncertainty

**What's tested:**

- ✅ bd ready supports `--sort` flag with hybrid/priority/oldest (verified: `bd ready --help`)
- ✅ Daemon sorts by priority in daemon_queue.go (verified: code inspection)
- ✅ Frontier calculates transitive leverage (verified: `pkg/frontier/frontier.go`)
- ✅ ~33% area label coverage on open issues (verified: `bd list --json`)
- ✅ ReadyArgs.SortPolicy field exists but is unused by orch (verified: types.go:125)

**What's untested:**

- ⚠️ Performance of caching frontier results in daemon poll loop (not benchmarked)
- ⚠️ Whether 5 strategies are the right set (based on investigation, not user feedback)
- ⚠️ Whether neutral-score for missing data produces good sort ordering in practice

**What would change this:**

- If frontier computation becomes too expensive for 60s polling, need event-driven leverage updates
- If users request sort composition (unblock + locality), named-presets-only would need extension
- If area labels never reach >50% coverage, locality-based strategies may need deprioritization

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Sort lives in orch daemon | architectural | Cross-component (daemon + frontier + beads), establishes new subsystem |
| Named presets | implementation | API design within daemon scope |
| Go code + config | implementation | Standard Go patterns |
| Graceful degradation | implementation | Within sort function scope |
| Frontier cache | architectural | Changes daemon poll loop structure |
| Require-going-forward | implementation | Label convention, no enforcement needed |

### Recommended Approach ⭐

**Daemon Sort Strategies via `pkg/daemon/sort/`** — Named sort presets as Go functions in a new package, selected by daemon config, consuming cached frontier data.

**Why this approach:**
- Reuses existing infrastructure (frontier leverage, beads metadata, session queries)
- Clear mental model for operators ("unblock mode" vs "flow state mode")
- Testable Go functions with well-defined inputs and outputs
- Graceful degradation for sparse metadata

**Trade-offs accepted:**
- No composability (presets only) — observe if needed
- 60s stale leverage data — acceptable for batch daemon
- New strategies require code change — strategies change rarely

**Implementation sequence:**

1. **Create `pkg/daemon/sort/` package** — Define `SortFunc` interface, implement `Priority` (current behavior) and `Unblock` strategies. Unblock reuses `frontier.calculateLeverage`.

2. **Add frontier cache to daemon** — Compute frontier once per poll cycle, store in `Daemon.cachedFrontier`. Sort functions read from cache.

3. **Wire into `NextIssueExcluding()`** — Replace `sort.Slice(issues, priority)` with `d.activeSort.Sort(issues)`. Config selects strategy via `sort_mode` field.

4. **Implement remaining 3 strategies** — Flow State, Firefight, Debt Reduction, Decision Harvest. Each builds on the package foundation.

5. **Add CLI flag** — `orch daemon --sort-mode unblock` and `orch daemon preview` shows sort mode.

### Strategy Specifications

**1. Unblock Mode** (maximize throughput by clearing bottlenecks)
- Primary: `dependency_leverage` (descending — highest leverage first)
- Secondary: `authority_level` (daemon-traversable before orchestrator/human)
- Tertiary: `priority` (tiebreaker)
- Use case: Overnight daemon batch runs, clearing blocked subgraphs

**2. Flow State Mode** (minimize context switching)
- Primary: `context_locality` (same area as last completed issue)
- Secondary: `priority` within area
- Tertiary: `staleness` (older issues first within same area)
- Use case: Focused sprints on one system area

**3. Firefight Mode** (triage by urgency and blast radius)
- Primary: `priority` (P0 > P1 > P2)
- Secondary: `dependency_leverage` (higher leverage = more impact)
- Tertiary: `issue_type` (bugs before features before tasks)
- Use case: Production incidents, critical bug bursts

**4. Debt Reduction Mode** (clean up accumulated work)
- Primary: `staleness × touch_count` (old untouched issues first)
- Secondary: `effort:small` label (quick wins first)
- Tertiary: `priority`
- Use case: Periodic cleanup sessions, reducing backlog debt

**5. Decision Harvest Mode** (resolve questions blocking the graph)
- Primary: `issue_type == question` first
- Secondary: `dependency_leverage` (which questions unblock the most)
- Tertiary: `subtype:factual` before `subtype:judgment` (daemon-resolvable first)
- Use case: When graph is question-blocked, daemon focuses on resolvable questions

### Interface Design

```go
// pkg/daemon/sort/sort.go

// Strategy defines a named sort strategy.
type Strategy interface {
    // Name returns the strategy identifier (e.g., "unblock", "flow-state")
    Name() string
    
    // Sort orders issues by this strategy's criteria.
    // ctx provides cached cross-system data (frontier, sessions).
    Sort(issues []Issue, ctx *SortContext) []Issue
}

// SortContext provides pre-computed data for sort strategies.
type SortContext struct {
    // FrontierState cached from last poll cycle
    Frontier *frontier.FrontierState
    
    // ActiveAreas derived from current agent sessions
    ActiveAreas map[string]int // area label → count of active agents
    
    // LastCompletedArea from most recent orch complete
    LastCompletedArea string
}
```

### Alternative Approaches Considered

**Option B: Sort in bd CLI via plugins**
- **Pros:** bd CLI already has `--sort`; keeps beads as single sort authority
- **Cons:** bd CLI lacks frontier data, session data, cross-system context. Would need to shell out to orch from bd — inverts the dependency.
- **When to use instead:** If beads gains plugin architecture with data source adapters

**Option C: Composable flag-based sorting**
- **Pros:** Maximum flexibility, users define their own sort priority chain
- **Cons:** Conflicting optimization targets produce incoherent results. High cognitive load ("what combination of 5 dimensions should I use?"). Named presets are easier to reason about.
- **When to use instead:** If the 5 named strategies prove insufficient after 3+ months of usage

**Rationale for recommendation:** Named presets in orch daemon provide the right level of abstraction — coherent operational modes that integrate cross-system data, without the complexity of composable sort DSLs or the limitation of single-source sorting.

---

### Implementation Details

**What to implement first:**
- `pkg/daemon/sort/` package with Strategy interface
- `Priority` strategy (current behavior, ensures no regression)
- `Unblock` strategy (highest value, reuses frontier)
- Frontier cache in daemon poll loop
- Config/CLI flag to select strategy

**Things to watch out for:**
- ⚠️ Frontier computation cost — cache per poll cycle, don't recompute per sort call
- ⚠️ Sort must not block daemon poll loop — set timeout on frontier computation
- ⚠️ `daemon_crossproject.go` has its own sort — must also use the strategy
- ⚠️ Test with empty/nil frontier (daemon may start before frontier data available)

**Areas needing further investigation:**
- Touch count tracking — beads doesn't track this today; needs bd feature or proxy via comment count
- Verification cost estimation — no data source exists; defer this dimension
- Active area detection — how to determine "area of last completed issue" reliably

**Success criteria:**
- ✅ `orch daemon preview` shows sort mode and explains ordering
- ✅ `orch daemon --sort-mode unblock` changes spawn order
- ✅ Issues with higher leverage spawn first in unblock mode (testable with mock)
- ✅ Missing area labels don't crash or produce degenerate sort ordering
- ✅ No performance regression in daemon poll loop

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This recommendation resolves recurring "daemon picks wrong next issue" friction
- Future sort strategy additions should reference this design

**Suggested blocks keywords:**
- "sort strategy"
- "daemon spawn order"
- "prioritization"
- "work selection"

---

## References

**Files Examined:**
- `pkg/daemon/daemon_queue.go` — Current daemon queue sort (line 49)
- `pkg/daemon/daemon_crossproject.go` — Cross-project sort (line 58)
- `pkg/beads/types.go` — ReadyArgs.SortPolicy field (line 125)
- `pkg/frontier/frontier.go` — Leverage calculation
- `pkg/attention/types.go` — AttentionItem structure
- `.kb/models/decidability-graph.md` — Decidability model
- `.kb/models/daemon-autonomous-operation.md` — Daemon model
- `.kb/decisions/2026-02-03-attention-model-single-priority.md` — Attention priority decision

**Commands Run:**
```bash
# Check bd ready sort capabilities
bd ready --help

# Check area label coverage
bd list --json --limit 0 --status open | jq -r '.[].labels // [] | .[] | select(startswith("area:"))' | sort | uniq -c

# Count open issues
bd list --status open --limit 0 --json | jq 'length'

# Check available issue fields
bd show orch-go-21413 --json | jq '.[0] | keys'
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-03-attention-model-single-priority.md` — Daemon/attention separation
- **Model:** `.kb/models/decidability-graph.md` — Leverage and authority concepts
- **Model:** `.kb/models/daemon-autonomous-operation.md` — Daemon poll-spawn-complete cycle

---

## Investigation History

**2026-02-06 22:45:** Investigation started
- Initial question: How should bead prioritization sorting be designed?
- Context: Investigation found 5 named sort strategies needing design decisions

**2026-02-06 23:00:** Exploration complete - 6 forks identified
- All forks navigable with available substrate (principles, models, decisions)
- Key finding: sort logic must live in orch daemon due to cross-system data needs

**2026-02-06 23:15:** Investigation completed
- Status: Complete
- Key outcome: Named presets in pkg/daemon/sort/, starting with Unblock and Flow State modes
