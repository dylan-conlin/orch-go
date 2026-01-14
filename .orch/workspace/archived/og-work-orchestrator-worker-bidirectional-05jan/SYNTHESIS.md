# Session Synthesis

**Agent:** og-work-orchestrator-worker-bidirectional-05jan
**Issue:** orch-go-pn7q
**Duration:** 2026-01-05 14:00 → 2026-01-05 15:30
**Outcome:** success

---

## TLDR

Designed bidirectional communication patterns for orchestrator-worker interaction. Found 5 existing mechanisms (orch send, orch question, bd comment, SSE events, Phase reporting) and identified 5 gaps (session health detection, question notification, response visibility, course correction, acknowledgment). Recommends incremental CLI-based enhancement, not new infrastructure.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-05-inv-orchestrator-worker-bidirectional-communication-interaction.md` - Full design investigation with findings, synthesis, and implementation recommendations

### Files Modified
- None (design investigation only)

### Commits
- (Pending) Investigation file creation

---

## Evidence (What Was Observed)

- `orch send` has dual-path (API + tmux fallback) but is fire-and-forget by default (`send_cmd.go:96-102`)
- `orch question` is pull-based, requires explicit polling (`question_cmd.go:35-36`)
- SSE events proxy exists (`serve_agents_events.go`) but doesn't parse for questions
- Phase reporting via `bd comment` is reliable but one-way (worker→orchestrator)
- Prior decision exists: "MCP for agent-internal use, CLI for orchestrator/scripts/humans"
- Status dashboard guide confirms "Only Phase: Complete indicates actual completion"

### Tests Run
```bash
# Context gathering - all knowledge search commands completed successfully
kb context "orch send"  # Found prior investigation on send vs spawn
kb context "orch question"  # Found question command implementation details
kb context "bidirectional"  # Found completion lifecycle design
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-05-inv-orchestrator-worker-bidirectional-communication-interaction.md` - Complete design analysis with 6 findings, synthesis, and implementation recommendations

### Decisions Made
- Decision 1: **CLI-based approach** because prior constraint says "CLI for orchestrator/scripts/humans" and orchestrators are the primary users
- Decision 2: **Incremental enhancement** over new infrastructure because building blocks exist (SSE, beads, events.jsonl) - they need integration, not replacement
- Decision 3: **Session health detection is highest priority** because orchestrators cannot currently distinguish healthy-idle from stuck/dead

### Constraints Discovered
- SSE events are OpenCode-level, not parsed for semantic content (questions, progress)
- tmux fallback agents have reduced functionality (no API session ID)
- events.jsonl exists but is not surfaced to orchestrators

### Externalized via `kn`
- `kn decide "Bidirectional communication uses CLI, not MCP" --reason "Prior constraint: CLI for orchestrators/scripts, MCP for agent-internal"` - (to be run)
- `kn decide "Session health detection via orch probe" --reason "No current mechanism distinguishes healthy-idle from stuck"` - (to be run)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up (create epic with children)

### If Spawn Follow-up

**Issue:** Epic: Orchestrator-Worker Bidirectional Communication
**Skill:** feature-impl (for implementation work)
**Context:**
```
Design investigation complete. Create epic with 6 children:
1. orch probe - session health detection (P0)
2. Question notification via SSE parsing (P0)
3. orch history - response visibility (P1)
4. orch redirect - course correction (P1)
5. Acknowledgment protocol (P2)
6. Dashboard integration for above (P2)

Investigation file: .kb/investigations/2026-01-05-inv-orchestrator-worker-bidirectional-communication-interaction.md
```

### Implementation Priority Order

1. **Session Health Detection** (P0 - blocking issue)
   - Add `orch probe <id>` command
   - Parse response to determine health
   - Dashboard: show "unresponsive" after probe timeout

2. **Question Notification** (P0 - frequent pain point)
   - Extend `serve_agents_events.go` to parse for AskUserQuestion
   - Emit `event: question` SSE event
   - Dashboard: show pending question badge

3. **Response Visibility** (P1)
   - Add `orch history <id>` to view send/response pairs
   - Data already in events.jsonl, just needs surfacing

4. **Course Correction** (P1)
   - Add `orch redirect <id>` as priority-flagged send
   - Update SPAWN_CONTEXT.md template with redirect acknowledgment protocol

5. **Acknowledgment Protocol** (P2)
   - Add `orch ack <id>` or use bd comment for orchestrator→worker ack
   - Optional protocol, lower priority

6. **Dashboard Integration** (P2)
   - Pending question badge
   - Message history view
   - Health status indicator

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How should multiple orchestrators coordinate when talking to the same agent?
- Should acknowledgments be explicit (orch ack) or implicit (no ack = continue)?
- How long should `orch probe` wait before declaring session unresponsive?

**Areas worth exploring further:**
- Integration with Glass for visual verification of dashboard communication features
- Rate limiting strategy for probe commands to avoid API abuse
- Event log rotation for events.jsonl (currently unbounded)

**What remains unclear:**
- Whether tmux-only agents can support all features (may have reduced functionality)
- Optimal probe timeout threshold (5 min proposed but not validated)

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-work-orchestrator-worker-bidirectional-05jan/`
**Investigation:** `.kb/investigations/2026-01-05-inv-orchestrator-worker-bidirectional-communication-interaction.md`
**Beads:** `bd show orch-go-pn7q`
