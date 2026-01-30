<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrators should receive a compact git log summary at session start to prevent amnesia-driven duplicate work - 10-15 recent commits (~500-800 tokens) injected via ORCHESTRATOR_CONTEXT.md template provides the best balance of context vs token cost.

**Evidence:** kb quick entries explicitly state "Check git log before starting work" as a decision; existing context injection (~125KB for orchestrators) has room for ~800 tokens; Signal-Aware Spawn Gating already parses git log for completion detection; Session Amnesia is the foundational principle.

**Knowledge:** Three existing context injection points exist (ORCHESTRATOR_CONTEXT.md, SPAWN_CONTEXT.md, hooks) - only orchestrators need git log (workers have scoped SPAWN_CONTEXT); the `kb context` command already provides relevant prior decisions/investigations but misses what was *done* recently.

**Next:** Implement git log injection in OrchestratorContextTemplate via new `GenerateGitLogContext()` function with 15 commits, 7-day window, focused format.

**Authority:** implementation - Follows existing context injection patterns, reversible, clear success criteria (fewer duplicate investigations)

---

# Investigation: Design Git Log Context Orchestrator

**Question:** Should orchestrators start with recent git history to prevent amnesia (re-investigating, duplicating work, missing context)? What format, how much history, where to inject, token budget?

**Started:** 2026-01-30
**Updated:** 2026-01-30
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** Implement in pkg/spawn/orchestrator_context.go
**Status:** Complete

---

## Findings

### Finding 1: Session Amnesia Already Drives Design, Git Log Fills a Gap

**Evidence:** The principles document (`~/.kb/principles.md:56-77`) establishes Session Amnesia as THE foundational constraint: "This is THE constraint. When principles conflict, session amnesia wins." The `kb context` command provides prior decisions, constraints, investigations - but this is *knowledge*, not *actions*. Two `kn decide` entries explicitly state:
- "Verify issue status before starting work - Check bd show and git log before implementing to avoid duplicate effort" (kb-455e87)
- "Check git log before starting work on dependent phases - Parallel agents may complete work before spawned tasks run" (kb-ca2f45)

**Source:** 
- `~/.kb/principles.md:56-77` (Session Amnesia principle)
- `.kb/quick/entries.jsonl` (kb-455e87, kb-ca2f45)
- `kb context "session resume orchestrator"` output showing guides/models but no recent activity

**Significance:** The system already recognizes agents should check git log - but this is a reminder, not a gate. Per "Gate Over Remind" principle, reminders fail under cognitive load. Injecting git log context automatically at spawn makes this behavior structural.

---

### Finding 2: Context Injection Architecture Supports Addition

**Evidence:** Three distinct context injection paths exist:
1. **SPAWN_CONTEXT.md** for workers (~27KB base + skill content)
2. **ORCHESTRATOR_CONTEXT.md** for orchestrators (~8KB base)
3. **Session hooks** for interactive sessions (session-start.sh, bd prime)

The orchestrator context template (`pkg/spawn/orchestrator_context.go:19-127`) already includes dynamic sections: KBContext, ServerContext, RegisteredProjects. Adding GitLogContext follows the same pattern.

Current orchestrator session injection totals ~125KB (~31K tokens). Adding ~800 tokens (15 commits, ~50 chars each) is <3% increase - negligible impact.

**Source:**
- `pkg/spawn/orchestrator_context.go:19-127` (template structure)
- `pkg/spawn/context.go:489-552` (GenerateContext with dynamic sections)
- `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md:151` (125KB total for manual sessions)

**Significance:** The architecture is designed for additive context sections. Git log fits naturally in the existing pattern without architectural changes.

---

### Finding 3: Signal-Aware Spawn Gating Already Parses Git Log

**Evidence:** The `docs/designs/2026-01-18-signal-aware-spawn-gating.md` design document shows git log parsing is already planned for spawn-time completion detection:

```go
// Check git log for recent commits containing the beads ID
since := time.Now().Add(-6 * time.Hour).Format("2006-01-02 15:04:05")
cmd := exec.Command("git", "log", "--since="+since, "--grep="+issueID, "--oneline")
```

This proves:
1. Git log parsing at spawn-time is architecturally accepted
2. The format (`--oneline`) is established for machine parsing
3. Time windows (6h) are used for relevance filtering

