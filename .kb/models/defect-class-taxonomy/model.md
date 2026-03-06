# Model: Defect Class Taxonomy

**Domain:** Structural failure patterns in orch-go
**Last Updated:** 2026-03-03
**Synthesized From:** 459 fix commits (Dec 2025–Mar 2026), 60+ closed bugs, scope expansion investigation, three-code-paths probe, cross-model blind spots probe

---

## Summary (30 seconds)

Orch-go's bugs cluster into 7 named defect classes — structural patterns that recur across features and time. These aren't independent: they form a causal dependency graph where upstream classes (stale artifact accumulation, cross-project boundary bleed) create conditions that downstream classes (scope expansion, premature destruction) exploit. Naming them creates a shared vocabulary for architect reviews, spawn context, and automated checks. The taxonomy is a living artifact: classes may be resolved (instance rate drops to zero), merged (two classes turn out to be one), or discovered (new pattern emerges from commits).

---

## Quick Reference

| # | Class Name | Definition (one line) | Instances | Fix Pattern |
|---|------------|----------------------|-----------|-------------|
| 0 | Scope Expansion Without Assumption Validation | Scanner widens, consumer's implicit assumptions break | 8+ | Allowlist scanner pattern |
| 1 | Filter Amnesia | Filter exists in path A, missing in new path B | 15+ | Canonical filter functions |
| 2 | Multi-Backend Blindness | Code works for OpenCode OR Claude CLI, not both | 15+ | Backend-aware query interface |
| 3 | Stale Artifact Accumulation | Dead state never cleaned up, interferes with new features | 20+ | Lifecycle cleanup discipline |
| 4 | Cross-Project Boundary Bleed | Single-project code breaks in multi-project context | 20+ | Eliminate global state, thread projectDir |
| 5 | Contradictory Authority Signals | Multiple sources of truth disagree, fixes oscillate | 10+ | Single canonical derivation function |
| 6 | Duplicate Action Without Idempotency | Same action performed multiple times, no dedup | 12+ | System-level idempotency layer |
| 7 | Premature/Wrong-Target Destruction | Resource killed based on stale/incomplete info | 8+ | Liveness verification before destruction |

---

## Core Mechanism

### Dependency Graph

The 7 classes are not independent. Upstream classes create conditions that downstream classes exploit:

```
Stale Artifact Accumulation (3)
    ↓ creates data that
Scope Expansion (0) finds unexpectedly
    ↓ manifests via
Filter Amnesia (1) — missing exclusion in new consumer

Multi-Backend Blindness (2) — structurally similar to filter amnesia
    but at architecture level, not data level

Cross-Project Boundary Bleed (4)
    ↓ multiplies exposure for
Scope Expansion (0) — cross-project data is the highest-risk expansion vector

Contradictory Authority Signals (5)
    ↓ causes
Premature/Wrong-Target Destruction (7) — wrong status → wrong action

Duplicate Action (6) — independent, caused by missing idempotency
```

**Implication:** Fixing upstream classes reduces the surface area for downstream classes. Stale artifact cleanup (3) eliminates the data that scope expansion (0) finds. Single canonical status derivation (5) prevents the wrong-status-wrong-action chain that causes premature destruction (7).

### Class Definitions

#### Class 0: Scope Expansion Without Assumption Validation

A scanner/query expands its scope (wider search, cross-project, new data source). Downstream consumers have implicit assumptions about what the scanner returns. The expanded scope surfaces a data class the consumer doesn't expect. Consumer breaks.

**Representative instances:**
- Cross-project workspace scan found 19 untracked workspaces with synthetic IDs → verification counter inflated past threshold
- Hotspot bloat scanner walked build directories → false positives blocked spawns
- `bd list -l orch:agent` returned closed issues (label never removed) → active count inflated

**Structural prevention:** Allowlist scanner pattern — scanners declare what they return, consumers validate against the declaration. Three-layer defense (allowlist scanner, scan inventory, daemon self-check invariants).

**Relationship:** Filter amnesia (1) is a subclass. Stale artifact accumulation (3) creates the data that triggers this class.

---

#### Class 1: Filter Amnesia

