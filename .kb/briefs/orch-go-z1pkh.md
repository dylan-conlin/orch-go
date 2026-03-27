# Brief: orch-go-z1pkh

## Frame

The question looked simple: can the new liveness checker accidentally declare a worker dead before its five-minute grace period expires? In this repo, that is the kind of question that can waste a lot of time, because a timing bug in one place often turns out to be a missing input or split responsibility somewhere else.

## Resolution

The turn was realizing that `pkg/verify/liveness.go` is stricter than it first appears, but not actually wrong on its own. It does exactly what the tests say: with a real spawn time, anything under five minutes is still alive, exactly five minutes is already out of grace, and no spawn time means the function has no basis for granting grace at all. So the feared bug is not "bad arithmetic in the checker."

The real leak is in how the checker gets called. `orch complete` reads spawn time from the workspace manifest before asking for liveness, but `orch abandon` does not. That means a freshly spawned worker with no phase comment yet gets judged as dead immediately in the abandon safety path, not because the checker forgot the grace period, but because the caller never handed it the one fact the grace-period branch depends on. That is a more interesting bug than the original suspicion because it says the contract around liveness inputs is fragile in a hotspot area, not just off by one.

## Tension

The open design question is whether callers should be allowed to construct partial `LivenessInput` at all when safety behavior depends on it. Dylan may want an architectural pass here, because this kind of omission is easy to repeat anywhere the code treats spawn metadata as optional.