**Source:**
- `docs/designs/2026-01-18-signal-aware-spawn-gating.md:125-139`
- `pkg/spawn/context.go:19-31` (existing `getGitBaseline()` function)

**Significance:** Git log context injection is a natural extension of existing patterns, not a new architectural concern.

---

### Finding 4: Workers Don't Need Git Log - SPAWN_CONTEXT Provides Scoped Context

**Evidence:** Workers receive task-specific context via SPAWN_CONTEXT.md including:
- Specific beads ID and task description
- KB context for relevant decisions/investigations
- Skill guidance for their specific task

Workers are scoped to a single task. They don't need to know what happened across the project - they need to know what's relevant to their task. The kb context query at spawn already surfaces relevant prior work.

In contrast, orchestrators need the "what happened recently" view because they:
- Synthesize across multiple agents
- Make decisions about what to spawn next
- Avoid re-investigating recently completed work

**Source:**
- `pkg/spawn/context.go:54-356` (SpawnContextTemplate - task-scoped)
- `pkg/spawn/orchestrator_context.go:19-127` (ORCHESTRATOR_CONTEXT - session-goal scoped)

**Significance:** Only orchestrators need git log injection. Adding it to SPAWN_CONTEXT.md would bloat worker context unnecessarily.

---

## Synthesis

**Key Insights:**

1. **Gap exists between knowledge and actions** - `kb context` surfaces prior decisions and investigations (knowledge), but not what was done (commits). Orchestrators start sessions not knowing recent work completed by other agents.

2. **Architecture supports additive sections** - OrchestratorContextTemplate is designed for dynamic context sections (KBContext, ServerContext, RegisteredProjects). GitLogContext is the same pattern.

3. **Orchestrators only** - Workers are task-scoped; orchestrators are session-goal scoped. Only orchestrators need the "what happened recently" cross-agent view.

**Answer to Investigation Question:**

