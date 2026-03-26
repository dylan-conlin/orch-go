# Brief: orch-go-cig47

## Frame

An investigation found two bugs in the stall tracker — it was resetting its timer on every poll, and it depended on the wrong token type. The orchestrator routed this to architect review because the file is a hotspot and the `IsStalled` flag mixes different kinds of stall. The question was: how should stall detection be redesigned before patching?

## Resolution

I went in expecting to design a fix for two bugs and came out realizing both were already fixed. Commit `c4d4aa496` corrected the timestamp reset so cumulative polling now works. Commit `5062bdf08` unified everything on `execution.TokenStats`. All tests pass, project compiles clean.

The surprise was what the investigation *didn't* catch: `IsStalled` means different things depending on where you read it. The dashboard handler sets it from four separate code paths — token stall (3 min no progress), phase stall (15 min no update), never-started agents, and stale spawns. The CLI sets it only from the token tracker. The doc comments on the two structs literally describe different conditions. This is textbook Defect Class 5 — the same field name carrying contradictory meaning across consumers. The attention system reads `is_stalled` from the dashboard API and has no way to know *which* kind of stall it's looking at.

The fix is boring in the best way: add a `StallReason` string field alongside the existing boolean. Set it to `"token_stall"`, `"phase_stall"`, `"never_started"`, or `"spawn_stale"` at each setter site. The boolean stays for backward compatibility. I created orch-go-8e15i for the implementation (~4 touch points).

## Tension

Both prior bugs were fixed by different agents in recent commits, but neither noticed the semantic overloading problem. This raises a question about whether the current investigation→architect pipeline is too narrowly scoped — the investigation found *what was broken* but not *what was conflated*. The distinction matters: fixing a bug restores intended behavior, but noticing conflation is how you prevent the next class of bugs in the same area.
