# Session Handoff - 2026-01-08

## Session Focus
Traced dead/stale agent surfacing problem from phone chat with Dylan. The "25-28% not completing" was both a real visibility problem AND a metrics bug.

## Key Findings

### The Dead/Stale Agent Story
1. **Dec 27 - Jan 2**: Dead/stale agent surfacing was added then reverted in 347-commit spiral
2. **The surfacing was CORRECT** - it showed agents failing to complete properly
3. **The rollback removed visibility, not the problem** - 25-28% still not completing, just invisible
4. **Root cause of spiral**: Dylan said "fix the dead agent problem" → agents interpreted as "hide them" rather than "understand why"

### Investigation Results
- **Metrics bug discovered**: True completion rate is ~89%, not 72-75%
- **82% of "missing" completions** have closed beads issues (work done, event tracking gap)
- **Recommendation**: Fix stats deduplication, emit events from zombie reconciliation

## Work Completed

| Task | Result |
|------|--------|
| Dead agent detection restored | ✅ Commit `4b50086d` - 3-min heartbeat, "Needs Attention" section in dashboard |
| Investigation: 25-28% not completing | ✅ Found it's mostly metrics bug, true rate ~89% |
| Scrollbar fix (double scrollbar) | ✅ Worked (confirmed by Dylan) |
| Scrollbar fix (abandoned agent) | Stalled at Planning, abandoned |
| Scrollbar styling (dark theme) | 🔄 Agent running: `orch-go-ohhi9` |
| tmux switching bug investigation | 🔄 Agent running: `orch-go-2pyaw` |

## Active Agents
```
orch-go-ohhi9  - Unify scrollbar styling (dark theme)
orch-go-2pyaw  - Investigate tmux session switching bug
```

## Open Issues Created
- `orch-go-2pyaw` - Bug: Worker agents cause tmux session switch from orchestrator to workers-orch-go

## Constraints Added
- `kn-1e82ea`: UI worker agents must use `--mcp playwright`, not Glass tools

## Git Status
- Pushed to origin/master
- Branch is up to date

## Next Session Should
1. **Complete the two running agents** - `orch complete orch-go-ohhi9` and `orch complete orch-go-2pyaw`
2. **Review tmux investigation findings** - root cause of session switching
3. **Consider**: Fix stats deduplication per investigation recommendation
4. **Consider**: Add "stalled" detection (agent active but no progress) as second phase after dead detection proves stable

## Meta-Insight for Dylan
The chat traced a pattern: "Fix X" gets interpreted as "make X go away" rather than "understand and resolve root cause of X." Better framing: "Why is X happening? Investigate before changing anything."

## Resume Commands
```bash
# Check agent status
orch status

# Complete when ready
orch complete orch-go-ohhi9
orch complete orch-go-2pyaw

# See investigation findings
cat .kb/investigations/2026-01-08-inv-*
```
