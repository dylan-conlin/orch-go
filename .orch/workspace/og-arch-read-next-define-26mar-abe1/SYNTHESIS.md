# Session Synthesis

**Agent:** og-arch-read-next-define-26mar-abe1
**Issue:** orch-go-1r7ih
**Duration:** 2026-03-26
**Outcome:** success

---

## Plain-Language Summary

The openness boundary matrix classified "ranking intelligence" as a single held-back future product surface. I found that's wrong — it's actually three things. The bottom layer (sort by recency and read-state) already exists. The top layer (learned ranking from feedback data, cross-artifact intelligence) is correctly held-back. But the middle layer — grouping briefs by their parent thread and surfacing ones with open questions first — is part of the product's method commitments (thread-first, uncertainty treatment) and should ship as open, fixed core. Without this middle layer, the reading surface contradicts the product's own identity: it says threads are primary, but reading ignores thread structure entirely.

## TLDR

Decomposed "ranking intelligence" from a single held-back surface into a 3-layer model. Layer 2 (thread-grouping + tension surfacing) is method-defining and should ship now. Produced concrete API enrichment spec for implementation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-design-ranking-attention-layer-boundary.md` — Full architectural investigation with 3-layer decomposition, ordering principles, implementation spec
- `.kb/models/dashboard-architecture/probes/2026-03-26-probe-ranking-attention-layer-boundary.md` — Probe confirming attention pipeline is work-only, reading surface has no ordering intelligence

### Files Modified
- `.kb/models/dashboard-architecture/model.md` — Added attention gap paragraph (work vs reading), updated merged probes table (15→16), added serve_briefs.go and comprehension_queue.go to key components

---

## Evidence (What Was Observed)

- `serve_briefs.go:254` sorts exclusively by mod time — no thread context, no content signals
- `comprehension_queue.go` manages state lifecycle but provides no ordering within states
- `pkg/thread/backprop.go` moves beads IDs to thread's `resolved_by` during `orch complete` — the brief→thread connection exists in data but is not surfaced in any API
- All 11 attention collectors in `pkg/attention/` produce work signals; zero produce reading/comprehension signals
- All 30 briefs in `.kb/briefs/` have Tension sections (template requires it), but no parsing occurs
- Brief feedback mechanism exists (`shallow/good` in `.kb/briefs/feedback/`) but has no consumers

---

## Architectural Choices

### Three-layer decomposition rather than unified ranking engine
- **What I chose:** Decompose ranking into substrate (exists), method-expressing (missing), learned (future) — each with different product classifications
- **What I rejected:** Building a single ranking engine that handles all signals
- **Why:** The openness boundary matrix's own 4-question test classifies these layers differently. Thread-grouping fails Q1/Q2 (user can't remove without dissolving method); learned ranking passes Q4 (leverage only when integrated). Treating them identically misclassifies method as future.
- **Risk accepted:** Layer 2 may not improve Dylan's actual reading comprehension if his reading pattern is already effective with chronological ordering

### Thread-grouping as primary sort, not tension surfacing
- **What I chose:** Primary: group by thread. Secondary: tension-first within group. Tertiary: recency.
- **What I rejected:** Tension-first as primary (all items with open questions above all items without, regardless of thread)
- **Why:** Thread coherence enables batch synthesis — reading 3 briefs from one thread produces understanding. Tension-first across threads produces urgency-driven scattered reading. The product values synthesis over urgency.
- **Risk accepted:** A time-sensitive tension in a low-activity thread may be buried below routine briefs from a high-activity thread

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-design-ranking-attention-layer-boundary.md` — 3-layer ranking decomposition with implementation spec
- `.kb/models/dashboard-architecture/probes/2026-03-26-probe-ranking-attention-layer-boundary.md` — Attention pipeline work-only gap

### Decisions Made
- Decision: Thread-grouping and tension surfacing are method core (fixed, open), not held-back future product
- Decision: Ordering principles are: thread coherence > tension priority > comprehension state > recency

### Constraints Discovered
- The brief→thread reverse-lookup is O(threads × resolved_by_entries) — needs caching for >50 threads
- Brief feedback (shallow/good) exists but has zero consumers — unused signal

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation, probe, model merge, SYNTHESIS, BRIEF)
- [x] Probe status: Complete
- [x] Investigation status: Complete
- [x] Model merge: done (attention gap paragraph, probes table updated)
- [x] Ready for `orch complete orch-go-1r7ih`

**Follow-up work:** Layer 2 implementation (enrich `GET /api/briefs` with thread_slug, thread_title, has_tension; sort by thread groups) — this is implementation-authority work that can be spawned as feature-impl.

---

## Unexplored Questions

- Whether Dylan's actual reading pattern benefits from thread-grouped ordering vs chronological (assumption untested)
- Whether tension-section quality variance is sufficient for useful ranking signal (every brief has one by template requirement)
- Whether the attention pipeline should eventually include reading-signal collectors alongside work-signal collectors, or keep them architecturally separate
- How cross-artifact ranking (Layer 3) would join briefs, threads, and attention items — different data shapes, urgency semantics

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-read-next-define-26mar-abe1/`
**Investigation:** `.kb/investigations/2026-03-26-design-ranking-attention-layer-boundary.md`
**Beads:** `bd show orch-go-1r7ih`

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 6 outcomes verified, 3 manual checks for orchestrator.
