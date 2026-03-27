---
# THREAD METADATA
# title: Human-readable name. Threads are named by their core claim,
#   not their topic — "open-loop systems cause X" not "about open loops."
title: "Open-loop systems — the unifying pattern behind epistemic integrity failures"

# status: open | resolved | stale
#   open = still developing, new entries expected
#   resolved = thinking merged elsewhere (see resolved_to)
#   stale = no updates in 2+ weeks, may be abandoned or superseded
status: resolved

# created/updated: ISO dates. Updated changes with each new entry.
created: 2026-03-20
updated: 2026-03-22

# resolved_to: Where this thread's thinking landed.
#   Can be another thread, a decision, or a model.
#   Empty string if still open.
resolved_to: ".kb/threads/2026-03-22-coordination-self-awareness.md"
---

# Open-loop systems — the unifying pattern behind epistemic integrity failures

<!-- ABOUT THREADS
    Threads are living documents that track an evolving line of thinking.
    Each dated entry adds to the understanding. Entries are append-only —
    you don't edit old entries, you add new ones that refine or contradict.
    The value is in the *development* of the idea, not just the conclusion.
-->

## 2026-03-20

<!-- First entry: the initial observation or claim.
     Be specific about what you're claiming and what evidence supports it. -->

Core claim: every epistemic integrity failure we've encountered is an instance of an open-loop system — a system where the action path and the observation path are disconnected. Instances identified so far:

1. Behavioral accretion — code acts on growing data, doesn't observe the growth
2. Stale decisions — decisions constrain the codebase, nobody observes enforcement
3. Silent account restrictions — platform degrades users, no feedback channel
4. Gate bypass rates — gates deny actions, don't observe displacement
5. Architectural displacement — hooks prevent wrong action, don't observe what agents do instead
6. Measurement command identity crisis — 10 commands measure things, nobody measures who uses them
7. Knowledge decay — models claim things about code, nobody observes whether claims are still true

The framing shifts from "are we measuring enough" to "does the system close the loop between action and consequence." Measurement is one loop-closing mechanism. Gates, feedback channels, probes are others. The question is always: where is the loop open?

## Auto-Linked Investigations

<!-- Investigations that reference or were spawned from this thread.
     These are auto-populated by the knowledge system — you don't
     maintain this list manually. -->

Investigation complete. Verdict: diagnostic lens, not standalone model. 75% of models show the pattern but each has 60-75% domain-specific content open-loop can't express. The actionable finding: 14/16 open loops are missing SENSORS specifically. System is actuator-rich (spawn, gates, hooks), reference-signal-clear (project instructions, skills), but sensor-poor (no systematic consequence observation). The tautology guard: "failures persist because consequences aren't observed" is near-circular. The useful version is always asking WHICH control component is missing. Best home: diagnostic section in a measurement model. Daemon periodic tasks are already partially the sensor layer — just incompletely.

Connection to compositional correctness gap: the sensor gap IS a compositional correctness gap. Each component sensor validates its own concern, but no sensor checks composition. The structural prescription from both lenses is the same: every gate stack needs a gate one abstraction level above the others.
