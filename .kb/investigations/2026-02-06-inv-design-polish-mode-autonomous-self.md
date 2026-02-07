## Summary (D.E.K.N.)

**Delta:** Polish Mode is a daemon extension that auto-generates improvement work when the queue is empty, using 4 audit types (code quality, metadata hygiene, knowledge health, test coverage) with a priority ceiling (P3) ensuring polish never preempts real work, rate-limited to 3 issues/cycle with daemon-safe execution for all audits.

**Evidence:** Existing daemon infrastructure (poll-spawn-complete cycle, cleanup, reflection, dead session detection, grace period, sort strategies design) provides 80%+ of needed foundation. The daemon already handles idle state detection (no ready issues → sleep), capacity management, and cross-project operation. Cleanup and reflection prove the daemon can safely execute self-improvement when idle.

**Knowledge:** Polish Mode is "preventive maintenance" for the orchestration system — analogous to a manufacturing plant running diagnostics and maintenance when no production orders are queued. The key insight is that ALL polish audit types are daemon-safe because they only CREATE issues (with `triage:review` labels), never modify production code directly. Human judgment gates promotion to `triage:ready`.

**Next:** Implement Polish Mode as a new daemon periodic operation in `pkg/daemon/polish.go` with 4 audit functions, priority ceiling P3, rate limit 3 issues/cycle, and `--polish` daemon flag.

**Authority:** architectural - New daemon subsystem, affects cross-component behavior (daemon + beads + kb), establishes pattern for autonomous self-improvement

---

# Investigation: Design Polish Mode - Autonomous Self-Improvement for Orchestration

**Question:** How should Polish Mode work — autonomous self-improvement when the daemon queue is empty? What audits generate polish issues? How to rate-limit, prioritize, gate, and verify?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** Architect agent (orch-go-21414)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Daemon Autonomous Operation model | extends | Yes - poll-spawn-complete cycle confirmed in daemon_spawn.go | No |
| Daemon Guide | extends | Yes - cleanup/reflection infrastructure confirmed | No |
| Design Bead Prioritization Sorting System | extends | Yes - sort strategies + named presets pattern confirmed | No |
| Daemon Resilience decision (2026-02-06) | extends | Yes - retry/escalation/staging already implemented | No |
| Completion Workflow Guide | confirms | Yes - escalation model and auto-completion verified | No |

---

## Findings

### Finding 1: Daemon already detects idle state — "queue empty" is a well-defined moment

**Evidence:** In `daemon_spawn.go:479-499`, `Run()` processes issues in a loop until `!result.Processed` (queue empty). In the main poll loop, when `bd ready` returns zero `triage:ready` issues, the daemon simply sleeps. This is the natural insertion point for polish work — when the daemon would otherwise do nothing.

**Source:** `pkg/daemon/daemon_spawn.go:490-492`

**Significance:** No new "idle detection" needed. The existing condition `!result.Processed` after filtering for `triage:ready` issues is the trigger. Polish mode activates when this condition is true AND the daemon has available capacity.

---

### Finding 2: Daemon periodic operations provide the exact pattern needed

**Evidence:** The daemon already runs periodic self-improvement operations:
- `RunPeriodicReflection()` — runs `kb reflect` to surface synthesis opportunities
- `RunPeriodicCleanup()` — archives stale sessions, workspaces, investigations
- `RunDeadSessionDetection()` — finds and escalates stuck agents
- `RunServerRecovery()` — detects and recovers from server restarts

Each follows the pattern: `ShouldRunX()` → `RunX()` → log result. All are interval-gated and idempotent.

**Source:** `pkg/daemon/daemon_periodic.go:14-77`, `pkg/daemon/cleanup.go`

**Significance:** Polish Mode fits naturally as another periodic operation. Same pattern: `ShouldRunPolish()` → `RunPolish()` → create issues with `triage:review` labels. The pattern is proven, tested, and understood.

---

### Finding 3: The `triage:review` label provides an existing human gate

