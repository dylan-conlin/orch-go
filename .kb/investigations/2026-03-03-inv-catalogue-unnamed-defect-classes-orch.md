## Summary (D.E.K.N.)

**Delta:** Seven recurring defect classes identified in orch-go (459 fix commits, 3 months). One was already named (scope expansion). Six are newly named: multi-backend blindness (15+ instances), cross-project boundary bleed (20+ instances), stale artifact accumulation (20+ instances), contradictory authority signals (10+ instances), duplicate action without idempotency (12+ instances), and premature/wrong-target destruction (8+ instances). Filter amnesia validated as a subclass of scope expansion, not independent.

**Evidence:** Mined all 459 `fix:` commits from Dec 2025–Mar 2026, 60+ closed bugs from beads, and probe findings. Each class has 8+ instances with traced root causes from commit messages.

**Knowledge:** These classes overlap in a dependency graph: stale artifact accumulation creates the data that scope expansion finds; multi-backend blindness is structurally similar to filter amnesia but at the architecture level; contradictory authority signals are the root cause of the status oscillation bugs that consumed 40+ commits. Naming these creates a shared vocabulary for spawn context, architect reviews, and daemon self-checks.

**Next:** Promote to decision. Create `orch doctor --defect-scan` that checks for known instances of each class. Add class names to architect skill context so new features are reviewed against them.

**Authority:** strategic — Naming defect classes creates irreversible framing that shapes all future development and review. This is a value judgment about which patterns matter.

---

# Investigation: Catalogue of Recurring Defect Classes in orch-go

**Question:** What recurring structural failure patterns exist in orch-go beyond the already-named "scope expansion without assumption validation"? Can we name them, define them, and count their instances to enable structural prevention?

**Defect-Class:** meta (cross-class catalogue)

**Started:** 2026-03-03
**Updated:** 2026-03-03
**Owner:** og-inv-catalogue-unnamed-defect-03mar-5e6a
**Phase:** Complete
**Next Step:** None — promote to decision when accepted
**Status:** Complete

**Patches-Decision:** N/A (new catalogue — no prior decision exists)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-03-design-scope-expansion-failure-mode-defense.md | extends | Yes — 8 instances catalogued there are included here as Class 1 | None — this investigation adds 6 new classes |
| 2026-02-16-probe-three-code-paths-verification-state.md | deepens | Yes — probe finding is instance of Class 5 (contradictory authority signals) | None |
| 2026-03-01-probe-cross-model-blind-spots.md | extends | Yes — blind spot analysis applies to multi-backend blindness | None |

---

## The Catalogue

### Class 0: Scope Expansion Without Assumption Validation (ALREADY NAMED)

**Definition:** A scanner/query expands its scope (wider search, cross-project, new data source). Downstream consumers have implicit assumptions about what the scanner returns. The expanded scope surfaces a data class the consumer doesn't expect. Consumer breaks.

**Instance Count:** 8 (catalogued in prior investigation)

**Representative Examples:**
- **ihc4**: Cross-project workspace scan found 19 untracked workspaces with synthetic IDs → verification counter inflated past threshold
- **1229**: Hotspot bloat scanner walked build directories → false positives blocked spawns
- **1096**: `bd list -l orch:agent` returned closed issues (label never removed) → active count inflated

**Structural Prevention:** Three-layer defense (allowlist scanner, scan inventory, daemon self-check invariants) — designed, not yet implemented.

**Relationship to Other Classes:** Filter amnesia (Class 1) is a subclass. Stale artifact accumulation (Class 3) creates the data that triggers this class.

---

### Class 1: Filter Amnesia

**Definition:** A filter, guard, or exclusion exists in one code path. A new consumer of the same data is added without applying the same filter. The new consumer processes items the original consumer correctly excluded.

**Instance Count:** 15+

**Representative Examples:**

