# Session Handoff - 2025-12-30

## Focus
System reliability + root cause thinking

## Completed This Session

| Issue | Description |
|-------|-------------|
| ✅ orch-go-6vr6 | Headless spawn race condition - message verification |
| ✅ orch-go-n8xo | Stale architect recommendations - filter fix |
| ✅ orch-go-dz88 | Agent error reporting investigation (fix in orch-knowledge) |
| ✅ orch-go-rhcs | Untracked agent visibility - dashboard stalled detection |
| ✅ orch-go-zdja | Cross-project phase fetching fix |
| ✅ orch-go-jb0w | Root cause thinking epic - investigation complete |
| ✅ orch-go-cqa5 | Why agents stop - Claude conversational pattern, not skill guidance |
| ✅ orch-go-i0l4 | Design-session principles requirement |
| ✅ orch-go-ss7o | No silent waiting instruction in SPAWN_CONTEXT |

## Key Outcomes

### Process Change: Design-First Gate
Added to orchestrator skill. Before creating issues or spawning:
1. What's the symptom?
2. What design assumption does this symptom reveal?
3. Is this a fix or a conversation?

If can't answer #2, discuss before proceeding.

### Root Cause Findings
1. **Why system fixes symptoms:** Completion gates reward correctness not depth. Design questioning only triggers on failure.
2. **Why agents stop mid-execution:** Claude's conversational pattern treats step-lists as proposals awaiting confirmation. Fixed with "no silent waiting" instruction.

### Open Question
When should design discussions happen in orchestrator chat vs spawning design-session? Captured via `kb quick question`.

## Created Issues (Not Yet Spawned)

| Issue | Priority | Description |
|-------|----------|-------------|
| orch-go-8zfz | P1 | orch complete should check for unfiled discovered work |
| orch-go-s8lr | P2 | Agent stall visibility (superseded by cqa5 root cause) |
| orch-go-5kms | P2 | kb reflect semantic clustering |
| orch-go-xumh | P2 | Spawn context include related investigations |
| orch-go-pebz | P2 | Supersedes/lineage reminder in spawn template |
| orch-go-h7qx | P2 | Worker-base consolidation |
| orch-go-hn8j | P2 | Capability skill type (document-handling) |
| orch-go-5mv6 | P2 | Glass hover capability |

## Pushed
- orch-go: 4719803b (up to date with origin)
- orch-knowledge: ed82119 (local only, no remote)

## Notes
- Deleted stale `skills/src/policy/` directory in orch-knowledge
- 67% usage on personal account (resets in 2d 9h)
- Investigation showed 2-day-old recommendations from orch-go-s03z were never filed - created orch-go-8zfz to gate on this
