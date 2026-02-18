# Design: orch rework Command

**Date:** 2026-02-17
**Type:** Design
**Phase:** Complete
**Status:** Active

## Problem Statement

Completion review regularly finds gaps in agent work that require rework. Current examples:
1. Agent shipped "Phase: Complete" but deprecated flags weren't removed from `spawn_cmd.go` — only backend logic was cleaned up
2. Agent shipped sort feature but sorting has no visible effect on API responses

The orchestrator works around this by spawning fresh agents with verbose REWORK prefixes, but this is:
- **Lossy** — No automatic inclusion of prior SYNTHESIS.md, no structured feedback injection
- **Disconnected** — New issue loses history, or same issue but no formal rework tracking
- **Unmeasurable** — Rework count/rate not tracked in events
- **Manual** — Orchestrator must construct rework context by hand every time

**Design Question:** How should `orch rework` formalize the completion-review → rework loop?

**Success Criteria:**
1. Rework preserves connection to original work (same beads ID)
2. New agent gets structured rework context (what was wrong, what was tried)
3. Rework count is trackable for quality metrics
4. Command is ergonomic for orchestrators (single command, minimal flags)
5. Integrates with daemon for potential auto-rework

**Constraints:**
- Must work with existing workspace lifecycle (new workspace, not resume old session)
- Must respect Session Amnesia (new agent, full context)
- Must not break completion verification (reworked agents still go through gates)

---

## Fork Navigation

### Fork 1: New Workspace vs Resume Same Session

**Decision Question:** Should rework resume the same workspace/session or create a new one?

**Options:**
- A: Resume same workspace (inject message into existing session)
- B: Create new workspace (fresh agent, with context from prior)
- C: Hybrid — new workspace, linked to archived workspace

**Substrate says:**
- Principle: **Session Amnesia** — Claude has no memory between sessions. Even if we inject into an existing session, context may be degraded or exhausted.
- Principle: **Self-Describing Artifacts** — New agent needs full context in its workspace.
- Model: **Agent Lifecycle State Model** — Sessions are transient infrastructure; workspace state is persistent. Sessions may be deleted by the time rework happens.
- Decision: **File-Based Workspace State Detection** — Workspace state determined from filesystem, not session state.

**RECOMMENDATION:** Option C — New workspace with link to prior archived workspace.

**Reasoning:**
- The original session is almost certainly dead (completed, context exhausted, or deleted)
- `orch resume` already handles "paused agent" — rework is fundamentally different (work was wrong, not paused)
- New workspace gives fresh context budget for the rework agent
- Archived workspace preserves original evidence (SYNTHESIS.md, commits, etc.)
- Link via `.prior_workspace` file enables tracing the rework chain

**Trade-off accepted:** New session costs tokens to re-establish context. This is acceptable because the prior context was insufficient anyway — that's why rework is needed.

---

### Fork 2: How Rework Feedback Gets to the Agent

**Decision Question:** How should rework instructions reach the new agent?

**Options:**
- A: Separate REWORK_CONTEXT.md alongside SPAWN_CONTEXT.md
- B: Embedded section in SPAWN_CONTEXT.md
- C: Beads comment that agent reads via `bd show`

**Substrate says:**
- Principle: **Surfacing Over Browsing** — Bring relevant state to the agent; don't require navigation.
- Principle: **Self-Describing Artifacts** — SPAWN_CONTEXT.md is the agent's single entry point.
- Current pattern: SPAWN_CONTEXT.md already contains all context (skill, task, KB context, beads ID). Template has extensibility via `contextData` struct.

**RECOMMENDATION:** Option B — Embedded `## REWORK CONTEXT` section in SPAWN_CONTEXT.md.

**Reasoning:**
- Agents read SPAWN_CONTEXT.md as their first action. Adding a visible rework section ensures the feedback is seen immediately.
- A separate file requires the agent to discover and read it — violates Surfacing Over Browsing.
- Beads comments require an API call — adds latency and failure mode.
- The template's `contextData` struct in `pkg/spawn/context.go` already supports optional sections (DesignWorkspace, KBContext, etc.) — rework is another optional section.

