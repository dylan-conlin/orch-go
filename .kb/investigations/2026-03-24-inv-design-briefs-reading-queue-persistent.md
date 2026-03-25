## Summary (D.E.K.N.)

**Delta:** Briefs disappear from the dashboard after `orch complete` clears `comprehension:pending` — a persistent reading queue needs a dedicated `/briefs` page with a list-all-briefs API endpoint, independent of the completion lifecycle.

**Evidence:** Current review-queue-section.svelte renders briefs only for `comprehension:pending` items; `.kb/briefs/` files persist but have no list endpoint; the Thinking page (`/thinking`) provides a proven pattern for reading-product pages with expand/collapse, state filters, and mark-as-read.

**Knowledge:** The system already has all building blocks: per-brief GET+POST endpoints, `hasBriefFile()`, `MarkdownContent` component, in-memory read state. The gap is (1) a `GET /api/briefs` list endpoint that scans `.kb/briefs/`, and (2) a SvelteKit page that consumes it.

**Next:** Implement via 3-component decomposition: API endpoint, Svelte store, page component.

**Authority:** implementation — Extends existing brief infrastructure with no new architectural patterns.

---

# Investigation: Design Briefs Reading Queue — Persistent Dashboard View

**Question:** How should the dashboard present briefs as a persistent reading queue that survives the completion lifecycle?

**Started:** 2026-03-24
**Updated:** 2026-03-24
**Owner:** orch-go-swrwn (architect)
**Phase:** Complete
**Next Step:** Implementation via decomposed issues
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-24-inv-design-brief-md-comprehension-artifact.md | extends | Yes — BRIEF.md template, lifecycle copy to `.kb/briefs/` | None |

---

## Findings

### Finding 1: Briefs are lifecycle-gated in the current UI

**Evidence:** `review-queue-section.svelte` renders brief expand/fetch/mark-as-read UI only for items returned by `/api/beads/review-queue`, which uses `verify.ListUnverifiedWork()` — items with `comprehension:pending` labels. Once `orch complete` removes this label, the item and its brief button disappear from the queue.

**Source:** `web/src/lib/components/review-queue-section/review-queue-section.svelte:130` (`{#if $reviewQueue && $reviewQueue.count > 0}`), `cmd/orch/serve_beads.go:165-216` (review queue handler)

**Significance:** This is the core problem. Briefs are reading artifacts, not verification artifacts. Their visibility shouldn't be coupled to the completion gate.

---

### Finding 2: All per-brief infrastructure exists but there is no list endpoint

**Evidence:**
- `GET /api/briefs/{beads-id}` returns brief content + read state (serve_briefs.go:68-92)
- `POST /api/briefs/{beads-id}` marks brief as read (serve_briefs.go:94-113)
- `hasBriefFile(beadsID)` checks existence (serve_briefs.go:116-120)
- In-memory `briefReadState` map tracks read/unread (serve_briefs.go:31-34)
- `.kb/briefs/{beads-id}.md` files persist on disk after completion

What's missing: `GET /api/briefs` that lists all briefs from `.kb/briefs/`, returning ID, read state, and optionally a preview/title extracted from the Frame section.

**Source:** `cmd/orch/serve_briefs.go`, `cmd/orch/serve.go:432`

**Significance:** A list endpoint is the foundation. The frontend needs to know what briefs exist without querying beads review-queue.

---

### Finding 3: The Thinking page provides a proven reading-product pattern

**Evidence:** `/thinking` route (`web/src/routes/thinking/+page.svelte`) implements: state filters (unread/read/starred/archived), type filters, expand/collapse with mark-as-read on expand, archive flow, 60s polling refresh. It uses the digest store with fetch + updateState methods.

**Source:** `web/src/routes/thinking/+page.svelte:1-274`

**Significance:** The briefs page can follow this exact pattern but simpler — briefs have fewer states (unread/read) and no type variants. The Thinking page validates the pattern works for Dylan's reading workflow.

---

### Finding 4: Read state is in-memory only — acceptable for V1

