# Model: Agent Lifecycle State Model

**Domain:** Agent Lifecycle / State Management
**Last Updated:** 2026-02-13
**Synthesized From:** 17 investigations (Dec 20, 2025 - Jan 6, 2026) into agent state, completion detection, cross-project visibility, and dashboard status display

---

## Summary (30 seconds)

Agent state exists across **four independent layers** (tmux windows, OpenCode in-memory, OpenCode on-disk, beads comments). These layers fall into two distinct categories: **state layers** (beads, workspace files) that represent what work was done, and **infrastructure layers** (OpenCode sessions, tmux windows) that represent transient execution resources. The dashboard reconciles these via a **Priority Cascade**: check beads issue status first (highest authority), then Phase comments, then registry state, then session existence. Status can appear "wrong" at the dashboard level while being "correct" at each individual layer - this is a measurement artifact from combining multiple sources of truth.

---

## Core Mechanism

### Four-Layer State Model

Agent state is distributed across four independent systems:

| Layer | Category | Storage | Lifecycle | What It Knows | Authority Level |
|-------|----------|---------|-----------|---------------|-----------------|
| **Beads comments** | **State** | `.beads/issues.jsonl` | Persistent | Phase transitions, metadata | Highest (canonical) |
| **Workspace files** | **State** | `.orch/workspace/` | Persistent | SPAWN_CONTEXT, SYNTHESIS, .tier | High (artifact record) |
| **OpenCode on-disk** | **Infrastructure** | `~/.local/share/opencode/storage/` | Persistent (no TTL) | Full message history | Medium (historical) |
| **OpenCode in-memory** | **Infrastructure** | Server process | Until restart | Session ID, current status | Medium (operational) |
| **Tmux windows** | **Infrastructure** | Runtime (volatile) | Until window closed | Agent visible, window ID | Low (UI only) |

**Key insight:** The registry (`~/.orch/registry.json`) was a fifth layer attempting to cache all four, which caused drift. The solution is to query authoritative sources directly and reconcile at query time.

### Source of Truth by Concern

Different questions have different authoritative sources:

| Question | Source | NOT this |
|----------|--------|----------|
| Is agent complete? | Beads issue `status = closed` | OpenCode session exists |
| What phase is agent in? | Beads comments (`Phase: X`) | Dashboard shows "active" |
| Did agent finish? | `Phase: Complete` comment exists | Session went idle |
| Is agent processing? | SSE `session.status = busy` | Session exists |
| Is agent visible? | Tmux window exists | Session exists |

**Beads is the source of truth for completion.** OpenCode sessions persist to disk indefinitely. Session existence means nothing about whether the agent is done. Only beads matters.

### State vs Infrastructure: Why This Distinction Matters

The four-layer model (above) conflates two fundamentally different concerns. Recognizing the difference clarifies what orch should *own* versus what it should merely *use*.

**State layers** (beads comments, workspace files) represent *what work was done* and *what phase it's in*. They are persistent, orch-controlled, and survive infrastructure restarts. Orch owns their lifecycle entirely.

**Infrastructure layers** (OpenCode sessions, tmux windows) represent *execution resources*. They are transient, externally-controlled (by OpenCode server and tmux respectively), and have no inherent connection to work completion. Orch uses them but doesn't control their lifecycle.

**The reconciliation burden comes from treating infrastructure as state.** When orch tries to derive agent status from session existence or tmux window presence, it must constantly reconcile infrastructure reality against state truth. This is the root cause of phantom agents (tmux window exists but session exited), ghost sessions (OpenCode session persists after work completed), and orphan infrastructure (resources with no corresponding state).

**Ownership model (Own / Accept / Lobby):**

| Bucket | What | Implication |
|--------|------|-------------|
| **Own** | State layers (beads, workspaces), verification gates, skill integration | Orch's domain — design, maintain, evolve |
| **Accept** | Infrastructure constraints (sessions persist, no metadata API, SSE-only) | Work within them — periodic cleanup, workspace-as-metadata |
| **Lobby** | Missing infrastructure features (session TTL, metadata API, state endpoint) | File upstream — would eliminate reconciliation burden |

See: `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` for the full decision and implementation plan.

