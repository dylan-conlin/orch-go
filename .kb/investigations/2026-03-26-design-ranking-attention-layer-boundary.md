## Summary (D.E.K.N.)

**Delta:** Decomposed "ranking intelligence" from a single held-back surface into three distinct layers with different product classifications: substrate ordering (exists, open), method-expressing ordering (missing, should be fixed/open), and learned ranking (missing, correctly held-back). Produced concrete implementation spec for the method layer.

**Evidence:** Audited `serve_briefs.go` (mod-time sort only), `comprehension_queue.go` (state lifecycle without ordering), `pkg/thread/backprop.go` (thread→work linkage exists but unreachable from briefs API), `pkg/attention/` (11 work-signal collectors, zero reading-signal collectors), and 30 brief files in `.kb/briefs/` (all have Tension sections, none are parsed).

**Knowledge:** The openness boundary matrix conflates two fundamentally different things under "ranking intelligence": method-expressing ordering (thread-grouping, tension surfacing) that is part of the product's core commitments, and learned/integrated ranking that is legitimately future product leverage. The conflation makes the entire ranking surface look like future work, when half of it is method that should ship now.

**Next:** Implement Layer 2 (method-expressing ordering): enrich `GET /api/briefs` with `thread_slug` and `has_tension`, group briefs by thread in the reading surface. This is implementation-authority work within existing patterns.

**Authority:** strategic — This reclassifies a surface from the openness boundary matrix. The matrix classified ranking as a single held-back surface; this investigation finds part of it should be fixed/open (method core). That changes what ships with the first release.

---

# Investigation: Define Ranking/Attention Layer Boundary — What Should Be Read Next

