# Investigation: Comparison View Takes 14.78s

## Summary (D.E.K.N.)

**Delta:** The production comparison page is slow because the initial request still renders two very heavy categories (Aluminum/Steel) into a 7.33MB HTML response, and each cell includes large tooltip markup plus per-cell helper work.

**Evidence:** Replayed authenticated production request from HAR cookie: `x-runtime=15.544330`, `bytes=7333454`, `cell-tooltip=2376`, `tooltip-row=17928`; lazy category endpoints are much faster (`x-runtime 0.53s-6.88s`) and the page contains `loading="lazy"` + category frame URLs.

**Knowledge:** Lazy loading is deployed and working for non-priority categories, but current architecture still front-loads massive server-side render work and payload for priority categories.

**Next:** Implement a two-step response strategy: keep initial grid lightweight (defer rich tooltip + trace lookups), add fragment caching for category HTML, then evaluate JSON-on-demand for cell detail if runtime still exceeds target.

**Authority:** architectural - solution spans controller query strategy, helper behavior, rendering/caching boundaries, and possible server-rendered-to-JSON migration for cell detail.

---

**Question:** What is the true bottleneck behind the 14.78s production load, is lazy loading deployed, and which optimization class (SQL, streaming, caching, JSON migration, architecture) should be chosen while preserving all current features?

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** Agent
**Phase:** Complete
**Next Step:** Orchestrator review and implementation planning
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation                                                                                                                                              | Relationship | Verified | Conflicts |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | -------- | --------- |
| `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2026-01-17-inv-implement-lazy-loading-load-scroll.md`   | extends      | yes      | none      |
| `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2026-01-17-inv-review-optimize-lazy-loaded-category.md` | deepens      | yes      | none      |

---

## Findings

### Finding 1: Lazy loading is deployed in production, but only for non-priority categories

**Evidence:** Authenticated production HTML contains lazy-load markers: `/quotes/comparison/category/...`, `loading="lazy"`, `data-controller="lazy-load-category"`; `category_header_row_count=2` confirms only two fully rendered categories on initial page.

**Source:** Command output from replayed production request using HAR cookie (`python requests`), plus current view implementation in `backend/app/views/price_quotes/comparison.html.erb:229` and `backend/app/views/price_quotes/comparison.html.erb:241`.

**Significance:** Prior lazy-loading work is deployed and functional. The current incident is not "lazy loading failed to deploy"; it is "priority categories are still too expensive."

---

### Finding 2: Initial payload is dominated by server-rendered cell tooltip HTML, not static assets

**Evidence:** Production initial response is `7,333,454` bytes with `x-runtime=15.544330`; HTML element counts: `cell-tooltip=2376`, `tooltip-row=17928`, `price-cell=2583`, `data-row=287`. Removing tooltip blocks from the response string reduces size by `4,834,314` bytes (65.9%).

**Source:** Command outputs from HAR/live-response analysis scripts, and tooltip markup loop in `backend/app/views/price_quotes/_comparison_category.html.erb:200` through `backend/app/views/price_quotes/_comparison_category.html.erb:288`.

**Significance:** The largest bottleneck is render architecture + response shape. Even with lazy categories, the "first two categories" are too rich to render inline.

---

### Finding 3: Per-cell trace helper introduces high-cardinality DB lookup work in the view

**Evidence:** For each cell with quote data, template executes `trace_path_for_quote(current_quote)` (and similar for unavailable quotes), and helper does `ScrapeJob.find_by(job_id: quote.job_id)` with no memoization.

**Source:** `backend/app/views/price_quotes/_comparison_category.html.erb:272`, `backend/app/views/price_quotes/_comparison_category.html.erb:311`, and `backend/app/helpers/price_quotes_helper.rb:496` through `backend/app/helpers/price_quotes_helper.rb:500`.

**Significance:** This creates N+1-style query risk at render time. In captured production response, `trace_link_count=0`, so this repeated lookup often does work that yields no visible output.

---

### Finding 4: Controller-side aggregation is also heavy, but secondary to response/render inflation