**Evidence:** The daemon's triage workflow already distinguishes `triage:ready` (daemon can auto-spawn) from `triage:review` (needs orchestrator review). Polish-generated issues should use `triage:review` by default — they enter the system as proposals, not mandates. The orchestrator reviews and either promotes to `triage:ready` or closes as not-needed.

**Source:** Daemon Guide, Triage Labels section; `pkg/daemon/daemon_queue.go` label filtering

**Significance:** Polish Mode's safety comes from the existing triage gate. Polish audits ONLY create issues — they never modify code, never auto-spawn agents, never bypass human review. The human gate is the `triage:review` → `triage:ready` promotion step.

---

### Finding 4: The existing cleanup, reflection, and hotspot infrastructure covers 2 of 4 audit types

**Evidence:**
- **Cleanup** (`pkg/daemon/cleanup.go`) already archives stale sessions, workspaces, and empty investigations — this IS metadata hygiene.
- **Reflection** (`pkg/daemon/reflect.go`) already surfaces synthesis opportunities via `kb reflect` — this IS knowledge health.
- **Hotspot detection** (`pkg/daemon/hotspot.go`) already identifies fix-density hotspots — this IS code quality signal.

What's MISSING:
- Generating beads issues from cleanup/reflection findings (currently only logs)
- Test coverage audit (no existing infrastructure)
- Metadata completeness checks (missing area labels, missing types, etc.)

**Source:** `pkg/daemon/cleanup.go`, `pkg/daemon/reflect.go`, `pkg/daemon/hotspot.go`

**Significance:** ~50% of polish audits are already computed but not converted into trackable issues. The main work is converting existing signals into beads issues, not building new detection infrastructure.

---

### Finding 5: Priority system supports ceiling concept natively

**Evidence:** Beads issues have a numeric `Priority` field (lower = higher priority). The daemon sorts by priority and processes highest-priority issues first. Setting all polish-generated issues to P3 (or a custom "polish" priority tier) ensures they sort below any human-created P0/P1/P2 work. The sort strategies design (prioritization investigation) already handles multi-dimension sorting.

**Source:** `pkg/daemon/daemon_queue.go:49` (priority sort), beads priority field in `pkg/beads/types.go`

**Significance:** Priority ceiling is trivially implementable. Polish issues at P3 will never preempt P0-P2 real work. If any real work arrives in the queue, it naturally sorts above polish work.

---

## Decision Forks

### Fork 1: What audits should generate polish issues?

**Options:**
- A: 4 audit types (code quality, metadata hygiene, knowledge health, test coverage)
- B: 2 audit types (code quality + metadata) — minimal viable
- C: 6+ audit types (add security, performance, accessibility, documentation)

**Substrate says:**
- Principle: Compose Over Monolith — small focused audits, each does one thing
- Principle: Graceful Degradation — start with what works, add later
- Model: Daemon periodic operations — each is independent, interval-gated
- Finding 4: 2 of 4 audits already have infrastructure

**RECOMMENDATION:** Option A — 4 audit types, phased implementation. Start with the 2 that have infrastructure (metadata hygiene, knowledge health), add code quality and test coverage in later phases.

**Trade-off accepted:** Not exhaustive. Security, performance, etc. can be added later as new audit modules.

**When this would change:** If even 4 types generate too much noise, reduce to 2.

**The 4 audit types:**

| Audit Type | What It Checks | Issue Generated | Existing Infrastructure |
|-----------|----------------|-----------------|------------------------|
| **Metadata Hygiene** | Issues missing area labels, missing types, stale open issues (>30 days, no activity) | `bd create "Issue X missing area label" --type chore -l triage:review -l polish:metadata` | Partial — cleanup already detects stale items |
| **Knowledge Health** | Investigation clusters without synthesis, stale decisions, empty investigations | `bd create "10+ investigations on 'daemon' — needs synthesis" --type task -l triage:review -l polish:knowledge` | Yes — `kb reflect --type synthesis` |
| **Code Quality** | Files with 5+ fix commits (hotspots), functions >300 lines, duplicate investigation topics | `bd create "Hotspot: status.go has 8 fix commits" --type task -l triage:review -l polish:quality` | Partial — hotspot.go exists |
| **Test Coverage** | Go packages with 0% test coverage, test files not updated in 60+ days | `bd create "pkg/X has no tests" --type task -l triage:review -l polish:tests` | No — needs `go test -cover` parsing |

