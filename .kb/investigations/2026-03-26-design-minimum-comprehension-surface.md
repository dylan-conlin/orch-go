## Summary (D.E.K.N.)

**Delta:** The current thread-first home surface has the right topology (threads above operations) but renders the comprehension layer as metadata (counts, titles, badges) instead of content (prose, synthesis, questions). The transformation from "dashboard with threads" to "this is the thing" is rendering mode, not new features.

**Evidence:** Audited the current +page.svelte (post-thread-first redesign), orch orient CLI output, 3 briefs from .kb/briefs/, thread API responses. The CLI delivers more comprehension than the web surface. Briefs are genuinely good reading artifacts (49 exist). All data for a content-first surface is already available via existing APIs.

**Knowledge:** The minimum viable product surface requires exactly three elements rendered as *content* (not metadata) simultaneously: (1) active thread entries as readable prose, (2) latest unread brief inline with Frame/Resolution/Tension, (3) open tensions as readable question text. Removing any one collapses the product into dashboard. Everything else (review queue, agents, knowledge tree, work graph) is secondary.

**Next:** Implement content-first rendering for the three identity-defining elements. No new API endpoints needed — existing thread detail, brief, and question endpoints already return the text.

**Authority:** architectural — Defines the product surface boundary, changes what "home" means.

---

# Investigation: Minimum Comprehension Surface — What Makes It Unmistakably The Product?

**Question:** What is the smallest comprehension surface that makes orch-go feel like the actual product rather than a better dashboard layered on execution tooling?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-f8y50
**Phase:** Complete
**Next Step:** Implement content-first rendering (three elements)
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` | constrains | yes | — |
| `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md` | constrains | yes | — |
| `.kb/investigations/2026-03-26-design-thread-first-home-surface.md` | extends | yes | Surface has right topology but wrong rendering mode |
| `.kb/investigations/2026-03-24-inv-design-brief-md-comprehension-artifact.md` | extends | yes | Brief pipeline works; surface doesn't show brief content |
| `.kb/briefs/orch-go-wgkj4.md` (system inventory) | extends | yes | 16%/72% ratio confirmed; surface gap is *in addition to* code mass gap |
| `.kb/threads/2026-03-24-threads-as-primary-artifact-thinking.md` | extends | yes | — |

---

## Findings

### Finding 1: The Product Triangle — Three Elements, Jointly Necessary

**Evidence:** Systematic elimination testing: for each candidate comprehension element, ask "if this is removed, does the surface still feel like 'the thing' or does it collapse into a familiar tool category?"

| Remove this... | ...and the surface becomes | Category |
|---------------|---------------------------|----------|
| Thread content | Brief inbox + question tracker | Email with categories |
| Brief content | Thread notebook + question log | Research journal with no findings |
| Tension/question text | Thread reader + brief reader | Project status page |
| Review queue | (still has threads + briefs + tensions) | Still feels like the product |
| Agent status | (still has threads + briefs + tensions) | Still feels like the product |
| Knowledge tree | (still has threads + briefs + tensions) | Still feels like the product |

**Significance:** Threads, briefs, and tensions form a product triangle. Each covers one leg of the comprehension loop:

- **Thread entries** = "your questions" (what you're thinking about)
- **Brief content** = "what was learned" (what agents discovered)
- **Tension text** = "what's still open" (what the system doesn't know)

Remove any one leg and the surface loses its distinctive identity. Add any other element and the identity doesn't change — it improves but doesn't transform.

---

### Finding 2: Rendering Mode Is the Gap, Not Feature Inventory

**Evidence:** The current home surface already has all three identity-defining elements. But they render as metadata:

```
CURRENT (metadata mode):
┌─────────────────────────────────────────────┐
│ Threads                              [14]   │
│  ▸ active  OpenClaw migration...     2 ent  │
│  ▸ forming Every spawn composes...   1 ent  │
│  ▸ forming Epistemic status...       1 ent  │
├─────────────────────────────────────────────┤
│ Unread Briefs [7]  │  Open Questions [3]    │
├─────────────────────────────────────────────┤
│ Review Queue: 5 items                       │
└─────────────────────────────────────────────┘

