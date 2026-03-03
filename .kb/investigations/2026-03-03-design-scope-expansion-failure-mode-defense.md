## Summary (D.E.K.N.)

**Delta:** Scope-expansion failures are a distinct defect class ("scope expansion without assumption validation") that has caused 8+ production incidents. The structural root cause: scanners return all results, consumers assume properties about them, and those assumptions are implicit — never declared, never validated against real state.

**Evidence:** 8 catalogued incidents (ihc4, 1098, 1229, 1096, rfru, vos2, rexs, xptz) all follow the same pattern: scanner widens → finds unexpected data class → consumer's implicit assumption breaks. Every fix was "add a filter" — reactive, not preventive.

**Knowledge:** Three-layer defense: (1) allowlist scanner pattern makes new data classes excluded by default, (2) `orch doctor --scan-inventory` reveals what scanners actually find in production, (3) daemon self-check invariants catch violations at runtime. Layer 1 prevents the class; layers 2-3 catch what slips through.

**Next:** Promote to decision. Implement in three phases: daemon self-check first (highest value, catches existing bugs), then allowlist pattern for new scanner code, then scan-inventory tooling.

**Authority:** architectural — Cross-component pattern affecting daemon, spawn, completion, and workspace scanning. Requires structural change to how scanner functions are designed across the codebase.

---

# Investigation: Scope Expansion Failure Mode Defense

**Question:** How do we prevent features that expand scope (wider queries, cross-project scanning, new data sources) from breaking downstream consumers when accumulated real-world state contains entries the feature wasn't designed for?

**Defect-Class:** integration-mismatch (subclass: scope-expansion-without-assumption-validation)

**Started:** 2026-03-03
**Updated:** 2026-03-03
**Owner:** og-arch-recurring-failure-mode-03mar-7799
**Phase:** Complete
**Next Step:** None — promote to decision when accepted
**Status:** Complete

**Patches-Decision:** N/A (new defect class — no prior decision exists)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-07 unbounded-resource-consumption constraints | extends | Yes — the defect class blindness principle originated there | None — different defect class but same meta-pattern |
| 2026-02-18 two-lane-agent-discovery | deepens | Yes — two-lane ADR drove cross-project scanning that triggered several instances | None |
| 2026-02-15 daemon-unified-config-construction | extends | Yes — config path divergence is a related failure mode | None |

---

## Findings

### Finding 1: Eight Catalogued Instances Form a Distinct Defect Class

**Evidence:** Every instance follows the same structure:

| # | Issue ID | What Expanded | What Broke | Data Class Found | Fix Applied |
|---|----------|---------------|------------|------------------|-------------|
| 1 | ihc4 | Cross-project workspace scanning → verification counter | Counter inflated past threshold, daemon paused | 19 untracked workspaces with synthetic `*-untracked-*` IDs | `isUntrackedBeadsID()` filter in `SeedFromBacklog`/`RecordCompletion` |
| 2 | 1098 | Workspace scan scope (all `.orch/workspace/`) | 5 scan functions O(1438) instead of O(124) | 1,314 archived workspaces in `archived/` subdir | `if entry.Name() == "archived" { continue }` |
| 3 | 1229 | Hotspot bloat scanner (full filesystem walk) | False positives blocked spawns on non-source files | `.svelte-kit/`, `.opencode/`, `dist/`, 13 build dirs | `skipBloatDirs` map + `buildOutputPrefixes` slice |
| 4 | 1096 | `bd list -l orch:agent` (never cleaned on close) | Active-agent count inflated with historical agents | Closed issues retaining `orch:agent` label | `bd label remove` in `on_close` hook |
| 5 | rfru | Cross-project completion processing (comments) | 739+ failures/session — bd calls against wrong project | Cross-project agents without `ProjectDir` | `ProjectDir string` field on `CompletedAgent` struct |
| 6 | vos2 | Verification counter (poll-cycle re-encounters) | One agent across 3 poll cycles hits threshold=3 | Same agent returned by `ListCompletedAgents()` every poll | `seenIDs map[string]bool` deduplication |
| 7 | rexs | Double `RecordCompletion()` call path | Daemon paused after 2 completions (counted as 4) | Each completion counted in two code paths | Remove duplicate call site |
| 8 | xptz | Cross-project `queryTrackedAgents` via kb projects | Ghost agents with stale `orch:agent` labels | Historical residue from other projects | `orch clean --ghosts` + cross-project `orch abandon` |

