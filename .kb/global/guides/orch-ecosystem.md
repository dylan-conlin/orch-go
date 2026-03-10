# Orch Ecosystem Architecture

**Purpose:** Comprehensive guide to how orchestration ecosystem repos fit together, enabling agents to understand cross-repo architecture at session start.

**Last Updated:** 2025-12-19

---

## The Big Picture

You're looking at general-purpose AI workforce infrastructure. The surface presentation is "coordinate Claude Code agents" but the underlying architecture is:
- **Dependency-aware work queue** (beads)
- **Worker pool with supervisor layer** (orch + opencode)  
- **Persistent state and knowledge** (kb, kn, git-backed everything)

This can run research pipelines, competitive intel, content production, batch analysis - anything decomposable into tasks with artifacts. The code orchestration is just the current application.

**The constraint that shaped everything:** Claude has no memory between sessions. Every pattern compensates for this amnesia - workspaces, artifacts, SPAWN_CONTEXT.md, all of it.

---

## Quick Reference: What Each Tool Does

```
orch        → "agent coordination"     (spawn, monitor, complete, daemon)
beads (bd)  → "what work needs doing"  (issues, dependencies, tracking)
kb          → "deep documentation"     (investigations, decisions)
kn          → "what we've learned"     (quick decisions, constraints, failures)
agentlog    → "what happened"          (error/event logging for agents)
opencode    → "agent execution"        (Claude frontend, session management)
playwright  → "browser automation"     (MCP server for UI testing)
```

