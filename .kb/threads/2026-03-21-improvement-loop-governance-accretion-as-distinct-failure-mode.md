---
title: "Improvement loop — governance accretion as a distinct failure mode"
created: 2026-03-21
status: active
---

# Improvement loop — governance accretion as a distinct failure mode

## 2026-03-21

There are (at least) two ways a multi-agent system fails:

**Degradation loop (entropy spiral):** Agents break each other's work. Fix:feat ratio climbs. The system visibly gets worse. You can see it happening and hit the brakes. This is what happened three times in orch-go (1,625 lost commits). The harness engineering model was built to explain and prevent this.

**Improvement loop (governance accretion):** Agents build governance infrastructure that works. Each addition is locally justified, tested, completed successfully. Nothing breaks. The system looks healthier the more you measure it. But measurement becomes the main activity. 35% of the codebase exists to watch the other 65%. The daemon spends 85% of its cycles on meta-work. 22 of 26 periodic tasks are governance. The cure metastasizes.

These are structurally different:

| Property | Degradation loop | Improvement loop |
|----------|-----------------|------------------|
| Visible? | Yes — things break | No — things work |
| Danger signal | Fix:feat ratio climbing | None obvious |
| Agents' output | Individually broken | Individually correct |
| Composition | Destructive interference | Constructive but unbounded |
| Feels like | Crisis | Productivity, then directionlessness |
| Stopping signal | Pain | ... nothing? |

The improvement loop has no natural brake. There's no metric that says "you have too much measurement." Every new piece of measurement infrastructure *improves* the metrics it reports on. The system reports itself as healthier the more governance you add.

**The discovery path was itself the proof.** This wasn't found by a probe, a tension cluster scan, or a model drift detector — all of which are instances of the pattern. It was found by the human stepping back and saying "why do I feel directionless?" The feeling of having nothing urgent to do, in a system with 777 orphaned investigations and 38 stale decisions and 22 daemon tasks churning, was the signal.

**Connection to "gates producing gates" (Layer 4):** The harness engineering model aspired to build self-extending governance — gates that generate gates. The irony: this was already happening organically. Amnesiac agents were emergently building coordination infrastructure, one justified commit at a time. Layer 4 wasn't needed because the improvement loop *is* Layer 4, uncontrolled.

**The brake is the human in the loop.** The open question was "what would a brake look like?" and the answer turned out to be simple: the human making decisions. The improvement loop ran unchecked precisely during the period when Dylan wasn't actively engaging with the system's output — not reading the investigations, not acting on recommendations, not making strategic calls. The agents kept producing because nothing told them to stop. The governance kept growing because every addition was locally justified and no one was asking "but is this producing meaning?"

This reframes the failure mode. It's not that the system lacked a metric or a gate. It's that multi-agent systems without active human decision-making will fill the vacuum with self-referential infrastructure. The agents don't know they're done. They can't feel directionlessness. Only the human can, and only if they're paying attention.

The degradation loop forces engagement — things break, you have to respond. The improvement loop doesn't. It feels like the system is working, so you let it run. And it does run. It just runs in circles.

**Implication for harness engineering:** No amount of automated governance substitutes for a human who is present and making decisions. The harness can prevent destruction (entropy spirals). It cannot prevent purposelessness (improvement loops). That requires someone asking "why are we doing this?" — and meaning it.

**Follow-on: orchestrator as scoping agent, daemon as sole executor.** The improvement loop partly resulted from the orchestrator having access to `orch spawn` — bypassing the daemon whenever friction arose. This meant the daemon never experienced the pressure it needed to improve as an attractor. New design: orchestrator's only output is well-scoped beads issues. Daemon is the sole spawn mechanism. This forces scoping quality (the issue IS the work product), separates judgment from execution, and routes all friction through the daemon where it can drive improvement. See thread: `2026-03-21-orchestrator-scoping-daemon-pressure`.

## Auto-Linked Investigations

- .kb/investigations/2026-03-25-inv-investigate-operationalizing-ralph-loop-orch.md
- .kb/investigations/archived/2025-12-22-inv-debug-session-id-write.md
- .kb/investigations/archived/2025-12-21-inv-failure-mode-artifacts.md
