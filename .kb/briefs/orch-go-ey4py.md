# Brief: orch-go-ey4py

## Frame

You accepted that the comprehension layer is the product, not the execution layer. But pkg/daemon/ is the single largest surface in the repo — 40K lines, 69 files — and the question was: how much of that mass is actually serving comprehension, and how much is generic job-runner plumbing? Without a concrete answer, the daemon's sheer size would keep pulling investment and attention by gravitational default, regardless of the product decision.

## Resolution

I expected a messy spectrum. What I found was cleaner than that. About 93% of the daemon is pure execution infrastructure — pool management, spawn pipelines, OODA cycles, cleanup, rate limiting. Standard job-queue stuff that any orchestration system needs. But there's a thin vein running through it, about 7%, that implements something distinctive: the principle that execution should pace itself to human understanding.

This vein clusters at three seams. The first is the comprehension queue — the daemon counts how much unread work has piled up and stops spawning when the pile gets too high. The second is the knowledge quality loop — reflect analysis that checks whether what the system "knows" is still coherent, audit selection that spot-checks whether completed work was trustworthy. The third is the human-attention handoff — question detection that surfaces "an agent needs you to think," completion routing that feeds the review queue.

None of these are large. Together they're about 3,500 lines. But they're where the daemon stops being a generic task runner and starts implementing a method. The boundary test turned out to be one question: "Does this behavior change what the human understands, or only what the system executes?" It resolves cleanly for 95% of behaviors.

## Tension

The classification says 93% of daemon work is substrate. That's fine — substrate is supposed to be big. But it means daemon tickets will keep appearing in the backlog, looking like product work because they live in a large important package, when almost all of them are maintenance of replaceable infrastructure. The filter exists now. The question is whether it actually changes how work gets prioritized in practice, or whether the daemon's mass continues to capture attention by default.