**Evidence:** `?format=json` request still takes `x-runtime=12.199132` and returns `~2.17MB`; HTML path is slower at `x-runtime=15.544330` and `~7.33MB`. Category endpoints benchmarked at `x-runtime` from `0.536677` to `6.883843` depending on category size.

**Source:** Live production requests replayed with HAR auth context; controller aggregation loops in `backend/app/controllers/price_quotes_controller.rb:360` through `backend/app/controllers/price_quotes_controller.rb:494`.

**Significance:** SQL-only tuning will not solve the full problem. There is both data-assembly cost and substantial server-render payload cost.

---

## Synthesis

**Key Insights:**

1. **Deployed but insufficient lazy loading** - The system already defers non-priority categories, but initial response still renders too much detail for Aluminum/Steel.

2. **Response architecture is the root class** - The 7.33MB server-rendered HTML (mostly cell tooltip/detail markup) is the primary lever, not JS/CSS assets or network transfer.

3. **N+1 helper pattern amplifies latency** - Per-cell trace lookup in helper adds avoidable DB work during template rendering.

**Answer to Investigation Question:**

This does not require response streaming as the primary fix, and SQL optimization alone is insufficient. The right path is an architectural response-shape change: keep initial comparison grid lean, defer rich cell detail/trace lookups, and add fragment caching for category HTML. If this still misses SLO, migrate cell-detail tooltip content to on-demand JSON endpoint calls while preserving all user-visible features (same data, delayed fetch for detail).

---

## Structured Uncertainty

**What's tested:**

- ✅ Production lazy-loading markers exist in authenticated HTML (verified by direct request replay with HAR cookie).
- ✅ Production baseline reproduced at ~16.4s wall time with `x-runtime=15.544330` and 7.33MB HTML.
- ✅ Lazy category endpoint timings measured by category (`brass/copper/stainless-steel/aluminum/steel`).
- ✅ Tooltip markup contribution measured by removing tooltip blocks from captured HTML string (~65.9% size reduction).

**What's untested:**

- ⚠️ Exact DB query count per request in production (no SQL log access in this session).
- ⚠️ CPU-time split between controller aggregation vs ERB rendering inside Rails process.
- ⚠️ Effect of fragment caching hit rates under real-user traffic patterns.

**What would change this:**

- If production tracing shows DB dominates total time even after tooltip deferral, SQL/index strategy would move up in priority.
- If removing trace lookups/tooltip render does not materially reduce runtime, controller data model assembly must be redesigned first.
- If stakeholders require all rich tooltip fields at first paint, JSON-on-demand cannot be the primary path.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation                                                                                          | Authority     | Rationale                                                                                                    |
| ------------------------------------------------------------------------------------------------------- | ------------- | ------------------------------------------------------------------------------------------------------------ |
| Make initial HTML lightweight by deferring rich tooltip/trace detail and adding category fragment cache | architectural | Crosses view, helper, controller, caching strategy; affects render contract but not user-visible feature set |

### Recommended Approach ⭐

**Lean Initial Render + Deferred Detail + Category Fragment Cache** - Preserve all features, but stop emitting full per-cell detail on initial HTML and avoid per-cell DB lookups at render time.

**Why this approach:**

- Directly targets measured bottleneck (7.33MB response, huge tooltip markup density).
- Preserves current UX/features (tooltips, trace links, rescrape controls) with progressive loading.
- Avoids high-risk full rewrite while creating clear path to JSON detail if needed.

**Trade-offs accepted:**

- First tooltip open may incur small fetch delay.
- Slightly higher frontend complexity for detail hydration.

**Implementation sequence:**

1. Remove per-cell `trace_path_for_quote` DB lookup from server render path (preload map or deferred lookup endpoint).
2. Render compact cells initially (value + status), move detailed tooltip sections behind on-hover/on-focus fetch.
3. Add fragment caching for category partial output keyed by date-range/config/include_flagged/show_all_data and category.
4. Re-measure x-runtime and payload; only then evaluate deeper SQL/index work.

### Alternative Approaches Considered

**Option B: SQL optimization only**