A filter, guard, or exclusion exists in one code path. A new consumer of the same data is added without applying the same filter. The new consumer processes items the original consumer correctly excluded.

**Representative instances:**
- `isUntrackedBeadsID()` existed in active_count.go but not verification_tracker.go
- Open/in_progress filter in agent listing but not dashboard agent discovery
- Labels/LabelsAny in ListArgs struct silently ignored by CLIClient.List()
- Closed-issue filter in review but not /api/pending-reviews
- Time-window filter in CLI but /api/sessions defaulted to 12h returning empty

**Structural prevention:** Canonical filter functions shared by all consumers rather than each consumer reimplementing. Same allowlist scanner pattern as Class 0.

**Relationship:** Subclass of scope expansion (0). The "new consumer" is the expansion; the "missing filter" is the unvalidated assumption. Separated because the fix pattern differs — Class 0 is about scanner design, Class 1 is about consumer discipline.

---

#### Class 2: Multi-Backend Blindness

Code is written and tested against one spawn backend (OpenCode API or Claude CLI/tmux) but doesn't account for agents running on the other backend. The code works for its tested path and silently produces wrong results for the other.

**Representative instances:**
- Status detection checked SessionID before SpawnMode → Claude CLI agents always marked idle
- Daemon cleanup only checked OpenCode sessions → killed active Claude CLI workers on restart
- `DefaultActiveCount()` only queried OpenCode API → Claude CLI agents invisible to capacity checks
- Session dedup only checked OpenCode API → Claude CLI agents bypassed dedup, 10 duplicates in 20 min
- Agent metadata read from OpenCode session → Claude CLI agents use AGENT_MANIFEST.json, wrong source prioritized

**Structural prevention:** Backend-aware query interface: `AgentDiscovery.ListActive()` that dispatches to both backends and merges results. The `DiscoverLiveAgents()` function (formerly `CombinedActiveCount`) is the embryo of this pattern. Test requirement: any new agent query must have test cases for both backends.

**Historical context:** Claude CLI became the default backend on Feb 19, 2026 (Anthropic OAuth ban). Before that, most agents were OpenCode-only, so OpenCode-only code was correct. The transition exposed every OpenCode-only code path as a bug. This class is a **migration defect** — it will recur if a third backend is added unless the backend-aware interface is built.

---

#### Class 3: Stale Artifact Accumulation

State artifacts (workspaces, labels, sessions, PID files, cache entries, DB records) are created during normal operations but never cleaned up on lifecycle transitions. Over time, accumulated stale artifacts interact unpredictably with new features — inflating counts, slowing scans, causing false matches, or resurrecting dead state.

**Representative instances:**
- `orch:agent` label on closed issues → active agent count inflated with historical agents
- 1,314 archived workspaces in `.orch/workspace/` → 5 scan functions O(1438) instead of O(124)
- OpenCode sessions not deleted after completion → `orch status` showed completed agents as "running"
- Daemon PID file from dead process → status readers showed stale daemon info
- 49 issues stuck in in_progress → blocked respawns, inflated all counters

**Structural prevention:** Lifecycle hook discipline — every state transition (spawn → active → complete → archived) must clean up artifacts from the prior state. TTL-based expiry for transient artifacts (spawn cache, PID files, session metadata). `orch doctor` checks for accumulated artifacts.

**Why this matters beyond individual fixes:** Every scope expansion (0) is only dangerous because accumulated state contains unexpected entries. If lifecycle transitions reliably cleaned up artifacts, scanners would only find current state. This class is the **enabling condition** for scope expansion failures.

---

#### Class 4: Cross-Project Boundary Bleed

Code assumes single-project context (current working directory, beads database, git repo, .kb/ directory). When invoked for a different project (via daemon cross-project spawn, `--workdir` flag, or multi-project dashboard), it operates on the wrong project's data without error.

**Sub-patterns:**

**4a: Global State Corruption (beads.DefaultDir)** — The package-level variable controls which project's beads database is targeted. Setting it without defer-restore causes all subsequent beads operations to target the wrong project. 4 instances with identical root cause.

