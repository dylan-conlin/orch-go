---
title: "Architectural displacement — hard deny hooks prevent wrong action but don't ensure right action, agents route code to wrong packages"
status: resolved
created: 2026-03-19
updated: 2026-03-28
resolved_to: "culled: operational, create issue if resurfaces"
---

# Architectural displacement — hard deny hooks prevent wrong action but don't ensure right action, agents route code to wrong packages

## 2026-03-19

Two enforcement failure modes with opposite displacement effects. Advisory gates (accretion, hotspot) fail by letting code through — code lands in right place, constraint ignored. Hard deny hooks (governance file protection) fail by displacing code — agent can't write to pkg/spawn/gates/, puts concurrency logic in pkg/orch/spawn_preflight.go instead. Gate is respected, architecture is violated. Evidence: SCS agent scs-sp-8dm blocked from writing pkg/spawn/gates/concurrency.go, reported CONSTRAINT via beads comment, added check to spawn_preflight.go. 12 cross-project hook blocks recorded, 63 total hook denials, zero tracking of what agents did INSTEAD. This is a measurement blind spot: we measure the block but not the downstream architectural effect. Connects to: gates-without-attractors (architectural-enforcement model), gate passability decision (Jan 4), measurement honesty (unmeasured side-effects = absent signal). The question: should hard denies include an attractor ('put this code HERE instead') or should we accept displacement as the cost of control plane immutability?

Cross-model synthesis from exploration completion (3 parallel investigations). Displacement is the same failure mode appearing across 4 models: (1) Architectural enforcement: gates-as-signaling without attractors = death spiral variant. Prevention > detection > rejection layers 2+3 were missing. (2) Measurement honesty: extended with two-gap independence + structural undetectability ceiling — generalize beyond governance to all enforcement metrics. (3) Agent trust: displacement is a policy gap not enforcement gap — hook works perfectly but policy layer (skill docs) gives wrong info (69 files claimed protected, 2 actually hooked). Weakest link determines actual trust. (4) Skill content transfer: phantom protection zone is wrong knowledge transferring reliably — worse than behavioral dilution because agents faithfully learn incorrect constraints. Cross-cutting: system optimized to prevent wrong actions but not enable right ones. Enforcement without guidance = displacement. Guidance without accuracy = phantom constraints. Measurement without consequence-tracking = false confidence.
