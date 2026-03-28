# Brief: orch-go-cn3j0

## Frame

The daemon was silently deferring a design research issue behind a hotspot checker bug because it thought they were related. They weren't — they just happened to live in the same project. The sibling sequencing logic that prevents test-before-implementation ordering was treating every orch-go issue as a sibling of every other orch-go issue.

## Resolution

The sibling detection was built for the scrape project, where three issues came from a single `--explore` decomposition. For that context, "same project prefix" was a fine proxy for "same effort." But orch-go has dozens of independent issues, and the proxy broke down completely — any test-like issue got stuck behind any unrelated bug.

The fix was narrower than I expected. The relationship signal already existed: `epicChildIDs`, populated during Orient by expanding `triage:ready` epics. The sibling function just wasn't using it. Adding an `epicChildIDs` parameter and requiring both issues to be epic children restored the original intent (sequence --explore siblings) while eliminating the false matches. No new state, no new queries, no architectural change.

## Tension

The fix scopes sibling detection to epic children, which is correct for --explore spawns. But `epicChildIDs` is a flat set — it doesn't track which parent. If two unrelated epics in the same project both have test+impl children in the ready queue simultaneously, there could still be cross-epic false matches. In practice this seems vanishingly unlikely, but the structural gap is there. Worth noting if you ever see deferral happening between children of different epics.
