---
title: "Every spawn composes knowledge — task completion as side effect, not primary output"
status: forming
created: 2026-03-26
updated: 2026-03-26
resolved_to: ""
---

# Every spawn composes knowledge — task completion as side effect, not primary output

## 2026-03-26

Reading through 37+ briefs, Dylan noticed the brief 'frame' section reveals a real thought process — agents aren't just completing tasks, they're reasoning about the system. This sparked a question: what if knowledge composition was the contract for every spawn, not an optional side-effect?

Current state: agents produce work artifacts (code, fixes, designs) as primary output, with SYNTHESIS.md and BRIEF.md as completion artifacts. The brief is a report on what happened.

Forming idea: invert this. The brief — the composed understanding of what was learned, how it connects to threads, what changed in the model — is the primary deliverable. Task completion (code shipped, bug fixed) is substrate that enables the knowledge output.

This connects to:
- Decision 2026-03-26 (thread/comprehension layer is primary product) — this would operationalize it at the spawn level
- System inventory (16% core / 72% substrate) — if every spawn composes knowledge, the product surface expands without adding code
- Today's enrichment pipeline finding — thin issues produce blind agents precisely because the system treats knowledge as optional metadata, not primary output

The question isn't 'should agents also capture knowledge?' (they already do, inconsistently). The question is: 'what changes if knowledge composition is the acceptance criterion, not task completion?'

Reading all 13 briefs from today through the knowledge-composition lens surfaced five findings:

**1. The briefs are already doing it — unevenly.** The best briefs (account capacity routing, stall tracker) compose knowledge by correcting false beliefs. The weakest (kb context trace, duplicate spawn) report accurately but change nothing. The difference: whether the agent started from a genuine gap or wrong belief vs a procedure to follow.

**2. Frame quality determines composition quality.** When the Frame articulates a wrong belief, the Resolution naturally composes knowledge. When the Frame is 'we needed to know X,' the Resolution is a report. This means the enrichment pipeline's real leverage isn't description length — it's giving agents a belief to test rather than a task to execute.

**3. Tension sections are orphaned knowledge seeds.** Every brief ends with an open question. These are the forming questions for next-round work. Right now they get archived with the brief. Nothing structurally picks them up. If Tension sections fed into the thread graph, the system would accumulate open questions as a natural byproduct of work.

**4. The acceptance criterion question.** If knowledge composition is primary, agents that ship code but produce no brief haven't met the contract (6/17 workspaces today had no SYNTHESIS). 'Phase: Complete' would need to mean 'understanding composed,' not 'task done.'

**5. The 16%/72% ratio might not need to change.** If every substrate spawn also composes knowledge about the substrate, then substrate work IS simultaneously core work. The ratio stays but the meaning inverts — execution becomes the occasion for knowledge composition, not a separate activity.

Identity gap cluster: 8 briefs from today (f8y50, 1r7ih, z4h7s, wgkj4, vo51p, ey4py, pityd, bw0y6) independently discovered that every surface in the system still introduces it as something it isn't. Code is 16% core / 72% substrate. Daemon is 93% substrate. Dashboard has 3-4x execution over comprehension. README, architecture guide, CLAUDE.md all carry the old identity. The comprehension surface shows metadata not content. No individual brief names this pattern — it only becomes visible when you read the cluster together. This is the strongest evidence yet for the thread's thesis: the composition of findings IS the valuable output, and the system has no mechanism to produce it. The identity-gap insight happened manually in an orchestrator session. Architect issue orch-go-c5ha1 released to design the missing composition layer.
