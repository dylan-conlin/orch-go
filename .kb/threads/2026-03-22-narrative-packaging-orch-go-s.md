---
title: "Narrative packaging — what is orch-go's while you sleep"
status: open
created: 2026-03-22
updated: 2026-03-28
resolved_to: ""
---

# Narrative packaging — what is orch-go's while you sleep

## 2026-03-22

The harness engineering blog post got 0 HN traction. autoresearch got 48k stars with 'AI agents running research while you sleep.' The gap isn't technical — it's that autoresearch has a sentence anyone can feel. orch-go's capabilities (multi-agent orchestration, skill routing, governance, comprehension queues) are real but they're mechanism descriptions, not stories. The AI infra role search needs a calling card with a hook: 'I built X that does Y while you Z.' Candidates to explore: 'ship features while you sleep' (too generic), 'AI agents that know when they're wrong' (harness engineering angle), 'the system that teaches itself what to audit' (quality audit bootstrapping). None of these land yet. The autoresearch comparison is useful as a foil — it shows the boundary where orchestration isn't needed, which frames why it IS needed for everything else.

## 2026-03-27

Validation step clarified: the methodology isn't ready to hand to someone else because it's still forming (thread→model promotion was built today). The one-paragraph test: can you explain how thinking compounds without referencing orch-go internals? Current best: 'you name what you don't understand, keep it visible, and let separate not-understandings find each other.' Lea at SCS is the candidate first user — she liked Cowork, which has hook infrastructure. But the methodology needs to finish crystallizing first. Product phase: forming, not ready.

## 2026-03-28

Shift: narrative needs evidence first, not framing. Bibliometrics finding (p=0.0086) is the first concrete result — papers' open questions cluster tighter than findings, nobody has shown this before. The tool (embed gaps separately), the finding (gaps self-organize), and the method (research cycle) are all expressions of the same principle. The story writes itself when the evidence exists.

Market signal (Reddit, 2026-03-28): Claude computer use + OpenClaw major update same day. Reddit frames it as 'two approaches to AI that does work instead of talking about it' (76 upvotes). OpenClaw ships plugin SDK, skill auto-mapping, multi-model routing. Key insight: both are execution orchestration — route tasks to models, manage tools. Neither addresses understanding orchestration — knowledge that composes, gaps that self-organize, research cycles that pressure their own models. The market is crowding 'AI that does work.' Nobody is claiming 'AI that understands what it's doing.' That's the differentiated position. The bibliometrics result is more valuable than product announcements because it's evidence for a claim nobody else is making.

Session 2026-03-28: named incompleteness absorbed 6 threads, gained a cross-domain instance (sketchybar — human attention, not AI), and had 3 claims probed (2 confirmed, 1 directionally supported). Research cycle went from design to running end-to-end in one session. The model's core holds; measurement limitations are at the instrument layer (TF-IDF), not the theory layer. Evidence base is now strong enough to write from.