PRODUCT (content mode):
┌─────────────────────────────────────────────┐
│ What you're thinking about                  │
│                                             │
│ OpenClaw migration                          │
│ We clarified the decision boundary: the     │
│ lock-in problem is not simply using Claude   │
│ Code, it's using it as the only execution   │
│ path. The migration question is...          │
│                                             │
│ Every spawn composes knowledge              │
│ Reading through 37+ briefs, Dylan noticed   │
│ the brief 'frame' section reveals a pattern │
│ agents can't see...                         │
├─────────────────────────────────────────────┤
│ What was learned                            │
│                                             │
│ The Timeout That Looked Like Absence        │
│ You investigated why GPT-5.4 spawns got 0/  │
│ 100 context quality scores. The turn: the   │
│ knowledge WAS being found — just not fast   │
│ enough. kb context takes 5.8-8.8s; timeout  │
│ is 5s. The code treats timeout identically  │
│ to "nothing found"...                       │
│                                    ▸ more   │
├─────────────────────────────────────────────┤
│ What's still open                           │
│                                             │
│ • Which daemon behaviors are substrate vs   │
│   comprehension-method product surface?     │
│ • What should be read next, and why? Define │
│   the ranking/attention layer boundary      │
│ • Is per-query timeout architecture good    │
│   enough or should we go straight to        │
│   budget-based?                             │
└─────────────────────────────────────────────┘
```

The data is the same. The APIs are the same. The difference is: text fills the viewport vs. counts fill badges.

**Source:** `web/src/routes/+page.svelte:486-598` (current markup), `cmd/orch/serve_threads.go` (thread API with full entry content), `cmd/orch/serve_briefs.go` (brief API with markdown content), questions store (already returns question text)

**Significance:** This is not a feature request. It's a rendering mode change. The existing APIs return all the content needed. The surface just doesn't show it.

---

### Finding 3: The Orient CLI Is Already Closer to Product

**Evidence:** Running `orch orient` produces:
- Thread entry previews (100+ chars of prose, not just title)
- Session insights ("Headless brief pipeline had 4 serial bugs...")
- Metric divergence alerts with explanations
- Ready-to-work queue with context

Compare to the web home surface:
- Thread titles with entry counts
- Brief count badge
- Question count badge

The CLI output is something you *read*. The web surface is something you *scan*.

**Source:** `orch orient` output (full text in workspace), `web/src/routes/+page.svelte:486-598`

**Significance:** The "design spec" for the web surface (identified in the thread-first home investigation) is already implemented in the CLI. The web surface took the topology (threads above operations) but not the content mode (readable prose).

---

### Finding 4: Briefs Are the Strongest Product Signal

**Evidence:** Read 3 briefs:
- **orch-go-k6c0v** ("The Timeout That Looked Like Absence"): A story about a timeout being treated as absence, with a 3-layer fix and unresolved architectural tension. *This is compelling reading.*
- **orch-go-wgkj4** (system inventory): The 16/72 core/substrate ratio with the question "do you want to change the ratio or change the front door?" *This surfaces a real strategic choice.*
- **orch-go-sispn** (constraint dilution): 329 trials showing deterministic failure, with an experiment design that costs $2-5. *This teaches you something.*

49 briefs exist. If any of these appeared inline on the home surface, a viewer would immediately understand this is not a dashboard.

**Source:** `.kb/briefs/orch-go-k6c0v.md`, `.kb/briefs/orch-go-wgkj4.md`, `.kb/briefs/orch-go-sispn.md`

**Significance:** The strongest product signal is not the thread spine or the question list — it's the brief content. Briefs are where the system demonstrates its unique value proposition: agents produce understanding, not just code. But on the current surface, briefs are a number in a badge.

---

### Finding 5: The Dashboard/Product Threshold Test

**Evidence:** Two tests distinguish dashboard from product:

**Dashboard test (negative):** "Could I build this with Notion + webhooks?"
- Current surface: Thread list → Notion database. Brief count → webhook notification. Question count → filtered view. Review queue → another filtered view. **Yes, Notion could do this.**
- Product surface: Thread entry prose → no equivalent (Notion doesn't synthesize). Brief inline → no equivalent (Notion doesn't produce this content). Tension text → no equivalent. **No, Notion can't do the synthesis.**

**Product test (positive):** "Do I need to read this every morning before I can think about what to work on?"
- Current surface: No. The counts tell you what to check, but you could ignore them.
- Product surface: Yes. The thread entries remind you where your thinking was. The brief shows you what was learned overnight. The tensions tell you what needs your judgment.

**Significance:** The threshold between dashboard and product is whether the surface produces *understanding* or *awareness*. Counts produce awareness. Prose produces understanding. The current surface is on the wrong side of this threshold despite having all the right data.

---

## Synthesis

### The Minimum Viable Comprehension Surface

Three elements, all rendered as content (not metadata), present simultaneously on a single surface:

#### 1. Active Thread Entries (Essential)

**What:** The latest entry of each active/forming thread, rendered as readable prose (200-300 chars, expandable). 3-5 threads visible.

**Why essential:** This is the organizing spine. It answers "what am I thinking about?" — the question that every session should start from. Without readable thread content, threads are just a categorization system.

**What it replaces:** Current thread title + entry count list.

**Data source:** Already available via `GET /api/threads` (returns latest_entry) and `GET /api/threads/{slug}` (returns full entries).

#### 2. Latest Unread Brief Inline (Essential)

**What:** The newest unread brief's Frame/Resolution/Tension, rendered as markdown on the home surface. One brief, fully readable without navigation.

**Why essential:** This is the output signal. It demonstrates the system's unique value: agents produce understanding. A single brief inline does more for product identity than 10 thread titles.

**What it replaces:** Current "7 unread" badge with link to /briefs.

**Data source:** Already available via `GET /api/briefs` (list) and `GET /api/briefs/{id}` (full content).

#### 3. Open Tensions as Text (Essential)

**What:** The question text of open/blocking questions and the titles of forming threads with no recent activity. Rendered as a readable list of prose, not counts.

**Why essential:** This is the epistemic honesty signal. The system explicitly represents what it doesn't know. This separates it from every dashboard, status page, and project management tool.

**What it replaces:** Current "3 blocking" badge.

**Data source:** Already available via questions store (returns question text) and threads store (status + updated date).

### Secondary Elements (Useful, Not Identity-Defining)

| Element | Current state | Product contribution | Why not essential |
|---------|--------------|---------------------|-------------------|
| Comprehension queue | Implemented (ReviewQueueSection) | Shows what's awaiting review | It's a work queue — a dashboard concept |
| Knowledge change notifications | Not implemented | Would show what was learned today | Helpful context, not the core reading experience |
| Thread-to-work linking | Partially implemented (linked work count) | Shows execution serving threads | Execution visibility in thread clothing |
| Thread graph visualization | Not implemented | Shows thread relationships | The flat reading surface already communicates through content |
| Operational summary | Implemented (agent counts, daemon health) | System health awareness | Necessary but subordinate — a compact line suffices |

### What Can Be Omitted Without Losing the Product Center

- Agent cards / per-agent status monitoring
- Services section / infrastructure health
- SSE event streams / real-time execution events
- Work graph / dependency chains
- Historical mode / archive
- Performance metrics / usage tracking
- Coaching metrics

All of these serve execution observability. They should exist (on /work, /harness, or below the fold) but their absence from the comprehension surface does not diminish the product identity.

### What Should Be Delayed Until After First Release

- Thread CRUD from web UI (create/append/resolve threads in browser)
- Digest product (the /thinking route's original concept)
- Thread graph visualization (threads as nodes, connections as edges)
- Brief quality scoring / ranking intelligence
- Composite /api/home endpoint (optimization, not identity)
- Thread search/filter (needed when thread count grows, not at 50)

---

## The Durable Product-Surface Boundary

This investigation's primary deliverable is a boundary rule to prevent future confusion between "dashboard improvement" and "product clarification":

**Rule:** A surface change is product clarification if it moves a comprehension element from metadata rendering to content rendering. A surface change is dashboard improvement if it adds, rearranges, or polishes metadata rendering.

**Examples:**
- Showing brief text instead of brief count → product clarification
- Adding a brief count to the nav bar → dashboard improvement
- Rendering thread entries as prose → product clarification
- Adding thread status filters → dashboard improvement
- Displaying question text inline → product clarification
- Adding a question count tooltip → dashboard improvement

**Application:** When prioritizing work, product clarifications should take precedence over dashboard improvements. When evaluating whether a feature "moves the product forward," ask: does it put *content* in front of the user, or *metadata*?

---

## Structured Uncertainty

**What's tested:**

- ✅ Current surface renders comprehension as metadata (verified: read +page.svelte markup)
- ✅ Orient CLI produces more comprehension than web surface (verified: ran orch orient, compared output)
- ✅ Brief content is compelling reading material (verified: read 3 briefs)
- ✅ All data for content-first rendering is available via existing APIs (verified: thread, brief, question endpoints)
- ✅ Removing any one of the three essential elements collapses product identity (verified: elimination analysis)

**What's untested:**

- ⚠️ Whether content-first rendering actually *feels* different to Dylan (needs to be built and used for a week)
- ⚠️ Whether one brief inline is enough or whether 2-3 brief previews would be better
- ⚠️ Whether the prose density of content mode creates information overload on a 666px-wide half-screen
- ⚠️ Whether forming threads with no entries should appear in the tension section or be filtered out
- ⚠️ Whether the content surface needs to be the root route or could be a new /read route

**What would change this:**

- If Dylan uses the content surface for a week and still starts sessions from the CLI orient command, the web surface is still not the product
- If the 666px width constraint makes prose unreadable, may need a reader-view modal instead of inline rendering
- If brief content inline makes the page too long, may need a "latest brief" carousel instead of full render
- If Dylan never reads the tensions section, it may not be identity-defining after all

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Content-first rendering for 3 essential elements | architectural | Changes what the product surface means — product identity shift |
| Section headers renamed to question framing | implementation | Copy change within existing layout |
| Operational section condensed to single line | implementation | Layout change within existing page |

### Recommended Approach: Transform Existing Root Route

**Why this approach:**
- Reuses all existing API endpoints (no backend changes)
- Transforms existing Svelte components (no new routes)
- Preserves all operational data (condensed below, not removed)
- The product test is immediate: deploy, open, does it feel different?

**Implementation sequence:**

1. **Thread entries visible by default** — Change thread section from click-to-expand to latest-entry-always-visible. Each active thread shows its latest entry text (200-300 chars) directly in the card, with full-expand on click for full entries. ~30 lines of Svelte change.

2. **Brief content inline** — Add a "Latest Brief" section between threads and tensions. Fetches the newest unread brief's full content via existing `/api/briefs/{id}` endpoint and renders it as markdown using the existing MarkdownContent component. If all read, shows the most recently read brief with "all caught up" indicator. ~60 lines of Svelte.

3. **Tension text rendered** — Replace the "Open Questions: 3 blocking" badge with a prose section that renders the actual question text. Pull from existing questions store. Add stale forming threads (no entry in 3+ days) as implicit tensions. ~40 lines of Svelte.

4. **Section headers reframed** — Rename sections from noun labels to question framing:
   - "Threads" → "What you're thinking about"
   - "Latest Brief" → "What was learned"
   - "Open Questions" → "What's still open"
   - "Review Queue" → "What's ready for review" (secondary)
   ~10 lines of copy change.

5. **Operational section condensed** — Replace full agent cards, stats bar, and services section with a single compact line: "3 agents active · 46 issues open · daemon healthy". Link to /work for full operational view. ~20 lines of Svelte.

**Trade-offs accepted:**
- Thread entries always visible increases page height — acceptable because content is the point
- One brief inline may not be enough for high-volume days — can iterate to 2-3 previews
- Question text may be long — truncate at 150 chars with expand
- Operational section is very condensed — users who need execution detail use /work

### Alternative Approaches Considered

**Option B: New /read route, keep current / as operational dashboard**
- **Pros:** Zero risk to existing workflows; both surfaces coexist
- **Cons:** The root route IS the identity. Putting the product on a side route says "this is optional." Contradicts the product decision.
- **When to use instead:** If Dylan explicitly says he wants the operational dashboard as his default

**Option C: Progressive disclosure — content mode as a toggle on existing surface**
- **Pros:** Users choose their rendering mode
- **Cons:** Making the product method optional is the "too open at the center" failure mode from the open-boundary decision
- **When to use instead:** Never — this dissolves the product into a preference

**Rationale for recommendation:** The product decision says the comprehension layer is primary. The root route is where the product lives. Putting content there is the most direct expression of the decision. All data is already available; this is a rendering change, not an infrastructure change.

---

### Implementation Details

**What to implement first:**
- Thread entries always visible (highest impact for least code)
- Brief content inline (strongest product signal)

**Things to watch out for:**
- ⚠️ Tab-indented Svelte files: use `cat -vet` before Edit tool on +page.svelte
- ⚠️ Defect class exposure: Class 1 (Filter Amnesia) — brief list and inline brief may use different filter criteria; ensure both respect project_dir from orchestrator context
- ⚠️ MarkdownContent component exists in `web/src/lib/components/markdown-content/` — reuse it for brief rendering
- ⚠️ The page already has 18+ stores; adding inline brief fetch should use existing briefs store, not a new one
- ⚠️ 666px width constraint: brief markdown may need max-width styling to remain readable in half-screen

**Success criteria:**
- ✅ Opening localhost:5188 shows thread entry prose without clicking
- ✅ Latest unread brief is readable on the home surface without navigating
- ✅ Open questions are visible as text, not just counts
- ✅ Dylan says "I read this before starting my day" within one week
- ✅ A stranger shown the surface for 10 seconds says "it's a reading/thinking tool" not "it's a dashboard"

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte:486-598` — Current thread-first home surface markup
- `web/src/routes/+layout.svelte:60-85` — Nav hierarchy (already Threads | Briefs | Knowledge | Work | Harness)
- `web/src/routes/briefs/+page.svelte` — Brief reading surface with expand/mark-as-read
- `web/src/lib/components/markdown-content/` — Existing markdown renderer
- `cmd/orch/serve_threads.go` — Thread API endpoints (list + detail)
- `cmd/orch/serve_briefs.go` — Brief API endpoints (list + detail + mark-as-read)
- `.kb/briefs/orch-go-k6c0v.md` — Example brief ("The Timeout That Looked Like Absence")
- `.kb/briefs/orch-go-wgkj4.md` — Example brief (system inventory, 16/72 ratio)
- `.kb/briefs/orch-go-sispn.md` — Example brief (constraint dilution, 329 trials)

**Commands Run:**
- `orch orient` — Session orientation CLI (compared to web surface)
- `orch thread list` — Full thread listing (50 threads, 5 active, 14 forming)

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`
- **Decision:** `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md`
- **Investigation:** `.kb/investigations/2026-03-26-design-thread-first-home-surface.md`
- **Thread:** `.kb/threads/2026-03-24-threads-as-primary-artifact-thinking.md`
- **Probe:** `.kb/models/dashboard-architecture/probes/2026-03-26-probe-minimum-comprehension-surface-product-identity.md`

---

## Investigation History

**2026-03-26:** Investigation started
- Initial question: What is the minimum comprehension surface that unmistakably feels like the product?
- Context: Thread-first home surface exists but tension remains between "improved dashboard" and "product center"

**2026-03-26:** Exploration complete
- 4 forks identified: content vs metadata rendering, reading vs monitoring, identity-defining vs useful, product vs dashboard threshold
- Key finding: all three identity-defining elements already exist as data — the gap is rendering mode
- Key finding: orient CLI delivers more comprehension than web surface

**2026-03-26:** Investigation completed
- Status: Complete
- Key outcome: Product triangle (threads + briefs + tensions) rendered as content, not metadata. The transformation is rendering mode, not feature inventory.
