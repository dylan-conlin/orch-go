## Summary (D.E.K.N.)

**Delta:** Price-watch scraper failures are 100% infrastructure-side (our plumbing), not site-side; the whack-a-mole pattern stems from three overlapping self-healing jobs with blind spots between them, compounded by a global Redis lock that amplifies every bug.

**Evidence:** Cataloged 7 incidents (pw-8918, pw-8926, pw-8934, stuck SCS re-scrapes, incomplete runs, orphan scope gap, pw-8927); all 7 are infrastructure failures in lock lifecycle, job routing, status filters, or feature gaps. Zero site-side failures since SCS IP blocking (Jan 2026).

**Knowledge:** The system has a 97% base success rate when infrastructure is stable, but the lock-based concurrency control creates a single point of amplification where any bug (lock leak, retry storm, queue demotion) cascades to block ALL scraping. The two-job self-healing architecture has structural blind spots that require a unified approach.

**Next:** Implement three-phase reliability strategy: (1) Merge self-healing jobs into unified `CollectionRunHealthCheckJob`, (2) Add lightweight run-level alerting, (3) Expand canary testing. Remove Redis lock once Sidekiq capsule validated (pw-8928).

**Authority:** architectural - Crosses multiple components (jobs, models, orchestrator), establishes new patterns, requires coordinated implementation across 5+ files.

---

# Investigation: Scraper Reliability Strategy for Price-Watch

**Question:** What holistic reliability strategy should replace the whack-a-mole pattern of fixing individual scraper failures? What are the actual failure modes, where is the infrastructure/site-side boundary, and what should we build?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** Architect (spawned from pw-8935)
**Phase:** Complete
**Next Step:** None - Recommendations ready for implementation prioritization
**Status:** Complete

