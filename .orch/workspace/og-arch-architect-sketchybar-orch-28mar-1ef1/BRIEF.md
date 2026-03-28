# Brief: orch-go-ngzhu

## Frame

The sketchybar widget had four reliability bugs in a week — stale zeros, wrong comprehension counts, unreliable logs, repeated accuracy investigations. Each time, the fix was on the daemon side, not the widget side. This session asked the obvious structural question: is the widget architecture wrong, or were we just fixing the wrong thing four times?

## Resolution

I expected to find a flawed architecture that needed redundant data sources or health checks. Instead I found the opposite: the widget's design — poll a file, check its mtime for liveness, fall back to beads when the daemon is dead — is exactly right. Every failure traced to daemon-status.json containing bad data, not to the widget misinterpreting good data.

The surprising part is that the structural fix already happened, distributed across four separate sessions without anyone naming it as a redesign. The throttle collapse removed the in-memory counter that diverged. The mtime check catches daemon death. The bd fallback provides live data when needed. The integration test catches Go/bash drift. Put together, these are the redesign — they just didn't arrive as a single coherent change. The widget was never the problem. It was a faithful mirror of a daemon that kept getting things wrong, and the daemon is now getting things right.

## Tension

The widget works because it trusts one file written by one process. If that trust is ever wrong again — a new field sourced from in-memory state, a regression in the atomic write path, a new subsystem that doesn't flow through daemon-status.json — the widget will faithfully display the wrong answer again, and the instinct will again be to "fix the widget." The question is whether the contract between daemon and widget (daemon-status.json as the single canonical interface, sourced from authoritative stores) should be documented explicitly enough that the next developer who adds a daemon feature knows what they're signing up for.
