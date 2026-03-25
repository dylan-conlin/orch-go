# Brief: orch-go-swrwn

## Frame

The first real brief landed today — a half-page on the coordination experiment confound. It was good. Dylan read it. Then `orch complete` ran, the review queue cleared, and the brief vanished from the dashboard. The thing designed to be read over coffee had no place to be found over coffee.

## Resolution

The review queue is doing its job — it shows completions that need verification, and once verified, they clear out. That's correct operational behavior. The problem isn't the review queue; it's that briefs were bolted onto it as their only UI surface. Briefs are reading artifacts, not verification artifacts. They need their own home.

The design gives them one: a `/briefs` page, following the same pattern as the existing `/thinking` page. One new API endpoint scans `.kb/briefs/` to list what exists. A Svelte store fetches and caches. The page shows all briefs with expand/collapse and mark-as-read — independent of whether the originating issue is still open, pending review, or long closed. The existing review-queue brief button stays (it's useful for fresh completions), but the briefs page is the persistent reading queue.

Three implementation issues, clean dependency chain: API endpoint first, frontend in parallel, integration verification last.

## Tension

The in-memory read state resets when the server restarts. For now that's fine — briefs accumulate slowly and restarts are rare. But the deeper question: is binary read/unread enough? The thread says briefs should provoke conversation, not replace it. "Mark as read" might create false comprehension — Dylan reads, marks, moves on, but never has the reactive moment. Should the reading queue eventually support annotation (a "let's discuss" button, a "question" tag) that feeds back into the next orchestrator session? That's a design space this V1 intentionally doesn't enter.
