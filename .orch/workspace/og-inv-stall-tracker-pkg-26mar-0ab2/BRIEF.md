# Brief: orch-go-acohg

## Frame

This looked simple at first: read the stall tracker and explain when an agent gets marked stalled. The surprise was that the real question was not just "what threshold is configured," but "what clock is the code actually measuring" once the tracker is exercised by the dashboard and status paths.

## Resolution

The tracker says "3 minutes without token progress," but the code does something narrower. Every `Update` call replaces the saved snapshot before checking for a stall, so the timer is really measuring the gap between two identical samples, not the total time an agent has been flat. When I ran a small probe against the real code, repeated unchanged 1 second samples never crossed a 3 second threshold; only a later 4 second gap did. That turns the warning into a missed-poll detector more than a sustained-no-progress detector.

The downstream path is also milder than the name suggests. `orch status` and `/api/agents` flip `IsStalled`, the formatter prints `STALLED`, and the attention collector can surface the agent for human review. But there is no automatic recovery in the traced code, and the same flag is also used for a separate 15 minute phase-timeout path. So the system is carrying one warning bit that currently means at least two different things.

## Tension

The open question is whether stall detection should be defined around token inactivity, polling gaps, or a richer liveness contract that separates those two. There is also a migration smell here: the tracker now expects `execution` token types while the current callers and tests still speak `opencode`, which means the semantic fix and the type-boundary fix probably want the same architectural pass.