**YES**, orchestrators should receive recent git history at session start to prevent amnesia. The format should be:
- **Count:** 15 commits (enough to show a day's work, not overwhelming)
- **Time window:** 7 days (captures recent context, excludes ancient history)
- **Format:** One-line with hash, message prefix, and date (matches existing `--oneline` patterns)
- **Injection point:** New `{{.GitLogContext}}` section in OrchestratorContextTemplate
- **Token budget:** ~500-800 tokens (3% of typical orchestrator context - negligible)

---

## Structured Uncertainty

**What's tested:**

- ✅ Context injection architecture supports dynamic sections (verified: KBContext, ServerContext patterns exist)
- ✅ Git log parsing at spawn-time is architecturally accepted (verified: Signal-Aware Spawn Gating design)
- ✅ kb quick entries explicitly say "check git log" (verified: kb-455e87, kb-ca2f45)
- ✅ Session Amnesia is foundational principle (verified: principles.md)

**What's untested:**

- ⚠️ Actual token impact of 15 commits (estimated 500-800, not measured)
- ⚠️ Whether orchestrators actually use the git log context (requires behavioral observation)
- ⚠️ Optimal commit count (15 is heuristic, may need tuning)
- ⚠️ Whether date filtering (7 days) is the right window

**What would change this:**

- Finding would be wrong if orchestrators don't read the git log section (monitor via coaching plugin patterns)
- Finding would be wrong if 15 commits pushes context over limits (would need to reduce)
- Finding would be wrong if workers actually need cross-agent context (would need broader injection)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add GitLogContext to OrchestratorContextTemplate | implementation | Follows existing context injection pattern, reversible, single-component change |

### Recommended Approach: Focused Git Log Section in OrchestratorContextTemplate

**Why this approach:**
- Follows existing dynamic context section pattern (KBContext, ServerContext)
- Addresses Session Amnesia without reminders that can be ignored
- Minimal token cost (~800 tokens, <3% increase)
- Orchestrator-only (doesn't bloat worker context)

**Trade-offs accepted:**
- Some commits may be irrelevant (not all recent work matters to all orchestrators)
- Fixed commit count (15) may be suboptimal for some sessions

**Implementation sequence:**

1. **Add `GenerateGitLogContext()` function** - Mirror existing `GenerateServerContext()` pattern
2. **Add `{{.GitLogContext}}` to OrchestratorContextTemplate** - Place after KBContext section
3. **Add GitLogContext field to orchestratorContextData struct** - Standard template data
4. **Call GenerateGitLogContext in GenerateOrchestratorContext()** - Like other dynamic sections

### Alternative Approaches Considered

**Option B: Add to all spawn contexts (workers + orchestrators)**
- **Pros:** Consistent context across all agents
- **Cons:** Workers don't need cross-agent context; bloats SPAWN_CONTEXT unnecessarily
- **When to use instead:** If workers frequently encounter duplicate work (evidence doesn't show this)

**Option C: Add via Claude Code hooks (session-start.sh)**
- **Pros:** Applies to interactive sessions too
- **Cons:** Spawned agents already have context via templates; hooks add complexity; role-aware filtering needed
- **When to use instead:** If interactive orchestrator sessions also need git log context

**Option D: Extend `kb context` to include recent commits**
- **Pros:** Single source for all context
- **Cons:** Changes kb tooling scope; kb focuses on knowledge not actions; doesn't help orchestrator spawn context
- **When to use instead:** If kb should become the universal context provider

**Rationale for recommendation:** Option A (focused OrchestratorContextTemplate addition) is the minimal change that addresses the problem. It follows existing patterns, stays within orchestrator scope, and is easily reversible.

---

### Implementation Details

**What to implement first:**
1. `GenerateGitLogContext()` function in `pkg/spawn/orchestrator_context.go`
2. Template addition and struct field

**Git command to use:**
```bash
git log --oneline --since="7 days ago" -15 --format="%h %s (%ar)"
```

This produces output like:
```
7b7b91c investigation: SPAWN_CONTEXT generation issues (2 hours ago)
64af1e9 fix: skip tmux select-window for daemon spawns (3 hours ago)
...
```

**Template section format:**
```markdown
## Recent Activity

Recent commits in this project (last 7 days):

{{.GitLogContext}}

Use this context to avoid duplicate work and understand recent changes.
```

**Things to watch out for:**
- `git log` can fail if not in a git repo - handle gracefully (return empty string)
- Very active projects may have many commits - cap at 15 to control token cost
- Format should be human-readable but compact

**Success criteria:**
- Orchestrators can see recent commits without manually running `git log`
- No duplicate investigations of recently-completed work
- Token cost stays under 1000 tokens for the git log section

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Session Amnesia and related principles
- `.kb/quick/entries.jsonl` - kb-455e87, kb-ca2f45 decisions about git log
- `pkg/spawn/orchestrator_context.go` - Current orchestrator context template
- `pkg/spawn/context.go` - Worker context and dynamic section patterns
- `docs/designs/2026-01-18-signal-aware-spawn-gating.md` - Git log parsing patterns
- `.kb/models/context-injection.md` - Context injection architecture
- `.kb/guides/session-resume-protocol.md` - Historical session resume approach
- `~/.claude/hooks/session-start.sh` - Hook-based context injection

**Commands Run:**
```bash
# View recent commits for context
git log --oneline -20

# Get kb context for session resume patterns
kb context "session resume orchestrator context"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-19-remove-session-handoff-machinery.md` - Context now relies on kb/beads, not session handoffs
- **Design:** `docs/designs/2026-01-18-signal-aware-spawn-gating.md` - Git log parsing for spawn gating
- **Model:** `.kb/models/context-injection.md` - Context injection architecture
- **Model:** `.kb/models/spawn-architecture.md` - Spawn architecture and SPAWN_CONTEXT.md

---

## Investigation History

**2026-01-30 Initial:** Investigation started
- Initial question: Should orchestrators start with recent git history to prevent amnesia?
- Context: Orchestrators frequently re-investigate work that was recently completed by other agents

**2026-01-30 Analysis:** Examined existing context injection patterns
- Found three distinct templates: SPAWN_CONTEXT, ORCHESTRATOR_CONTEXT, META_ORCHESTRATOR_CONTEXT
- kb quick entries explicitly recommend checking git log
- Signal-Aware Spawn Gating already establishes git log parsing pattern

**2026-01-30 Complete:** Investigation completed
- Status: Complete
- Key outcome: Recommend adding GitLogContext section to OrchestratorContextTemplate - 15 commits, 7-day window, ~800 tokens