**Implementation:** Add to `contextData`:
```go
ReworkFeedback    string  // Orchestrator's rework instructions
ReworkNumber      int     // Which rework attempt (1, 2, ...)
PriorSynthesis    string  // TLDR + Delta from prior SYNTHESIS.md
PriorWorkspace    string  // Path to archived workspace
```

**Template section:**
```
{{if .ReworkFeedback}}
## 🔄 REWORK CONTEXT (Attempt #{{.ReworkNumber}})

**This is rework** — a prior agent attempted this task but the work was insufficient.

### What Was Wrong
{{.ReworkFeedback}}

### Prior Attempt Summary
{{.PriorSynthesis}}

### Prior Workspace
Full prior artifacts at: {{.PriorWorkspace}}

### Rework Instructions
1. Read the feedback above carefully
2. Read the prior SYNTHESIS.md for full context on what was tried
3. Focus specifically on the identified gaps
4. Do NOT re-do work that was correct — build on it
5. Report via `bd comment {{.BeadsID}} "Phase: Planning - Rework #{{.ReworkNumber}}: [brief plan]"`
{{end}}
```

**Trade-off accepted:** SPAWN_CONTEXT.md gets longer for rework agents. Acceptable because rework context is critical and shouldn't be buried.

---

### Fork 3: How Much Prior Work to Include

**Decision Question:** How much of the original SYNTHESIS.md should be included in the rework context?

**Options:**
- A: Full SYNTHESIS.md inline
- B: TLDR only
- C: TLDR + Delta + Evidence sections
- D: TLDR + Delta inline, full path for deep dive

**Substrate says:**
- Principle: **Progressive Disclosure** — TLDR first, key sections next, full details available.
- Principle: **Session Amnesia** — Agent needs enough context but not so much it overwhelms.
- SYNTHESIS template has sections: TLDR, Delta, Evidence, Knowledge, Next.

**RECOMMENDATION:** Option D — TLDR + Delta inline, with full archived workspace path.

**Reasoning:**
- TLDR tells the agent what was attempted and what the outcome was
- Delta tells the agent what files were created/modified (critical for knowing what to fix)
- Evidence and Knowledge are available via the archived workspace path if needed
- Full SYNTHESIS inline could be 200+ lines, consuming context budget

**Implementation:** Extract from prior SYNTHESIS.md:
```go
func ExtractReworkSummary(synthesisPath string) (string, error) {
    content, err := os.ReadFile(synthesisPath)
    // Parse TLDR section (between "## TLDR" and next "---")
    // Parse Delta section (between "## Delta" and next "---")
    // Return combined
}
```

---

### Fork 4: Beads Integration

**Decision Question:** How should rework interact with the beads issue lifecycle?

**Options:**
- A: Reopen the original issue
- B: Create new issue linked to original
- C: Reopen original + add rework comment + label

**Substrate says:**
- Model: **Agent Lifecycle State Model** — "Beads issue closed = canonical completion"
- Decision: **Registry Contract** — Registry is spawn-time cache, beads is source of truth
- Principle: **Track Actions, Not Just State** — Rework is an action that should be visible

**RECOMMENDATION:** Option C — Reopen original + rework comment + rework label.

**Reasoning:**
- Reopening maintains the **full comment history** (original Phase: Planning, Phase: Complete, rework feedback)
- The agent gets the **same beads ID**, so all `bd comment` calls go to the same place
- Adding a `rework:N` label enables **filtering** (find all issues that required rework)
- A rework comment documents **why** rework was needed

**Implementation:**
```go
// 1. Reopen
beads.FallbackUpdate(beadsID, "open")

// 2. Add rework comment
comment := fmt.Sprintf("REWORK #%d: %s", reworkNum, feedback)
beads.FallbackAddComment(beadsID, comment)

// 3. Add label (if bd supports it)
// bd update <id> --label rework:N
```

