# Brief: orch-go-1r7ih

## Frame

Your openness boundary matrix from earlier today classified "ranking intelligence" as held-back — one of three future product surfaces alongside hosted comprehension UX and collaborative knowledge. That felt right in the abstract. But when I went to answer the concrete question — what should you actually read next, and why? — I couldn't find the answer anywhere in the system. Briefs sort by modification time. That's it. Thirty briefs, all served in the order their files were last touched, with no awareness of which thread they belong to or whether they contain a question that needs your judgment.

## Resolution

The turn: "ranking intelligence" isn't one thing. When I ran each ranking signal through your own 4-question structural test from the openness matrix, three of them — thread-grouping, tension surfacing, batch coherence — failed the test for held-back. They're not future product leverage. They're the product's core commitments (thread-first, uncertainty treatment) expressing themselves through reading order. If threads are the primary artifact but the reading surface ignores thread structure, the product contradicts itself. That's not a ranking optimization — it's a method gap.

What surprised me: the infrastructure to close this gap already exists. When `orch complete` runs, `BackPropagateCompletion()` moves the beads ID into the thread's `resolved_by` list. Every brief already has a Tension section (the template requires it). The data model connects briefs to threads and flags open questions. The API just doesn't expose any of it. The briefs endpoint returns `{beads_id, marked_read}` — two fields, zero semantic context. The fix is enrichment: add `thread_slug`, `thread_title`, and `has_tension` to the response, then sort by thread clusters with tension items first. This is an afternoon of implementation, not a research project.

The genuinely held-back part is real but smaller than the matrix suggested: learned ranking from feedback data (the shallow/good mechanism exists but has zero consumers), cross-artifact attention integration (ranking briefs against threads and investigations), and collaborative priority. These need usage data that doesn't exist yet.

## Tension

This decomposition reclassifies a surface from the openness boundary matrix you accepted hours ago. The matrix treated ranking as a single held-back item; I'm saying half of it should be method core that ships now. That either means the matrix was wrong about granularity (likely — it was inventorying 28 surfaces in one pass, fine to miss a sub-decomposition), or my "thread-grouping is method-defining" claim is stronger than warranted and it's actually just a nice-to-have. The test: does chronological ordering of briefs feel wrong to you? If it doesn't, thread-grouping is a preference, not a commitment, and the matrix was right the first time.