- **Pros:** Lower DB latency, minimal UI change.
- **Cons:** Does not address 7.33MB HTML payload and heavy ERB/detail generation.
- **When to use instead:** If profiling shows DB time dominates after render-shape fixes.

**Option C: Response streaming**

- **Pros:** Better first-byte/first-paint perception.
- **Cons:** Total server work and payload size remain largely unchanged; complexity/correctness risk in Rails templates.
- **When to use instead:** If perceived load remains poor after reducing payload and computation.

**Option D: Full JSON API migration immediately**

- **Pros:** Strong long-term control over payload shape.
- **Cons:** Higher migration risk and larger implementation scope for a daily-use operational screen.
- **When to use instead:** If phased lean-render approach cannot meet performance target.

**Rationale for recommendation:** Option A is the fastest path to meaningful wins with lowest product risk and keeps a clean escalation path to Option D only if needed.

---

### Implementation Details

**What to implement first:**

- Replace `trace_path_for_quote` per-cell helper query with precomputed hash or deferred endpoint lookup.
- Introduce compact tooltip skeleton data in initial ERB (no full nested tooltip rows per cell).
- Add fragment caching around `_comparison_category` blocks.

**Things to watch out for:**

- ⚠️ Cache invalidation: include period boundaries, target config, include_flagged, data mode, and category.
- ⚠️ Accessibility: deferred tooltip content must still support keyboard and screen reader usage.
- ⚠️ Interaction parity: trace links and rescrape state must remain available with deferred details.

**Areas needing further investigation:**

- Collect production SQL query counts for comparison action before/after helper/query changes.
- Determine whether `@comparison_data` precomputation can be pruned for initial request without breaking filter/header behavior.

**Success criteria:**

- ✅ Initial HTML payload reduced from ~7.3MB to <=1.5MB.
- ✅ `x-runtime` for initial comparison request reduced from ~15.5s to <=4s (target) under same account/context.
- ✅ No feature regression in tooltips, trace links, rescrape controls, and lazy category loading.

---

## References

**Files Examined:**

- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/backend/app/controllers/price_quotes_controller.rb` - Comparison data assembly and lazy category flow.
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/backend/app/views/price_quotes/comparison.html.erb` - Initial page/category lazy frame structure.
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/backend/app/views/price_quotes/_comparison_category.html.erb` - Per-cell rendering + tooltip bloat source.
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/backend/app/helpers/price_quotes_helper.rb` - Trace lookup helper causing per-cell DB query risk.
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/shared/har/comparison-view-2026-02-08.har` - Original production HAR artifact.

**Commands Run:**

```bash
# HAR analysis by entry size/timing and root response content metrics
python3 - <<'PY' ...

# Authenticated replay of production request from HAR cookie (verify lazy markers, payload, runtime)
python3 - <<'PY' ...

# Benchmark category endpoint runtimes and payload sizes
python3 - <<'PY' ...

# Estimate payload contribution from tooltip markup removal
python3 - <<'PY' ...
```

**External Documentation:**

- N/A

**Related Artifacts:**

- **Investigation:** `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2026-01-17-inv-implement-lazy-loading-load-scroll.md` - Initial lazy-loading implementation.
- **Investigation:** `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.kb/investigations/2026-01-17-inv-review-optimize-lazy-loaded-category.md` - Prior lazy-category optimization.
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-arch-comparison-view-takes-08feb-29a5/` - Session synthesis destination.

---

## Investigation History

**2026-02-08 20:00:** Investigation started

- Initial question: why comparison view still takes ~14-15s in production after prior lazy-loading work.

**2026-02-08 20:45:** Production replay validated

- Reproduced authenticated initial load with `x-runtime=15.544330` and `~7.33MB` HTML.

**2026-02-08 21:15:** Root bottleneck isolated

- Confirmed lazy loading is deployed for non-priority categories; identified render/payload inflation and per-cell trace lookup pattern as primary levers.

**2026-02-08 21:30:** Investigation completed

- Key outcome: recommend architectural payload-shape reduction + deferred detail/caching before any full API migration.
