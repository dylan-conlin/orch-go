# Decision: Orchestrator Session Lifecycle

**Date:** 2026-01-04
**Status:** Accepted
**Context:** Establishing the interaction model for meta-orchestrator, orchestrator, and worker sessions

## Decision

### Hierarchy and Interaction Model

| Level | Spawned By | Interacts With | Completes |
|-------|------------|----------------|-----------|
| Meta-orchestrator | Dylan (or prior meta-orch) | Dylan | Next meta-orchestrator (or Dylan) |
| Orchestrator | Meta-orchestrator | Dylan (directly) | Meta-orchestrator |
| Worker | Orchestrator | Orchestrator (or Dylan) | Orchestrator |

**Key insight:** Moving "up" a level gives pattern visibility. Moving "down" gives execution detail. Dylan can interact at ANY level, choosing the frame that gives the right perspective.

### Orchestrator Session Rules

1. **Orchestrators spawn into `orchestrator` tmux session** - Not `workers-*` sessions. There's a single `orchestrator` session for all orchestrator spawns.

2. **Orchestrators are interactive** - Dylan works WITH them directly, not through the meta-orchestrator. They're not autonomous workers.

3. **Completion flows down** - The level above completes the level below:
   - Next meta-orchestrator (or Dylan) runs `orch complete` on meta-orchestrators
   - Meta-orchestrator runs `orch complete` on orchestrators
   - Orchestrators run `orch complete` on workers

4. **Transcript export on complete** - When completing an orchestrator, `orch complete` automatically:
   - Sends `/export` to the tmux window (while session still active)
   - Waits for export file
   - Moves it to workspace as `TRANSCRIPT.md`
   - Then closes beads issue and kills tmux window

### Meta-orchestrator Role

The meta-orchestrator:
- Spawns orchestrator sessions when requested
- Discusses orchestrator patterns/performance with Dylan
- Reviews SESSION_HANDOFF.md after orchestrator sessions complete
- Does NOT replace Dylan's direct interaction with orchestrators

### Historical Context

This parallels the evolution of the orchestration system:
1. **Phase 1:** Dylan worked directly with single workers
2. **Phase 2:** Dylan worked with orchestrator, orchestrator managed workers
3. **Phase 3:** Dylan can work at any level (meta-orchestrator, orchestrator, worker), choosing the frame that provides the needed perspective

## Implementation

- `orch spawn orchestrator "task"` routes to `orchestrator` tmux session
- `.beads_id` file written to workspace for reliable lookup during complete
- `FindWindowByBeadsIDAllSessions` searches both `workers-*` and `orchestrator` sessions
- `exportOrchestratorTranscript()` handles transcript export before cleanup

## Consequences

- Orchestrators and meta-orchestrators don't try to end their own sessions (no `orch session end`)
- They write SESSION_HANDOFF.md when goal reached, then wait
- Session end happens when the level above runs `orch complete`
- Transcripts preserved for later reflection
- Meta-orchestrators are completed the same way as orchestrators (uniform lifecycle)