**When this would change:** If beads gets "linked issues" support, creating a child issue for each rework attempt would provide cleaner separation while maintaining the link.

---

### Fork 5: Event Tracking

**Decision Question:** How should rework be tracked for metrics?

**Options:**
- A: New `agent.reworked` event type
- B: Track via beads comments only
- C: Both event + beads tracking

**Substrate says:**
- Principle: **Observation Infrastructure** — If the system can't observe it, it can't manage it.
- Principle: **Track Actions, Not Just State**
- Pattern: `events.jsonl` tracks all lifecycle transitions (spawn, complete, abandon)

**RECOMMENDATION:** Option C — Both.

**Reasoning:**
- `agent.reworked` event enables **metrics**: rework rate by skill, by model, over time
- Beads comments provide **human-readable history**
- The event should capture: prior workspace, rework number, feedback summary, skill

**Implementation:**
```go
// New event type
const EventTypeAgentReworked = "agent.reworked"

// Event data
type AgentReworkedData struct {
    BeadsID        string `json:"beads_id"`
    PriorWorkspace string `json:"prior_workspace"`
    NewWorkspace   string `json:"new_workspace"`
    ReworkNumber   int    `json:"rework_number"`
    Feedback       string `json:"feedback"`
    Skill          string `json:"skill"`
    Model          string `json:"model,omitempty"`
}
```

**Metrics enabled:**
- Rework rate = reworked / completed (overall and per-skill)
- Average rework count per issue
- Rework frequency by model
- First-pass success rate = 1 - (issues with any rework / total issues)

---

### Fork 6: Command Signature

**Decision Question:** What should the command interface look like?

**Options:**
- A: `orch rework <beads-id> "feedback"` (positional)
- B: `orch rework <beads-id> --feedback "feedback"` (flag-based)
- C: `orch rework <beads-id>` with interactive prompt for feedback

**Substrate says:**
- Current CLI pattern: `orch spawn <skill> "task"` uses positional args
- `orch complete <beads-id>` uses positional beads ID
- `orch send <session-id> "message"` uses positional message

**RECOMMENDATION:** Option A — `orch rework <beads-id> "feedback"`

**Reasoning:**
- Consistent with existing patterns (`spawn`, `send`)
- Feedback is required (not optional), so positional makes sense
- Interactive prompt adds friction for automation/daemon integration

**Full signature:**
```
orch rework <beads-id> "feedback message"

Flags:
  --model <alias>     Override model for rework agent (default: inherit from original)
  --skill <skill>     Override skill (default: inherit from original)
  --tmux              Spawn in tmux for visual monitoring
  --bypass-triage     Required for manual rework (consistent with spawn)
  --force             Override safety checks (e.g., issue not yet closed)
```

**Why `--bypass-triage`:** Consistent with spawn's manual/daemon distinction. Future daemon integration could auto-rework without this flag.

**Flag inheritance:** By default, rework inherits the model and skill from the original spawn (read from AGENT_MANIFEST.json in archived workspace). Flags allow override when the orchestrator knows a different model/skill would be better.

---

### Fork 7: orch rework vs orch resume

**Decision Question:** How do these two commands differ and avoid confusion?

| Aspect | `orch resume` | `orch rework` |
|--------|---------------|---------------|
| **When** | Agent paused mid-work | Agent completed but work is wrong |
| **Session** | Sends message to EXISTING session | Creates NEW session |
| **Workspace** | Uses EXISTING workspace | Creates NEW workspace |
| **Context** | "Continue from where you left off" | "Here's what was wrong, fix it" |
| **Beads** | No state change | Reopens closed issue |
| **Event** | `agent.resumed` | `agent.reworked` |
| **Feedback** | None (just wake up) | Specific rework instructions |
| **Issue state** | Still open (in_progress) | Was closed, gets reopened |

**This is clear enough.** Resume = same session, continue. Rework = new session, redo.