**Question:** What should the system surface for reading/review next, where does that ranking logic belong (substrate, method core, or future product), and what are the ordering principles?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-1r7ih
**Phase:** Complete
**Next Step:** Implementation issue for Layer 2 enrichment
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-26-inv-define-openness-boundary-matrix-productization.md` | extends | yes — read full document, this investigation decomposes one of its held-back surfaces | Partially contradicts: matrix treats "routing/ranking intelligence" as single held-back item; this finds it's 3 layers |
| `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` | respects | yes — thread-first commitment drives Layer 2 classification | none |
| `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md` | extends | yes — uses the 4-question structural test to reclassify ranking sub-surfaces | none |
| `.kb/investigations/2026-03-24-inv-design-briefs-reading-queue-persistent.md` | extends | yes — that investigation designed the briefs page; this investigation defines the ordering within it | none |
| `.kb/models/dashboard-architecture/probes/2026-02-16-attention-pipeline-full-audit.md` | extends | yes — attention pipeline is work-focused; reading surface needs separate attention logic | none |

---

## Findings

### Finding 1: Current "read next" has no intelligence — it's chronological

**Evidence:** `serve_briefs.go:254` sorts by `items[i].modTime > items[j].modTime` (newest-first). The `BriefListItem` struct returns only `{beads_id, marked_read}` — no thread context, no content signals, no priority. The comprehension queue (`comprehension_queue.go`) manages state transitions but provides no ordering within states.

**Source:** `cmd/orch/serve_briefs.go:235-270`, `pkg/daemon/comprehension_queue.go:1-241`

**Significance:** Newest-first is the default sort for *all notification systems*. It's what you get when you don't think about ordering. For a product whose identity is "turning agent work into durable, legible understanding," chronological sort treats all agent output as equally worth reading in temporal order. This is correct for email; it's wrong for a comprehension system.

---

### Finding 2: "Ranking intelligence" decomposes into three layers

**Evidence:** Applying the openness boundary matrix's 4-question test to specific ranking signals:

| Signal | Can user change without dissolving method? | Would making optional = generic infra? | Lowers adoption fear? | Leverage only when integrated? | Classification |
|--------|---------------------------------------------|----------------------------------------|-----------------------|-------------------------------|---------------|
| Comprehension state (unread first) | No — this IS the review discipline | Yes — without it, system is just a feed | N/A (substrate) | No — standalone useful | **Substrate (open)** |
| Recency within state | Yes — some users prefer oldest-first | No — sort direction isn't identity | N/A (substrate) | No — standalone useful | **Substrate (configurable)** |
| Thread-grouping | No — this IS thread-first commitment | Yes — without it, briefs are scattered notifications | N/A (method) | No — useful with basic briefs | **Method core (fixed)** |
| Tension surfacing | No — this IS uncertainty treatment | Yes — without it, questions hide in brief bodies | N/A (method) | No — useful with basic briefs | **Method core (fixed)** |
| Batch coherence | No — this IS how synthesis happens | Yes — without it, reading is random | N/A (method) | No — useful with basic briefs | **Method core (fixed)** |
| Feedback-learned ordering | Yes — it's optimization | No — absence doesn't dissolve product | No | Yes — requires usage data | **Held-back (future)** |
| Cross-artifact ranking | Yes — it's integration | No — absence doesn't dissolve product | No | Yes — requires multi-system integration | **Held-back (future)** |
| Collaborative priority | Yes — it's team feature | No — absence doesn't dissolve product | No | Yes — requires multi-user | **Held-back (future)** |

**Source:** `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md` (4-question test, lines 77-82)

**Significance:** The matrix's single "held-back" classification for ranking is wrong by its own structural test. Thread-grouping and tension surfacing fail question 1 ("can the user change it without dissolving the method?") and question 2 ("would making optional = generic infra?"). They should be classified as method core (fixed), not held-back.

---

### Finding 3: Thread-grouping infrastructure exists but isn't wired to briefs

**Evidence:** The connection path is:
1. Each brief is named `{beads-id}.md` in `.kb/briefs/`
2. Threads track completed work in `resolved_by: [beads-ids]` frontmatter (verified: `pkg/thread/thread.go:577`)
3. `BackPropagateCompletion()` moves beads IDs from `active_work` to `resolved_by` during `orch complete` (`pkg/thread/backprop.go:20-69`)
4. Thread list API (`GET /api/threads`) returns all threads with their frontmatter

**The gap:** No code performs the reverse-lookup: given a beads_id, find which thread(s) it appears in. `GET /api/briefs` returns briefs without any thread context. The frontend has no way to group briefs by thread.

**Source:** `pkg/thread/backprop.go:20-69`, `cmd/orch/serve_briefs.go:209-271`, `cmd/orch/serve_threads.go:26-68`

**Significance:** This is a wiring gap, not an architecture gap. The data model already connects briefs to threads. The API just doesn't expose the connection.

---

### Finding 4: Tension is already a structural section — just not parsed

**Evidence:** Every brief follows the template (`.orch/templates/BRIEF.md`) with three required sections: Frame, Resolution, Tension. The template explicitly states: "This section is required — a brief without tension is a summary, and summaries create false comprehension."

Examining `.kb/briefs/`:
- 30 briefs exist (all from recent sessions)
- All follow the 3-section structure
- Tension sections contain explicit open questions, unresolved tradeoffs, and judgment calls

But `handleBriefsList()` reads only filenames and mod times. No brief content is parsed. A brief with "The entire auth system might need redesign — this needs your judgment" ranks identically to "No significant tensions — straightforward implementation."

**Source:** `.orch/templates/BRIEF.md:30`, `cmd/orch/serve_briefs.go:240-251`

**Significance:** Tension detection is trivially implementable: check if `## Tension` section is non-empty or contains question markers. The signal already exists in every brief by template design.

---

### Finding 5: The attention pipeline is exclusively work-focused

**Evidence:** All 11 attention collectors in `pkg/attention/` produce work signals:
- `issue-ready`, `likely-done`, `recently-closed` (beads lifecycle)
- `agent-stuck`, `verify-failed`, `epic-orphaned` (agent health)
- `duplicate-candidate`, `competing`, `stale`, `unblocked` (issue hygiene)

None produce reading/comprehension signals. There is no collector for "brief needs reading" or "thread has new evidence" or "question needs Dylan's judgment."

**Source:** `pkg/attention/` (11 collector files)

**Significance:** The attention system and the reading surface serve different audiences with different needs. The attention system answers "what work needs intervention." The reading surface should answer "what should Dylan understand next." These are not the same question, and the answer to the second should not be derived from the first.

