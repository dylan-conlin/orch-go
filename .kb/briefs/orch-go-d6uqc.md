# Brief: orch-go-d6uqc

## Frame

Orient was trying to be everything at once. Fifteen sections of operational metrics, model freshness checks, daemon health signals, adoption drift warnings — all injected into the orchestrator's first breath of every session. The decision that the product is a thinking surface, not an ops dashboard, meant most of this was noise. The question was whether you could strip it cleanly without losing information or breaking the JSON API that other tools consume.

## Resolution

The split turned out to be surgically clean. FormatOrientation now renders five things: what you're thinking about (threads), what was recently learned (briefs — new), what's still contested (tensions), what's ready to work on, and where you're focused. Everything operational — throughput, daemon health, model freshness, adoption drift, divergence alerts, explore candidates, reflection suggestions — moved to a new FormatHealth renderer that `orch health` calls after its existing score output.

The brief scanning was the only genuinely new capability. It reads `.kb/briefs/`, extracts the first sentence of the Frame section as a title, checks for tension presence, and consults the same read-state file the web UI uses. The result is a compact list: 21 unread, here are the latest 5, these have open tensions.

The struct that carries all this data didn't change shape — it just got comments marking which sections belong to which renderer. JSON output still returns everything. The `--hook` path gets the thinking surface. The split is purely at the rendering layer, which is why 80+ existing tests only needed their function call changed from FormatOrientation to FormatHealth, not their assertions rewritten.

## Tension

The operational sections still get *collected* even when orient only renders the thinking surface, because the JSON output path needs them. For the `--hook` path (the common case), this is wasted work — daemon health queries, git numstat, reflect suggestions, all computed and discarded. Whether to skip operational collection for `--hook` is a performance question that only matters if orient hook latency becomes noticeable. Right now it's not — but it's the obvious next optimization if it does.
