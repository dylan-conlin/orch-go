# Brief: orch-go-qz5y5

## Frame

Seven new implementation issues — all spawned from the same probe's recommendations — were dead on arrival. The daemon's commit dedup gate saw that each issue's description mentioned a completed investigation, found commits for that investigation, and concluded the work was already done. It couldn't tell the difference between "this is the same thing" and "this cites the thing it came from."

## Resolution

The gate was checking: "does any beads ID in this issue's description have commits?" But descriptions naturally cite their origin — an implementation task born from an investigation will always reference that investigation. The fix adds type awareness. When a task references an investigation (or any different-type issue), the gate recognizes it as a citation and lets it through. A task referencing another task with commits is still flagged as a likely duplicate. The key insight was that the `Issue` struct already carried type information — it was right there in the data, just unused by the gate. The production wiring reuses the same RPC-first, CLI-fallback pattern as the existing status lookup, failing open if the type can't be resolved.

## Tension

The fix handles the clear case (investigation→task is never a duplicate) but the ambiguous middle ground exists: what about feature→feature references where one is genuinely building on the other? Same-type follow-up work that cites its predecessor will still get blocked. This hasn't shown up yet — all 7 false positives were cross-type — but if the pattern of spawning follow-up issues from completed work grows, the same-type case may need its own heuristic.
