# Brief: orch-go-sfzdv

## Frame

You completed 8 agents via `orch complete` and the comprehension queue counter stayed at 8. The daemon gates spawning on this counter — if it never decrements, the daemon stays paused forever. This is the same shape as the verification tracker stale-state bug (orch-go-zem67): a counter that only goes up.

## Resolution

The comprehension label transition (`comprehension:unread` -> `comprehension:processed`) was inside a code block that only ran for open issues. But auto-completed agents — the majority of daemon work — are already closed by the time `orch complete` reviews them. So the transition was silently skipped every time. The fix is one structural change: move the transition outside the `!target.IsClosed` guard. "Orchestrator has reviewed this work" is a comprehension state, not an issue lifecycle state — it shouldn't be gated on whether the issue is open.

The tell was two items that had *both* labels simultaneously: `comprehension:unread` AND `comprehension:processed`. The daemon added `unread`, then the headless completion goroutine added `processed` (via its own `orch complete --headless` which also hit the same `IsClosed` guard and failed to remove `unread`). The other five items had only `unread` — no transition was ever attempted.

## Tension

The `!target.IsClosed` guard was likely added intentionally to prevent double-closing, but it swallowed three unrelated operations (triage removal, comprehension transition, verification signal) in one block. Any future post-lifecycle operation added inside that `if` block will silently fail for auto-completed agents. The pattern of "one guard, many operations" is a recurring source of silent failures in the completion pipeline.
