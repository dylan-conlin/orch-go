# Probe: Ranking/Attention Layer Boundary — Where "Read Next" Logic Belongs

**Model:** dashboard-architecture
**Date:** 2026-03-26
**Status:** Complete

---

## Question

The openness boundary matrix (orch-go-pityd) classifies "higher-order routing/ranking/review intelligence" as HELD-BACK. But the system still needs a principled answer to a simpler question: what should Dylan read next, and why? Is "ranking intelligence" one thing or several things at different layers? Where does the boundary fall between method-defining ordering (which should be fixed/open) and future product intelligence (which is legitimately held back)?

---

## What I Tested

1. **Examined current ordering logic in `serve_briefs.go`:**
   - `GET /api/briefs` sorts by file modification time, newest-first (line 254: `items[i].modTime > items[j].modTime`)
   - Returns `{beads_id, marked_read}` — no thread context, no content signals
   - Read state is binary (marked_read bool), persisted in `~/.orch/briefs-read-state.json`

2. **Examined comprehension queue lifecycle in `comprehension_queue.go`:**
   - Two-state: `comprehension:unread` → `comprehension:processed`
   - Throttle at 5 unread items (daemon pauses spawning)
   - No ordering within states — items are returned in whatever order `bd list` provides
   - Feedback mechanism exists (shallow/good) but is not wired to ordering

3. **Examined thread-to-work linkage in `pkg/thread/`:**
   - Threads have `active_work` (in-progress) and `resolved_by` (completed) beads ID lists
   - `BackPropagateCompletion()` moves beads IDs from active_work → resolved_by on completion
   - `LinkWork()` adds beads IDs to threads
   - Connection exists: brief beads_id → thread resolved_by. But this reverse-lookup is NOT surfaced in the briefs API.

4. **Examined attention pipeline (`pkg/attention/`):**
   - 11 collectors produce `AttentionItem` structs with priority scores
   - These are about *work* signals (stuck agents, ready issues, verify failures), NOT reading signals
   - No collector exists for "brief needs reading" or "thread has new evidence"
   - The attention system answers "what work needs attention" not "what should Dylan read"

5. **Examined brief content for structural signals:**
   - Briefs have 3 required sections: Frame, Resolution, Tension
   - Tension section contains open questions requiring Dylan's judgment
   - No parsing of brief content occurs in the API — briefs are served as raw markdown

6. **Cross-referenced the openness boundary matrix:**
   - Matrix classifies "routing/ranking intelligence" as single held-back surface
   - 4-question test: "Does it create leverage only when integrated?" → held-back candidate
   - But this test doesn't distinguish between ordering that IS the method vs ordering that IMPROVES the method

---

## What I Observed

### The conflation

"Ranking intelligence" in the openness boundary matrix is treated as a single held-back surface. But auditing the actual system reveals it decomposes into three distinct layers with different product classifications:

**Layer 1 — Substrate ordering (already exists, open):**
- Sort by comprehension state: unread > processed > read
- Sort by recency within state: newest-first (mod time)
- This is implemented. It's basic. It works.

**Layer 2 — Method-expressing ordering (does NOT exist, should be fixed/open):**
- Thread-grouping: briefs from the same thread should cluster together in reading order
- Tension surfacing: briefs with explicit open questions (## Tension section) should rank above briefs without
- Batch coherence: reading 3 briefs from one thread produces synthesis; reading 3 scattered briefs produces information

**Layer 3 — Learned/integrated ranking (does NOT exist, legitimately held-back):**
- Quality-informed ordering: feedback ratings (shallow/good) inform which brief styles to surface first
- Cross-artifact integration: rank briefs alongside threads, attention items, investigations
- Collaborative ranking: team-level "what matters most"

### Why this matters

Layer 2 is the one the matrix missed. Thread-grouping is not "routing/ranking intelligence" in the held-back sense — it's the thread-first commitment expressing itself through reading order. If threads are the primary artifact (per the accepted decision), then the reading surface must follow thread structure. Making this optional or future-work contradicts the product's core commitment.

Similarly, tension surfacing is the uncertainty-treatment commitment expressing itself. The method says "surface unknowns explicitly." If a brief says "here's an open question requiring your judgment" and the reading surface doesn't prioritize it, the method is contradicted.

### Existing infrastructure gaps for Layer 2

The pieces exist but aren't wired:

| What's needed | What exists | Gap |
|---------------|-------------|-----|
| Brief → thread reverse-lookup | Thread frontmatter has `resolved_by: [beads-ids]` | No API endpoint exposes this linkage; `GET /api/briefs` doesn't include thread context |
| Tension detection | Briefs have `## Tension` section by template | No parsing occurs; API serves raw markdown without structural analysis |
| Thread-grouped display | Thread list API exists (`GET /api/threads`), brief API exists | Two APIs are disconnected; no joined view |

### Layer 3 is genuinely future

Learned ranking requires feedback data (only 2 ratings: shallow/good, stored in `.kb/briefs/feedback/`). Cross-artifact integration requires ranking briefs against threads, attention items, investigations — different data shapes, different urgency semantics. Collaborative ranking requires multi-user. None of these are close to ready. The matrix's held-back classification is correct for this layer.

---

## Model Impact

- [x] **Extends** model with: The dashboard architecture model's attention pipeline (11 collectors → `/api/attention`) is exclusively work-focused. The reading/comprehension surface has no equivalent attention system — briefs sort by mod time only. The model should note that the attention system serves two distinct audiences (work monitoring vs. reading/comprehension) and currently only serves the first. Additionally, the "ranking intelligence" surface identified as held-back in the openness boundary matrix decomposes into three layers: substrate ordering (exists, open), method-expressing ordering (missing, should be fixed/open), and learned ranking (missing, correctly held-back). The model should track this decomposition.

- [x] **Contradicts** implicit model assumption: The attention pipeline audit probe (2026-02-16) found "11 real collectors, only 2 firing" and focused on work signals. The model treats attention as a work-monitoring concern. But the reading surface needs its own attention signals (thread-coherence, tension detection, batch grouping) that are categorically different from work signals. These are not additional collectors for the existing pipeline — they're a separate concern.

---

## Notes

- The comprehension queue throttle (5 unread items) functions as crude attention management: "don't produce more than Dylan can read." But it controls *volume*, not *order*. Volume control without ordering is half a solution.
- The brief feedback mechanism (shallow/good) exists in `comprehension_queue.go` and `comprehension_cmd.go` but has no consumers. It records to `.kb/briefs/feedback/{beads-id}.txt` and is never read by the ordering logic.
- The `BackPropagateCompletion()` function in `pkg/thread/backprop.go` is called during `orch complete` — meaning by the time a brief exists, its beads_id has already been moved from `active_work` to `resolved_by` in the parent thread. The reverse-lookup is reliable at brief-reading time.
