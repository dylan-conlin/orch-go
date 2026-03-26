# Probe: Minimum Comprehension Surface — What Makes It Feel Like The Product?

**Date:** 2026-03-26
**Model:** dashboard-architecture
**Status:** Complete
**Question:** What is the smallest comprehension surface that makes orch-go unmistakably feel like the product rather than a dashboard layered on execution tooling?

## What I Tested

Examined the current implemented home surface (`web/src/routes/+page.svelte`), the orient CLI output, the briefs reading surface, the thread API and data model, and 3 briefs from `.kb/briefs/` to assess:

1. **What the current surface actually renders** — thread titles with entry counts, brief count as a number, question count as a number, review queue as a work list, then operational sections below
2. **What the brief artifacts actually contain** — Frame/Resolution/Tension format with story-first writing, producing genuine comprehension artifacts (e.g., orch-go-k6c0v: "The timeout that looked like absence")
3. **What the orient command produces** — readable thread entry previews, changelog, model state, session insights — the CLI is closer to "reading surface" than the web UI is
4. **What the thread data model supports** — full Entry content with dates, linked work, status lifecycle — the data is rich, the surface renders it as metadata

## What I Observed

### Observation 1: The Current Surface Renders Comprehension as Metadata

The implemented thread-first home surface (committed `923eb16cc`) has the correct *topology* — threads above operations, nav reordered to Threads | Briefs | Knowledge | Work | Harness. But it renders the comprehension layer as dashboard metadata:

- Threads section: titles with entry counts and status labels. Content hidden behind click-to-expand.
- Briefs: "7 unread" as a badge. Content hidden behind navigation to /briefs page.
- Questions: "3 blocking" as a badge. Content hidden behind another surface.
- Review queue: list of issue IDs with gate status indicators.

All four comprehension elements are *referenced* (counts, titles, badges) rather than *rendered* (text, synthesis, prose).

### Observation 2: The Orient CLI Is Closer to Product Than the Web UI

`orch orient` outputs:
- Thread entry previews (truncated but readable prose, not just titles)
- Changelog with commit messages (what changed)
- Session insights (what was learned last time)
- Ready-to-work queue (what to do next)

The CLI gives you something to *read*. The web UI gives you something to *scan*. The product decision says the comprehension layer is primary, but the CLI is delivering more comprehension than the web surface.

### Observation 3: Briefs Are Genuinely Good Comprehension Artifacts

Reading orch-go-k6c0v, orch-go-wgkj4, orch-go-sispn — these are real stories with turns, surprises, and unresolved tensions. They're not summaries dressed up as narratives. The Frame/Resolution/Tension structure produces artifacts worth reading. There are 49 of them.

This matters because it means the *content* for a product-feel surface already exists. The system is producing understanding — the surface just isn't showing it.

### Observation 4: Three Elements Together Create the Product Triangle

Through elimination testing (what happens if you remove each candidate element?), three elements emerged as jointly necessary and individually insufficient:

**Remove thread content → inbox app.** You have briefs arriving and questions pending, but no organizing spine. It's email with categories.

**Remove brief content → notebook app.** You have questions and tensions, but no answers. It's a research journal with no findings.

**Remove tension/question text → reporting tool.** You know what was asked and what was learned, but not what's unresolved. It's a project status page.

All three present → "this is where I think." The surface shows your questions (threads), what was learned about them (briefs), and what's still open (tensions). That's the comprehension loop made visible.

### Observation 5: Content-First Rendering Is the Transformation, Not New Features

The current surface has all three identity-defining elements — threads, briefs, and questions all exist. The problem is rendering mode:

| Element | Current rendering | Product rendering |
|---------|------------------|-------------------|
| Threads | Titles + entry counts | Latest entry as readable prose |
| Briefs | "7 unread" badge | Newest brief Frame/Resolution/Tension inline |
| Questions | "3 blocking" badge | Question text rendered as prose |

No new data primitives needed. No new API endpoints needed. The transformation is: show the text, not the count.

## Model Impact

### Extends Dashboard Architecture Model

The dashboard-architecture model describes a 3-page SPA with execution-centric root route. The thread-first redesign (Finding 3 of the home surface investigation) was a topology fix — correct ordering. This probe identifies a *rendering mode* gap that the topology fix doesn't address.

**New claim to add:** Dashboard surfaces have two rendering modes — *metadata mode* (counts, badges, titles) and *content mode* (prose, synthesis, readable text). The product feel requires content mode for the comprehension layer. Metadata mode is appropriate for operational/execution surfaces.

### Confirms Core/Substrate Ratio Finding

The 16%/72% ratio (orch-go-wgkj4) is a code mass observation. This probe adds a surface rendering observation: even the 16% core code that exists renders as metadata on the surface. The product identity gap is double: not enough code AND the code that exists doesn't render as content.

### Extends Product Boundary Decision

The decision lists "thread graph / thread-centric reading surfaces" as core. The word "reading" is key — and the current surface is scanning, not reading. This probe operationalizes "reading surface" as: content fills the viewport, user's default interaction is reading prose, not expanding lists.
