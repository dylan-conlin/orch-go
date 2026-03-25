# Brief: orch-go-z5z3u

## Frame

The daemon was treating completed agents as if they were still "recently spawned" — showing them as blocked in the spawn cache for up to 6 hours after finishing their work. This meant CountSpawnable returned wrong numbers, and if an issue needed to be re-spawned (verification failure, reopened), it was silently blocked. The symptom: kxtrd completed and the completion loop processed it, but the daemon kept logging "recently spawned, awaiting status update" on every cycle.

## Resolution

The spawn cache has three cleanup mechanisms: TTL expiry (6h), session reconciliation (startup only), and issue-status reconciliation (dead code, never called). None of them fired during normal operation when an agent completed. The fix was a 3-line addition: when `CompletionOnce` successfully processes a completion, call `Unmark()` to immediately remove the spawn cache entry. This is where the unmark logically belongs — the completion processing path already clears verification retry state and marks the dedup tracker, so clearing the spawn cache is the same kind of post-success cleanup. The daemon guide already documented `ReconcileWithIssues()` as dead code; this fix takes the simpler path of cleaning on success rather than reconciling on a schedule.

## Tension

`ReconcileWithIssues()` is still dead code. The direct unmark fixes the immediate bug, but the spawn cache still has only one active cleanup for edge cases: the 6h TTL. If completion processing fails persistently for an agent (verification keeps erroring), the spawn cache entry sits there blocking the issue until TTL. Adding periodic session reconciliation to the main loop would cover this, but it adds network I/O (OpenCode API + tmux check per entry) on every poll cycle. Whether that tradeoff is worth it depends on how often verification fails persistently — worth monitoring.
