---
title: "Threads as primary artifact — thinking has structure and it should be first-class"
status: active
created: 2026-03-24
updated: 2026-03-24
resolved_to: ""
spawned_from: comprehension-artifacts-async-synthesis-delivery
spawned:
  - artifact-attractors-agents-naturally-externalize
active_work: [orch-go-jwteq, orch-go-zsj78, orch-go-gfka3, orch-go-3htj3, orch-go-xwbby, orch-go-abz6a, orch-go-gvo79]
resolved_by: [orch-go-jwteq, orch-go-zsj78, orch-go-gfka3, orch-go-3htj3, orch-go-xwbby, orch-go-abz6a, orch-go-gvo79]
---

# Threads as primary artifact — thinking has structure and it should be first-class

## 2026-03-24

Lineage: decidability graph (decisions have dependencies) → question entities in beads (not all work is tasks, some is questions) → architect fork design (ideas branch, system should track it) → thread connective tissue (today). Each touched a different face of the same need: thinking has structure, and that structure should be first-class. Today's insight: threads aren't a side artifact — they're the primary artifact. Investigations gather evidence for threads. Decisions resolve branches of threads. Models formalize what threads have learned. Agent work tests thread claims. The thread is the spine; everything else hangs off it. This inverts orch-go's current model where beads issues are primary (work to be done) and threads are captured mid-session as a side effect. The way Dylan actually thinks — and the way this session demonstrated — threads are the driver and work is what threads produce when they need evidence. Minimal structure needed: spawned_from/spawned (thread-to-thread lineage), active_work/resolved_by (thread-to-work grounding). Together these make the thinking path traversable: where did this thought come from, what work tested it, what did that work find, where did the thought go next. Full lifecycle of an idea from forming question to evidence to next question.

The work graph (localhost:5188/work-graph) shows the daemon's view: dependency chains, issue types, agent status. Dylan says it never resonated — it's not how he thinks. He thinks in questions that branch and sharpen, not in dependency chains. The visualization that would resonate is a thread graph: threads as nodes, spawned_from/spawned as edges, active work hanging off each thread as evidence. The work graph becomes a sub-view of the thread graph — what the system is doing in service of each thread. This inverts the current dashboard architecture: instead of work-centric with threads as metadata, it becomes thread-centric with work as grounding. The thread graph is where Dylan would start his day — 'what am I thinking about, what has the system learned about each question, where do I want to go next' — instead of 'what agents are running.'

Session rhythm change shipped: orchestrator skill session protocol now thread-first. Start protocol inverted — threads are step 1 (not step 6), fires only surface if blocking thread progress, work proposed in service of chosen thread. End protocol leads with thread trajectory (moved/formed/converged) before task summary. Fast path table updated. This is the first behavioral test of the thesis: if threads are primary, the session cadence should follow threads, not backlog. Living with this for a few sessions to see what changes.

Briefs-in-isolation problem confirms thread-centric UI need. BRIEF.md with only a beads ID is an orphaned artifact — same problem threads solve for investigations and decisions. The natural fix: briefs render as evidence within their thread context, not as a flat list. When Dylan reads a thread, he sees its briefs alongside the work that produced them. This means the thread graph UI should be the primary reading surface, not a separate briefs page. The briefs page is scaffolding; the thread view is the destination.