| # | Issue | Filter That Existed | New Consumer Missing It | Commit |
|---|-------|---------------------|------------------------|--------|
| 1 | ihc4 | `isUntrackedBeadsID()` in active_count.go | verification_tracker.go | 190fe365d |
| 2 | bf0a | open/in_progress filter in agent listing | dashboard agent discovery (missing `blocked` status) | e448c5673 |
| 3 | ym0m | Labels/LabelsAny in ListArgs struct | CLIClient.List() silently ignored them | 880c3864b |
| 4 | 155e | closed-issue filter in review | orch status architect recommendations | 155e1771b |
| 5 | fc1c | closed-issue filter in other endpoints | /api/pending-reviews | fc1c8482a |
| 6 | 1094 | time-window filter in CLI | /api/sessions defaulted to 12h, returning empty | a10a93f27 |
| 7 | 43bb | closed-issue filter | orch review NEEDS_REVIEW output | 43bbeb234 |
| 8 | three-code-paths | open-issues filter in daemon+review | spawn gate checked CLOSED issues instead | probe 2026-02-16 |

**Structural Prevention:** Allowlist scanner pattern (same as Class 0 prevention). Also: canonical filter functions used by all consumers rather than each consumer reimplementing.

**Relationship to Other Classes:** This IS a subclass of scope expansion (Class 0). The "new consumer" is the expansion; the "missing filter" is the unvalidated assumption. Separating it is useful because the fix pattern differs — Class 0 is about scanner design, while filter amnesia is about consumer discipline.

---

### Class 2: Multi-Backend Blindness

**Definition:** Code is written and tested against one spawn backend (OpenCode API or Claude CLI/tmux) but doesn't account for agents running on the other backend. The code works for its tested path and silently produces wrong results for the other.

**Instance Count:** 15+ (commits touching this: 42)

**Representative Examples:**

| # | Issue | What Was Backend-Blind | What Broke | Commit |
|---|-------|----------------------|------------|--------|
| 1 | 456h | Status detection checked SessionID before SpawnMode | Claude CLI agents (tmux window ID as session_id) always marked idle | deb24530e |
| 2 | a774 | Daemon cleanup only checked OpenCode sessions | Killed active Claude CLI workers on restart | 108495ddc |
| 3 | 15a2 | `orch clean` only checked OpenCode sessions + beads | Killed active Claude Code agents in tmux | ea01657eb |
| 4 | 4uz | `DefaultActiveCount()` only queried OpenCode API | Claude CLI agents invisible to capacity checks, unlimited spawns | f03c05220 |
| 5 | eqjn | Session dedup only checked OpenCode API | Claude CLI agents bypassed dedup, 10 duplicates in 20 min | ba6b612fd |
| 6 | 1183 | Three endpoints checked tmux state independently | Dashboard oscillated — tmux failures caused disagreement | 8f3cffbb8 |
| 7 | 6ad2 | Workspace scan, beads enrichment, agent map all OpenCode-only | Dashboard blind to tmux agents — three compounding bugs | 6ad20dde0 |
| 8 | kqmf | Agent metadata read from OpenCode session | Claude CLI agents use AGENT_MANIFEST.json — wrong source prioritized | 9eb8204f5 |

**Why This Is a Distinct Class (Not Just Filter Amnesia):**
Filter amnesia is about missing a filter on data. Multi-backend blindness is about missing an entire execution pathway. The structural fix is different: filter amnesia needs canonical filter functions; multi-backend blindness needs a backend-aware dispatch interface where adding a new query for one backend forces implementing it for the other (or explicitly opting out).

**Structural Prevention:**
- Backend-aware query interface: `AgentDiscovery.ListActive()` that dispatches to both backends and merges results, rather than callers querying OpenCode directly
- The `CombinedActiveCount()` function (from 4uz fix) is the embryo of this pattern
- Test requirement: any new agent query must have test cases for both backends

**Historical Note:** Claude CLI became the default backend on Feb 19, 2026 (OAuth ban). Before that, most agents were OpenCode. The backend transition exposed every OpenCode-only code path as a bug. The bug velocity in this class spiked in late Feb / early Mar.

---

### Class 3: Stale Artifact Accumulation

**Definition:** State artifacts (workspaces, labels, sessions, PID files, cache entries, DB records) are created during normal operations but never cleaned up on lifecycle transitions. Over time, accumulated stale artifacts interact unpredictably with new features — inflating counts, slowing scans, causing false matches, or resurrecting dead state.

**Instance Count:** 20+ (commits touching this: 45+)

**Representative Examples:**