---

## Synthesis

### The Three Layers

```
┌─────────────────────────────────────────────────────────┐
│  Layer 3: Learned/Integrated Ranking (HELD-BACK)        │
│  - Feedback-learned quality ordering                     │
│  - Cross-artifact attention (briefs + threads + signals) │
│  - Collaborative team priority                           │
│  - Requires: usage data, multi-system integration        │
├─────────────────────────────────────────────────────────┤
│  Layer 2: Method-Expressing Ordering (FIXED, OPEN)      │
│  - Thread-grouping: cluster briefs by parent thread      │
│  - Tension surfacing: boost items with open questions     │
│  - Batch coherence: related briefs read together          │
│  - Requires: reverse-lookup (brief → thread), parsing    │
├─────────────────────────────────────────────────────────┤
│  Layer 1: Substrate Ordering (OPEN, EXISTS)              │
│  - Comprehension state: unread > read                    │
│  - Recency: newest-first within state                    │
│  - Exists in: serve_briefs.go, comprehension_queue.go    │
└─────────────────────────────────────────────────────────┘
```

### Ordering Principles for "Read Next"

**Primary sort: Thread coherence**
Briefs from the same thread cluster together. Within a cluster, newest-first. A thread with 3 unread briefs is a batch — read the batch, not 3 scattered items.

**Secondary sort: Tension priority**
Within a thread cluster (or across unclustered briefs), items with explicit open questions rank above items without. The rationale: open questions may block other work or require time-sensitive judgment.

**Tertiary sort: Comprehension state**
Unread > read. This is already implicit in the comprehension queue but should be formalized in the briefs API.

**Quaternary sort: Recency**
Newest-first within equal-priority groups. The current default behavior, now as a tie-breaker rather than the primary sort.

### Which signals are method-defining vs optional heuristics

**Method-defining (must ship, not configurable):**

1. **Thread coherence** — The product says threads are primary. If reading ignores thread structure, the product contradicts itself. This is not a ranking heuristic; it's the method operating.

2. **Tension surfacing** — The product says uncertainty should be explicit. If a brief says "here's a question that needs your judgment" and the reading surface doesn't prioritize it, the method is contradicted.

3. **Comprehension state** — The product says review before new work. Unread items must surface above read items. This is already how the comprehension queue works; it should be formalized in the reading surface.

**Optional heuristics (can be configurable):**

4. **Recency direction** — Newest-first vs oldest-first is a preference, not a method commitment. Default newest-first.

5. **Thread-cluster ordering** — Which thread cluster shows first? Currently: most-recently-updated thread first. Could be configurable (most active, most briefs, alphabetical).

### What ships now vs future

**Ships now (Layer 1 — already exists):**
- Mod-time sort: `serve_briefs.go:254`
- Read state: `briefReadState` map
- Comprehension lifecycle: `comprehension_queue.go`
- No code changes needed. Just document the contract.

**Ships soon (Layer 2 — method core, requires implementation):**

Enrich `GET /api/briefs` response:

```go
type BriefListItem struct {
    BeadsID     string `json:"beads_id"`
    MarkedRead  bool   `json:"marked_read"`
    ThreadSlug  string `json:"thread_slug,omitempty"`  // NEW: parent thread
    ThreadTitle string `json:"thread_title,omitempty"` // NEW: for display
    HasTension  bool   `json:"has_tension"`            // NEW: non-empty ## Tension
}
```

Implementation:
1. **Reverse-lookup function:** Scan threads for `resolved_by` containing a beads_id. Cache the result (threads change slowly).
2. **Tension detection:** Read first 200 bytes after `## Tension` header. If non-empty and not just template text, `has_tension = true`.
3. **Sort change:** Group by thread_slug, then within groups sort by has_tension (true first), then by mod time.

**Ships later (Layer 3 — held-back, requires data):**
- Feedback-learned ordering (requires accumulated shallow/good ratings)
- Cross-artifact ranking (requires joining briefs, threads, attention items, investigations)
- Collaborative priority (requires multi-user)

### Correction to openness boundary matrix

The matrix's row for "Routing/ranking intelligence" should be split:

