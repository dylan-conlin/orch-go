# Session Synthesis

**Agent:** og-arch-design-thread-first-26mar-3809
**Issue:** orch-go-q6ykb
**Duration:** 2026-03-26T12:36 → 2026-03-26T13:15
**Outcome:** success

---

## Plain-Language Summary

The web dashboard says "I'm an execution monitoring tool" while the product decision says "I'm a comprehension layer." This design bridges that gap. The concrete recommendation: redesign the root route to show active threads, new evidence (unread briefs + recent knowledge changes), and open tensions above a condensed operational summary. This requires only 2 new API endpoints (thread list and thread detail) because the thread domain model already exists in Go — it just has no HTTP exposure. The nav reorders from Dashboard/Work Graph/Knowledge/Briefs/Harness to Threads/Briefs/Knowledge/Work/Harness.

## TLDR

Designed the thread-first home surface: 2 new API endpoints to expose pkg/thread via HTTP, a redesigned root route with 3 comprehension sections above condensed operational data, and a nav reorder. No new data primitives needed — the orient CLI command already models the ideal information architecture.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-design-thread-first-home-surface.md` - Full design investigation with forks, recommendations, implementation sequence

### Files Modified
- None (design-only session)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- pkg/thread/ has 2,337 lines of well-structured domain code (Thread, ThreadSummary, lifecycle states, relations) but serve.go has zero thread endpoints among 50+ registered handlers
- `orch orient` already produces the ideal session-start orientation: active threads, changelog, metrics, ready queue — this is the design spec for the home surface
- Current root route (+page.svelte) imports 18 stores and 12 components, all execution-centric. Zero thread content.
- The /thinking route has digest product UI but the `/api/digest` backend doesn't exist (no Go handlers)
- Nav order (Dashboard | Work Graph | Knowledge | Briefs | Harness) puts execution first
- Briefs and knowledge tree already have full API coverage — ready to compose into home surface

### Tests Run
```bash
# Verified thread list output format
go run ./cmd/orch thread list
# 14 threads with rich metadata

# Verified orient command produces target IA
go run ./cmd/orch orient
# Active threads, changelog, metrics, ready queue — exactly the home surface spec
```

---

## Architectural Choices

### Root route redesign vs new /threads route
- **What I chose:** Redesign root route in-place (thread sections above, condensed execution below)
- **What I rejected:** New /threads route with redirect from /
- **Why:** The root route IS the product identity. A redirect is indecisive. And in-place redesign preserves bookmarks.
- **Risk accepted:** Execution-first users lose their current landing page. Mitigated by keeping operational data below the fold.

### Orient as design spec vs inventing new IA
- **What I chose:** Mirror orient's information architecture in the dashboard
- **What I rejected:** Novel home surface design
- **Why:** Orient already answers "what should Dylan see first?" — tested by real use. Reinventing is waste.
- **Risk accepted:** Dashboard becomes a visual rendering of CLI output. If orient's IA evolves, the dashboard should follow.

### Thread API endpoints vs composite /api/home
- **What I chose:** Separate /api/threads endpoints for v1, composite endpoint as Phase 2
- **What I rejected:** Single composite endpoint from the start
- **Why:** Separate endpoints are simpler, testable, reusable. Composite can be added later if HTTP overhead matters.
- **Risk accepted:** Multiple parallel fetches on page load (threads, briefs, questions) — acceptable for single-user dashboard.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-design-thread-first-home-surface.md` - Full design with 4 forks navigated, implementation sequence, success criteria

### Decisions Made
- Decision 1: Root route gets redesigned (not replaced by new route) because it IS the product identity
- Decision 2: Nav reorder to Threads | Briefs | Knowledge | Work | Harness
- Decision 3: /thinking route becomes orphaned (broken backend, wrong abstraction) — fold its intent into home

### Constraints Discovered
- Digest product API (/api/digest) has no Go implementation — /thinking page is a UI-only prototype
- Thread data is CLI-only — zero HTTP exposure blocks all web thread features

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

Key outcomes:
- Investigation produced with complete design recommendation
- 4 design forks identified and navigated
- Implementation sequence defined (Thread API → Store → Root route → Nav)
- Success criteria specified (7 testable acceptance criteria)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Implementation Issues

The design decomposes into 4 implementation tasks:

1. **Thread API endpoints** — `cmd/orch/serve_threads.go` exposing GET /api/threads and GET /api/threads/{slug}
2. **Thread Svelte store** — `web/src/lib/stores/threads.ts` with ThreadSummary type, fetch, polling
3. **Root route redesign** — Replace +page.svelte with thread-first sections above condensed operational view
4. **Nav reorder** — Update +layout.svelte nav order

---

## Unexplored Questions

- Whether a composite `/api/home` endpoint is worth building vs. parallel fetches (performance question for Phase 2)
- How thread entries should render in cards (full markdown? plain text? first N words?)
- Whether stale forming threads (no entry in 3+ days) is a useful "tension" signal or just noise
- What happens to the /thinking route — redirect to /, delete, or keep as alternate view?

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-thread-first-26mar-3809/`
**Investigation:** `.kb/investigations/2026-03-26-design-thread-first-home-surface.md`
**Beads:** `bd show orch-go-q6ykb`