---

### Fork 2: How to rate-limit polish issue generation?

**Options:**
- A: Fixed limit per poll cycle (e.g., max 3 issues per cycle)
- B: Fixed limit per time window (e.g., max 10 issues per day)
- C: Exponential backoff (fewer issues if previous polish issues aren't actioned)
- D: No limit (generate all detected issues)

**Substrate says:**
- Principle: Verification Bottleneck — system cannot change faster than human can verify
- Principle: Pressure Over Compensation — create gentle pressure, not flood
- Model: Daemon rate limiter — `pkg/daemon/rate_limiter.go` already implements hourly spawn limits
- Constraint: Dylan reviews issues manually

**RECOMMENDATION:** Option A with elements of B — Max 3 issues per polish cycle, max 10 per day. Simple, predictable. Matches the existing rate limiter pattern in the daemon. 3 issues is reviewable in <5 minutes. 10/day prevents overnight floods.

**Trade-off accepted:** May miss some issues on busy audit days. Acceptable because polish is continuous — missed items will surface next cycle.

**When this would change:** If polish issues are consistently high quality and always actioned, remove daily cap.

---

### Fork 3: Priority ceiling — how to ensure polish never preempts real work?

**Options:**
- A: Fixed priority P3 for all polish issues
- B: Custom priority tier "P-polish" (new priority level below P2)
- C: Label-based filtering — daemon skips polish issues when real work exists
- D: Priority P3 + daemon stops spawning polish when any `triage:ready` real work exists

**Substrate says:**
- Principle: Infrastructure Over Instruction — enforce via mechanism, not reminder
- Model: Daemon queue — sorts by priority, processes highest first
- Finding 5: Priority system handles ceiling natively

**RECOMMENDATION:** Option D — Priority P3 + daemon pauses polish spawning when any `triage:ready` real work exists. P3 ensures sort order, but the spawn pause is the real gate. If real work arrives mid-polish, daemon doesn't spawn more polish agents (existing ones finish naturally).

Implementation:
```go
func (d *Daemon) ShouldRunPolish() bool {
    // Only polish when queue is empty of real work
    realWorkCount := d.countReadyIssuesExcluding("polish:")
    if realWorkCount > 0 {
        return false
    }
    // Plus standard interval/capacity checks
    return d.polishEnabled && time.Since(d.lastPolish) >= d.Config.PolishInterval
}
```

**Trade-off accepted:** Polish agents that are already running continue even when real work arrives. Acceptable because: (a) polish work is small/fast, (b) they occupy at most 1-2 capacity slots, (c) killing running agents is worse than waiting.

**When this would change:** If polish agents regularly take >1 hour and block urgent work, add preemption.

---

### Fork 4: Which polish is daemon-safe vs needs human gate?

**Options:**
- A: All polish creates issues only (never executes directly)
- B: Some polish executes directly (cleanup, label fixes), others create issues
- C: All polish executes directly via daemon-spawned agents

**Substrate says:**
- Principle: Verification Bottleneck — changes need human verification
- Principle: Gate Over Remind — gates prevent unverified changes
- Model: Daemon triage workflow — `triage:review` is the human gate
- Constraint: "Ask 'should we' before 'how do we'" — polish proposes, doesn't act

**RECOMMENDATION:** Option A — All polish creates issues only. The daemon runs audits (read-only analysis), generates issue descriptions, and creates beads issues with `triage:review` labels. No polish audit modifies code, files, or system state. This makes all audits daemon-safe by definition.

The existing cleanup operations (session archival, workspace archival) are NOT polish — they're infrastructure maintenance and remain separate.

**Trade-off accepted:** Even obvious fixes (e.g., adding a missing area label) require human review. This is intentional friction — it ensures quality and prevents autonomous drift.

**When this would change:** If review burden becomes too high and polish quality is consistently good (>90% acceptance rate over 30 days), consider auto-executing trivial polish (label fixes).

---

### Fork 5: How to verify improvements?

**Options:**
- A: Standard completion verification (existing `orch complete` gates)
- B: Before/after metrics (measure improvement)
- C: Spot-check review by orchestrator
- D: No verification beyond standard completion

**Substrate says:**
- Principle: Verification Bottleneck — don't create more changes than can be verified
- Principle: Observation Infrastructure — if you can't observe it, you can't manage it
- Model: Completion workflow — existing 3-layer verification

**RECOMMENDATION:** Option A + lightweight metrics — Standard `orch complete` for polish agents that get promoted to `triage:ready` and spawned. Additionally, track polish metrics:
- Issues generated per audit type
- Acceptance rate (promoted to `triage:ready` / total generated)
- Completion rate of promoted polish issues

These metrics feed back into audit tuning — low-acceptance-rate audits get tuned or disabled.

**Trade-off accepted:** No before/after measurement of "system health improvement." Acceptable because polish is continuous, not a one-time fix. Cumulative effect is visible through existing dashboards.

**When this would change:** If Dylan wants to measure ROI of polish mode, add a health score.

---

### Fork 6: Where in the daemon poll loop does polish execute?

**Options:**
- A: After all other periodic operations, only if queue empty
- B: As a separate loop/goroutine
- C: Replace idle sleep with polish audit

**Substrate says:**
- Principle: Compose Over Monolith — polish is another periodic op, not a new loop
- Model: Daemon periodic operations — sequential execution within poll cycle
- Finding 2: Existing pattern is `ShouldRunX()` → `RunX()`

**RECOMMENDATION:** Option A — After all other periodic operations, only when queue is empty. This means:

```
Daemon Poll Cycle:
1. CheckServerHealth()
2. ReconcileWithOpenCode()
3. CompletionOnce()
4. RunPeriodicReflection()
5. RunPeriodicCleanup()
6. RunDeadSessionDetection()
7. RunServerRecovery()
8. WriteDaemonStatus()
9. Check capacity → spawn ready issues
10. IF no ready issues spawned AND capacity available:
    → RunPolishAudits()  ← NEW
11. Sleep for poll interval
```

**Trade-off accepted:** Polish audits add to poll cycle time. Acceptable because audits are read-only and fast (grep, count, compare). Issue creation via `bd create` is the most expensive operation (~100ms each).

**When this would change:** If audits become computationally expensive, move to separate goroutine (Option B).

---

### Fork 7: How to prevent duplicate polish issues?

**Options:**
- A: Check existing open issues before creating (dedup by title/content)
- B: Use `polish:*` labels + `bd search` to find existing polish issues
- C: Track created polish issues in daemon memory with TTL
- D: Rely on human review to close duplicates

**Substrate says:**
- Principle: Observation Infrastructure — prevent noise, not just observe it
- Model: Daemon spawn tracker — existing dedup via `SpawnedIssueTracker`
- Finding: Daemon already has `ProcessedCache` for dedup

**RECOMMENDATION:** Option B + C — Label-based dedup plus in-memory tracking. Before creating a polish issue:
1. `bd search "missing area label issue-X"` with `polish:metadata` label filter
2. If similar open issue exists, skip
3. Track created issues in `polishCache` (TTL = 24 hours) to avoid re-checking within same day

**Trade-off accepted:** Not perfect dedup. Slightly different descriptions may pass through. Acceptable because human review catches duplicates, and `triage:review` prevents auto-execution.

**When this would change:** If duplicate rate exceeds 20%, add content hashing for dedup.

---

## Synthesis

**Key Insights:**

1. **Polish Mode is "preventive maintenance" — it fills idle time productively.** The manufacturing analogy is precise: when the production floor has no orders, the team runs equipment diagnostics, cleans tooling, and organizes inventory. Polish Mode does the same — metadata hygiene, knowledge synthesis, code quality checks, test coverage gaps.

2. **Safety comes from the existing triage gate, not from audit restrictions.** All polish audits are read-only (analyze + create issue). The `triage:review` label ensures no polish work executes without human promotion. This makes the entire system daemon-safe without needing per-audit safety classification.

3. **~50% of polish infrastructure already exists.** Cleanup detects stale items, reflection surfaces synthesis needs, hotspot detection identifies code quality issues. The gap is converting these signals into trackable beads issues. Polish Mode is mostly a "signal-to-issue converter."

4. **Priority ceiling + spawn pause is the correct double gate.** P3 priority ensures sort order (polish issues sort below real work). The spawn pause ensures the daemon doesn't waste capacity on polish when real work exists. Together they guarantee polish never preempts production work.

5. **Rate limiting prevents review burden.** 3 issues per cycle, 10 per day. This is reviewable in <5 minutes. The daemon runs continuously — anything missed surfaces next cycle. There's no urgency to polish.

**Answer to Investigation Question:**

Polish Mode should be implemented as a new daemon periodic operation (`pkg/daemon/polish.go`) with 4 read-only audit types that create beads issues labeled `triage:review polish:*`. The daemon runs audits only when the `triage:ready` queue is empty, rate-limited to 3 issues per cycle and 10 per day, at priority P3. All audits are daemon-safe because they only create issues, never modify code. Human review gates promotion to `triage:ready`. Verification uses existing `orch complete` infrastructure plus acceptance rate tracking for audit tuning.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon detects empty queue via `!result.Processed` (verified: `daemon_spawn.go:490-492`)
- ✅ Periodic operations pattern works for self-improvement (verified: reflection + cleanup in `daemon_periodic.go`)
- ✅ `triage:review` label prevents auto-spawning (verified: daemon queue filtering in `daemon_queue.go`)
- ✅ Priority sorting handles P3 ceiling (verified: `daemon_queue.go:49`)
- ✅ Hotspot detection exists (verified: `pkg/daemon/hotspot.go`)
- ✅ kb reflect surfaces synthesis needs (verified: `pkg/daemon/reflect.go`)

**What's untested:**

- ⚠️ Whether 4 audit types generate enough signal to be useful (hypothesis based on manual observation)
- ⚠️ Whether 3 issues/cycle is the right rate limit (needs tuning after deployment)
- ⚠️ Whether label-based dedup is sufficient to prevent floods (not benchmarked)
- ⚠️ Whether acceptance rate tracking is worth the implementation cost (metrics TBD)
- ⚠️ Whether test coverage audit via `go test -cover` is fast enough for daemon poll cycle

**What would change this:**

- If polish issues are consistently rejected (>50%), the audit generating them should be disabled or retuned
- If polish issues generate more noise than signal, reduce to 2 audit types (metadata + knowledge only)
- If the daemon poll cycle becomes too slow (>5s), move audits to a separate goroutine
- If Dylan wants auto-execution of trivial polish, the safety model changes from "create issue only" to "execute + verify"

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Polish Mode as daemon periodic op | architectural | New subsystem in daemon, affects cross-component behavior |
| 4 audit types with phased rollout | implementation | Audit implementations are self-contained modules |
| Priority P3 ceiling | implementation | Uses existing priority system |
| Rate limit 3/cycle, 10/day | implementation | Configuration within daemon scope |
| `triage:review` as human gate | implementation | Uses existing label convention |
| Acceptance rate metrics | architectural | New observability concern, affects dashboard |

### Recommended Approach ⭐

**Polish Mode as Daemon Periodic Operation** — A new module in `pkg/daemon/polish.go` that runs read-only audits when the queue is empty, creating beads issues at P3 priority with `triage:review polish:*` labels.

**Why this approach:**
- Reuses existing daemon patterns (periodic operations, triage labels, priority sorting)
- All audits are inherently safe (read-only analysis + issue creation)
- Priority ceiling + spawn pause double-gate prevents interference with real work
- Rate limiting prevents review burden
- ~50% of infrastructure already exists (cleanup, reflection, hotspot detection)

**Trade-offs accepted:**
- Polish requires human review (no auto-execution) — ensures quality but adds review overhead
- Not exhaustive (4 audit types) — extensible later
- Priority P3 is coarse (all polish same priority) — can add sub-priorities later

**Implementation sequence:**

1. **Phase 1: Infrastructure** (1-2 hours)
   - Create `pkg/daemon/polish.go` with `PolishConfig`, `PolishResult`, `ShouldRunPolish()`, `RunPolish()`
   - Add `--polish` flag to daemon CLI
   - Add polish rate limiter (3/cycle, 10/day)
   - Add dedup via `polish:*` label search

2. **Phase 2: Metadata Hygiene Audit** (1 hour)
   - Check issues missing area labels
   - Check issues with null/empty type
   - Check stale open issues (>30 days, no comments)
   - Create issues with `polish:metadata` label

3. **Phase 3: Knowledge Health Audit** (1 hour)
   - Integrate with existing `kb reflect --type synthesis` output
   - Check for empty/unfilled investigation files
   - Check for stale decisions (>90 days, never cited)
   - Create issues with `polish:knowledge` label

4. **Phase 4: Code Quality Audit** (1-2 hours)
   - Integrate with existing hotspot detection
   - Add large-function detection (>300 lines)
   - Add repeated-investigation-topic detection
   - Create issues with `polish:quality` label

5. **Phase 5: Test Coverage Audit** (1-2 hours)
   - Parse `go test -cover` output
   - Identify 0% coverage packages
   - Identify stale test files (>60 days without modification)
   - Create issues with `polish:tests` label

6. **Phase 6: Metrics & Tuning** (1 hour)
   - Track issues generated per audit type
   - Track acceptance rate (promoted / total)
   - Add polish section to `orch daemon preview`
   - Add polish section to daemon status output

### Alternative Approaches Considered

**Option B: Polish as separate process (not daemon)**
- **Pros:** No daemon modification, can run independently
- **Cons:** Duplicates daemon infrastructure (polling, capacity, beads integration). Misses the key trigger (queue empty). Adds operational complexity (two processes to manage).
- **When to use instead:** If daemon becomes too complex (>2000 lines in daemon_periodic.go)

**Option C: Polish executed directly (no issues, just fixes)**
- **Pros:** Faster improvement cycle, no review overhead
- **Cons:** Violates Verification Bottleneck principle. Autonomous code changes without human review risk system degradation. The 347-commit spiral was exactly this — autonomous changes without verification.
- **When to use instead:** Only for trivially safe operations (label additions) after proving >90% acceptance rate over 30 days

**Rationale for recommendation:** Polish Mode as a daemon periodic operation provides maximum safety (read-only + triage:review gate) with minimum infrastructure cost (reuses 80%+ of existing daemon patterns). The phased implementation allows early validation before investing in all 4 audit types.

---

### Implementation Details

**What to implement first:**
- `pkg/daemon/polish.go` — Core module with `ShouldRunPolish()` and `RunPolish()`
- Metadata Hygiene audit — highest signal (obvious issues with concrete fixes)
- Knowledge Health audit — leverages existing `kb reflect` infrastructure

**Things to watch out for:**
- ⚠️ `bd create` invocations are ~100ms each — with 3 issues/cycle, adds ~300ms to poll loop (acceptable)
- ⚠️ `bd search` for dedup may be slow on large issue databases — add timeout
- ⚠️ `go test -cover` for test audit is expensive (~5-10s per package) — run only every 6 hours, not every poll
- ⚠️ Polish labels (`polish:metadata`, etc.) need to be consistent — consider a `PolishLabel` type
- ⚠️ Cross-project polish needs project-specific audit context — run audits per-project

**Areas needing further investigation:**
- How to detect "stale decisions" — need citation tracking in kb
- Whether test coverage threshold should be 0% or configurable
- Whether polish should have its own capacity allocation (e.g., max 1 polish agent at a time)
- How to surface polish metrics in dashboard

**Success criteria:**
- ✅ `orch daemon preview` shows "Polish: X audits ready, Y issues would be created"
- ✅ `orch daemon run --polish` generates issues when queue is empty
- ✅ Polish issues have `triage:review polish:*` labels and P3 priority
- ✅ No polish issues created when `triage:ready` real work exists
- ✅ Rate limit enforced (max 3/cycle, 10/day)
- ✅ Existing daemon tests still pass (no regression)

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision establishes the pattern for autonomous self-improvement
- Future audit type additions should reference this design
- Daemon behavior changes should check whether they conflict with polish mode

**Suggested blocks keywords:**
- "polish mode"
- "daemon idle"
- "autonomous improvement"
- "preventive maintenance"
- "self-improvement"

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` — Main daemon package structure
- `pkg/daemon/daemon_spawn.go` — Run loop, capacity management, queue empty detection
- `pkg/daemon/daemon_periodic.go` — Periodic operations pattern (reflection, cleanup, recovery)
- `pkg/daemon/daemon_queue.go` — Issue filtering, priority sorting, label-based selection
- `pkg/daemon/cleanup.go` — Stale session/workspace archival infrastructure
- `pkg/daemon/reflect.go` — kb reflect integration
- `pkg/daemon/hotspot.go` — Hotspot detection interface
- `pkg/daemon/rate_limiter.go` — Spawn rate limiting
- `pkg/daemon/spawn_tracker.go` — Spawn dedup tracking
- `pkg/daemon/grace_period.go` — Grace period for staging

**Commands Run:**
```bash
# Check daemon package structure
ls pkg/daemon/*.go

# Search for idle/empty queue handling
grep -r "queue.*empty\|no.*ready\|idle\|polish" pkg/daemon/

# Review existing periodic operations
cat pkg/daemon/daemon_periodic.go | head -80

# Review cleanup infrastructure
cat pkg/daemon/cleanup.go
```

**Related Artifacts:**
- **Model:** `.kb/models/daemon-autonomous-operation.md` — Daemon poll-spawn-complete cycle
- **Guide:** `.kb/guides/daemon.md` — Daemon operational guide
- **Guide:** `.kb/guides/completion.md` — Completion and verification workflow
- **Investigation:** `.kb/investigations/2026-02-06-inv-design-bead-prioritization-sorting-system.md` — Sort strategies design
- **Decision:** `.kb/decisions/2026-02-06-daemon-resilience-retry-staging-model-routing.md` — Daemon resilience patterns

---

## Investigation History

**2026-02-06 23:30:** Investigation started
- Initial question: How should Polish Mode work for autonomous self-improvement when queue empty?
- Context: Daemon needs productive idle behavior — currently sleeps when no ready issues exist

**2026-02-06 23:45:** Problem framing complete
- Defined 5 key design questions from spawn context
- Identified manufacturing "preventive maintenance" as the right analogy
- Gathered context from daemon model, guide, completion guide, recent decisions

**2026-02-07 00:00:** Exploration complete — 7 forks identified
- All forks navigable with available substrate (principles, models, existing daemon code)
- Key finding: ~50% of polish infrastructure already exists in daemon periodic operations
- Key finding: ALL audits are daemon-safe because they only create issues, never modify code

**2026-02-07 00:15:** Synthesis complete
- Recommended phased implementation starting with metadata hygiene and knowledge health audits
- Priority ceiling P3 + spawn pause provides double gate against preemption
- Rate limit 3/cycle, 10/day prevents review burden
- Status: Complete