| Surface | Old Classification | New Classification | Rationale |
|---------|-------------------|-------------------|-----------|
| Thread-coherent reading order | HELD-BACK | FIXED (method core) | Fails the matrix's own Q1 and Q2: user can't remove it without dissolving thread-first commitment |
| Tension surfacing in reading order | HELD-BACK | FIXED (method core) | Fails Q1 and Q2: removing it contradicts uncertainty-treatment commitment |
| Learned quality ranking | HELD-BACK | HELD-BACK (unchanged) | Passes Q4: leverage only when integrated with usage data |
| Cross-artifact attention integration | HELD-BACK | HELD-BACK (unchanged) | Passes Q4: leverage only when integrated with full comprehension pipeline |
| Collaborative ranking | HELD-BACK | HELD-BACK (unchanged) | Passes Q4: leverage only when multi-user |

---

## Structured Uncertainty

**What's tested:**

- ✅ Current briefs API returns only beads_id and marked_read — no thread context, no content signals (verified: `serve_briefs.go:259-265`)
- ✅ Thread→work linkage exists via resolved_by frontmatter (verified: `pkg/thread/backprop.go`)
- ✅ Tension section is required and present in all 30 existing briefs (verified: `.kb/briefs/*.md`, `.orch/templates/BRIEF.md`)
- ✅ Attention pipeline is exclusively work-focused — zero reading/comprehension collectors (verified: 11 files in `pkg/attention/`)
- ✅ BackPropagateCompletion moves IDs to resolved_by during orch complete, before brief creation (verified: `pkg/thread/backprop.go:20-69`)

**What's untested:**

- ⚠️ Performance of thread reverse-lookup across 24+ threads with many resolved_by entries — may need caching
- ⚠️ Whether thread-grouped reading actually improves comprehension vs. chronological (assumption based on product thesis, not measurement)
- ⚠️ Whether tension-section parsing produces useful signal or just "every brief has tension" (template requires it, so the question is quality variance)
- ⚠️ Whether Dylan's actual reading pattern follows thread coherence or is more spontaneous

**What would change this:**

- If Dylan reads briefs in chronological order regardless of thread grouping, Layer 2 is overhead without value → keep it but make thread-grouping a view option
- If tension sections are uniformly non-trivial (every brief has a real open question), tension surfacing provides no ranking signal → remove has_tension or add severity levels
- If the reading surface becomes multi-user before the comprehension pipeline is proven, collaborative ranking jumps from Layer 3 to Layer 2

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Split "ranking intelligence" into 3 layers | strategic | Reclassifies a surface from the openness boundary matrix |
| Enrich briefs API with thread_slug and has_tension | implementation | Extends existing API within established patterns |
| Thread-grouped reading order | strategic | Operationalizes the thread-first product commitment in the reading surface |
| Defer Layer 3 (learned ranking) | strategic | Confirms held-back classification for the genuinely future part |

### Recommended Approach ⭐

**Layer 2 Implementation — Method-expressing ordering**

Enrich `GET /api/briefs` with thread context and tension detection. Frontend groups briefs by thread. This operationalizes two product commitments (thread-first, uncertainty treatment) in the reading surface.

**Why this approach:**
- The product says threads are primary — the reading surface must follow
- The product says uncertainty should be explicit — the reading surface must surface it
- The infrastructure exists (thread→work linkage, brief templates with Tension section)
- The implementation is bounded: one reverse-lookup function, one section parser, one API enrichment

