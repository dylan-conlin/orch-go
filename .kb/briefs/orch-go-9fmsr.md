# Brief: orch-go-9fmsr

## Frame

The daemon was refusing to spawn new work because its comprehension throttle showed 6/3 — way over the limit. But those items had already been completed. The labels just never got removed, so the daemon thought there was a backlog of unreviewed work when there wasn't one.

## Resolution

Two things were wrong with how `orch complete` handled comprehension labels. First, daemon-triggered completions (`--headless`) were explicitly excluded from label removal — the original design assumed headless completion wasn't "real" comprehension, so the label should stay. But that meant every daemon auto-complete left a `comprehension:unread` label on a closed issue, permanently incrementing the throttle counter. Second, the label removal happened after `bd close`, which meant it was operating on a closed issue — a fragile ordering that could silently fail.

The fix is simple: strip all comprehension labels before closing the issue, and do it for every completion path. Completion is comprehension — there's no reason for any comprehension label to survive it. Two closed issues (`orch-go-nxfxi`, `orch-go-b3twh`) were found with stale labels and cleaned up.

## Tension

The original `!completeHeadless` guard was an intentional design choice, not an oversight — someone decided that automated completions shouldn't count as human review. That was probably right in principle, but it created a leak: labels accumulated on closed issues with no mechanism to drain them. The daemon's comprehension throttle assumes all `comprehension:unread` items are actionable; closed issues with stale labels break that assumption. Is there a case where we'd want comprehension labels to survive completion, or is "completion = comprehension" the right invariant going forward?