| # | Issue | What Accumulated | What Broke | Commit |
|---|-------|-----------------|------------|--------|
| 1 | 1096 | `orch:agent` label on closed issues | Active agent count inflated with historical agents | f7523d21d |
| 2 | 1098 | 1,314 archived workspaces in `.orch/workspace/` | 5 scan functions O(1438) instead of O(124) | d8604c5a5 |
| 3 | ghost | OpenCode sessions not deleted after completion | `orch status` showed completed agents as "running" | 5d26e1f59 |
| 4 | 4omh | Daemon PID file from dead process | Status readers showed stale daemon info | 8d227c80b |
| 5 | 1zyp | pollTime not refreshed before status write | Daemon always showed "stalled" | a95e62d0b |
| 6 | ahif | Spawn cache cleared on orphan detection | Recently-spawned agents re-spawned (thrash loop) | 142c23ae2 |
| 7 | 4yyr | Debrief queries with no time cap | Stale session data from weeks ago included | 1b2ce020f |
| 8 | xptz | `orch:agent` labels on cross-project issues | Ghost agents from historical residue in other projects | 8de20511b |
| 9 | jcyl | Cross-repo failed spawn issues in queue | Infinite retry every 15s poll cycle (queue poisoning) | d8a5a459c |
| 10 | zombie | 49 issues stuck in in_progress | Blocked respawns, inflated all counters | d767a2f68 |

**Why This Matters Beyond Individual Fixes:**
Every scope expansion (Class 0) is only dangerous because accumulated state contains unexpected entries. If lifecycle transitions reliably cleaned up artifacts, scanners would only find current state. Stale artifact accumulation is the **enabling condition** for scope expansion failures.

**Structural Prevention:**
- Lifecycle hook discipline: every state transition (spawn → active → complete → archived) must clean up artifacts from the prior state
- `orch clean` as a periodic maintenance command (exists but needs to cover all artifact types)
- TTL-based expiry for transient artifacts (spawn cache, PID files, session metadata)
- `orch doctor` checks for accumulated artifacts (partially exists)

---

### Class 4: Cross-Project Boundary Bleed

**Definition:** Code assumes single-project context (current working directory, beads database, git repo, .kb/ directory). When invoked for a different project (via daemon cross-project spawn, `--workdir` flag, or multi-project dashboard), it operates on the wrong project's data without error.

**Instance Count:** 20+ (commits touching this: 27)

**Sub-patterns:**

#### 4a: Global State Corruption (beads.DefaultDir)

The `beads.DefaultDir` package-level variable controls which project's beads database is targeted. Setting it without defer-restore causes all subsequent beads operations in the same process to target the wrong project.

| # | Issue | Where DefaultDir Was Set Without Restore | Impact | Commit |
|---|-------|----------------------------------------|--------|--------|
| 1 | obdv | complete_pipeline.go:103 | Error paths leave wrong project active | 98c6fed48 |
| 2 | 82ge | rework_cmd.go, abandon_cmd.go | Same pattern in two more commands | 18553654c |
| 3 | 1230 | runWork() (before --workdir check) | 103 consecutive cross-project spawn failures | 9d6f97e47 |
| 4 | vv7l | loadBeadsLabels in spawn | Labels (needs:playwright) silently lost | d61ef549a |

#### 4b: CWD Assumption (hardcoded working directory)

Functions use `os.Getwd()` or process CWD instead of an explicit `projectDir` parameter.

| # | Issue | Function Using CWD | Should Have Used | Commit |
|---|-------|-------------------|------------------|--------|
| 1 | rfru | Daemon completion comment fetching | agent.ProjectDir | 5f2605c31 |
| 2 | t3ll | runKBContextQuery `cmd.Dir` | projectDir parameter | 61d2a8df3 |
| 3 | 3go1 | Gap analysis quality scoring | projectDir for path matching | dcc9f7bd2 |
| 4 | mwkh | VerifyGitDiff baseline SHA | manifest.ProjectDir | c940cebe2 |

#### 4c: Missing Project Context in API/Shell-outs

| # | Issue | What Lacked Project Context | Commit |
|---|-------|---------------------------|--------|
| 1 | nw73 | BEADS_DIR not injected for cross-repo phase reporting | f6e276dde |
| 2 | 1152 | Deliverable path detection for cross-repo agents | 6f3ed5c82 |
| 3 | 1231 | Work-graph 'unassigned' for cross-project issues | 7b6831287 |
| 4 | c09bb | kb context group resolution for cross-project spawns | c09bb9079 |