**4b: CWD Assumption** — Functions use `os.Getwd()` or process CWD instead of an explicit `projectDir` parameter. Functions like daemon completion comment fetching, kb context query, gap analysis quality scoring.

**4c: Missing Project Context in API/Shell-outs** — BEADS_DIR not injected for cross-repo phase reporting, deliverable path detection missing for cross-repo agents, work-graph 'unassigned' for cross-project issues.

**Structural prevention:** Eliminate `beads.DefaultDir` global — replace with explicit `projectDir` parameter on all beads client methods. Require `ProjectDir` field on all structs that cross project boundaries. Lint rule: flag any code that calls `beads.DefaultDir =` without `defer` on the next line.

---

#### Class 5: Contradictory Authority Signals

Multiple sources of truth exist for the same state (typically agent status). Different code paths read different signals and reach contradictory conclusions. Fixes oscillate between "signal A is authoritative" and "signal B is authoritative" without resolving which one wins.

**The canonical example — agent completion status** has 4+ signals:
1. **Phase: Complete** — beads comment from agent
2. **SYNTHESIS.md** — file existence in workspace
3. **OpenCode session state** — active/idle/completed
4. **Tmux window liveness** — process running or not

These signals disagree in real scenarios. The fix history oscillated:
- Dec 28: Phase Complete > session state
- Jan 6: Session state > Phase Complete
- Jan 6: Duplicate of the same fix
- Jan 8: Session state > artifacts

Each fix was correct for its triggering bug but created conditions for the next bug. **The oscillation is the defect class** — not any individual fix.

**Structural prevention:** Single canonical status derivation function used by all consumers (the `ListUnverifiedWork()` function is the embryo). Explicit precedence hierarchy documented and enforced. Status should be computed, not stored — derive from primary signals each time.

**Cost:** This is the most expensive class per-instance. 41 commits touch status derivation. It's the only class where reactive fixes are actively counterproductive.

---

#### Class 6: Duplicate Action Without Idempotency

An action (spawn, completion recording, event emission, issue creation) is performed multiple times because the dedup mechanism is absent, too narrow, or has a race window. The duplicate causes visible damage.

**The duplicate spawn saga — 7 separate fixes over 3 months:**
1. No dedup at all → add `SpawnedIssueTracker` with 5-min TTL
2. Race between daemon poll and beads status update → fresh beads status check
3. TTL expired while agent still running → session-level dedup via OpenCode query (6h)
4. OpenCode-only dedup — Claude CLI agents invisible → tmux window check
5. Orphan detector cleared spawn cache → remove Unmark() from orphan path
6. Different beads IDs, same title → content-aware dedup (title matching)
7. in_progress issues not filtered → skip in_progress in NextIssue()

Each fix patched one dedup gap, which revealed the next. Dedup is a cross-cutting concern that resists point fixes.

**Structural prevention:** Idempotency keys — every spawn gets a unique key; second spawn with same key is a no-op. Two-phase commit for daemon — mark issue as "spawn pending" atomically before spawning. Event dedup layer keyed on (event_type, entity_id, time_window).

---

#### Class 7: Premature/Wrong-Target Destruction

A resource (tmux window, OpenCode session, workspace) is destroyed based on stale or incomplete information, killing active work or destroying state needed for inspection.

**Representative instances:**
- Tmux window killed by index (unstable) → window indices shift, killed wrong window
- Tmux window killed via defer (all exit paths) → killed before orchestrator could inspect gate failures
- Active Claude CLI workers killed → daemon cleanup only checked OpenCode (multi-backend blindness)
- Active Claude Code agents killed → `orch clean --sessions` only checked OpenCode + beads
- Current session deleted → `orch clean --verify-opencode` deleted self

**Structural prevention:**
- **Liveness verification before destruction**: Check at least 2 independent signals (beads issue status + process liveness) before killing anything
- **Stable identifiers**: Use tmux window IDs (@-prefixed), not indices
- **Destruction as last step**: Never destroy in defer or early-return paths; destroy only on the explicit success path after all gates pass
- **Fail-safe on uncertainty**: If liveness check fails, preserve the resource

---

## Why This Fails

### New instances of known classes

The most common failure mode: a new feature introduces a new instance of an existing class because the developer didn't know the class existed.

