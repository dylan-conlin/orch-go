# Brief: orch-go-dy8ev

## Frame

When a thread gets promoted to a model, the scaffold creates `model.md` and a `probes/` directory — but not `claims.yaml`. This matters because the entire claims infrastructure (orient edges, daemon probe generation, the completion pipeline) expects `claims.yaml` to exist. The result: every newly promoted model was born broken. The completion pipeline would crash trying to update claims, and orient couldn't surface the model's knowledge edges.

## Resolution

The fix was eight lines in `scaffoldPromotionArtifact()`: import the claims package, create a seed `claims.yaml` with the model name and an empty claims array. The plumbing already existed — `claims.SaveFile()` handles serialization — the scaffold just never called it. What made this click was seeing that model directories are supposed to be *knowledge units* (model.md for understanding, claims.yaml for assertions, probes/ for evidence), and the scaffold was only creating two of the three. The test I wrote first confirmed the gap: `TestThreadPromoteCmd_Model` now verifies claims.yaml exists with the correct model name and version.

## Tension

`kb create model` has the exact same bug — it creates model.md + probes/ but no claims.yaml. That's a separate code path I didn't touch. Should both paths share a single "create model directory" function, or is the duplication acceptable given how small the creation logic is?
