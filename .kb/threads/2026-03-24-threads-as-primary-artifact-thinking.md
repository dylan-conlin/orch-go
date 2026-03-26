---
title: "Threads as primary artifact — agents produce knowledge, threads compose it, the system is the attractor field"
status: active
created: 2026-03-24
updated: 2026-03-25
resolved_to: ""
spawned_from: comprehension-artifacts-async-synthesis-delivery
spawned: []
absorbed:
  - artifact-attractors-agents-naturally-externalize (merged 2026-03-25)
active_work: [orch-go-jwteq, orch-go-zsj78, orch-go-gfka3, orch-go-3htj3, orch-go-xwbby, orch-go-abz6a, orch-go-gvo79]
resolved_by: [orch-go-jwteq, orch-go-zsj78, orch-go-gfka3, orch-go-3htj3, orch-go-xwbby, orch-go-abz6a, orch-go-gvo79]
---

# Threads as primary artifact — agents produce knowledge, threads compose it, the system is the attractor field

## 2026-03-24

Lineage: decidability graph (decisions have dependencies) → question entities in beads (not all work is tasks, some is questions) → architect fork design (ideas branch, system should track it) → thread connective tissue (today). Each touched a different face of the same need: thinking has structure, and that structure should be first-class. Today's insight: threads aren't a side artifact — they're the primary artifact. Investigations gather evidence for threads. Decisions resolve branches of threads. Models formalize what threads have learned. Agent work tests thread claims. The thread is the spine; everything else hangs off it. This inverts orch-go's current model where beads issues are primary (work to be done) and threads are captured mid-session as a side effect. The way Dylan actually thinks — and the way this session demonstrated — threads are the driver and work is what threads produce when they need evidence. Minimal structure needed: spawned_from/spawned (thread-to-thread lineage), active_work/resolved_by (thread-to-work grounding). Together these make the thinking path traversable: where did this thought come from, what work tested it, what did that work find, where did the thought go next. Full lifecycle of an idea from forming question to evidence to next question.

The work graph (localhost:5188/work-graph) shows the daemon's view: dependency chains, issue types, agent status. Dylan says it never resonated — it's not how he thinks. He thinks in questions that branch and sharpen, not in dependency chains. The visualization that would resonate is a thread graph: threads as nodes, spawned_from/spawned as edges, active work hanging off each thread as evidence. The work graph becomes a sub-view of the thread graph — what the system is doing in service of each thread. This inverts the current dashboard architecture: instead of work-centric with threads as metadata, it becomes thread-centric with work as grounding. The thread graph is where Dylan would start his day — 'what am I thinking about, what has the system learned about each question, where do I want to go next' — instead of 'what agents are running.'

Session rhythm change shipped: orchestrator skill session protocol now thread-first. Start protocol inverted — threads are step 1 (not step 6), fires only surface if blocking thread progress, work proposed in service of chosen thread. End protocol leads with thread trajectory (moved/formed/converged) before task summary. Fast path table updated. This is the first behavioral test of the thesis: if threads are primary, the session cadence should follow threads, not backlog. Living with this for a few sessions to see what changes.

Briefs-in-isolation problem confirms thread-centric UI need. BRIEF.md with only a beads ID is an orphaned artifact — same problem threads solve for investigations and decisions. The natural fix: briefs render as evidence within their thread context, not as a flat list. When Dylan reads a thread, he sees its briefs alongside the work that produced them. This means the thread graph UI should be the primary reading surface, not a separate briefs page. The briefs page is scaffolding; the thread view is the destination.

## 2026-03-25 — Unified model (merged artifact-attractors thread)

Two threads converged into one argument with supply and demand sides:

**Supply side (artifact attractors):** Claude agents naturally produce artifacts — PLAN.md, NOTES.md, SYNTHESIS.md. It's trained behavior, not convention. Without structure, these accrete at project root, get cleaned up, reappear next session. The agent is doing the right thing in the wrong place. The community treats this output as disposable: run agent, get code change, discard context. This misses that process artifacts (what it learned, tried, decided, what's unresolved) are the valuable part. Agents are knowledge generators that happen to also write code.

**Demand side (threads as primary):** The composing structure where artifacts become understanding is the thread. Investigations gather evidence for threads. Decisions resolve branches. Models formalize what threads have learned. The thread is the spine; everything else hangs off it.

**The system is the attractor field between them.** .kb/investigations/, .orch/workspace/, .kb/threads/ — these are structural placement for knowledge. Same pattern as the coordination experiments: agents without placement put output in the gravitational default (project root, end of file). Agents with attractors put it where it composes.

**What this means for orch-go's identity:** The interesting thing orch-go does is NOT "orchestrate AI agents" — plenty of tools do that. The interesting thing is treating agent-produced knowledge as a first-class, composable resource. The thread graph, the knowledge base, the brief pipeline — these are all infrastructure for the claim that what agents learn matters as much as what they build. That's a genuinely different bet than the market is making.

## 2026-03-26 — Decision boundary accepted

This thread crossed from emerging thesis to explicit product boundary. After reviewing the repo as a whole, the clearest characterization was:

- the strongest layer is thread/comprehension/coordination
- the weakest current pattern is that the repo still presents itself as an orchestration/execution system first
- the sprawl is mostly discovery residue from figuring out what the project is

The important move was not "invent a new insight" but "believe the best existing one enough to reorganize around it." That led to a direct product decision:

**orch-go is primarily the thread/comprehension layer, not the execution layer.**

Execution remains necessary, but it is substrate. The differentiated value is the system that turns agent work into durable, legible understanding. This gives a useful deletion criterion for future work: if a feature does not strengthen comprehension, synthesis, knowledge composition, or portability of the substrate beneath them, it needs a very strong reason to stay first-class.

Decision recorded at:
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-04-design-too-many-knowledge-artifact-types.md