**Source:** Git commits: `190fe365d`, `d8604c5a5`, `03839d7eb`, `f7523d21d`, `5f2605c31`, `fa3a43feb`, `eb53a2a2f`, `8de20511b`

**Significance:** This is not 8 unrelated bugs. It's one defect class — **scope expansion without assumption validation** — manifesting across different components. The Defect Class Blindness principle (from `~/.kb/principles.md`) applies: "Investigations that fix individual symptoms without connecting shared root causes allow the same defect class to ship repeatedly."

---

### Finding 2: The Fix Is Always "Add a Filter" — And It's Always Reactive

**Evidence:** In all 8 cases, the fix was structurally identical:

```
scanner returns items → consumer iterates all → BREAK
                                                   ↓
fix: add filter(item) → skip items that don't match consumer assumptions
```

The filter addresses a specific data class discovered in production:
- `isUntrackedBeadsID()` for synthetic beads IDs
- `archived/` skip for historical workspaces
- `skipBloatDirs` for build output
- Label removal for lifecycle cleanup
- `ProjectDir` threading for cross-project routing
- `seenIDs` dedup for poll-cycle re-encounters

Every filter was added **after** the production breakage. Tests passed because they tested against expected inputs — the bug lived in data classes the tests didn't model.

**Source:** Code review of all 8 fix commits

**Significance:** The reactive pattern means each new scope expansion introduces a period of production vulnerability. The 19-untracked-workspace incident (ihc4) is the latest example: feature shipped, tests passed, production broke within the first daemon cycle. The structural problem: **consumer assumptions about scanner data are implicit** — they live in the developer's head, not in the code.

---

### Finding 3: The `isUntrackedBeadsID()` Filter Already Existed — But Wasn't Applied to New Code

**Evidence:** When cross-project workspace scanning was added (orch-go-x1ln), the `isUntrackedBeadsID()` function already existed in `active_count.go:157`, used by `DefaultActiveCount()` (line 67) and `CombinedActiveCount()` (line 239). The ihc4 fix simply extended the same filter to `verification_tracker.go` — two code paths in the same package that should have had the same exclusion.

**Source:** `pkg/daemon/active_count.go:67,157,239`, `pkg/daemon/verification_tracker.go:75,140`

**Significance:** The filter existed but was not discoverable by the feature developer. There's no mechanism that says "when adding a new consumer of workspace scan results, apply these exclusions." Each consumer must independently discover and implement the correct filters. This is the denylist problem: every new consumer must know about every data class to exclude. Missed one? Production breaks.

---

### Finding 4: Tests Can't Catch What They Don't Model

**Evidence:** The x1ln feature (cross-project workspace scanning) included tests that passed. The tests created mock workspaces with expected beads IDs. No test created an "untracked workspace with synthetic beads ID" because the developer didn't know those existed in production.

The fundamental constraint: **you can't write a test for data you don't know exists.** Tests verify behavior against expected inputs. Scope-expansion bugs live in the gap between expected and actual production state.

**Source:** Task description, commit history, test review

**Significance:** More testing won't solve this. Better tests require knowing what to test for, which requires knowing what data exists in production. The solution must operate at a different level: either making assumptions explicit (so they can be validated), or running against real state (so unknown data classes surface naturally).

---

### Finding 5: Cross-Project Features Are the Primary Expansion Vector

**Evidence:** Of the 8 catalogued instances, 4 (ihc4, rfru, xptz, and the daemon preview reformat dj34) were triggered by cross-project scanning features. The two-lane agent discovery ADR (2026-02-18) established domain boundaries but didn't anticipate the data classes that cross-project scanning would surface.