**Structural Prevention:**
- Eliminate `beads.DefaultDir` global — replace with explicit `projectDir` parameter on all beads client methods
- Require `ProjectDir` field on all structs that cross project boundaries (agent, workspace, completion target)
- Lint rule: flag any code that calls `beads.DefaultDir =` without `defer` on the next line

---

### Class 5: Contradictory Authority Signals

**Definition:** Multiple sources of truth exist for the same state (typically agent status). Different code paths read different signals and reach contradictory conclusions. Fixes oscillate between "signal A is authoritative" and "signal B is authoritative" without resolving which one wins.

**Instance Count:** 10+ (commits touching status derivation: 41)

**The Canonical Example — Agent Completion Status:**

The system has 4+ signals for "is this agent done?":
1. **Phase: Complete** — beads comment from agent
2. **SYNTHESIS.md** — file existence in workspace
3. **OpenCode session state** — active/idle/completed
4. **Tmux window liveness** — process running or not

These signals disagree in real scenarios:
- Agent reports Phase: Complete but OpenCode session still open (hasn't exited yet)
- SYNTHESIS.md exists from previous spawn but agent was re-spawned
- OpenCode session is idle but agent is still thinking (Claude CLI)
- Tmux window exists but process is dead

**The Fix Oscillation:**

| Date | Commit | Direction | Fix |
|------|--------|-----------|-----|
| Dec 28 | eed04d690 | Phase Complete > session state | "Phase Complete is authoritative regardless of session" |
| Jan 6 | a4dfcff32 | Session state > Phase Complete | "Active session overrides Phase Complete" |
| Jan 6 | c23ddbf8e | Same as a4dfcff32 | Duplicate of the same fix |
| Jan 8 | 125a69de6 | Session state > artifacts | "Status detection respects session state over artifacts" |
| Feb 16 | probe | Three paths disagree | Spawn gate: CLOSED issues; Daemon: OPEN issues; Review: workspace + OPEN |

Each fix was correct for its triggering bug but created the conditions for the next bug. The oscillation is the defect class — not any individual fix.

**Other Instances:**
- **1183**: Three dashboard endpoints checked tmux liveness independently → oscillation between correct and "unassigned" each poll cycle
- **lvu0**: "Active for concurrency" vs "active for listing" — idle agents counted against cap in one path, not the other
- **three-code-paths probe**: Spawn verification, daemon verification, and review each used different definitions of "unverified"

**Structural Prevention:**
- Single canonical status derivation function used by all consumers (the `ListUnverifiedWork()` created in the three-code-paths probe is the embryo)
- Explicit precedence hierarchy documented and enforced: define once which signal wins when they disagree
- Status should be computed, not stored — derive from primary signals each time instead of caching a derived status that goes stale

---

### Class 6: Duplicate Action Without Idempotency

**Definition:** An action (spawn, completion recording, event emission, issue creation) is performed multiple times because the dedup mechanism is absent, too narrow, or has a race window. The duplicate causes visible damage (duplicate agents, inflated counters, wasted resources).

**Instance Count:** 12+ (commits touching dedup: 23)

**The Duplicate Spawn Saga (7 separate fixes):**

| # | Issue | Why Dedup Failed | Fix | Commit |
|---|-------|-----------------|-----|--------|
| 1 | 2ma | No dedup at all | `SpawnedIssueTracker` with 5-min TTL | 48b850cca |
| 2 | 09cc | Race between daemon poll and beads status update | Fresh beads status check before spawn | 4e6609909 |
| 3 | 2nru | TTL expired while agent still running | Session-level dedup via OpenCode query (6h) | 674912984 |
| 4 | eqjn | OpenCode-only dedup — Claude CLI agents invisible | Layer 2 tmux window check | ba6b612fd |
| 5 | ahif | Orphan detector cleared spawn cache | Remove Unmark() from orphan path | 142c23ae2 |
| 6 | d29a | Different beads IDs, same title | Content-aware dedup (title matching) | d29a3c8af |
| 7 | 13774 | in_progress issues not filtered | Skip in_progress in NextIssue() | 13774ca50 |

Each fix patched one dedup gap, which revealed the next. The sequence shows the class clearly: dedup is a cross-cutting concern that can't be solved with point fixes.

**Other Duplicate Action Instances:**
- **rexs**: Double `RecordCompletion()` call in daemon loop → counter inflated 2x
- **e3oi**: Duplicate abandonment events in stats
- **ed1c71c**: Duplicate knowledge maintenance issue creation
- **1209**: `--skip-phase-complete` not propagated → bd close double-gated

**Structural Prevention:**
- Idempotency keys: every spawn gets a unique key; second spawn with same key is a no-op
- Two-phase commit for daemon: mark issue as "spawn pending" atomically before actually spawning
- Event dedup layer: all event emissions go through a dedup filter keyed on (event_type, entity_id, time_window)

---

### Class 7: Premature/Wrong-Target Destruction

**Definition:** A resource (tmux window, OpenCode session, workspace) is destroyed based on stale or incomplete information, killing active work or destroying state needed for inspection.

**Instance Count:** 8+

**Representative Examples:**

| # | Issue | What Was Destroyed | Why It Was Wrong | Commit |
|---|-------|-------------------|------------------|--------|
| 1 | 1216 | Tmux window by index (unstable) | Window indices shift; killed wrong window | 681157d07 |
| 2 | v454 | Tmux window via defer (all exit paths) | Killed before orchestrator could inspect gate failures | f7b0868bb |
| 3 | a774 | Active Claude CLI workers | Daemon cleanup only checked OpenCode (multi-backend blindness) | 108495ddc |
| 4 | 15a2 | Active Claude Code agents | `orch clean --sessions` only checked OpenCode + beads | ea01657eb |
| 5 | 1221 | Daemon-spawned tmux windows | `orch clean --sessions` didn't protect daemon windows | b88cd9db3 |
| 6 | c855 | Current session | `orch clean --verify-opencode` deleted self | c855f5974 |
| 7 | 722db | Tmux window on early return | Early returns from verification skipped cleanup → phantom OR killed too early | 722ddb6bb |
| 8 | ghost→5d26 | Nothing (inverse) — failed to destroy | OpenCode session left alive → ghost agent | 5d26e1f59 |

**Structural Prevention:**
- **Liveness verification before destruction**: Always check at least 2 independent signals (beads issue status + process liveness) before killing anything
- **Stable identifiers**: Use tmux window IDs (@-prefixed), not indices
- **Destruction as last step**: Never destroy in defer or early-return paths; destroy only on the explicit success path after all gates pass
- **Fail-safe on uncertainty**: If liveness check fails (beads unavailable, tmux unresponsive), preserve the resource

---

## Findings

### Finding 1: The Seven Classes Are Not Independent — They Form a Dependency Graph

**Evidence:** Mapping the causal relationships:

```
Stale Artifact Accumulation (Class 3)
    ↓ creates data that
Scope Expansion (Class 0) finds unexpectedly
    ↓ manifests via
Filter Amnesia (Class 1) — missing exclusion in new consumer

Multi-Backend Blindness (Class 2) — structurally similar to filter amnesia
    but at architecture level, not data level

Cross-Project Boundary Bleed (Class 4)
    ↓ multiplies exposure for
Scope Expansion (Class 0) — cross-project data is the highest-risk expansion vector

Contradictory Authority Signals (Class 5)
    ↓ causes
Premature/Wrong-Target Destruction (Class 7) — wrong status → wrong action

Duplicate Action (Class 6) — independent, caused by missing idempotency
```

**Source:** Cross-referencing all instance tables above

**Significance:** Fixing one class reduces the exposure of related classes. For example, fixing stale artifact accumulation (Class 3) would eliminate the data that makes scope expansion (Class 0) dangerous. The dependency graph suggests a priority order for structural prevention.

---

### Finding 2: Multi-Backend Blindness Spiked After the Backend Transition

**Evidence:** Claude CLI became the default backend on Feb 19, 2026. The 42 commits touching multi-backend concerns are concentrated in late Jan through early Mar. Before the transition, OpenCode was the only backend, so OpenCode-only code was correct. After the transition, every OpenCode-only code path became a latent bug.

Key timeline:
- Jan 10: Backend independence principle discovered (OpenCode crashed, Claude CLI survived)
- Feb 19: Claude CLI becomes default (Anthropic OAuth ban)
- Feb-Mar: 15+ multi-backend blindness fixes

**Source:** `git log` dates for backend-related fixes

**Significance:** This class is a **migration defect** — it exists because the system transitioned from one architecture to another. The individual fixes are correct, but the class will recur if a third backend is ever added (e.g., direct API calls) unless the backend-aware interface pattern is implemented.

---

### Finding 3: The Contradictory Authority Oscillation Is the Most Expensive Class

**Evidence:** 41 commits touch status derivation logic. The Phase Complete vs session state oscillation produced at least 4 contradictory fixes over 2 weeks (Dec 28 – Jan 8). Each fix was correct for its trigger bug but wrong for the opposite scenario. The three-code-paths probe (Feb 16) found that even after months of fixes, three consumers still disagreed on "unverified."

**Source:** Commits eed04d690, a4dfcff32, c23ddbf8e, 125a69de6; probe 2026-02-16

**Significance:** This class is expensive because fixes create new bugs. It's the only class where the fix pattern itself is the problem. The structural fix (single canonical derivation function with explicit precedence) is the only way to break the cycle.

---

### Finding 4: Duplicate Spawn Required 7 Separate Fixes Over 3 Months

**Evidence:** The duplicate spawn problem was first fixed Dec 2025 (SpawnedIssueTracker) and most recently fixed Mar 2026 (spawn cache retained across orphan detection). Each fix patched one dedup gap:
1. No tracker → add tracker
2. Race condition → add fresh status check
3. TTL too short → extend TTL + add session check
4. OpenCode-only → add tmux check
5. Cache cleared by orphan detector → stop clearing
6. Same title, different ID → add content dedup
7. in_progress not filtered → add status filter

**Source:** 7 commit messages in the Duplicate Action catalogue above

**Significance:** Dedup is a cross-cutting concern that resists point fixes. Each fix addresses one gap while the system finds new gaps to exploit. This suggests the need for a unified dedup layer rather than patching individual code paths.

---

### Finding 5: Candidate Pattern Validation

**Validating the 3 candidate patterns from the task:**

1. **Filter Amnesia — VALIDATED as subclass of scope expansion.** 15+ instances. The pattern is real and distinct enough to name separately. It specifically addresses the "new consumer of existing data" scenario where the filter already exists but isn't applied. The existing scope expansion investigation (Finding 3) identified this exact pattern with `isUntrackedBeadsID()`.

2. **Stale Artifact Accumulation — VALIDATED as independent class.** 20+ instances. This is the *enabling condition* for scope expansion. It's worth naming separately because its fix (lifecycle cleanup discipline) is different from the scope expansion fix (allowlist scanners). The 1,314-workspace accumulation (1098) and 49-zombie-issue accumulation are the clearest examples.

3. **Cross-Project Boundary Bleed — VALIDATED as independent class.** 20+ instances with three distinct sub-patterns (global state corruption, CWD assumption, missing project context). The `beads.DefaultDir` sub-pattern alone has 4 instances with identical root cause. This overlaps with scope expansion (cross-project data as unexpected data class) but the fix is architectural — eliminating global state, not adding filters.

---

## Synthesis

**Key Insights:**

1. **Seven defect classes, not isolated bugs.** The 459 fix commits from 3 months cluster into 7 structural patterns. Naming them creates a shared vocabulary that can be injected into spawn context, architect reviews, and automated checks. "This is a multi-backend blindness risk" is more useful than "make sure to check tmux too."

2. **The classes form a causal dependency graph.** Stale artifacts enable scope expansion. Cross-project features multiply scope expansion risk. Contradictory signals cause premature destruction. Fixing upstream classes (stale artifact cleanup, single authority source) reduces the surface area for downstream classes.

3. **Multi-backend blindness is a migration-induced class.** It exists because the system transitioned from OpenCode to Claude CLI. It will recur on the next architectural transition unless a backend-aware abstraction is built. This is the highest-volume class (42 commits) and the most recent.

4. **Contradictory authority signals are the most expensive per-instance.** Each fix creates conditions for the next bug. This is the only class where reactive fixes are actively counterproductive.

5. **Duplicate spawn resistance to point fixes proves the need for cross-cutting dedup.** Seven fixes over 3 months, each addressing a different gap. The pattern won't stop until dedup is a system-level concern rather than a per-code-path concern.

**Answer to Investigation Question:**

Six defect classes have been newly named and catalogued alongside the one already named. Together, these seven classes account for the majority of structural bugs in orch-go over the past 3 months. The candidate patterns (filter amnesia, stale artifact accumulation, cross-project boundary bleed) are all validated. Additionally, three classes were discovered that weren't in the candidate list: multi-backend blindness, contradictory authority signals, and duplicate action without idempotency. Premature/wrong-target destruction was identified as a consequence class driven by the others.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 7 classes have 8+ instances traced to specific commits with root cause analysis (verified: read commit messages for all representative examples)
- ✅ Filter amnesia is a subclass of scope expansion (verified: Finding 3 of prior investigation explicitly identifies this)
- ✅ Contradictory authority signal oscillation produced 4 contradictory fixes in 2 weeks (verified: commits eed04d690, a4dfcff32, c23ddbf8e, 125a69de6 traced)
- ✅ Duplicate spawn required 7 separate fixes (verified: all 7 commit messages read and sequenced)
- ✅ Multi-backend blindness spiked after Feb 19 backend transition (verified: commit dates)

**What's untested:**

- ⚠️ These 7 classes account for "the majority" of structural bugs (not quantified — some fixes may not fit any class)
- ⚠️ The dependency graph is directional (stale artifacts → scope expansion) rather than bidirectional (could go both ways in edge cases)
- ⚠️ The proposed structural preventions would actually prevent future instances (design-level claims, not implemented)
- ⚠️ There may be an 8th+ class hiding in the 459 commits that I didn't identify (sampling, not exhaustive classification)

**What would change this:**

- If a significant cluster of bugs doesn't fit any of the 7 classes, a new class should be added
- If the structural preventions for one class make another class worse (e.g., allowlist pattern increases code complexity enough to cause new bugs), the dependency graph needs revision
- If the multi-backend blindness class stops producing bugs after the transition stabilizes, it may be a one-time migration artifact rather than a recurring class

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Name and catalogue 7 defect classes | strategic | Irreversible framing choice that shapes future development vocabulary |
| Add class names to architect skill context | architectural | Cross-component: affects how all future features are reviewed |
| Implement `orch doctor --defect-scan` | architectural | Cross-component: scans across daemon, spawn, completion, dashboard |
| Eliminate `beads.DefaultDir` global | architectural | Structural change to how all beads operations work |
| Backend-aware query interface | architectural | Structural change to how all agent queries work |

### Recommended Approach ⭐

**Defect Class Vocabulary + Automated Detection** — Name the classes (this investigation), inject them into architect review context, and build `orch doctor --defect-scan` to check for known instances.

**Why this approach:**
- Naming is cheap and immediately useful — architects reviewing features can check "is this a multi-backend blindness risk?"
- Automated detection catches instances that humans miss (the whole point of naming the class)
- Builds on existing `orch doctor` infrastructure

**Trade-offs accepted:**
- Naming creates commitment — a bad name is worse than no name (mitigated: names are descriptive and grounded in instances)
- Automated detection only catches known patterns, not novel instances (mitigated: architect review covers novel cases)

**Implementation sequence:**
1. **Phase 1: Vocabulary** — Accept this catalogue. Add class names to architect skill context so new features are reviewed against them.
2. **Phase 2: `orch doctor --defect-scan`** — For each class, implement a check that detects known anti-patterns in the current codebase.
3. **Phase 3: Structural fixes** — Tackle the highest-value structural preventions: single canonical status derivation (Class 5), backend-aware query interface (Class 2), eliminate `beads.DefaultDir` global (Class 4).

### Alternative Approaches Considered

**Option B: Fix only the highest-instance classes (2 and 3)**
- **Pros:** Focused effort, highest ROI
- **Cons:** Leaves other classes unnamed and untracked; naming cost is near-zero
- **When to use instead:** If implementation capacity is extremely limited

**Option C: Build comprehensive prevention infrastructure for all 7 classes simultaneously**
- **Pros:** Prevents all classes at once
- **Cons:** Massive scope; most classes only need naming + awareness, not infrastructure
- **When to use instead:** Never — incremental is better given the dependency graph

**Rationale for recommendation:** Naming is essentially free and immediately valuable. Automated detection is medium-cost and catches what naming misses. Structural fixes should be prioritized by the dependency graph: fix upstream classes (stale artifacts, authority signals) to reduce exposure of downstream classes.

---

### Implementation Details

**What to implement first:**
- Add the 7 class names to architect skill context (immediate, <1h)
- Build `orch doctor --defect-scan` basic checks for Class 2 (multi-backend) and Class 5 (contradictory authority)

**Things to watch out for:**
- ⚠️ Naming creates anchoring — resist the temptation to force-classify every bug into one of the 7 classes. New classes may emerge.
- ⚠️ The dependency graph is a hypothesis. Verify it holds as structural fixes are implemented.
- ⚠️ Multi-backend blindness may stabilize naturally as the codebase matures post-transition. Monitor before investing heavily in backend abstraction.

**Areas needing further investigation:**
- What percentage of the 459 fix commits fit one of the 7 classes vs. being truly one-off? (Quantification would strengthen the case)
- Are there cross-project defect classes (same class appearing in beads, opencode, orch-go) that this project-scoped analysis missed?
- Does the fix oscillation in Class 5 correlate with specific agent models (Opus vs Sonnet) or is it model-independent?

**Success criteria:**
- ✅ Architect reviews reference defect class names when evaluating new features
- ✅ `orch doctor --defect-scan` detects at least 1 instance that would otherwise ship
- ✅ Defect class instance counts decline over the next 3 months (tracked via git log mining)

---

## Defect Class Quick Reference

| # | Class Name | Definition (one line) | Instance Count | Fix Pattern |
|---|------------|----------------------|----------------|-------------|
| 0 | Scope Expansion Without Assumption Validation | Scanner widens, consumer's implicit assumptions break | 8+ | Allowlist scanner pattern |
| 1 | Filter Amnesia | Filter exists in path A, missing in new path B | 15+ | Canonical filter functions |
| 2 | Multi-Backend Blindness | Code works for OpenCode OR Claude CLI, not both | 15+ | Backend-aware query interface |
| 3 | Stale Artifact Accumulation | Dead state never cleaned up, interferes with new features | 20+ | Lifecycle cleanup discipline |
| 4 | Cross-Project Boundary Bleed | Single-project code breaks in multi-project context | 20+ | Eliminate global state, thread projectDir |
| 5 | Contradictory Authority Signals | Multiple sources of truth disagree, fixes oscillate | 10+ | Single canonical derivation function |
| 6 | Duplicate Action Without Idempotency | Same action performed multiple times, no dedup | 12+ | System-level idempotency layer |
| 7 | Premature/Wrong-Target Destruction | Resource killed based on stale/incomplete info | 8+ | Liveness verification before destruction |

---

## References

**Files Examined:**
- 459 `fix:` commit messages from Dec 2025–Mar 2026
- 60+ closed bug issues from `bd list --status=closed --type=bug`
- `.kb/investigations/2026-03-03-design-scope-expansion-failure-mode-defense.md` — prior class 0 catalogue
- `.kb/models/completion-verification/probes/2026-02-16-probe-three-code-paths-verification-state.md` — class 5 instance
- `.kb/models/completion-verification/probes/2026-03-01-probe-cross-model-blind-spots.md` — cross-model analysis

**Commands Run:**
```bash
# Mine all bug fix commits from last 3 months
git log --since="2025-12-01" --grep="fix:" --format="%h %s"

# Read commit messages for representative examples per class
git show --format="%h %s%n%b" --no-patch <sha>

# Count instances per defect class pattern
git log --since="2025-12-01" --grep="fix:" --format="%h %s" | grep -ciE "<pattern>"

# List closed bug issues from beads
bd list --status=closed --type=bug
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-03-design-scope-expansion-failure-mode-defense.md` — Class 0 analysis and three-layer defense design
- **Probe:** `.kb/models/completion-verification/probes/2026-02-16-probe-three-code-paths-verification-state.md` — Class 5 instance (contradictory authority)
- **Decision:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Established domain boundaries, drove Class 4 exposure

---

## Investigation History

**2026-03-03 14:00:** Investigation started
- Initial question: What recurring structural failure patterns beyond "scope expansion" exist in orch-go?
- Context: 459 fix commits in 3 months. One class formally named, suspected more.

**2026-03-03 14:30:** Data mining complete
- Mined all fix commits, closed bugs, probes. Categorized by grep patterns for cross-project, multi-backend, duplicate, status, stale, filter, destruction.

**2026-03-03 15:00:** Seven classes identified and catalogued
- Validated 3 candidate patterns (filter amnesia, stale artifacts, cross-project)
- Discovered 3 additional classes (multi-backend blindness, contradictory authority, duplicate action without idempotency)
- Identified premature destruction as consequence class

**2026-03-03 15:30:** Investigation completed
- Status: Complete
- Key outcome: Seven named defect classes with 100+ total instances, dependency graph, and structural prevention recommendations
