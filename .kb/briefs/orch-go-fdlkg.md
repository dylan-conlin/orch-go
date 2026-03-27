# Brief: orch-go-fdlkg

## Frame

The sketchybar widget had three open questions that kept getting deferred: should the bar show "2/5" or just "2" with color? Should it include comprehension count even though that costs 500ms per poll? Should the scripts live in the orch-go repo? They looked like three independent choices, but they kept blocking each other because nobody had named what the widget actually *is*.

## Resolution

The turn came when I stopped trying to answer the questions individually and asked: what's the widget's relationship to the orchestration system? It's a passive status reflector — like a shell prompt or terminal theme. It reads state files; it doesn't participate in the system. Once that's clear, each fork falls out naturally.

Active/max with color stays because it gives you two independent channels — the fraction tells you utilization, the color tells you health. Collapsing to active-only-with-color overloads one signal. The comprehension count belongs in daemon-status.json, not in a separate bd CLI call at the widget layer, because the daemon already computes it every cycle for spawn throttling and then throws the number away. Writing one more field to the status file turns a 500ms poll cost into zero. And the widget stays in `~/.config/sketchybar/` because the integration test already catches health-computation drift between Go and bash — repo ownership would add commit noise for color changes without improving correctness.

One follow-up issue: add `ComprehensionPending` to the `DaemonStatus` struct (orch-go-67ja3). Everything else is widget-layer work outside the repo.

## Tension

The daemon only calls `CheckPreSpawnGates` (where comprehension count lives) when it's about to spawn. If no work is ready, the count doesn't refresh. The implementation might need a separate periodic query rather than piggybacking on the gate check — and that means the "zero cost" claim isn't quite zero; it's "one bd CLI call per poll cycle regardless of spawn activity." Still cheaper than calling it from bash, but worth watching.