---

## Implementation Plan

### Phase 1: Core Command (MVP)

**Files to create/modify:**

1. **`cmd/orch/rework_cmd.go`** (new) — Cobra command implementation
   - `runRework(beadsID, feedback string)` main function
   - Find archived workspace by beads ID
   - Read AGENT_MANIFEST.json for skill/model/tier
   - Extract rework summary from SYNTHESIS.md
   - Determine rework number from beads comments
   - Reopen beads issue + add rework comment
   - Delegate to spawn pipeline with rework context

2. **`pkg/spawn/context.go`** (modify) — Add rework fields to `contextData` struct and template

3. **`pkg/spawn/rework.go`** (new) — Rework-specific helpers
   - `FindArchivedWorkspaceByBeadsID(projectDir, beadsID)` — scan archived/ for .beads_id match
   - `ExtractReworkSummary(synthesisPath)` — extract TLDR + Delta from prior SYNTHESIS.md
   - `CountReworks(beadsID, projectDir)` — count "REWORK #" comments on issue

4. **`pkg/events/logger.go`** (modify) — Add `EventTypeAgentReworked` and `AgentReworkedData`

5. **`cmd/orch/main.go`** (modify) — Register `reworkCmd`

### Phase 2: Daemon Integration (Future)

- Add `autoRework` to daemon completion loop
- When verification finds specific gate failures (e.g., missing test evidence, incomplete implementation), auto-trigger rework with structured feedback
- Configurable: which gates trigger auto-rework vs block for human review
- Max rework attempts (default: 2) before escalating to human

### Phase 3: Metrics Dashboard (Future)

- Add rework metrics to `orch stats`
- Dashboard visualization of rework rate over time
- Per-skill rework rate for identifying weak skills

---

## Acceptance Criteria

- [ ] `orch rework <beads-id> "feedback"` creates a new agent with rework context
- [ ] Prior SYNTHESIS.md TLDR + Delta are included in SPAWN_CONTEXT.md
- [ ] Beads issue is reopened with REWORK comment
- [ ] `agent.reworked` event is logged
- [ ] Rework count is tracked (REWORK #1, #2, etc.)
- [ ] Original skill/model inherited from AGENT_MANIFEST.json
- [ ] `--model` and `--skill` flags allow override
- [ ] `--tmux` flag works for visual monitoring
- [ ] Build passes: `go build ./cmd/orch/` and `go vet ./cmd/orch/`

## Out of Scope

- Daemon auto-rework (Phase 2)
- Dashboard metrics (Phase 3)
- Rework for orchestrator sessions (only worker agents)
- Multiple concurrent rework attempts on same issue

---

## Recommendations

⭐ **RECOMMENDED:** Implement as described above — new workspace, rework context in SPAWN_CONTEXT.md, reopen beads issue.

**Why:** This is the simplest design that closes the rework loop. It reuses 90% of the spawn pipeline, adds one new command file, and extends two existing files. The orchestrator's existing manual workflow becomes a single command.

**Trade-off:** New session means re-reading the full codebase. Accepted because the prior session's context was inadequate — that's the whole point of rework.

**Expected outcome:** Orchestrator can go from "this work is wrong" to "rework agent spawned with full context" in one command, with metrics tracking rework rate.

**Alternative: Use `orch send` to inject rework into existing session**
- **Pros:** No new session, agent has full conversation history
- **Cons:** Session may be dead/deleted, context may be exhausted, no fresh start
- **When to choose:** If the session is still alive AND context isn't exhausted AND the fix is small

**Alternative: Create new beads issue for each rework**
- **Pros:** Clean issue history, no reopening
- **Cons:** Loses connection to original work, rework rate metric is harder
- **When to choose:** If beads gets "linked issues" support

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision establishes the rework loop pattern for the completion verification model
- Future agents working on completion, daemon, or spawn should know about rework

**Suggested blocks keywords:**
- "rework"
- "completion review"
- "verification failure"
- "orch rework"