**Evidence:** `briefReadState` in `serve_briefs.go:31-34` is a `map[string]bool` protected by `sync.RWMutex`. Restarting `orch serve` resets all read state. This matches the thread's design decision that mark-as-read is UI-only state, separate from the `orch complete` comprehension gate.

**Source:** `cmd/orch/serve_briefs.go:31-34`, thread: 2026-03-24-comprehension-artifacts-async-synthesis-delivery

**Significance:** In-memory is fine for V1. Briefs accumulate slowly (one per full-tier completion). If persistence becomes needed, the existing pattern can be extended to a `.kb/briefs/.read-state.json` sidecar file. Not needed now — don't build for hypothetical requirements.

---

## Synthesis

**Key Insights:**

1. **Lifecycle decoupling is the key move** — The only change needed is giving briefs their own view that reads from `.kb/briefs/` directory directly, not from the review queue's `comprehension:pending` filter.

2. **The thinking page proves the pattern** — Expand/collapse reading with state tracking and filters already works in this dashboard. The briefs page is a simplified version.

3. **Three clean components** — API endpoint (scan `.kb/briefs/`), Svelte store (fetch/mark-read), page component (list/expand/read). No new packages, no new infrastructure patterns.

**Answer to Investigation Question:**

A dedicated `/briefs` route that lists all briefs from `.kb/briefs/` via a new `GET /api/briefs` endpoint, with expand/collapse inline reading and mark-as-read flow. The review-queue section's brief button should remain (it's useful for fresh completions) but the briefs page is the persistent reading queue that survives completion.

---

## Structured Uncertainty

**What's tested:**

- Existing `GET /api/briefs/{id}` and `POST /api/briefs/{id}` endpoints work correctly (verified via existing tests in serve_briefs_test.go)
- `.kb/briefs/` directory exists and contains brief files (verified: 1 brief currently exists — orch-go-3tyik.md)
- MarkdownContent component renders brief content (verified: used in review-queue-section.svelte)
- Thinking page pattern works as a reading product (verified: deployed and used)

**What's untested:**

- Performance of directory scanning for `GET /api/briefs` when many briefs accumulate (unlikely to be an issue — file count grows ~1-3 per day)
- Whether Dylan wants to navigate from review-queue brief button to the briefs page, or if the inline expand is sufficient in both locations
- Whether Frame section title extraction is worth the complexity vs. showing beads ID + read state

**What would change this:**

- If briefs accumulated at high volume (10+/day), we'd need pagination. Current rate (~1-3/day) makes this unlikely.
- If Dylan wants annotation (follow-up, question, done) beyond binary read/unread, the in-memory state pattern needs redesign. Thread notes this as future work.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `GET /api/briefs` list endpoint | implementation | Extends existing brief API within established patterns |
| Create briefs Svelte store | implementation | Follows existing store patterns (beads.ts, digest.ts) |
| Create `/briefs` route | implementation | Follows existing route patterns (/thinking, /knowledge-tree) |
| Add nav link to layout | implementation | Adds to existing nav, no structural change |

### Recommended Approach: Dedicated `/briefs` Page with List API

**Why this approach:**
- Decouples brief reading from completion lifecycle (addresses the core problem)
- Follows proven Thinking page pattern (validated reading-product UX)
- Minimal new code — reuses MarkdownContent, existing brief endpoints, in-memory read state