### State Transitions

**Normal lifecycle:**

```
spawned (orch spawn)
    ↓
Registry entry created (Status: running)
OpenCode session created
Beads issue created (Status: open)
Tmux window created (if --tmux)
    ↓
working (agent executes task)
    ↓
Phase transitions reported via bd comment
"Phase: Planning" → "Phase: Implementing" → "Phase: Complete"
    ↓
Phase: Complete reached (agent declares done)
SYNTHESIS.md written (if full tier)
Git commits created
    ↓
orch complete runs (orchestrator verification)
Verifies deliverables exist
Closes beads issue (Status: closed)
    ↓
completed (dashboard shows blue badge)
Session may remain in OpenCode storage
Tmux window may remain open
```

**Abandoned path:**

```
spawned → running
    ↓
orch abandon (human judgment)
    ↓
Registry updated (Status: abandoned)
Beads issue remains open (NOT closed)
    ↓
Dashboard shows abandoned (yellow badge)
Session remains in OpenCode
```

### Critical Invariants

1. **Phase: Complete is agent's declaration** - Only agent can reach this, not orchestrator
2. **Beads issue closed = canonical completion** - All status queries defer to beads
3. **Session existence ≠ agent still working** - Sessions persist indefinitely
4. **Status checks don't mutate state** - Calculation is read-only, no side effects
5. **Multiple sources must be reconciled** - No single source has complete truth
6. **Tmux windows are UI layer only** - Not authoritative for state

---

## Why This Fails

### Failure Mode 1: Dashboard Shows "Active" When Agent is Done

**Symptom:** Dashboard shows agent as active, but `bd show <id>` says status=closed

**Root cause:** Dashboard caching or SSE lag - hasn't received beads update yet

**Why it happens:**
- Agent reaches Phase: Complete
- `orch complete` closes beads issue
- Beads issue status = closed
- Dashboard hasn't refreshed or polled beads yet
- Dashboard still shows cached "active" state

**Fix:** Refresh dashboard browser tab (forces beads query)

**NOT the fix:** Deleting OpenCode session (treats symptom, not cause)

### Failure Mode 2: "Dead" Agents That Actually Completed

**Symptom:** Dashboard shows "dead", but work is done and beads issue closed

**Root cause:** Session cleanup happened before dashboard queried, cascade reached session check

**Why it happens:**
- Agent completed, beads issue closed
- Session cleanup ran (manual or automatic)
- Dashboard cascade: beads check → no issue (closed) → session check → no session → "dead"

**Fix (Jan 8):** Priority Cascade puts beads/Phase check before session existence check

### Failure Mode 3: Agent Went Idle But Not Complete

**Symptom:** Session status is "idle" but no `Phase: Complete` comment

**Root cause:** Agent ran out of context, crashed, or didn't follow completion protocol

**Why it happens:**
- Session exhausts context (150k tokens)
- Agent stops responding
- SSE event: `session.status = idle`
- No `Phase: Complete` was ever written
- Dashboard shows "idle" or "waiting"

**This is expected behavior.** Session idle ≠ work complete. Only agents that explicitly run `bd comment <id> "Phase: Complete"` are considered done.

**Fix:** Check workspace for what agent accomplished, then either:
- `orch complete <id> --force` if work is done
- `orch abandon <id>` if work is incomplete

### Failure Mode 4: Cross-Project Agents Not Visible

**Symptom:** Agent spawned with `--workdir /other/project` doesn't appear in dashboard

**Root cause:** Dashboard only scans current project's `.orch/workspace/` directory

**Why it happens:**
- Workspace created in `/other/project/.orch/workspace/`
- Dashboard running from `orch-go` only sees `orch-go/.orch/workspace/`
- Cross-project discovery requires querying OpenCode sessions for unique directories

**Fix (Jan 6):** Multi-project workspace cache built from OpenCode session directories

---

## Constraints

### Why Four Layers Instead of Single Source of Truth?

**Constraint:** Each layer serves a distinct purpose with different lifecycle requirements

**Implication:** State must be reconciled by combining sources, not stored in one place

