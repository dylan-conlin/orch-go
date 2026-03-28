---
title: "Open-loop systems — the unifying pattern behind epistemic integrity failures"
status: resolved
created: 2026-03-20
updated: 2026-03-22
resolved_to: ".kb/threads/2026-03-22-coordination-self-awareness-system-that.md"
---

# Open-loop systems — the unifying pattern behind epistemic integrity failures

## 2026-03-20

Core claim: every epistemic integrity failure we've encountered is an instance of an open-loop system — a system where the action path and the observation path are disconnected. Instances identified so far:

1. Behavioral accretion — code acts on growing data, doesn't observe the growth
2. Stale decisions — decisions constrain the codebase, nobody observes enforcement
3. Silent account restrictions (Anthropic) — platform degrades users, no feedback channel
4. Gate bypass rates — gates deny actions, don't observe displacement
5. Architectural displacement — hooks prevent wrong action, don't observe what agents do instead
6. Measurement command identity crisis — 10 commands measure things, nobody measures who uses them
7. Knowledge decay — models claim things about code, nobody observes whether claims are still true

The framing shifts from 'are we measuring enough' to 'does the system close the loop between action and consequence.' Measurement is one loop-closing mechanism. Gates, feedback channels, probes are others. The question is always: where is the loop open?

## Auto-Linked Investigations

- .kb/investigations/2026-03-01-dsl-design-principles-natural-language-embedded.md
- .kb/investigations/2026-03-20-inv-investigate-whether-open-loop-systems.md
- .kb/investigations/archived/2025-12-27-inv-pattern-analyzer-repeated-behavioral-failures.md

Investigation complete (orch-go-bhm8g). Verdict: diagnostic lens, not standalone model. 75% of models show the pattern but each has 60-75% domain-specific content open-loop can't express. The actionable finding: 14/16 open loops are missing SENSORS specifically. System is actuator-rich (spawn, gates, hooks), reference-signal-clear (CLAUDE.md, skills), but sensor-poor (no systematic consequence observation). The tautology guard: 'failures persist because consequences aren't observed' is near-circular. The useful version is always asking WHICH control component is missing. Best home: diagnostic section in measurement-honesty model. Daemon periodic tasks are already partially the sensor layer — just incompletely.

Connection to compositional correctness gap: the sensor gap IS a compositional correctness gap. Each component sensor validates its own concern, but no sensor checks composition. The structural prescription from both lenses is the same: every gate stack needs a gate one abstraction level above the others. The consequence sensor field (orch-go-3szd5) forces architects to think compositionally. The next step would be an actual composition gate in the completion pipeline — 'did this agent's output compose well with the system it landed in?' Even a cheap version (file-overlap detection between concurrent agents) would catch obvious cases.
