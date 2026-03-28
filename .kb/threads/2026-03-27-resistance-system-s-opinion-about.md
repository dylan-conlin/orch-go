---
title: "Resistance is the system's opinion about whether work belongs"
status: subsumed
created: 2026-03-27
updated: 2026-03-28
resolved_to: ""
---

# Resistance is the system's opinion about whether work belongs

## 2026-03-27

Dylan noticed two modes of working: flow (issues appear, resolve without friction, the system has a place for everything) and grind (one bug spawns another, patches need patches, you're willing something into existence). The flow sessions tend to involve discovered work — you see something, name it, the system already wants it. The grind sessions tend to involve willed work — you decided something should exist before asking if the system has a natural place for it. The compositional accretion audit found the same pattern at the artifact level: opt-out signals (natural to creation) hit 100% adoption, opt-in signals (bolted on) plateau at 20%. Flow and grind might be the experiential version of that same distinction. If so, resistance isn't just 'bugs happen' — it's a design signal about whether the work belongs here, belongs yet, or should exist at all.

Dylan asked the critical follow-up: how do you distinguish 'bolted on' from 'just hard'? Not all resistance means misfit — some things are genuinely complex but still belong. The distinction forming: bolted-on resistance lives in the seams (boundaries between new thing and everything else), each fix reveals more fixes, the backlog grows as you work. Just-hard resistance lives in the interior (the logic itself is complex), each fix makes remaining problems clearer and fewer, the backlog shrinks. Concrete examples: VerificationTracker was bolted-on — worked but needed ResyncWithBacklog, then stale-state patches, then diverged from beads, fix count kept growing. orch compose clustering was just-hard — single-linkage chaining was tricky, but once mid-band filtering solved it, the rest fell into place. The legibility signal might be: does the fix count go down or up as you work?

Dylan sharpened the question: he doesn't look at code. From his seat, 'bolted on' and 'just hard' resistance feel identical — the thing keeps breaking. He can't see whether bugs cluster in seams (misfit) or in the interior (genuine complexity). Right now the legibility depends entirely on the orchestrator reading code and making the call. But the system already has the data: beads tracks issue chains (does fix A spawn fix B?), area labels show where bugs cluster, fix-to-feature ratio is computable, whether fixes stay fixed is in the history. None of this is surfaced as a signal Dylan can act on. This is a product surface: the system should distinguish 'bolted-on pattern' (4 fixes in 3 weeks, each spawning follow-ups) from 'just-hard pattern' (1 hard fix that stayed solved) and present that distinction as part of how it shows work — not after he asks, but ambient, the way the comprehension count is ambient.