**Trade-offs accepted:**
- In-memory read state resets on server restart (acceptable for V1 — briefs accumulate slowly)
- No pagination (acceptable — brief count won't exceed ~100 in months of use)
- Frame section title not extracted (just show beads ID — keeps API simple, can add title later)

**Implementation sequence:**

1. **API: `GET /api/briefs`** — Scan `.kb/briefs/*.md`, return array of `{beads_id, marked_read}` sorted newest-first (by file mod time). Add to serve_briefs.go, register in serve.go.
2. **Store: `briefs` store** — Svelte writable store with `fetch()` (list), `fetchBrief(id)` (content), `markRead(id)` (POST). New file: `web/src/lib/stores/briefs.ts`.
3. **Page: `/briefs` route** — SvelteKit page with: header showing unread/total counts, unread/all filter toggle, list of briefs with expand/collapse, MarkdownContent for expanded briefs, mark-as-read on expand. New file: `web/src/routes/briefs/+page.svelte`.
4. **Nav: Add "Briefs" link** — Add to `+layout.svelte` nav, between Dashboard and Work Graph (or after Knowledge Tree). Show unread count badge.

### Alternative Approaches Considered

**Option B: Dashboard section (not a separate page)**
- **Pros:** No navigation change, everything on one page
- **Cons:** Dashboard is already dense (12+ sections). Adding a reading section fights for space with operational sections. The Thinking page proves reading products work better as their own page — different cognitive mode (reading vs. monitoring).
- **When to use instead:** If Dylan finds he never navigates to a separate page and wants briefs in his default view.

**Option C: Extend review-queue-section to show all briefs (not just pending)**
- **Pros:** No new page, reuses existing component
- **Cons:** Conflates two concerns: completion verification (review queue) and persistent reading (briefs). The review queue is gated on `comprehension:pending` for a reason — it's operational. Briefs are a reading product.
- **When to use instead:** If the number of briefs stays very low (<5 total) and a separate page feels heavy.

**Rationale for recommendation:** The thread explicitly says "the dashboard becomes a reading product, not a status board." A dedicated page makes the briefs a reading product. Embedding in the dashboard keeps them as a dashboard feature.

---

### Implementation Details

**What to implement first:**
- API endpoint (everything else depends on it)
- Store and page can be done in parallel after API

**Things to watch out for:**
- Defect class exposure: Class 0 (Scope Expansion) — the list endpoint scans a directory; make sure it only returns `.md` files with valid beads-ID names (use existing `validBeadsID` regex)
- Defect class exposure: Class 3 (Stale Artifact Accumulation) — briefs persist forever; V1 doesn't need cleanup, but track if accumulation becomes a problem
- Tab indentation: Svelte files use tabs. Use `cat -vet` before editing to verify tab counts.

**Success criteria:**
- `GET /api/briefs` returns all briefs from `.kb/briefs/` with correct read state
- `/briefs` page lists all briefs, expand/collapse works, mark-as-read works
- Brief that was in review queue before `orch complete` is still visible on `/briefs` after completion
- Nav link shows in header with optional unread badge

---

## References

**Files Examined:**
- `cmd/orch/serve_briefs.go` — Existing brief GET/POST handlers, read state, hasBriefFile
- `cmd/orch/serve_briefs_test.go` — Tests for brief endpoints
- `cmd/orch/serve_beads.go:165-216` — Review queue handler (shows has_brief enrichment)
- `cmd/orch/serve.go:431-432` — Route registration
- `web/src/lib/stores/beads.ts` — BriefResponse type, review queue store pattern
- `web/src/lib/components/review-queue-section/review-queue-section.svelte` — Existing brief expand/read UI
- `web/src/routes/thinking/+page.svelte` — Proven reading-product page pattern
- `web/src/routes/+layout.svelte` — Nav layout for adding Briefs link
- `web/src/routes/+page.svelte` — Dashboard layout, section state management
- `.kb/briefs/orch-go-3tyik.md` — Example brief content
- `.kb/threads/2026-03-24-comprehension-artifacts-async-synthesis-delivery.md` — Design thread

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-24-comprehension-artifacts-async-synthesis-delivery.md`
- **Investigation:** `.kb/investigations/2026-03-24-inv-design-brief-md-comprehension-artifact.md` — Prior design of BRIEF.md template and lifecycle

---

## Investigation History

**2026-03-24 17:45:** Investigation started
- Initial question: How to make briefs persist as a reading queue beyond the completion lifecycle?
- Context: Dylan's first brief was good but disappeared from dashboard after orch complete

**2026-03-24 17:55:** Exploration complete — 4 forks identified, all navigable
- Forks: page vs section, list API, read-state persistence, review-queue relationship
- All resolved via substrate (existing patterns, thread design intent, principles)

**2026-03-24 18:00:** Investigation completed
- Status: Complete
- Key outcome: Dedicated `/briefs` page with `GET /api/briefs` list endpoint, following Thinking page pattern
