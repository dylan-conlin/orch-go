# Brief: orch-go-uiv9d

## Frame

The daemon's completion pipeline has two paths that work (headless brief generation, label-ready-review) and one that doesn't (light auto-complete for effort:small agents). Today it went 0 for 12. The "50% failure rate" framing is misleading — it's not random flakiness. It's one path working perfectly and another failing every time.

## Resolution

I expected something subtle — maybe a synthesis/tier mismatch from the architect's recent tier-aware design, or a race condition in the completion pipeline. Instead it was mechanical: `CompleteLight` skips explain-back (gate1) but not behavioral verification (gate2). Both gates require a human. The daemon has no human. Gate2 fails every time.

The interesting part is *why* this wasn't caught. `CompleteLight` was written before `--headless` existed. It was designed to be "lighter" by manually skipping individual gates. When `--headless` was added later — which correctly handles the entire "no human present" case by forcing `review-tier=auto` — nobody went back to update `CompleteLight`. The two approaches coexist: the old one (skip individual gates) and the new one (declare the entire session non-interactive). The old one doesn't compose because gate skips are independent — skipping gate1 doesn't imply anything about gate2.

Fix: `CompleteLight` now uses `--headless`. One flag, correct semantics, brief generation included.

## Tension

`Complete()` (the *full* auto-complete path, for review-tier=auto/scan agents) has the same class of bug: it passes `--force` without the required `--reason` flag. It hasn't blown up yet because no agents currently route through that path. But the first time one does, it'll fail the same way. Filed as orch-go-o0nqu, but the pattern question is: how many other daemon-to-CLI interfaces have flag mismatches that only surface when the path is first exercised?
