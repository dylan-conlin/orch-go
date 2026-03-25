---
title: "Comprehension artifacts — async synthesis delivery replaces conversation-gated review"
status: forming
created: 2026-03-24
updated: 2026-03-24
resolved_to: ""
spawned_from: coordination-protocol-primitives-route-sequence
spawned:
  - threads-as-primary-artifact-thinking
active_work: [orch-go-o7c0u, orch-go-zn7fg, orch-go-gp0th, orch-go-ovfsc, orch-go-1l7us]
  - "orch-go-swrwn"
resolved_by: [orch-go-o7c0u, orch-go-zn7fg, orch-go-gp0th, orch-go-ovfsc, orch-go-1l7us]
---

# Comprehension artifacts — async synthesis delivery replaces conversation-gated review

## 2026-03-24

Current bottleneck: comprehension is gated on orchestrator conversation. Dylan can't process completions unless we're talking. This means the system's throughput is bounded by conversation time, not Dylan's reading speed. Insight from sketchybar design session: instead of the orchestrator narrating completions live, the system should produce comprehension artifacts — pre-written synthesis briefs in Dylan's preferred style and depth — that land in a dashboard reading queue. Dylan reads them async (over coffee, between meetings), then we start conversations from shared understanding instead of from zero. This changes: (1) the sketchybar widget becomes 'N unread' — simple, actionable, (2) the dashboard becomes a reading product, not a status board, (3) our conversations start at 'here's what I think about what I read' instead of 'let me walk you through what happened', (4) the comprehension queue becomes literal UI (reading queue with mark-as-read) instead of a counter in a hook. Template: Three-Layer Reconnection (frame/resolution/placement) written for reading, not conversation. Style: Dylan's four writing primers (story first, earn abstractions, say what it felt like, the turn is the piece). Open questions: who generates the artifact — the completing agent, the daemon, or a dedicated synthesis agent? How does 'mark as read' feed back into the comprehension gate? Does annotation (follow-up, done, question) replace the current orch complete flow?

Design convergence: the completing agent generates the comprehension brief as a second deliverable alongside SYNTHESIS.md. Zero new infrastructure — no extra API calls, no daemon transform step, no dedicated synthesis agent. The agent already has full context (it just did the work). Skill protocol adds BRIEF.md as a required artifact: written for Dylan, not the orchestrator. Style baked into protocol: story first, earn abstractions, say what it felt like. Daemon moves BRIEF.md to dashboard reading queue on completion. Max plan constraint resolved: no API calls needed, the agent session already exists. Remaining design work: (1) BRIEF.md template — what sections, what length, what style guide gets injected into the skill, (2) dashboard reading queue UI — where do briefs render, mark-as-read flow, annotation, (3) feedback loop — does mark-as-read decrement comprehension gate, or does Dylan still do orch complete separately, (4) skill protocol change — which skills get the BRIEF.md requirement (full-tier only? all?). Next step: architect issue to design the BRIEF.md template and skill protocol integration.

Design constraint surfaced during session: briefs must provoke conversation, not replace it. The brief handles facts and findings (what happened, what changed, what it means). But the highest-value output this session — the thread from coordination primitives to decomposition reframe to coord-bench to comprehension artifacts — emerged from Dylan reacting to synthesized material in real time. That's not in any SYNTHESIS.md. Risk: good briefs create false comprehension. Dylan reads, marks as read, moves on — but never has the reactive moment where one insight triggers a strategic reframe. Design implication: briefs should end with an open question or tension that requires Dylan's judgment, not a summary that feels complete. The brief is setup; the conversation is where the thinking happens.

First BRIEF.md produced and read (orch-go-3tyik, experiment harness fix). Quality was good — agent followed Frame/Resolution/Tension template with writing primers. Gap discovered: briefs vanish from dashboard after orch complete clears comprehension:pending. Architect issue created (orch-go-swrwn), daemon spawned 4 implementation agents, all completed. Briefs reading queue should now be live in dashboard. Second brief also produced (orch-go-f9xii, Sonnet cross-model validation) — even stronger quality, genuine surprise in Resolution, real strategic question in Tension. The BRIEF.md system works. The open question is whether Dylan actually reads them async or whether they just become another artifact.
