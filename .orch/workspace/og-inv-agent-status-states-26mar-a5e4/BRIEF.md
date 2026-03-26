# Brief: orch-go-8mw0y

## Frame

The question looked simple on the surface: what states can the liveness system put an agent in? But this mattered because the answer controls whether destructive commands like complete and abandon stop, warn, or proceed. If the state machine is fuzzy in our heads, every later bug report about "agent still running" turns into guesswork.

## Resolution

The useful turn was realizing that `pkg/verify/liveness.go` is much smaller than the surrounding behavior makes it feel. It is not a workflow engine that knows about Planning, Implementing, or Testing as distinct runtime states. It is a coarse classifier with three outputs: active, completed, and dead. The only things that move an agent between those states are the latest parseable `Phase:` comment, the special meaning of `Complete`, and a five-minute grace window after spawn when silence is still treated as activity.

What made this initially easy to misread is that the surrounding commands do more than the core classifier. `orch complete` uses liveness as a warning and confirmation prompt. `orch abandon` adds its own 30-minute recency rule for non-complete phases. So the bigger system feels richer than the underlying state machine actually is. The investigation now separates those layers cleanly: shared classifier here, caller policy there.

## Tension

The open question is whether that split is still serving us in this hotspot area. Keeping caller-specific policy outside `VerifyLiveness` keeps the core logic simple, but it also makes it easier for different commands to drift. If liveness-related confusion keeps resurfacing, the next move probably is not another tactical fix but an architectural pass on where freshness policy should live.