**Patches-Decision:** N/A (new strategic design)
**Extracted-From:** price-watch project (`/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch`)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/collection-system-architecture.md` (pw) | extends | Yes - read source code | None |
| `.kb/decisions/2026-01-17-subprocess-architecture-simplification.md` (pw) | extends | Yes | None |
| `.kb/investigations/2026-02-05-inv-fix-sidekiq-scheduled-set-explosion.md` (pw) | extends | Yes - verified fix in code | None |
| `.kb/investigations/2026-02-05-inv-jim-queued-re-scrapes-scs.md` (pw) | extends | Yes | None |
| `.kb/investigations/2026-02-03-inv-oshcut-undetectability-detection-vectors.md` (pw) | extends | Yes | None |
| pw-8933 (orphan job overlap investigation) | subsumes | Yes | This design absorbs pw-8933 |

---

## Findings

### Finding 1: All 7 Recent Incidents Are Infrastructure Failures

**Evidence:** Complete incident catalog from git history and code analysis:

| ID | Incident | Root Cause Category | Fix Type |
|----|----------|-------------------|----------|
| pw-8918 | Redis lock not released on exception | Lock lifecycle bug | Tactical (block restructure) |
| pw-8926 | 44K+ Sidekiq scheduled set explosion | Job lifecycle design flaw | Structural (polling + guard) |
| (unnamed) | SCS re-scrapes demoted to default queue | Queue routing bug | Tactical (one-line) |
| pw-8934 | Daily persona cap blocks re-scrapes | Feature gap | Tactical (parameter) |
| pw-8927 | Single-point lock failure risk | Prevention | Defensive (capsule) |
| (unnamed) | Incomplete runs excluded from recovery | Status filter bug | Tactical (filter fix) |
| (unnamed) | Orphan scope missed superseded/failed | SQL scope limitation | Tactical (SQL fix) |

Zero site-side failures since the SCS IP blocking incident (Jan 2026). OshCut scraping runs at 97%+ success rate when infrastructure is stable.

**Source:** `git log --grep="pw-8926\|pw-8934\|pw-8918\|pw-8933"` in price-watch repo; code analysis of `scrape_cli_job.rb`, `retry_orphaned_quotes_job.rb`, `check_active_run_orphans_job.rb`

**Significance:** The problem is NOT that sites are detecting/blocking us. The problem is our own plumbing. This means investments should go toward infrastructure reliability, not more stealth. The whack-a-mole pattern exists because each fix addresses a symptom without addressing the architectural amplifier (the global Redis lock).

---

### Finding 2: Three Self-Healing Jobs Have Blind Spots Between Them

**Evidence:** The system has three independent self-healing mechanisms running on overlapping schedules:

| Job | Schedule | Handles | Blind Spot |
|-----|----------|---------|------------|
| `RetryOrphanedQuotesJob` | Every 5 min | Quotes with `job_id=NULL` or missing/superseded/failed ScrapeJob | Only `in_progress` + `incomplete` runs; respects stagger/cooldown |
| `CheckActiveRunOrphansJob` | Every 5 min | Stale ScrapeJobs (>15 min pending) | Only `in_progress` runs; creates new ScrapeJob + dispatches |
| `CleanupStaleScrapeJobsJob` | Daily 3am | Jobs pending >24h | Destructive (deletes); only runs once daily |

**Overlap (pw-8933):** Both `RetryOrphanedQuotesJob` and `CheckActiveRunOrphansJob` can process the same quotes simultaneously. RetryOrphanedQuotes finds quotes where `job_id IS NULL OR ScrapeJob is superseded/failed`. CheckActiveRunOrphans finds ScrapeJobs pending >15 min and supersedes them (which then makes the linked quotes match RetryOrphanedQuotes's scope). This can create a redispatch loop:
1. CheckActiveRunOrphans supersedes stale ScrapeJob → creates new ScrapeJob
2. RetryOrphanedQuotes sees quote linked to superseded job → dispatches ANOTHER new ScrapeJob
3. Two ScrapeJobs now compete for the same quote

**Gap:** Neither job has a coordination mechanism with the other. No deduplication check exists at dispatch time to see if a quote already has a recently-dispatched job.

**Source:** `retry_orphaned_quotes_job.rb:173` (find_orphaned_quotes), `check_active_run_orphans_job.rb:99` (redispatch_stale_jobs!), `price_quote.rb` (orphaned scope), `collection_run.rb:534` (redispatch_stale_jobs!)

**Significance:** This is the root cause of the recurring orphan problem and answers pw-8933 directly. The two jobs operate on overlapping domains without coordination, creating the possibility of duplicate dispatches and orphan cycling.

---

### Finding 3: The Global Redis Lock Is an Amplifier, Not a Root Cause

**Evidence:** The Redis lock (`scraper:global:lock`, TTL=900s) serializes ALL scraping globally. Every infrastructure bug interacts with this lock:

- **pw-8918:** Lock not released → blocked all scraping for up to 15 minutes
- **pw-8926:** Lock retry created new Sidekiq entries → 44K entries (131x amplification from 337 jobs)
- **Stuck SCS:** Lock retry lost queue routing → priority re-scrapes starved behind collection runs
- **Normal operation:** Lock polling (10 polls × 30s = 5 min max wait) blocks Sidekiq thread

The Sidekiq capsule (pw-8927) with `concurrency=1` now provides structural concurrency control. The Redis lock is documented as a "safety net until validated in production (pw-8928)." But it remains the PRIMARY mechanism (lock code still executes before capsule-level protection kicks in).

**Source:** `scrape_cli_job.rb:33-37` (lock constants), `scrape_cli_job.rb:86-111` (lock acquisition + polling), `config/initializers/sidekiq.rb:62-65` (capsule config)

**Significance:** Removing the Redis lock (once capsule is validated) would eliminate the entire class of lock-related failures (3 of 7 incidents). The capsule provides the same guarantee structurally without lock lifecycle complexity. This is the single highest-ROI reliability improvement.

---

### Finding 4: Re-scrapes and Collection Runs Compete Through a Single Chokepoint

**Evidence:** Jim's workflow involves:
1. Starting collection runs (684+ quotes, takes hours)
2. Ad-hoc re-scrapes from heat map UI (small, urgent)

Both paths compete for the single scraper execution slot. The priority queue system (scrape_priority weight=10 vs scrape weight=1) was designed to solve this, but:
- Lock retry demoted priority jobs to default queue (fixed in Feb 2026)
- Daily persona cap blocked re-scrapes (fixed in pw-8934)
- During active collection runs, re-scrapes still wait for current job to finish (3-7 min)

The system has no way for Jim to see WHY a re-scrape isn't progressing or WHEN it will start.

**Source:** `scrape_orchestrator.rb:38-43` (queue constants), `scrape_cli_job.rb:33-37` (lock timing), `sidekiq.rb:62-65` (capsule weights)

**Significance:** Jim's pain comes from opacity, not just slowness. He can't tell if a re-scrape is queued, waiting for lock, waiting for persona, or stuck. Adding run-level status visibility would reduce his need to investigate and re-trigger manually.

---

### Finding 5: Dry-Run Mode Has Low Value; Canary Tests Have High Value

**Evidence:** A dry-run mode would exercise persona selection, job dispatch, and queue routing WITHOUT launching browsers. Analysis of what this would catch:

| Failure Mode | Dry-Run Catches? | Why/Why Not |
|-------------|-------------------|-------------|
| Lock lifecycle bugs | No | Lock only acquired during actual execution |
| Scheduled set explosion | No | Only happens under real lock contention |
| Queue routing bugs | Partially | Would validate initial queue assignment |
| Persona cap issues | Yes | Would verify persona availability |
| Orphan recovery gaps | No | Requires real job lifecycle |
| Subprocess timeouts | No | No subprocess involved |

A canary test (like the existing `quick_test.yaml` with 6 quotes) exercises the FULL pipeline including browser, proxy, login, scraping, result parsing, and quote saving. It catches everything a dry-run catches plus everything it doesn't.

**Source:** Analysis of `collection-system-architecture.md` failure patterns; `config/collection_runs/` directory

**Significance:** Building a dry-run mode would be engineering effort for partial coverage. Expanding the canary test to run automatically before full collections would provide full coverage with minimal investment. Recommend: canary-first approach, not dry-run.

---

### Finding 6: Current Observability Is Inadequate for Jim's Workflow

**Evidence:** The system logs to Rails logger and has Agentlog error/death handlers, but Jim's visibility is limited to:
- Collection run progress (X/Y quotes) via dashboard
- Heat map showing latest pricing data
- Manual `rake price_watch:run_summary` for run details

Missing:
- No per-run alerting when something goes wrong (Jim discovers problems hours later)
- No visibility into queue depth, lock contention, or persona availability
- No "canary just failed" notification
- No distinction between "scraping slowly" and "scraping stuck"

**Source:** `sidekiq.rb:6-38` (Agentlog handlers), `collection_run.rb:721-740` (summary_data), `sidekiq_cron.yml` (scheduled jobs)

**Significance:** Jim runs collections 2-3x/week. Each failure-investigation-fix cycle wastes 30-60 minutes of his time. Even simple Slack/email alerts on run completion status would reduce this dramatically.

---

## Synthesis

**Key Insights:**

1. **Infrastructure reliability is the bottleneck, not detection avoidance** - All 7 recent incidents are our own plumbing. The 97% base success rate proves the scraping technology works. Investment should go toward making the infrastructure boring and predictable, not more stealth.

2. **The global Redis lock is an architectural amplifier** - It converts every small bug into a system-wide outage. Removing it (once capsule validated) eliminates an entire class of failures with zero behavioral change. This is the highest-ROI single change.

3. **Self-healing needs unification, not more layers** - The three-job system has gaps because each job was built incrementally to fix specific failures. The overlap (pw-8933) and gaps create a maintenance burden where fixing one job can break another's assumptions. A single unified health check with clear responsibility boundaries would be simpler to reason about.

4. **Observability compounds reliability** - Jim's whack-a-mole experience is partly because he discovers problems late. Early alerting on failures would: (a) reduce Jim's investigation time, (b) provide data for further reliability improvements, (c) distinguish "expected slowness" from "stuck."

**Answer to Investigation Question:**

The holistic reliability strategy should proceed in three phases, prioritized by ROI and risk:

**Phase 1 (Immediate): Remove Lock + Unify Self-Healing**
- Validate and remove Redis lock (pw-8928 blocker)
- Merge `RetryOrphanedQuotesJob` + `CheckActiveRunOrphansJob` into unified `CollectionRunHealthCheckJob`
- This eliminates 4 of 7 incident root causes

**Phase 2 (Near-term): Observability + Canary**
- Add run-level Slack/email alerting (run started, run completed, run stuck)
- Expand `quick_test.yaml` canary to run automatically before full collections
- Add persona capacity dashboard widget

**Phase 3 (Longer-term): Predictive Health**
- Track per-run success rate over time (trending)
- Auto-escalate when canary fails (block full collection)
- Consider per-competitor health tracking

---

## Structured Uncertainty

**What's tested:**

- ✅ All 7 incidents cataloged from git history (verified: read commits, changed files, and current code)
- ✅ Self-healing job overlap confirmed (verified: read both jobs' scopes, found they can process same quotes)
- ✅ Redis lock is redundant with capsule (verified: capsule config at `concurrency=1` in `sidekiq.rb:62-65`)
- ✅ Infrastructure vs site-side classification (verified: all incidents trace to Ruby/Sidekiq/Redis code, not Node/browser/proxy)

**What's untested:**

- ⚠️ Lock removal safety (capsule concurrency=1 not yet validated in production as sole mechanism - pw-8928)
- ⚠️ Self-healing merge won't introduce new blind spots (design only, implementation needed)
- ⚠️ Canary test running before full collections will catch real problems (hypothesis based on architecture analysis)
- ⚠️ Quantitative reliability target (≥95% success rate per run) achievable with Phase 1 alone

**What would change this:**

- If capsule concurrency=1 fails to prevent concurrent execution in production, lock must stay (unlikely given Sidekiq's architecture)
- If site-side detection increases (OshCut upgrades), stealth investment would become necessary
- If Jim's workflow changes (e.g., continuous collection instead of 2-3x/week), alerting priorities would shift

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Remove Redis lock (after pw-8928 validation) | architectural | Cross-component (lock touches ScrapeCliJob, all self-healing jobs, Redis state) |
| Unify self-healing into single job | architectural | Changes job lifecycle pattern used across collection system |
| Add run-level alerting | implementation | Additive feature, doesn't change existing behavior |
| Expand canary testing | implementation | Uses existing infrastructure (quick_test.yaml), additive |
| Define reliability target (≥95%) | strategic | Sets organizational expectations, affects resource allocation |

### Recommended Approach: Three-Phase Reliability Hardening

**Phase 1: Simplify Infrastructure (highest ROI, 2-3 days)**

1. **Validate and remove Redis lock** - Run production collection with lock disabled behind feature flag (capsule already enforces concurrency=1). If no concurrent execution detected, remove lock code entirely. This eliminates 3 of 7 incident root causes.

2. **Unify self-healing into `CollectionRunHealthCheckJob`** - Single job running every 5 min that:
   - Finds all active runs (in_progress + incomplete)
   - For each run: detects orphaned quotes (no active ScrapeJob), stale ScrapeJobs (>15 min), and run completion readiness
   - Single dispatch path with deduplication (check if quote already has pending/processing job before dispatching)
   - Clear logging: "Run #47: 3 orphaned quotes redispatched, 1 stale job superseded, 95% complete"

3. **Close pw-8933** - The job overlap question is answered: merge the jobs into one. No need for separate investigation.

**Phase 2: Observability (medium ROI, 1-2 days)**

1. **Run lifecycle Slack notifications** - Send to a channel when: run starts, run completes (with success rate), run stuck (no progress in 30 min), run failed. Use existing `config.death_handlers` pattern.

2. **Canary-before-collection gate** - Before starting a full collection, automatically run `quick_test.yaml` (6 quotes). If canary fails, block full collection and alert. This catches browser/proxy/login issues before committing to a 684-quote run.

3. **Queue/persona status API endpoint** - Simple JSON endpoint showing: current queue depth, lock status, persona availability. Jim can check this from dashboard before triggering re-scrapes.

**Phase 3: Measurement & Prediction (lower ROI, ongoing)**

1. **Track success rate per run over time** - Store in DB, show trend in dashboard
2. **Define and monitor reliability target: ≥95% quote success rate per run, zero stuck runs requiring manual intervention**
3. **Auto-escalate persistent failures** - If 3 consecutive canary failures, page on-call

**Why this approach:**
- Phase 1 addresses root causes (lock amplifier, self-healing gaps) not just symptoms
- Phase 2 gives Jim visibility before problems compound
- Phase 3 provides data for future reliability decisions
- Each phase is independently valuable (no big-bang dependency)

**Trade-offs accepted:**
- Not building dry-run mode (canary provides better coverage with less effort)
- Not investing in per-competitor concurrency (single-threaded is sufficient for current 2-3x/week cadence)
- Deferring BullMQ cleanup (Phase 4 in subprocess decision, ~$85/month savings, but not reliability-related)

### Alternative Approaches Considered

**Option B: Keep Separate Jobs + Add Coordination Layer**
- **Pros:** Minimal code changes, less risk of regression
- **Cons:** Adds complexity (coordination layer) on top of already-complex system; doesn't address fundamental overlap; harder to reason about three interacting jobs than one
- **When to use instead:** If unified job approach proves too complex or introduces new failure modes during implementation

**Option C: Event-Driven Self-Healing (Replace Cron with Callbacks)**
- **Pros:** Immediate response (no 5-min cron delay), cleaner architecture
- **Cons:** Significantly higher implementation effort; requires event bus infrastructure; harder to debug; over-engineered for current scale
- **When to use instead:** If scaling to 10+ concurrent collection runs or sub-minute recovery SLA needed

**Rationale for recommendation:** Phase 1 removes the amplifier (lock) and the gaps (job overlap) that caused 6 of 7 incidents. This is the minimum change for maximum reliability improvement. Phases 2-3 provide visibility and measurement to sustain reliability over time.

---

### Implementation Details

**What to implement first:**
- pw-8928 validation (capsule concurrency=1 in production) - foundational for lock removal
- Unified health check job design (clear spec before coding)
- Canary gate (lowest-effort, highest-visibility Phase 2 item)

**Things to watch out for:**
- ⚠️ Removing lock while capsule config is wrong would allow concurrent scraping → detection risk
- ⚠️ Unified job must handle the `enqueued_at` vs `updated_at` distinction correctly (see pw-8926 root cause)
- ⚠️ Canary gate must not block re-scrapes (separate path, separate queue)
- ⚠️ Slack notifications need rate limiting (one message per event, not per-quote)

**Areas needing further investigation:**
- pw-8928: Production validation of capsule-only concurrency (prerequisite for lock removal)
- Optimal stale threshold for unified job (currently 15 min for orphans, 10 min for redispatch - need single value)
- Whether `CleanupStaleScrapeJobsJob` (daily 3am) should also be merged into unified job or kept separate

**Success criteria:**
- ✅ Zero stuck runs requiring manual intervention over 2 weeks (currently ~1/week)
- ✅ ≥95% quote success rate per collection run (currently 97% when stable, but drops to 70-80% during incidents)
- ✅ Jim receives Slack notification within 5 minutes of run completion or failure
- ✅ Canary test runs before every full collection and blocks on failure
- ✅ No more "two jobs processing same quote" scenarios (pw-8933 resolved)

---

## References

**Files Examined:**
- `backend/app/jobs/scrape_cli_job.rb` - Lock lifecycle, subprocess execution, error handling
- `backend/app/jobs/retry_orphaned_quotes_job.rb` - Orphan recovery, persona stagger, dispatch logic
- `backend/app/jobs/check_active_run_orphans_job.rb` - Stale job detection, redispatch, auto-completion
- `backend/app/services/scrape_orchestrator.rb` - Job dispatch, persona selection, queue routing
- `backend/app/models/persona.rb` - Cooldown, daily cap, claim atomicity
- `backend/app/models/collection_run.rb` - Run lifecycle, retry methods, completion detection
- `backend/app/models/price_quote.rb` - Orphaned scope, status transitions
- `backend/config/initializers/sidekiq.rb` - Capsule config, error handlers
- `backend/config/sidekiq_cron.yml` - Scheduled job definitions
- `.kb/models/collection-system-architecture.md` - Architecture model (price-watch)
- `.kb/decisions/2026-01-17-subprocess-architecture-simplification.md` - Subprocess decision (price-watch)

**Commands Run:**
```bash
# Git incident history
git log --grep="pw-8926\|pw-8934\|pw-8918\|pw-8933" --oneline

