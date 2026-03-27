---
title: "Epistemic dishonesty — the system conflates didn't-check with nothing-there"
status: forming
created: 2026-03-26
updated: 2026-03-26
resolved_to: ""
---

# Epistemic dishonesty — the system conflates didn't-check with nothing-there

## 2026-03-26

Five briefs from today share a pattern none of them name: the system routinely treats absence of evidence as evidence of absence. (1) kb context timeout returns nil, gap analysis scores 0/100 — 'I didn't wait long enough' becomes 'nothing exists' (304ta, k6c0v). (2) Stall tracker rewrites its timestamp every poll, so it measures poll spacing not liveness — 'I checked recently' becomes 'it's alive' (o5uih). (3) VERIFICATION_SPEC.yaml is evidence for humans but completion gates never read it — 'proof was supplied' becomes 'proof was verified' (fsikn). (4) Synthesis compliance counts an artifact the system stopped requiring at light tier — 'not produced' becomes 'not compliant' (n4uwb). (5) Liveness checker called without spawn time treats missing input as no-grace-period — 'caller forgot to provide context' becomes 'agent is dead' (z1pkh). Each was investigated independently as a bug. Together they're a design principle the system is missing: unknown and absent are different states that demand different responses. The structural fix isn't five patches — it's making the system's type system distinguish between 'checked: nothing' and 'not checked.' Connects to comprehension artifacts thread: if the composition layer conflates 'I clustered these briefs' with 'I understood them,' it would be the 6th instance.
