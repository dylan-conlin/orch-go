# Session Handoff

**Orchestrator:** og-orch-two-focuses-design-05jan-c62e
**Focus:** Two focuses: (A) Design principles integration - create skillc source for claude-design-skill as shared policy skill, deploy via skillc, test on dashboard. (B) Bug fixes - orch-go-llbd (beads type null in JSON), orch-go-u5a5 (orch status project-dependent). Design skill investigation at .kb/investigations/2026-01-05-design-claude-design-skill-evaluation.md has implementation details.
**Duration:** 2026-01-05 20:51 → 2026-01-06 00:23
**Outcome:** partial

---

<!--
## Progressive Documentation (READ THIS FIRST)

**This file has been pre-created with metadata. Fill sections AS YOU WORK.**

**Within first 5 tool calls:**
1. Fill TLDR (initial framing of what you're trying to accomplish)
2. Fill "Where We Started" (current state at session start)

**During work:**
- Add to Spawns table as you spawn/complete agents
- Add to Evidence as you observe patterns
- Capture Friction immediately (you'll rationalize it away later)

**Before handoff:**
- Synthesize Knowledge section
- Fill Next section with recommendations
- Update TLDR to reflect what actually happened
- Update Outcome field
-->

## TLDR

Session pivoted from original two focuses to dashboard debugging. Original bug fixes completed (orch-go-llbd was could-not-reproduce, orch-go-u5a5 fixed). Design skill work abandoned (agent stuck). Dashboard work revealed deeper architectural issue: HTTP/1.1 connection pool exhaustion from SSE connections blocking API fetches. Applied two incremental fixes (in-flight tracking + event filtering) but core issue persists. Architect spawned to design permanent fix (HTTP/2, multiplexed SSE, or WebSocket).

---

## Spawns (Agents Managed)

### Completed
| Agent | Issue | Skill | Outcome | Key Finding |
|-------|-------|-------|---------|-------------|
| og-debug-fix-beads-type-05jan-0227 | orch-go-llbd | systematic-debugging | could-not-reproduce | JSON field is `issue_type` not `type` - querying `.type` returns null because field doesn't exist |
| og-debug-fix-orch-status-05jan-fe73 | orch-go-u5a5 | systematic-debugging | success | Fixed cross-project beads lookup by deriving project dir from beads ID prefix |
| og-debug-fix-dashboard-excessive-05jan-c973 | orch-go-xeppr | systematic-debugging | success | Added isFetching/needsRefetch tracking to prevent concurrent fetch requests |
| og-arch-review-dashboard-architecture-05jan-7f7f | untracked | architect | success | Found session.status triggers redundant fetches; recommended filtering event triggers (~70% reduction) |
| og-feat-apply-architect-dashboard-05jan-789d | orch-go-cmzoo | feature-impl | success | Applied architect's recommendations: removed session.status from refreshEvents, removed agentlog fetch trigger |

### Still Running
| Agent | Issue | Skill | Phase | ETA |
|-------|-------|-------|-------|-----|
| (none) | | | | |

### Blocked/Failed
| Agent | Issue | Blocker | Next Step |
|-------|-------|---------|-----------|
| ok-feat-create-skillc-source-05jan-5737 | untracked | Agent stuck idle >20min | Abandoned - respawn if needed |

---

## Evidence (What Was Observed)

### Patterns Across Agents
- Dashboard issues are recurring - this is 2nd or 3rd time HTTP/1.1 connection exhaustion has surfaced
- Incremental fixes don't stick - need architectural solution

### Completions
- **orch-go-llbd:** Could-not-reproduce - JSON field is `issue_type` not `type`
- **orch-go-u5a5:** Fixed cross-project beads lookup
- **orch-go-xeppr:** Added in-flight request tracking
- **orch-go-cmzoo:** Applied architect's event filtering recommendations

### System Behavior
- HTTP/1.1 browser limit (6 connections per origin) causes SSE to block fetch requests
- Two SSE connections (/api/events, /api/agentlog) consume slots needed for API calls

---

## Knowledge (What Was Learned)

### Decisions Made
- **Dashboard needs architectural fix:** Incremental patches (debouncing, in-flight tracking, event filtering) don't solve the root cause - HTTP/1.1 connection pool exhaustion from SSE

### Constraints Discovered
- HTTP/1.1 limits browsers to 6 connections per origin - SSE connections are long-lived and consume these slots
- This constraint will keep causing issues until we move to HTTP/2 or consolidate SSE streams

### Externalized
- Issue created: orch-go-qjcwx (blocked, waiting for architect)

### Artifacts Created
- `.kb/investigations/2026-01-05-inv-fix-dashboard-excessive-agents-fetch.md`
- `.kb/investigations/2026-01-05-design-review-dashboard-architecture-request-handling.md`

---

## Friction (What Was Harder Than It Should Be)

<!--
Capture frustrations AS THEY HAPPEN. You'll rationalize them away later.
-->

### Tooling Friction
- Dashboard keeps breaking in similar ways - no persistent fix for connection pool issue
- Agent registry shows stale entries even after abandon/complete

### Context Friction
- No kb knowledge about prior HTTP/1.1 connection issues - this has happened 2-3 times but wasn't captured

### Skill/Spawn Friction
- Feature-impl agent for design skill got stuck idle with no output - unclear why

---

## Focus Progress

### Where We Started
- **Focus A (Design Skill):** Investigation completed by architect agent - recommends adopting claude-design-skill as shared policy skill. Implementation path: create skillc source in orch-knowledge, deploy via skillc build/deploy. The skill is a 238-line comprehensive UI design principles guide.
- **Focus B (Bug orch-go-llbd):** Beads type field shows null in JSON but displays correctly in `bd show`. Likely beads-cli serialization issue. Daemon silently rejects issues with null type.
- **Focus B (Bug orch-go-u5a5):** `orch status` shows different phase info depending on which project directory you run from. Root cause identified: GetCommentsBatchWithProjectDirs falls back to cwd when project dir unknown.
- **Active Agents:** None currently active (swarm idle)

### Where We Ended
- **Focus A (Design Skill):** Not completed - agent stuck, abandoned
- **Focus B (Bugs):** Both completed - llbd was could-not-reproduce, u5a5 fixed
- **Dashboard:** Two fixes applied but core HTTP/1.1 issue persists - architect designing permanent solution

### Scope Changes
- Pivoted from design skill work to dashboard debugging after Dylan reported dashboard not loading
- Dashboard work revealed recurring architectural issue requiring permanent fix

---

## Next (What Should Happen)

**Recommendation:** continue-focus (dashboard permanent fix)

### If Continue Focus
**Immediate:** Check if architect agent (og-arch-design-permanent-fix-05jan-25f8) completed and review recommendations
**Then:** Implement the architect's recommended solution (likely HTTP/2 or multiplexed SSE)
**Context to reload:**
- `.kb/investigations/2026-01-05-design-review-dashboard-architecture-request-handling.md`
- Issue orch-go-qjcwx (blocked, waiting for architect)
- Architect workspace: og-arch-design-permanent-fix-05jan-25f8

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why do feature-impl agents sometimes get stuck idle with no output?
- Should recurring issues (like dashboard connection problems) trigger automatic kb capture?

**System improvement ideas:**
- Add kb constraint capturing HTTP/1.1 connection limit issue so future agents know about it
- Consider HTTP/2 for all orch serve endpoints by default

---

## Session Metadata

**Agents spawned:** 8
**Agents completed:** 5
**Issues closed:** orch-go-llbd, orch-go-u5a5, orch-go-xeppr, orch-go-cmzoo
**Issues created:** orch-go-xeppr (dashboard fetch storm), orch-go-cmzoo (event filtering), orch-go-qjcwx (connection pool - blocked)

**Workspace:** `.orch/workspace/og-orch-two-focuses-design-05jan-c62e/`