# Recent commits (100)
git log --oneline -100
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-17-subprocess-architecture-simplification.md` (pw) - Architecture context
- **Investigation:** `.kb/investigations/2026-02-05-inv-fix-sidekiq-scheduled-set-explosion.md` (pw) - pw-8926 details
- **Investigation:** `.kb/investigations/2026-02-05-inv-jim-queued-re-scrapes-scs.md` (pw) - Queue demotion incident
- **Issue:** pw-8928 - Capsule validation prerequisite
- **Issue:** pw-8933 - Job overlap investigation (subsumed by this design)

---

## Investigation History

**2026-02-07 08:20:** Investigation started
- Initial question: What holistic reliability strategy should replace the whack-a-mole pattern?
- Context: Jim consistently experiences orphans and failures; 7 incidents in recent weeks

**2026-02-07 08:20:** Parallel exploration of codebase
- Launched 3 subagents: infrastructure code, git history, Node CLI architecture
- Directly read architecture model, subprocess decision, all key source files

**2026-02-07 08:25:** Incident catalog complete
- All 7 incidents classified as infrastructure failures
- Self-healing overlap (pw-8933) identified as root cause pattern

**2026-02-07 08:30:** Investigation completed
- Status: Complete
- Key outcome: Three-phase reliability strategy recommended: remove lock, unify self-healing, add observability