**Source:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md`, commits for ihc4, rfru, xptz, dj34

**Significance:** Cross-project features are structurally more dangerous than single-project features because they expand the data surface multiplicatively. A single project might have 20 workspaces; 5 projects might have 200 with different lifecycle patterns, labeling conventions, and accumulated state. The two-lane ADR correctly established domain boundaries but didn't address the "what data classes exist across those domains?" question.

---

## Synthesis

**Key Insights:**

1. **This is a defect class, not individual bugs.** "Scope expansion without assumption validation" is as distinct as "unbounded resource consumption" (which has its own principle). The 8 instances share: scanner widens → finds unexpected data class → consumer's implicit assumptions break → fix: add filter. Recognizing it as a class enables structural prevention rather than per-instance patching.

2. **The root cause is implicit consumer assumptions.** Every consumer of scanner data implicitly assumes properties about the items: "beads IDs are real," "workspaces are active," "labels are current." These assumptions are never declared in code. The scanner doesn't know what the consumer expects, and the consumer doesn't know what the scanner might return. The gap between these two creates the vulnerability.

3. **Testing can't bridge the gap because tests use expected data.** The solution must either make assumptions explicit (so new data classes trigger a compiler/lint error) or run against real state (so unknown classes surface naturally). These are complementary, not competing.

4. **Cross-project features multiply the exposure.** Single-project state is relatively well-known. Cross-project scanning reveals accumulated state from multiple projects with different histories, conventions, and lifecycles. This is the highest-risk expansion vector.

**Answer to Investigation Question:**

Prevent scope-expansion failures through a three-layer defense that doesn't require perfect human/agent judgment to activate:

1. **Layer 1 — Allowlist Scanner Pattern (structural prevention):** Invert the current denylist model. Instead of scanners returning everything and consumers filtering out what they don't want, scanners accept explicit inclusion criteria. New data classes are excluded by default. Consumers must explicitly opt in to each class they can handle.

2. **Layer 2 — Production State Inventory (`orch doctor --scan-inventory`):** Run scanner functions against real production state and classify every entry. Shows the landscape of actual data before and after scope changes. Catches "19 untracked workspaces" before they break the verification counter.

3. **Layer 3 — Daemon Self-Check Invariants (runtime detection):** Each consumer declares invariants about its inputs ("all beads IDs must be real," "counter must not exceed N"). The daemon validates these every poll cycle. Violations trigger alert → configurable pause. Catches what slips through layers 1-2 within one poll cycle (60s).

---

## Structured Uncertainty

**What's tested:**

- ✅ All 8 instances follow the scope-expansion-without-assumption-validation pattern (verified: read all fix commits and traced root causes)
- ✅ The `isUntrackedBeadsID()` filter existed before x1ln but wasn't applied to new code (verified: code review of `active_count.go` vs `verification_tracker.go`)
- ✅ Tests for x1ln passed despite the production-breaking data class existing (verified: review of x1ln completion comment)
- ✅ Cross-project features are the primary expansion vector (verified: 4/8 instances are cross-project)

**What's untested:**

- ⚠️ Allowlist scanner pattern would prevent future instances (not implemented — design-level claim)
- ⚠️ `orch doctor --scan-inventory` would have caught ihc4 pre-deployment (not benchmarked — reasonable inference from data)
- ⚠️ Daemon self-check invariants can run within poll-cycle time budget (not profiled)
- ⚠️ Refactoring existing scanners to allowlist pattern is feasible without breaking changes (not prototyped)

**What would change this:**

- If scope-expansion bugs primarily come from non-scanner sources (API changes, schema migrations), the scanner-focused solution would be insufficient
- If daemon poll-cycle time budget is already exhausted, runtime invariant checks would need a separate monitoring loop
- If the allowlist pattern creates excessive boilerplate, a lighter-weight annotation approach might be more practical

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Three-layer defense (allowlist + inventory + self-check) | architectural | Cross-component pattern change affecting daemon, spawn, completion, and workspace scanning. Multiple valid approaches; requires synthesis. |
| Defect class naming and principle promotion | strategic | New principle affects all future development. Irreversible framing choice. |

### Recommended Approach ⭐

**Three-Layer Scope Expansion Defense** — Prevent the class of bug through structural inversion (allowlist), make production state visible (inventory), and catch violations at runtime (self-check).

**Why this approach:**
- Addresses root cause (implicit assumptions) not symptoms (missing filters)
- Doesn't require perfect judgment to activate — allowlist is structural, inventory is automated, self-check is continuous
- Matches substrate: Gate Over Remind (self-check is a gate), Infrastructure Over Instruction (allowlist is code, not documentation), Coherence Over Patches (structural fix for 8+ instances of the same pattern)
- Each layer catches different failure modes: Layer 1 prevents known classes, Layer 2 reveals unknown classes, Layer 3 catches anything that slips through

**Trade-offs accepted:**
- Allowlist pattern requires refactoring existing scanner functions (migration cost)
- Self-check invariants add ~10ms per poll cycle (acceptable within 60s cycle)
- Inventory tooling is manual (not automatic) — developer must run it before deploying scope changes

**Implementation sequence:**

1. **Phase 1 — Daemon Self-Check Invariants (highest immediate value)**
   - Add invariant assertions to daemon poll loop
   - Check: verification counter items have valid beads IDs (not untracked, not synthetic)
   - Check: active agent count matches expected range (0 to MaxAgents, not negative, not >2x cap)
   - Check: completion candidates have valid ProjectDir (not empty for cross-project agents)
   - On violation: log warning with diagnostic detail, increment violation counter, pause after configurable threshold (default: 3 violations)
   - This catches existing bugs in existing code without refactoring
   - **Files:** `pkg/daemon/invariants.go` (new), `cmd/orch/daemon.go` (integrate into poll loop)

2. **Phase 2 — Allowlist Scanner Pattern for New Code**
   - Define `ScanScope` struct with explicit inclusion flags:
     ```go
     type ScanScope struct {
         IncludeTracked   bool  // agents with real beads IDs
         IncludeUntracked bool  // agents with synthetic IDs (*-untracked-*)
         IncludeArchived  bool  // agents in archived/ directories
         IncludeClosed    bool  // agents whose beads issues are closed
         ProjectDirs      []string // empty = current project only
     }
     ```
   - Default (zero-value) ScanScope includes nothing — consumers must explicitly opt in
   - Apply to new scanner functions immediately; migrate existing ones incrementally
   - The type system prevents "forgot to filter" — you can't accidentally get untracked agents without asking for them
   - **Files:** `pkg/daemon/scan_scope.go` (new), gradual adoption in `completion_processing.go`, `active_count.go`

3. **Phase 3 — Production State Inventory (`orch doctor --scan-inventory`)**
   - New `orch doctor` check that runs all scanner functions against real production state
   - Reports classification breakdown:
     ```
     verification-counter scan:
       Tracked agents: 12
       Untracked agents: 19 (excluded from count)
       Archived: 47 (excluded from scan)
     workspace scan:
       Active: 23
       Archived: 1,314
       Missing manifest: 3
     ```
   - Run manually before deploying scope-expanding features
   - Can also be integrated into daemon startup as a one-time health check
   - **Files:** `cmd/orch/doctor_scan.go` (new), integrate into existing `orch doctor` command

### Alternative Approaches Considered

**Option B: Shadow/Audit Mode (run old and new scanners side by side, diff results)**
- **Pros:** Catches exactly the delta between old and new scope; high confidence pre-deployment
- **Cons:** Requires maintaining both old and new code paths; heavy to implement; doesn't work for features where no old path exists; most of the value comes from the inventory aspect (Phase 3 above) without the dual-path complexity
- **When to use instead:** If scope-expanding features are rare enough to justify per-feature dual-path engineering (not the case in orch-go — 8 instances in 3 months)

**Option C: More Gates at Spawn/Completion (require agents to declare scope changes)**
- **Pros:** Would catch scope expansion at development time
- **Cons:** Task explicitly states "agents already skip gates that feel irrelevant." Requires perfect agent judgment to recognize "this is a scope expansion" — the hard part. Human developers also miss this (x1ln developer didn't recognize the untracked workspace risk). Infrastructure Over Instruction principle says this won't work.
- **When to use instead:** Never as a standalone solution. Could supplement structural measures.

**Option D: Comprehensive Production-State Test Fixtures**
- **Pros:** Tests would catch more data classes
- **Cons:** Requires knowing what data exists in production (the core challenge). Test fixtures become stale as production state evolves. Doesn't prevent unknown unknowns. "You can't write a test for data you don't know exists" — the premise of the task.
- **When to use instead:** As a supplement to Layer 1 (allowlist). Once the ScanScope struct exists, tests can verify that each consumer only requests what it can handle. But this doesn't replace running against real state.

**Rationale for recommendation:** Option A (three-layer defense) is the only approach that addresses the structural root cause (implicit assumptions), works without perfect judgment (allowlist is structural, self-check is automatic), and catches unknown unknowns (inventory reveals unexpected data classes). The other options either require judgment that doesn't exist (C), knowledge that doesn't exist (D), or excessive engineering per feature (B).

---

### Implementation Details

**What to implement first:**
- Daemon self-check invariants (Phase 1) — highest immediate value, catches existing bugs, no refactoring required
- Can be implemented as a single new file `pkg/daemon/invariants.go` with a `CheckInvariants()` function called once per poll cycle

**Things to watch out for:**
- ⚠️ Self-check invariants must not fail-closed on infrastructure issues (beads unavailable, tmux down). Use fail-open: if the check itself can't run, skip it and log, don't pause the daemon.
- ⚠️ ScanScope zero-value excluding everything is a breaking change if applied to existing functions. Migrate incrementally — new functions use ScanScope, existing functions keep current behavior until migrated.
- ⚠️ Inventory tooling should not modify state — read-only scan and report. Accidental side effects from "just checking" would be ironic.
- ⚠️ This is a hotspot area (daemon, spawn, verification, workspace). Phase 1 and 3 are safe (new files, non-breaking). Phase 2 (allowlist migration) should go through architect review per the three-layer hotspot enforcement decision.

**Areas needing further investigation:**
- What invariants should the daemon check? The three listed (valid beads IDs, count range, valid ProjectDir) are the minimum. More may emerge from analysis of other scanner consumers.
- Should self-check violations trigger notification (desktop notification) in addition to daemon pause?
- Should inventory be run automatically on daemon startup, or only manually?

**Success criteria:**
- ✅ New scope-expanding features using allowlist pattern cannot accidentally include unexpected data classes (structural prevention)
- ✅ `orch doctor --scan-inventory` reveals data classification breakdown for all scanner functions (visibility)
- ✅ Daemon self-check catches invariant violations within one poll cycle and pauses with diagnostic output (runtime detection)
- ✅ Zero production incidents from scope-expansion-without-assumption-validation defect class after all three layers deployed

---

## Decision Gate Guidance (if promoting to decision)

**Add `blocks:` frontmatter when promoting:**

This decision resolves a recurring issue (8+ prior incidents). Future agents implementing scope-expanding features might violate this pattern.

**Suggested `blocks:` keywords:**
- scanner expansion
- cross-project scanning
- workspace scanning
- daemon counter
- verification counter
- scope expansion

---

## References

**Files Examined:**
- `pkg/daemon/verification_tracker.go` — VerificationTracker with seenIDs dedup and untracked filter
- `pkg/daemon/active_count.go` — DefaultActiveCount, CombinedActiveCount, isUntrackedBeadsID
- `pkg/daemon/completion_processing.go` — listCompletedAgentsMultiProject, cross-project scanning
- `cmd/orch/daemon.go` — seedVerificationTracker, daemon poll loop
- `~/.kb/principles.md` — Defect Class Blindness, Gate Over Remind, Infrastructure Over Instruction, Coherence Over Patches

**Commands Run:**
```bash
# Show the triggering issue
bd show orch-go-x1ln

