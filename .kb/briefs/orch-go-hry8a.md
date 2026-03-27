# Brief: orch-go-hry8a

## Frame

The comprehension throttle — the thing that's supposed to pause the daemon after 3 agent completions so you can actually review what they did — wasn't working. The daemon logged "Human verification detected" every two minutes, but you hadn't touched anything. It had been doing this all day while 4 briefs sat unread and 34 thin-description agents spawned unchecked.

## Resolution

The mechanism itself was fine. `RecordHumanVerification()` correctly resets the counter, `IsPaused()` correctly blocks spawning. The problem was upstream: `WriteVerificationSignal()` — the function that tells the daemon "a human reviewed something" — was called from three places, and two of them aren't human. The dashboard API close endpoints (single and batch) fire when the orchestrator AI closes issues, not just when you do. And `orch complete --headless` (daemon-triggered completions) was also writing the signal. Every automated close looked like you sitting down to review.

The fix removes the signal from both dashboard API paths entirely (no reliable way to distinguish human from bot on an HTTP endpoint) and gates the CLI path on `!completeHeadless && !target.IsOrchestratorSession`. Now only interactive `orch complete` — you at a terminal, not headless, not an orchestrator session — writes the verification signal. Two structural tests scan the source files to prevent anyone from re-adding the calls.

## Tension

Closing issues from the web dashboard no longer resets the daemon's verification counter. If you review completed work through the dashboard UI instead of `orch complete`, the daemon won't notice. The escape hatch is `orch daemon resume`, but the gap is real: the dashboard is becoming a primary review surface, and now it can't express "I reviewed this." Worth thinking about whether the dashboard needs an explicit "I've reviewed, resume daemon" button — something that can't be triggered by automated orchestrator sessions.
