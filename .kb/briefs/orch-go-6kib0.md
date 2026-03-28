# Brief: orch-go-6kib0

## Frame

You've been here: the orchestrator detects its own frustration ("You're right. Let me stop"), then produces more of the same analysis. You export the session, close it, start fresh — and the new session immediately makes progress. The manual restart works because a new session means a new cognitive frame. This hook makes the system notice when it should suggest that restart, instead of waiting for you to do it yourself.

## Resolution

The hook is a bash script that fires on every message you type in a Claude Code session. It pattern-matches against 51 frustration signals across three categories: explicit frustration ("this isn't working"), repeated corrections ("I already said"), and abandon intent ("start over"). Each match increments a counter scoped to the tmux window. At three signals, Claude gets a nudge in its context: "This conversation may be fighting the user. Propose saving the question and starting fresh."

The boundary proposal, not the boundary itself, is the product. The hook surfaces the signal; you decide whether to act. If you do, a FRUSTRATION_BOUNDARY.md captures just the question and what didn't work — designed to be small enough that the next session gets the problem without inheriting the broken frame. 36 tests pass covering detection, false positives, thresholds, and JSON output.

One remaining step: registering the hook in settings.json. Workers can't modify their own hook config (sandbox protection), so that's a one-line command for you to run.

## Tension

The pattern list is handcrafted from the design doc's examples and intuition. Zero false positives in testing against clean messages, but real-world frustration has more texture than "this isn't working." First-week production observation will reveal whether the patterns catch what they should — and more importantly, whether they fire on things they shouldn't. The threshold of 3 is conservative by design, but tuning it requires data we don't have yet.