# Read fix commit
git show 190fe365d  # ihc4 fix: exclude untracked workspaces
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Established domain boundaries, drove cross-project features
- **Decision:** `.kb/decisions/2026-02-15-daemon-unified-config-construction.md` — Related: config path divergence is same meta-pattern
- **Decision:** `.kb/decisions/2026-02-14-verifiability-first-hard-constraint.md` — Verification bottleneck principle applies
- **Principle:** Defect Class Blindness (`~/.kb/principles.md`) — This investigation extends that principle to a new defect class

---

## Investigation History

**2026-03-03 10:00:** Investigation started
- Initial question: How to prevent scope-expansion features from breaking downstream consumers when they encounter unexpected production state?
- Context: 8+ incidents of the same pattern. Most recent: orch-go-ihc4 (cross-project workspace scanning found 19 untracked workspaces, inflated verification counter)

**2026-03-03 11:00:** Catalogued all 8 instances
- Each follows: scanner widens → finds unexpected data class → consumer's implicit assumptions break → fix: add filter
- Identified as defect class: "scope expansion without assumption validation"

**2026-03-03 11:30:** Substrate consultation complete
- Defect Class Blindness, Gate Over Remind, Infrastructure Over Instruction, Coherence Over Patches all point toward structural prevention + runtime detection
- Rejected: more gates (agents skip them), shadow mode (too heavy), comprehensive test fixtures (can't test for unknown data)

**2026-03-03 12:00:** Investigation completed
- Three-layer defense recommended: allowlist scanner pattern + production state inventory + daemon self-check invariants
- Key insight: invert from denylist (exclude what you don't want) to allowlist (include only what you opted into) — new data classes excluded by default
