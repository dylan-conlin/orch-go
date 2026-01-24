<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Existing two-layer dedup (session-level + TTL backup) is substrate-aligned but lacks observability for validation and monitoring.

**Evidence:** Session-level check uses OpenCode sessions (PRIMARY evidence per Evidence Hierarchy), fails open per Graceful Degradation, but doesn't emit events when API unavailable or dedup bypassed (violates Observation Infrastructure).

**Knowledge:** 6-hour window is estimate not validated by metrics; fail-open design trades correctness for availability; using Created timestamp (not Updated) is correct given "Session idle ≠ agent complete" constraint.

**Next:** Add observability layer (event emission + metrics) to validate window assumption and monitor API availability; collect baseline data for 1-2 weeks before adjusting thresholds.

**Promote to Decision:** recommend-no (tactical improvement, validates existing design but doesn't establish new architectural pattern)

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

# Investigation: Add Dedup Check Before Spawn

**Question:** What is the optimal design for preventing duplicate daemon spawns for the same beads issue?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** og-arch-add-dedup-check-16jan-6768
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Prior implementation already exists and is substrate-aligned

**Evidence:** Session-level dedup was implemented on Jan 15, 2026 with two-layer protection:
- Primary: Check OpenCode API for existing sessions with matching beads ID (6h max age)
- Backup: Extended SpawnedIssueTracker TTL from 5 minutes to 6 hours
- Fail-open design (allows spawn if API unavailable)
- Integrated into both `daemon.OnceExcluding()` and `daemon.OnceWithSlot()`

**Source:** 
- `.kb/investigations/2026-01-15-inv-implement-session-level-dedup-prevent.md`
- `pkg/daemon/session_dedup.go` (implementation)
- `pkg/daemon/daemon.go:735-750` (OnceExcluding integration)
- `pkg/daemon/daemon.go:840-855` (OnceWithSlot integration)

**Significance:** The fix was already implemented. This architect task is to review the design and validate it aligns with system principles. The two-layer approach (session check + TTL backup) follows Evidence Hierarchy principle (OpenCode sessions are PRIMARY evidence) and Graceful Degradation (fail-open when API unavailable).

---

### Finding 2: Session age calculation uses Created timestamp only

**Evidence:** Current implementation checks `s.Time.Created` against MaxAge (6 hours) to determine if a session is recent enough to block spawn. It does NOT consider `s.Time.Updated` for staleness detection.

**Source:**
- `pkg/daemon/session_dedup.go:87-92` (age calculation)
- `kb context "session deduplication"` constraint: "Session idle ≠ agent complete"

**Significance:** A session created 1 hour ago but not updated in 30 minutes might still be included in dedup check. However, the substrate constraint "Session idle ≠ agent complete" indicates agents legitimately go idle during normal operation. Using `Updated` timestamp could falsely exclude active-but-idle agents. The current design (Created-only) is correct given this constraint.

---

### Finding 3: 6-hour window is empirically chosen but not validated

**Evidence:** Both session dedup MaxAge and SpawnedIssueTracker TTL use 6 hours, described as "matching typical agent work duration" in code comments.

**Source:**
- `pkg/daemon/session_dedup.go:19` (MaxAge: 6 * time.Hour)
- `pkg/daemon/spawn_tracker.go:41` (TTL: 6 * time.Hour)
- Comment: "matching typical agent work duration"

**Significance:** The 6-hour window is an estimate, not based on observed metrics. If agents frequently run longer than 6 hours, duplicates could still occur. If agents typically complete in <2 hours, the window is unnecessarily long. This is an observability gap - the system doesn't track agent duration distributions to validate the chosen window.

---

### Finding 4: Fail-open design lacks observability

**Evidence:** When OpenCode API is unavailable, `HasExistingSession` returns false (fail-open) with no logging or metrics emission. The daemon proceeds to spawn without recording that the dedup check was bypassed.

**Source:**
- `pkg/daemon/session_dedup.go:67-76` (fail-open logic)
- No event emission or logging on API error

**Significance:** Fail-open is correct per Graceful Degradation principle, but lack of observability violates Observation Infrastructure principle ("If the system can't observe it, the system can't manage it"). When API failures cause duplicate spawns, we have no telemetry to detect the pattern or correlate duplicates with API outages.

---

## Synthesis

**Key Insights:**

1. **Two-layer dedup aligns with Evidence Hierarchy** - Using OpenCode sessions as primary check and TTL-based tracker as backup correctly implements Evidence Hierarchy principle (sessions are PRIMARY evidence of what's running). The backup layer provides resilience when primary source is unavailable.

2. **Fail-open design trades availability for correctness** - When API is unavailable, the system allows spawns to prevent blocking work. This follows Graceful Degradation but creates blind spot: we don't observe when/why dedup checks fail, violating Observation Infrastructure principle.

3. **Time window is estimate, not observation** - The 6-hour window is based on "typical agent work duration" without metrics to validate. We're guessing the right threshold rather than observing it. This is an observability gap that could lead to either false negatives (duplicates if agents run >6h) or false positives (blocking legitimate respawns if agents complete <6h but workspace exists).

4. **Session idle ≠ agent complete constrains age calculation** - Using `Created` timestamp (not `Updated`) for age calculation is correct given that agents legitimately go idle. Switching to `Updated` would falsely treat idle-but-active agents as stale.

**Answer to Investigation Question:**

The optimal design for preventing duplicate daemon spawns has three components:

1. **Primary dedup: Session-level check** - Query OpenCode API for existing sessions with matching beads ID (current: 6h window). This is PRIMARY evidence per Evidence Hierarchy.

2. **Backup dedup: TTL-based tracker** - In-memory tracking with extended TTL (current: 6h) provides protection when API unavailable. This implements Graceful Degradation.

3. **Observability layer: Event emission** - Log and emit events when dedup checks fail, when API is unavailable, or when edge cases occur. This satisfies Observation Infrastructure principle.

The current implementation (Finding 1) has components 1 and 2 correct but lacks component 3 (observability). The 6-hour window is an estimate that should be validated with metrics (Finding 3). The fail-open behavior is correct but needs telemetry (Finding 4).

---

## Structured Uncertainty

**What's tested:**

- ✅ Two-layer dedup implementation exists (verified: read session_dedup.go, spawn_tracker.go, daemon.go)
- ✅ Fail-open behavior on API error (verified: read test TestHasExistingSession_ServerError)
- ✅ Integration into both spawn paths (verified: daemon.go:735-750 and 840-855)
- ✅ Uses Created timestamp for age calculation (verified: session_dedup.go:87-92)

**What's untested:**

- ⚠️ 6-hour window matches typical agent work duration (estimate, not validated by metrics)
- ⚠️ Fail-open frequency and correlation with duplicates (no telemetry to observe)
- ⚠️ Dedup hit rate in production (no metrics to track)
- ⚠️ Session age distribution (no histogram data)

**What would change this:**

- Finding would be wrong if agents frequently run >12h (would need longer window)
- Finding would be wrong if OpenCode API frequently unavailable (fail-open might not be acceptable)
- Recommendation would change if Updated timestamp reliably distinguished dead from idle (would use it for staleness)
- Design would change if duplicate spawns caused data corruption (would need fail-closed behavior)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Keep existing two-layer dedup + add observability layer** - Preserve current session-level and TTL-based dedup, add event emission and metrics for monitoring.

**Why this approach:**
- Two-layer design is substrate-aligned (Evidence Hierarchy + Graceful Degradation)
- Primary issue is observability gap, not incorrect logic
- Adding telemetry is low-risk compared to redesigning dedup mechanism
- Enables data-driven validation of 6-hour window assumption

**Trade-offs accepted:**
- Still using estimated 6-hour window (not validated by metrics yet)
- Fail-open behavior remains (accepting duplicate risk when API down)
- Why acceptable: Observability enables validation later; fail-open prevents blocking work

**Implementation sequence:**
1. **Add event emission to session_dedup.go** - Log when API unavailable, when dedup check bypassed, when sessions found/not found
2. **Add metrics to daemon** - Track dedup hit rate, API availability, duplicate spawn rate
3. **Collect baseline data** - Run for 1-2 weeks to observe actual agent durations
4. **Validate/adjust window** - If data shows agents frequently run >6h or complete <2h, adjust MaxAge accordingly

### Alternative Approaches Considered

**Option B: Use Updated timestamp for staleness detection**
- **Pros:** Would exclude sessions that haven't updated in long time (likely dead)
- **Cons:** Violates substrate constraint "Session idle ≠ agent complete". Agents legitimately go idle during thinking, tool execution, loading. Would falsely treat active-but-idle agents as stale and allow duplicates.
- **When to use instead:** Only if we can reliably distinguish idle-but-active from dead (requires deeper session state inspection than just timestamps)

**Option C: Fail-closed (block spawn if API unavailable)**
- **Pros:** Prevents duplicates even when primary check fails
- **Cons:** Violates Graceful Degradation principle. API outage would stop all daemon spawning, blocking work even when no duplicates would occur.
- **When to use instead:** Only in systems where duplicate spawns cause data corruption or critical failures (not the case here - duplicates are inefficient but not catastrophic)

**Option D: Increase window to 12 or 24 hours**
- **Pros:** Covers longer-running agents, reduces duplicate risk
- **Cons:** Without metrics, still guessing the right threshold. Longer window means stale sessions block spawns longer. Increases false positive rate if agents typically complete quickly.
- **When to use instead:** After collecting metrics showing agents frequently run >6h

**Rationale for recommendation:** Option A addresses the observability gap (Observation Infrastructure principle) without changing the substrate-aligned logic. Options B and C violate system principles. Option D is premature optimization without data. The recommended approach enables data-driven refinement.

---

### Implementation Details

**What to implement first:**
- Add event emission to `SessionDedupChecker.HasExistingSession()` - log when API error, sessions found, sessions not found
- Add event emission to daemon.Once paths - record dedup check results (hit/miss/bypass)
- Add metrics: `dedup.api_failures`, `dedup.hits`, `dedup.misses`, `agent.duration_histogram`

**Things to watch out for:**
- ⚠️ Event emission must not block spawn path - use fire-and-forget pattern
- ⚠️ Metrics should be deduplicated by entity (beads ID), not counted by event
- ⚠️ API timeout in session_dedup is 10s - ensure this doesn't add >10s latency to spawn path
- ⚠️ Don't log on every successful dedup check (too noisy) - only log on API failures and duplicate detections

**Areas needing further investigation:**
- What is actual distribution of agent work durations? (need histogram data)
- How often does OpenCode API become unavailable? (need API availability metrics)
- Do we need different MaxAge for different skill types? (some skills may run longer)
- Should we consider session status (busy vs idle) in dedup logic? (blocked by "Session idle ≠ agent complete" constraint)

**Success criteria:**
- ✅ Events logged when API unavailable (can correlate with duplicate spawns)
- ✅ Metrics show dedup hit rate and API availability
- ✅ After 1-2 weeks, have data to validate/adjust 6-hour window
- ✅ No increase in spawn latency (event emission doesn't block)

---

## References

**Files Examined:**
- `pkg/daemon/session_dedup.go` - Session-level dedup implementation
- `pkg/daemon/session_dedup_test.go` - Test coverage for dedup logic
- `pkg/daemon/daemon.go:735-750` - OnceExcluding integration point
- `pkg/daemon/daemon.go:840-855` - OnceWithSlot integration point
- `pkg/daemon/spawn_tracker.go` - TTL-based backup tracker
- `pkg/daemon/active_count.go:141-151` - extractBeadsIDFromSessionTitle function
- `~/.kb/principles.md` - System principles for substrate consultation

**Commands Run:**
```bash
# Query substrate for session deduplication context
kb context "session deduplication"

# Find extractBeadsIDFromSessionTitle usage
grep -n "extractBeadsIDFromSessionTitle" pkg/daemon/*.go

# List daemon methods to identify spawn paths
grep -n "func (d \*Daemon)" pkg/daemon/daemon.go | head -20
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-15-inv-implement-session-level-dedup-prevent.md` - Prior implementation work
- **Decision:** `~/.kb/principles.md` (Evidence Hierarchy) - Guides primary/secondary source choices
- **Decision:** `~/.kb/principles.md` (Graceful Degradation) - Guides fail-open design
- **Decision:** `~/.kb/principles.md` (Observation Infrastructure) - Identifies observability gap

---

## Investigation History

**2026-01-16 10:00:** Investigation started
- Initial question: What is the optimal design for preventing duplicate daemon spawns for the same beads issue?
- Context: Bug fix issue orch-go-2nruy spawned with architect skill to review existing implementation

**2026-01-16 10:15:** Problem Framing complete
- Reviewed prior investigation from Jan 15 showing existing two-layer implementation
- Identified this as design validation task, not implementation task

**2026-01-16 10:30:** Exploration (Fork Navigation) complete
- Identified 5 decision forks: primary/backup strategy, time windows, fail-open design, session age calculation, integration points
- Consulted substrate (principles, constraints, decisions from kb context)

**2026-01-16 10:45:** Synthesis complete
- Navigated all forks with substrate-informed recommendations
- Key finding: Implementation is substrate-aligned but lacks observability layer
- Recommendation: Add event emission and metrics, collect data to validate 6-hour window