**Breakdown (state layers — orch owns):**
- **Beads** - Work tracking (survives everything, multi-session)
- **Workspace files** - Artifact record (SPAWN_CONTEXT, SYNTHESIS, tier metadata)

**Breakdown (infrastructure layers — orch uses):**
- **OpenCode disk** - Message history (debugging, resume)
- **OpenCode memory** - Real-time processing state (fast queries)
- **Tmux** - Visual monitoring (orchestrator needs to SEE work)

**This enables:** Each layer optimized for its purpose
**This constrains:** Must reconcile at query time (eventual consistency). The reconciliation burden is heaviest when infrastructure layers are treated as authoritative — they should be consulted only as fallback, after state layers.

### Why Can't We Infer Completion from Session State?

**Constraint:** Sessions go idle for many reasons (paused, waiting, crashed, context exhausted, completed)

**Implication:** Only explicit `Phase: Complete` signal is reliable

**Workaround:** Agents must follow completion protocol

**This enables:** Agents can pause/wait without being marked complete
**This constrains:** Agents that crash without reporting phase look "incomplete"

### Why Registry Caused Drift?

**Constraint:** Registry attempted to cache all four layers, but updates were async and incomplete

**Implication:** Registry state diverged from authoritative sources

**Root cause:**
- Beads issues closed via `bd close` (not `orch complete`) → registry not updated
- OpenCode sessions persist → registry shows "dead" when session exists
- Tmux windows close → registry still shows "running"

**Fix:** Query authoritative sources directly, use registry only for orchestrator-set metadata (abandoned status)

---

## Evolution

**Dec 20-21, 2025: Initial Implementation**
- Basic agent tracking via registry
- Tmux windows as primary UI
- OpenCode sessions for execution

**Dec 22-26, 2025: State Reconciliation Issues**
- "Dead" agents that actually completed
- "Active" agents when beads said closed
- Registry drift discovered

**Jan 4-6, 2026: Four-Layer Model**
- Investigation `2026-01-04-design-dashboard-agent-status-model.md` proposed Priority Cascade
- Beads established as canonical source for completion
- Registry demoted to metadata only

**Jan 6, 2026: Cross-Project Visibility**
- Multi-project workspace discovery
- Directory extraction from OpenCode sessions
- Beads queries routed to correct project

**Jan 12, 2026: Model Synthesis**
- 17 investigations synthesized into this model
- Four-layer architecture formalized
- Constraints made explicit

**Feb 13, 2026: State vs Infrastructure Distinction**
- Four-layer table reframed with Category column (State vs Infrastructure)
- Workspace files added as explicit state layer
- New section explaining why conflating state and infrastructure creates reconciliation burden
- Three-bucket ownership model (Own/Accept/Lobby) referenced
- Decision: `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`

---

## References

**Key Investigations:**
- `2026-01-04-design-dashboard-agent-status-model.md` - Priority Cascade design
- `2026-01-06-inv-cross-project-agent-visibility.md` - Multi-project discovery
- `2025-12-26-inv-registry-drift-analysis.md` - Why registry caching failed
- `2025-12-22-inv-completion-detection-race-condition.md` - Session idle ≠ complete
- ...and 13 others

**Decisions Informed by This Model:**
- Beads as canonical source of truth (completion)
- Priority Cascade for status calculation
- Four-layer architecture (no single source)
- Registry demoted to metadata only
- State vs infrastructure distinction (`.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`)

**Related Models:**
- `.kb/models/dashboard-agent-status.md` - How Priority Cascade calculates status
- `.kb/models/opencode-session-lifecycle.md` - How OpenCode sessions work
- `.kb/models/spawn-architecture.md` - How agents are created

**Related Guides:**
- `.kb/guides/agent-lifecycle.md` - How to use agent lifecycle commands (procedural)
- `.kb/guides/completion.md` - How to complete agents (procedural)
- `.kb/guides/status.md` - How to use orch status (procedural)

**Primary Evidence (Verify These):**
- `cmd/orch/serve_agents.go` - Status calculation implementation (~1400 lines)
- `pkg/registry/registry.go` - Registry structure (metadata only)
- `pkg/verify/check.go` - Phase parsing from beads comments
- `.beads/issues.jsonl` - Canonical completion source
