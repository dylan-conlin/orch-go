# Session Synthesis

**Agent:** og-arch-design-briefs-reading-24mar-b303
**Issue:** orch-go-swrwn
**Duration:** 2026-03-24T17:45 → 2026-03-24T18:10
**Outcome:** success

---

## Plain-Language Summary

Briefs — the half-page comprehension artifacts that agents write for Dylan — disappear from the dashboard the moment `orch complete` runs, because the review queue is gated on `comprehension:pending` labels. This design adds a persistent `/briefs` page (like the existing `/thinking` page) that lists all briefs from `.kb/briefs/` regardless of completion state. It needs one new API endpoint (`GET /api/briefs` to list all brief files), a small Svelte store, and a new page route. Dylan reads briefs over coffee; the briefs page is his reading queue.

---

## TLDR

Designed a persistent briefs reading queue: dedicated `/briefs` page + `GET /api/briefs` list endpoint, decoupled from the completion lifecycle. Three implementation issues created (API, frontend, integration).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-24-inv-design-briefs-reading-queue-persistent.md` — Full design investigation with recommendations
- `.orch/workspace/og-arch-design-briefs-reading-24mar-b303/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-design-briefs-reading-24mar-b303/VERIFICATION_SPEC.yaml` — Verification spec
- `.orch/workspace/og-arch-design-briefs-reading-24mar-b303/BRIEF.md` — Comprehension brief

### Issues Created
- `orch-go-wtah6` — API: GET /api/briefs list endpoint
- `orch-go-3o5rj` — Frontend: briefs Svelte store and /briefs page
- `orch-go-bpoph` — Integration: briefs reading queue end-to-end

---

## Evidence (What Was Observed)

- Review queue is gated on `comprehension:pending` — once cleared, briefs vanish from UI (serve_beads.go:165-216)
- `.kb/briefs/` files persist on disk after completion — only the UI link is lost
- Per-brief GET/POST endpoints already work (serve_briefs.go, 6 passing tests)
- Thinking page (`/thinking`) validates reading-product-as-page pattern in this dashboard
- In-memory read state (`briefReadState` map) matches thread's design intent for V1

---

## Architectural Choices

### Dedicated page vs. dashboard section
- **What I chose:** New `/briefs` route (separate page)
- **What I rejected:** Adding briefs section to main dashboard
- **Why:** Thread says "dashboard becomes a reading product, not a status board." Reading and monitoring are different cognitive modes. The Thinking page proves separate pages work. Dashboard already has 12+ sections.
- **Risk accepted:** Dylan might not navigate to a separate page. Mitigated by nav link with unread badge.

### In-memory read state vs. persistent state
- **What I chose:** Keep existing in-memory `briefReadState` map
- **What I rejected:** File-based persistence (`.read-state.json`)
- **Why:** Briefs accumulate at ~1-3/day. Server restarts are rare. Thread explicitly says mark-as-read is UI-only state. Don't build for hypothetical requirements.
- **Risk accepted:** Read state lost on server restart.

### Title extraction from brief content vs. beads ID only
- **What I chose:** Show beads ID in list, show full content on expand
- **What I rejected:** Parsing Frame section to extract a display title
- **Why:** Keeps the API simple. Frame section is 2-3 sentences — not a clean title source. Beads IDs are meaningful to Dylan (he uses them daily).
- **Risk accepted:** List view is less scannable without titles. Can add later.

---

## Knowledge (What Was Learned)

### Decisions Made
- Dedicated page over dashboard section (reading ≠ monitoring)
- Three-component decomposition: API, frontend, integration
- V1: in-memory read state, no pagination, no title extraction

### Constraints Discovered
- Review queue's lifecycle coupling is by design (it's operational), not a bug to fix — the fix is decoupling briefs into their own view

---

## Next (What Should Happen)

**Recommendation:** close — spawn implementation from decomposed issues

### Implementation Issues
- `orch-go-wtah6` — API endpoint (foundational — do first)
- `orch-go-3o5rj` — Frontend store + page (can parallel with API once types are agreed)
- `orch-go-bpoph` — Integration verification (depends on both above)

---

## Unexplored Questions

- Whether annotation beyond read/unread (follow-up, question, done) would improve the reading workflow — thread notes this as future work
- Whether the review-queue brief button should link to the `/briefs` page or remain self-contained

---

## Friction

Friction: none

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-briefs-reading-24mar-b303/`
**Investigation:** `.kb/investigations/2026-03-24-inv-design-briefs-reading-queue-persistent.md`
**Beads:** `bd show orch-go-swrwn`