**Trade-offs accepted:**
- Thread reverse-lookup adds latency to `GET /api/briefs` (mitigated by caching — threads change slowly)
- Tension detection is binary (has/doesn't have) — may need gradation later
- Briefs not linked to any thread fall into an "Unthreaded" group — acceptable for edge cases

**Implementation sequence:**
1. Add `threadForBeadsID(beadsID, threadsDir) (slug, title string)` — scan threads' resolved_by
2. Add `hasTensionSection(briefContent string) bool` — check for non-empty ## Tension
3. Enrich `BriefListItem` with `thread_slug`, `thread_title`, `has_tension`
4. Sort: group by thread_slug, then has_tension desc, then mod_time desc
5. Frontend: render briefs grouped by thread with thread title as header

### Alternative Approaches Considered

**Option B: Add reading signals to the existing attention pipeline**
- **Pros:** Reuses the 11-collector architecture; unified signal surface
- **Cons:** The attention pipeline answers "what work needs attention" not "what should be read." Forcing reading signals through work-signal types creates defect class 5 (Contradictory Authority Signals) — the same item would have a work priority and a reading priority that may conflict.
- **When to use instead:** If a future Layer 3 cross-artifact ranking engine needs to unify work and reading signals

**Option C: Build the full Layer 3 ranking engine now**
- **Pros:** Skip directly to the differentiated product surface
- **Cons:** No usage data to train on. The feedback mechanism has zero entries. Cross-artifact integration requires joining multiple data shapes that don't yet have a common priority model. Building Layer 3 before Layer 2 is proven inverts the evidence hierarchy.
- **When to use instead:** Never before Layer 2 is working and generating feedback data

---

### Things to watch out for:

- ⚠️ **Defect class 0 (Scope Expansion):** Thread reverse-lookup scans all threads for each brief. If thread count grows to 100+, this needs indexing or caching.
- ⚠️ **Defect class 5 (Contradictory Authority Signals):** If a brief appears in multiple threads' resolved_by lists, pick the most recently updated thread. Document this in the API contract.
- ⚠️ **Template compliance:** If future briefs skip the Tension section (template non-compliance), `has_tension` becomes unreliable. The method should enforce this at brief creation, not at reading time.

### Success criteria:

- ✅ `GET /api/briefs` returns thread_slug, thread_title, and has_tension for each brief
- ✅ Briefs from the same thread appear adjacent in the response
- ✅ Briefs with tension=true rank above tension=false within a thread group
- ✅ Reading surface displays briefs grouped by thread
- ✅ Unthreaded briefs appear in a separate "Other" group

---

## References

**Files Examined:**
- `cmd/orch/serve_briefs.go` — Brief list/fetch/mark-read API (mod-time sort, no thread context)
- `cmd/orch/serve_briefs_test.go` — Existing tests for brief endpoints
- `pkg/daemon/comprehension_queue.go` — Two-state lifecycle, throttle logic
- `cmd/orch/comprehension_cmd.go` — CLI for comprehension queue management
- `cmd/orch/serve_threads.go` — Thread list/show API
- `pkg/thread/thread.go` — Thread parsing, List() with updated-desc sort
- `pkg/thread/backprop.go` — BackPropagateCompletion (active_work → resolved_by)
- `pkg/thread/relations.go` — LinkWork, CreateWithParent
- `pkg/attention/types.go` — AttentionItem struct, Collector interface
- `pkg/attention/*.go` — All 11 collectors (work-signal only)
- `cmd/orch/serve_attention.go` — Attention API aggregation
- `.kb/briefs/*.md` — 30 existing briefs (all have Tension sections)
- `.orch/templates/BRIEF.md` — Brief template (3 required sections)
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`
- `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md`
- `.kb/investigations/2026-03-26-inv-define-openness-boundary-matrix-productization.md`
- `.kb/investigations/2026-03-24-inv-design-briefs-reading-queue-persistent.md`

**Related Artifacts:**
- **Decision:** Thread/comprehension is primary product
- **Decision:** Open boundary, opinionated core
- **Investigation:** Openness boundary matrix (this investigation refines one of its surfaces)
- **Model:** dashboard-architecture (this investigation extends with reading-surface attention gap)

---

## Investigation History

**2026-03-26:** Investigation started
- Question: Where does "read next" ranking logic belong — substrate, method core, or future product?
- Context: Openness boundary matrix classified "ranking intelligence" as held-back; system still has no principled reading order

**2026-03-26:** Codebase audit complete
- Audited serve_briefs.go, comprehension_queue.go, pkg/thread/, pkg/attention/, .kb/briefs/
- Key finding: ranking decomposes into 3 layers, not 1

**2026-03-26:** Investigation completed
- Status: Complete
- Key outcome: 3-layer decomposition with concrete Layer 2 implementation spec; partial contradiction of openness boundary matrix