**Design principles behind these tools:**
- Local-first: Files over databases, git over external services
- Compose over monolith: Small focused tools that combine
- AI-first: Surfacing over browsing (bring context to agent, don't require navigation)
- Gate over remind: Enforce capture through gates, not reminders that fail under load
- **Authority is Scoping:** Orchestrators exercise authority by defining context boundaries, not by micro-managing reasoning.

See `~/.kb/principles.md` for the full philosophy.

---

## Repository Overview

### Core Orchestration

#### orch-cli (GitHub: dylan-conlin/orch)
**Purpose:** Agent spawning, monitoring, completion, and the work daemon.

**What it does:**
- **Spawning:** `orch spawn SKILL "task"` creates agent in tmux with SPAWN_CONTEXT.md
- **Monitoring:** `orch status` shows running agents, `orch check <id>` inspects specific agent
- **Completion:** `orch complete <id>` verifies deliverables, closes beads issue, cleans up
- **Daemon:** `orch daemon run` autonomously processes `triage:ready` beads issues overnight
- **Strategic:** `orch focus`, `orch drift`, `orch next` for cross-project alignment

**Key concepts:**
- Every spawned agent gets a workspace in `.orch/workspace/{name}/`
- SPAWN_CONTEXT.md contains full skill guidance, authority, deliverables
- Agents report progress via `bd comment <issue-id> "Phase: X"`

**Implementation:** Currently Python (pipx at `~/.local/bin/orch`). Proposed migration to Go as HTTP client for OpenCode - see `.kb/decisions/2025-12-18-sdk-based-agent-management.md`.

**Current friction points:**
- Completion detection is manual (must run `orch status` or cycle tmux windows)
- No push notifications when agents complete
- Post-completion Q&A requires finding right tmux window

---

#### beads (bd) (Steve Yegge's project)
**Purpose:** Dependency-aware issue queue with git-backed persistence.

**Origin:** Steve Yegge's CLI. Built with the same AI-first insight - surfacing commands (`bd ready`) that bring relevant state to the agent rather than requiring navigation. Dylan uses it as the work queue layer in orch.

**What it does:**
- **Issue tracking:** `bd create`, `bd list`, `bd show`, `bd close`
- **Dependencies:** Issues can block/be blocked by other issues  
- **Progress tracking:** `bd comment <id> "message"` adds timestamped comments
- **Work discovery:** `bd ready` shows unblocked issues, `bd blocked` shows blocked ones
- **Sync:** Git-backed JSONL files in `.beads/` directory

**Key concepts:**
- Issues live in `.beads/issues.jsonl` (git-tracked)
- Each project has independent `.beads/` directory
- Triage labels: `triage:ready` (daemon can auto-spawn), `triage:review` (needs human)
- Agent lifecycle tracked via comments: `Phase: Planning` → `Phase: Complete`

**Source:** `~/Documents/personal/beads` (Steve's repo, Dylan has local copy)
**Implementation:** Go binary at `~/go/bin/bd`

---

#### beads-ui-svelte (Dylan's project)
**Purpose:** Web UI for viewing and managing beads issues.

**What it does:**
- Visual display of issues, dependencies, and progress
- **Auto-follows orchestrator's current working directory** (tmux-follower module, polls every 2s)
- Shows ready/blocked/in-progress issues
- Kanban board, epics view, issue table

**Key concepts:**
- Part of Dylan's three-window workflow:
  - Left Ghostty: `orchestrator` tmux session
  - Right Ghostty: `workers-{project}` tmux session  
  - Firefox: beads-ui at http://127.0.0.1:3333
- When you `cd` to different project, beads-ui automatically switches databases
- tmux-follower is embedded in this repo (not separate)

**Tech:** Svelte + Express + WebSocket

**Source:** `~/Documents/personal/beads-ui-svelte`

---

### Knowledge Management

#### kb-cli (Go binary)
**Purpose:** Knowledge base management for investigations and decisions.

**What it does:**
- **Create artifacts:** `kb create investigation auth-flow --model <name>`, `kb create decision session-arch`
- **Search:** `kb context "<topic>"` returns all knowledge (decisions + investigations)
- **Link:** `kb link <artifact> --issue <beads-id>` connects artifacts to issues
- **Publish:** `kb publish <artifact>` copies to global `~/.kb/`

**Key concepts:**
- Artifacts live in `.kb/` directory per project
- Investigations answer questions with evidence (test before concluding)
- Decisions record architectural choices with status (Proposed/Accepted)
- `kb context` is the primary pre-spawn knowledge check

**Implementation:** Go binary at `~/bin/kb`

---

#### kn (Go binary)
**Purpose:** Quick operational knowledge capture (decisions, constraints, failures).

**What it does:**
- **Quick decisions:** `kn decide "Use JWT" --reason "Need stateless auth"`
- **Failed attempts:** `kn tried "SQLite sessions" --failed "Race conditions"`
- **Constraints:** `kn constrain "Max 5MB payload" --reason "API gateway limit"`
- **Open questions:** `kn question "JWT or session cookies?"`
- **Search:** `kn context "rate limiting"` returns relevant entries

**Key concepts:**
- Entries stored in `.kn/entries.jsonl` per project
- Prevents retry loops by recording what didn't work
- Faster than kb (1-2 sentences vs full documents)
- Gates enforce capture: `--reason` flag required

**Implementation:** Go binary at `~/bin/kn`

---

### Observability

#### agentlog (GitHub: dylan-conlin/agentlog)
**Purpose:** AI-native development observability CLI.

**What it does:**
- Logs errors and events from agent sessions
- Provides crash logging for debugging agent failures
- Enables post-mortem analysis

**Key concepts:**
- Captures agent execution traces
- Helps identify patterns in agent failures
- Feeds back into spawn template improvements

**Implementation:** Go binary (public repo)

---

### Agent Execution

#### opencode (GitHub: sst/opencode)
**Purpose:** Alternative Claude frontend with HTTP API for programmatic control.

**What it does:**
- Provides Claude Code-like experience in terminal
- **HTTP API for session management** (critical for orch evolution)
- SSE for real-time events (completion, errors, tool calls)
- Native session persistence and Q&A (send message to existing session)

**Key concepts:**
- Alternative to Claude Code desktop app
- Configured via `opencode.json` in project root
- **Planned:** orch-cli will become Go HTTP client for OpenCode API
  - This enables: push notifications, completion detection, post-Q&A without context replay

**API endpoints:**
```
POST   /session                    - Create session
GET    /session                    - List sessions
POST   /session/{id}/prompt_async  - Send message
GET    /event                      - SSE event stream
```

---

#### playwright-mcp (MCP Server)
**Purpose:** Browser automation for agents via Model Context Protocol.

**What it does:**
- Provides agents access to browser for UI testing
- Enables smoke tests and visual verification
- Supports screenshots, navigation, form filling

**Key concepts:**
- Used via `--mcp playwright` when spawning agents
- Preferred over browser-use (avoids context explosion)
- Critical for UI feature validation

---

### Knowledge Archive

#### orch-knowledge (GitHub: dylan-conlin/orch-knowledge)
**Purpose:** Skills, decisions, investigations, and patterns archive.

**What it does:**
- **Skills source:** `skills/src/` contains skill templates that build to `~/.claude/skills/`
- **Decisions:** `.kb/decisions/` contains architectural decisions
- **Investigations:** `.kb/investigations/` contains research and analysis
- **Patterns:** `patterns-src/` and `docs/` contain reusable patterns
- **Guides:** Published to global `~/.kb/guides/`

**Key concepts:**
- Skills have source/distribution split: edit `skills/src/`, run `orch build skills`
- Never edit `~/.claude/skills/` directly (distribution, gets overwritten)
- Knowledge is curated: not all investigations become decisions
- Patterns emerge from real work, not theoretical design

---

## Dylan's Physical Environment

**Three-window workflow managed by five layers:**

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Left Ghostty   │  │ Right Ghostty   │  │    Firefox      │
│  orchestrator   │  │ workers-{proj}  │  │   beads-ui      │
│  tmux session   │  │ tmux session    │  │ :3333           │
└─────────────────┘  └─────────────────┘  └─────────────────┘
        │                    │                     │
        └────────── yabai manages positions ───────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
   skhd (keys)       tmux (sessions)    beads-ui follows
   alt+shift+w       auto-switching     orchestrator cwd
   for init          on spawn/cd
```

**Auto-switching behavior:**
- When you `cd` in orchestrator → right Ghostty switches to `workers-{project}`
- When `orch spawn` runs → switches to that project's workers session
- When orchestrator cwd changes → beads-ui switches to that project's `.beads/*.db`

**Menu bar (sketchybar):** Widget showing agent count + color-coded status (blue=working, yellow=attention, green=complete). Click to see all agents across projects.

**Full reference:** `.orch/docs/orchestration-window-setup.md`

---

## Epic Lifecycle Pattern

Epics are questions, not commitments. Discovery and delivery are separate phases.

```
1. CREATE EPIC with question/goal
   └── "How does the system develop institutional memory?"

2. SPAWN INVESTIGATIONS as children
   └── Each investigation answers one aspect of the question
   └── Investigations may spawn more investigations

3. SYNTHESIZE findings
   └── Orchestrator reviews completed investigations
   └── Patterns emerge, architecture becomes clear

4. CLOSE EPIC when question answered
   └── Epic delivered knowledge, not code
   └── Decision records capture architectural choices

5. CREATE NEW EPIC for implementation
   └── Implementation phases are well-scoped
   └── Each phase has clear deliverables
```

**Example from Dec 2025:**
- `ws4z` "System Self-Reflection - Temporal Pattern Awareness" → 7 investigations → closed
- `ivtg` "Implement Self-Reflection Protocol" → 5 implementation phases → in progress

**Why separate discovery from delivery:**
- Different success criteria (understanding vs working code)
- Conflating creates pressure to ship before understanding
- Investigation-first prevents building the wrong thing
- Clean epic closure enables honest progress tracking

**Anti-pattern:** Epic with mixed investigation and implementation tasks. This conflates "do we understand?" with "did we build?" and creates false progress signals.

---

## Data Flow

### Issue → Daemon → Spawn → Agent → Completion

```
1. ISSUE CREATION
   User reports symptom → orch spawn issue-creation "symptom"
   OR direct: bd create "title" "description"
   → Creates beads issue with triage label

2. DAEMON PROCESSING (autonomous)
   orch daemon run → polls bd ready for triage:ready issues
   → For each: orch spawn --issue <id>

3. SPAWN
   orch spawn SKILL "task" →
   - Creates .orch/workspace/{name}/SPAWN_CONTEXT.md
   - Opens tmux window for agent (or OpenCode session)
   - Loads skill guidance into spawn context
   
4. AGENT EXECUTION
   Agent works →
   - Reports progress: bd comment <id> "Phase: X"
   - Produces deliverables (code, investigations, etc.)
   - Commits changes locally (never pushes - orchestrator-exclusive)
   - Reports: bd comment <id> "Phase: Complete"

5. COMPLETION
   orch complete <id> →
   - Verifies deliverables exist
   - Checks tests pass (if applicable)
   - Closes beads issue
   - Cleans up session
```

### Knowledge Flow

```
Quick Learning (kn):
  Agent discovers constraint → kn constrain "X" --reason "Y"
  Agent makes decision → kn decide "X" --reason "Y"
  Agent fails attempt → kn tried "X" --failed "Y"
  
  ↓ (if recurs across projects, has teeth)
  
Principle candidate → ~/.kb/principles.md

Deep Documentation (kb):
  Create investigation → kb create investigation "topic" --model <name>  # or --orphan
  Promote to decision → kb create decision "topic" (when accepting recommendation)
  
Session Start:
  Pre-spawn check → kb context "task keywords"
  Returns: prior decisions, constraints, failed attempts, investigations
```

---

## Technology Stack Philosophy

**Per "AI-Native Technology Choice" principles (`~/.kb/guides/ai-native-technology-choice.md`):**

When AI writes the code, optimize for the artifact, not developer experience:

| Project Type | Choice | Why |
|--------------|--------|-----|
| CLIs (orch, bd, kn, kb) | Go | Single binary, no runtime deps, fast startup |
| beads-ui | Svelte/Bun | Smaller bundles, no virtual DOM |
| Scripts/glue | Python | Iteration speed, ecosystem |
| Backend APIs | Python/Node (usually) | Ecosystem richness |

**The insight:** Languages that were "too annoying" for humans (verbose, complex type systems) become viable. AI absorbs the annoyance; you get the artifact benefits.

---

## AI-First CLI Patterns

**Per "Rules for AI-First CLIs" (`~/.kb/guides/ai-first-cli-rules.md`):**

Key insight: AI-first ≠ JSON-first. LLMs read prose well. Design for both.

**Essential patterns:**
- TTY detection (auto-skip confirmations for agents)
- `--json` flag for scripts/pipelines
- Actionable error messages ("Run: git add . && git commit -m '...'")
- `prime` command for session startup context injection
- `context <topic>` for aggregated discovery

All ecosystem CLIs (orch, bd, kb, kn) implement these patterns.

---

## Communication Between Repos

### orch ↔ beads
- `orch work <id>` reads issue from beads, spawns agent
- `orch spawn --issue <id>` links spawn to beads issue
- `orch complete` closes beads issue on success
- `orch daemon` polls `bd ready` for work

### orch ↔ kb/kn
- Pre-spawn: `kb context` checks for prior knowledge
- Spawn context includes relevant constraints from kn
- Investigation skill produces artifacts in `.kb/`

### orch ↔ opencode
- Current: orch spawns tmux + Claude CLI
- Future: orch becomes Go HTTP client for OpenCode API
- Benefits: push notifications, native Q&A, structured events

### orch ↔ beads-ui
- Current: Independent, shares beads data via filesystem
- Future: SSE events for real-time agent status in dashboard

---

## Which Repo for What?

| I want to... | Use... |
|--------------|--------|
| Spawn/monitor/complete agents | `orch` |
| Create/track issues | `bd` (beads) |
| View issues visually | beads-ui-svelte |
| Create investigation/decision | `kb` |
| Record quick decision/constraint | `kn` |
| Analyze agent failures | agentlog |
| Automate browser testing | playwright-mcp |
| Edit skills/patterns | orch-knowledge |
| Run Claude sessions | opencode |

---

## Per-Project Architecture (Architecture B)

Each project is architecturally independent:
- **Independent `.orch/`:** Each project has its own orchestration context
- **Independent `.beads/`:** Each project has its own issue queue
- **Independent `.kb/`:** Each project has its own knowledge base
- **Independent `.kn/`:** Each project has its own operational knowledge

**Cross-project concerns are handled by:**
- Extracting reusable patterns to orch-knowledge
- Publishing guides to global `~/.kb/guides/`
- Skills shared via `~/.claude/skills/`

**Dylan switches contexts via `cd`** - when in `/project-name/`, the orchestrator IS that project's orchestrator.

---

## Quick Commands Reference

```bash
# Agent lifecycle
orch spawn investigation "topic"     # Create investigation agent
orch spawn feature-impl "feature"    # Create feature agent
orch status                          # See running agents
orch complete <id>                   # Complete and cleanup

# Issue tracking  
bd create "title" "description"      # Create issue
bd ready                             # Show ready work
bd comment <id> "message"            # Add progress comment
bd close <id> --reason "why"         # Close issue

# Knowledge
kb context "topic"                   # Get all knowledge about topic
kn decide "X" --reason "Y"           # Record decision
kn tried "X" --failed "Y"            # Record failure
kn constrain "X" --reason "Y"        # Record constraint

# Strategic
orch focus "goal" -p project         # Set focus
orch drift                           # Check alignment
orch next                            # Get next actions

# Environment
alt+shift+w                          # Initialize three-window layout
```

---

## Evolution & Roadmap

**Current friction being addressed:**
1. Completion detection manual → Go orch + OpenCode SSE for push notifications
2. Post-completion Q&A requires context → OpenCode session persistence
3. beads-ui passive → Real-time events via SSE

**Proposed architecture (from 2025-12-18 decision):**

```
beads-ui ← HTTP/SSE ← orch (Go) ← HTTP/SSE ← OpenCode Server
                          ↑
                     beads (.beads/)
```

Go orch becomes thin orchestration layer on OpenCode, gaining:
- Single binary distribution (matching bd, kn, kb)
- Native concurrent monitoring via goroutines
- Structured events without stdout parsing
- Q&A on completed sessions without conversation replay

---

## See Also

- **Principles:** `~/.kb/principles.md` (foundational values)
- **Technology choice:** `~/.kb/guides/ai-native-technology-choice.md`
- **CLI patterns:** `~/.kb/guides/ai-first-cli-rules.md`
- **Orchestrator skill:** `skills/src/policy/orchestrator/SKILL.md`
- **Window setup:** `.orch/docs/orchestration-window-setup.md`
- **orch-Go decision:** `orch-cli/.kb/decisions/2025-12-18-sdk-based-agent-management.md`