**Example:** Adding a new agent status endpoint that only queries OpenCode (Class 2). The fix is awareness — this taxonomy in architect review context.

### Fixes that create new bugs (Class 5 specific)

Contradictory authority signal fixes are uniquely dangerous because each fix is correct for its trigger but wrong for the opposite scenario. Reactive, per-bug fixes in this class are actively counterproductive. Only a structural fix (single canonical derivation) breaks the cycle.

### Point fixes for cross-cutting concerns (Class 6 specific)

Dedup resists point fixes because each patch addresses one gap while the system finds new gaps. Seven fixes over 3 months, each addressing a different gap. The concern must be solved at the system level.

### Upstream class creating downstream exposure

Stale artifact accumulation (3) makes scope expansion (0) dangerous. Cross-project boundary bleed (4) multiplies scope expansion risk. Fixing a downstream class without addressing its upstream enabler produces temporary relief.

---

## Constraints

### Why can't we just add more filters everywhere?

**Constraint:** Filter amnesia (1) is a consumer-side problem. Adding filters to every consumer doesn't scale — each new consumer is another opportunity to forget.

**Implication:** The fix must be structural — canonical filter functions that consumers import, or allowlist scanners that declare their output shape.

### Why can't we fix classes independently?

**Constraint:** The dependency graph means fixing a downstream class (e.g., scope expansion) without addressing its upstream enabler (stale artifacts) produces temporary relief at best.

**Implication:** Structural prevention should follow the dependency graph — fix upstream classes first to reduce downstream exposure.

### Why are instance counts approximate?

**Constraint:** Classification is based on commit message analysis and root cause tracing, not exhaustive annotation of all 459 commits. Some commits may fit multiple classes or none.

**Implication:** Instance counts are lower bounds. The dependency graph direction is a hypothesis that should be verified as structural fixes are implemented.

---

## Evolution

**2026-03-03:** Initial taxonomy created. Seven classes identified from 459 fix commits spanning Dec 2025–Mar 2026. Scope expansion (Class 0) was the only previously named class. Six new classes named: filter amnesia, multi-backend blindness, stale artifact accumulation, cross-project boundary bleed, contradictory authority signals, duplicate action without idempotency, premature/wrong-target destruction. Dependency graph established.

**Open questions:**
- What percentage of 459 fix commits fit one of the 7 classes vs. truly one-off? (Quantification would strengthen the model)
- Will multi-backend blindness stabilize naturally as the codebase matures post-transition, or is it permanent?
- Are there cross-project defect classes (same class in beads, opencode, orch-go) that this project-scoped analysis missed?
- Does the fix oscillation in Class 5 correlate with specific agent models (Opus vs Sonnet)?

**Probes directory:** `probes/` — future probes will test whether structural fixes reduce instance rates for specific classes.

---

## References

**Investigations:**
- `.kb/investigations/2026-03-03-inv-catalogue-unnamed-defect-classes-orch.md` — Source investigation with full instance tables and commit SHAs
- `.kb/investigations/2026-03-03-design-scope-expansion-failure-mode-defense.md` — Class 0 deep analysis and three-layer defense design

**Probes:**
- `.kb/models/completion-verification/probes/2026-02-16-probe-three-code-paths-verification-state.md` — Class 5 instance (contradictory authority across spawn gate, daemon, review)
- `.kb/models/completion-verification/probes/2026-03-01-probe-cross-model-blind-spots.md` — Cross-model analysis applicable to Class 2

**Decisions:**
- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Established domain boundaries, drove Class 4 exposure

**Related models:**
- `.kb/models/completion-verification/model.md` — Verification gates interact with Classes 1, 5, and 7
- `.kb/models/agent-lifecycle-state-model/model.md` — Agent state transitions are where Classes 3, 5, and 7 manifest
- `.kb/models/spawn-architecture/model.md` — Spawn is where Classes 2, 4, and 6 concentrate

**Primary evidence:**
- 459 `fix:` commits from `git log --since="2025-12-01" --grep="fix:"` — raw data
- 60+ closed bug issues from `bd list --status=closed --type=bug` — instance source
