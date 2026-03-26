# Session Synthesis

**Agent:** og-arch-minimum-comprehension-surface-26mar-4031
**Issue:** orch-go-f8y50
**Duration:** 2026-03-26T14:23 → 2026-03-26T14:50
**Outcome:** success

---

## Plain-Language Summary

The thread-first home surface has the right layout (threads above operations, nav reordered) but renders comprehension as dashboard metadata — counts, badges, and collapsed titles rather than readable prose. The minimum surface that feels like "the product" requires exactly three elements rendered as content simultaneously: active thread entries (readable prose, not titles), the latest unread brief (Frame/Resolution/Tension inline), and open tensions (question text, not "3 blocking"). Removing any one of these three collapses the product into a recognizable tool category (inbox, notebook, or status page). Everything else — review queue, agents, knowledge tree, work graph — is secondary. No new APIs needed; the transformation is rendering mode, not feature inventory.

---

## TLDR

Defined the minimum viable comprehension surface: the "product triangle" of thread entries + brief content + tension text, all rendered as content (not metadata) on a single surface. This is a rendering mode change using existing APIs, not new infrastructure.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-design-minimum-comprehension-surface.md` — Full investigation with recommendations
- `.kb/models/dashboard-architecture/probes/2026-03-26-probe-minimum-comprehension-surface-product-identity.md` — Probe with 5 observations

### Files Modified
- `.kb/models/dashboard-architecture/model.md` — Updated summary (5 routes), added invariant 10 (content mode vs metadata mode), added Mar 2026 evolution section, merged probe

### Commits
- (pending — will commit with investigation + probe + model update)

---

## Evidence (What Was Observed)

- Current +page.svelte renders threads as click-to-expand titles with entry counts, briefs as "7 unread" badge, questions as "3 blocking" badge (all metadata, no content)
- `orch orient` CLI produces readable thread entry previews, session insights as prose, metric divergence explanations — more comprehension than the web surface
- 49 briefs exist in .kb/briefs/ with genuine Frame/Resolution/Tension content; the brief pipeline is producing good comprehension artifacts
- All data for content-first rendering is available via existing endpoints: GET /api/threads returns latest_entry text, GET /api/briefs/{id} returns full markdown, questions store returns question text
- Elimination testing: removing any one of {thread entries, brief content, tension text} collapses identity; removing any of {review queue, agents, knowledge tree} does not

### Tests Run
```bash
# Verified current surface state
orch orient  # CLI produces more comprehension than web UI
orch thread list  # 50 threads, 5 active, 14 forming — rich data available
ls .kb/briefs/ | wc -l  # 49 briefs with content
```

---

## Architectural Choices

### Content-first rendering over new features
- **What I chose:** Transform existing elements from metadata to content rendering
- **What I rejected:** Adding new features/routes (e.g., /read route, digest product)
- **Why:** All data exists; the gap is rendering mode. Adding features without changing the rendering mode would produce a more featureful dashboard, not a product.
- **Risk accepted:** Content mode increases page height; may feel dense on 666px half-screen

### Three-element product triangle over comprehensive surface
- **What I chose:** Define exactly 3 essential elements (threads, briefs, tensions)
- **What I rejected:** Larger minimum surface including review queue, knowledge changes, thread graph
- **Why:** Elimination testing showed these three are jointly necessary and individually insufficient; everything else improves but doesn't transform
- **Risk accepted:** May be wrong about tensions being essential (needs validation)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-design-minimum-comprehension-surface.md` — Product surface boundary definition

### Decisions Made
- The product/dashboard distinction is rendering mode (content vs metadata), not feature inventory
- Three elements form the minimum product triangle: thread entries, brief content, tension text

### Constraints Discovered
- The orient CLI is closer to the product feel than the web surface — the web surface took topology but not content mode from the design spec

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement content-first rendering for home surface product triangle
**Skill:** feature-impl
**Context:**
```
Transform the root route (+page.svelte) from metadata rendering to content rendering for three elements: (1) thread entries always visible as prose, (2) latest unread brief rendered inline with markdown, (3) question text rendered instead of count badge. See .kb/investigations/2026-03-26-design-minimum-comprehension-surface.md for implementation sequence.
```

---

## Unexplored Questions

- Whether the content-first surface actually *feels* different after a week of use
- Whether one brief inline is sufficient or 2-3 previews would be better
- Whether the 666px width constraint makes prose unreadable (may need reader-view modal)
- Whether forming threads with no entries belong in the tension section

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-minimum-comprehension-surface-26mar-4031/`
**Investigation:** `.kb/investigations/2026-03-26-design-minimum-comprehension-surface.md`
**Beads:** `bd show orch-go-f8y50`

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.
