# Worker Patterns Guide

**Purpose:** Define patterns, protocols, and architectural constraints for worker agents. Synthesizes findings from 11 worker-related investigations into authoritative guidance.

**Last verified:** Jan 17, 2026

---

## Quick Reference

| Topic | Key Rule |
|-------|----------|
| Identity | ORCH_WORKER=1 env var + workspace path signals |
| Progress | `bd comment <issue-id> "Phase: X - summary"` |
| Completion | Report Phase: Complete → SYNTHESIS.md → commit → /exit |
| Authority | Implementation: decide. Architecture: escalate. |
| Servers | `orch servers` (project) OK. `orch serve` (infrastructure) never. |

---

## 1. Worker Identity & Detection

Workers are spawned agents that execute delegated tasks. They need to be distinguished from orchestrator sessions and manual OpenCode sessions for metrics and behavior filtering.

### The Three Detection Signals

Workers are identified by a combination of signals:

| Signal | Source | Detection Point |
|--------|--------|-----------------|
| `ORCH_WORKER=1` | Set by `orch spawn` at all spawn paths | Environment variable |
| `SPAWN_CONTEXT.md` exists | Created by spawn in `.orch/workspace/` | File existence check |
| Path contains `.orch/workspace/` | Workspace directory structure | Tool argument paths |

**Source:** `2025-12-23-inv-set-orch-worker-environment-variable.md`, `2026-01-10-inv-add-worker-filtering-coaching-ts.md`

### Critical Architecture Constraint

**Plugins run in server process, not per-agent.** This has major implications:

```
OpenCode Server (single process)
    ├── Plugin A (runs HERE, sees server env)
    ├── Plugin B (runs HERE, sees server env)
    │
    └── Spawns agents (SEPARATE processes)
        ├── Worker 1 (ORCH_WORKER=1 set HERE)
        ├── Worker 2 (ORCH_WORKER=1 set HERE)
        └── Orchestrator (no ORCH_WORKER)
```

**Implication:** Cannot detect workers via `process.env.ORCH_WORKER` at plugin initialization. Detection must happen **per-session** in tool hooks using observable signals (workdir paths, file reads).

**Pattern:** Check `input.args.workdir` or file paths in `tool.execute.after` hook, cache result per-sessionID.

**Source:** `2026-01-10-inv-debug-worker-filtering-coaching-ts.md`

### Light-Tier Spawn Detection Gap

**Issue:** Light-tier spawns inject context directly into the prompt rather than having agents read SPAWN_CONTEXT.md. This bypasses the "read SPAWN_CONTEXT.md" detection signal.

**Solution:** Force all spawn tiers to read SPAWN_CONTEXT.md at session start, or use path-based detection as primary signal.

**Source:** `2026-01-17-inv-verify-worker-metrics-perform-10.md`

---

## 2. Worker Authority Delegation

Workers have authority to make implementation decisions but must escalate strategic/architectural choices.

### Workers CAN Decide

- Implementation details (code structure, naming, file organization)
- Testing strategies (which tests, how to structure, test frameworks)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording
- Order of implementation steps

### Workers MUST Escalate

- Architectural decisions (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**Default:** When uncertain, err on side of escalation. Document question in workspace, set Status: QUESTION, and wait.

**Source:** `worker-base` skill, `decision-authority.md` guide

---

## 3. Progress Tracking Protocol

Workers report progress via beads comments for visibility across sessions.

### Phase Reporting

```bash
# Report at phase transitions
bd comment <issue-id> "Phase: Planning - Analyzing codebase structure"
bd comment <issue-id> "Phase: Implementing - Adding authentication middleware"
bd comment <issue-id> "Phase: Complete - Tests: go test ./... - 47 passed, 0 failed"

# Report blockers/questions immediately
bd comment <issue-id> "BLOCKED: Need clarification on API contract"
bd comment <issue-id> "QUESTION: Should we use JWT or session-based auth?"
```

### First 3 Actions Rule

Within first 3 tool calls, workers MUST:
1. Report via `bd comment <issue-id> "Phase: Planning - [brief description]"`
2. Read relevant codebase context for task
3. Begin planning

**Why:** If Phase is not reported within first 3 actions, agent will be flagged as unresponsive. Orchestrator monitors via beads comments.

### Test Evidence Requirement

When reporting Phase: Complete, include **actual test output**, not just "tests passing":

| Format | Example |
|--------|---------|
| `Tests: <command> - <output>` | `Tests: go test ./... - 47 passed, 0 failed (2.3s)` |
| Include coverage if available | `Tests: make test - PASS (coverage: 78%)` |

**Why:** `orch complete` validates test evidence. Vague claims like "all tests pass" trigger manual verification.

**Source:** `worker-base` skill, `2025-12-20-inv-update-all-worker-skills-include.md`

---

## 4. Session Complete Protocol

Precise sequence for worker session completion:

### Full-Tier Spawns

1. `bd comment <issue-id> "Phase: Complete - [summary]"` (report FIRST)
2. Create SYNTHESIS.md in workspace
3. `git add . && git commit -m "..."` (commit all changes)
4. `/exit` (close agent session)

### Light-Tier Spawns

1. `bd comment <issue-id> "Phase: Complete - [summary]"` (report FIRST)
2. `git add . && git commit -m "..."` (commit all changes)
3. `/exit` (close agent session)

SYNTHESIS.md is NOT required for light-tier spawns.

### Critical: What Workers NEVER Do

| Command | Why Never |
|---------|-----------|
| `bd close` | Only orchestrator closes issues via `orch complete` |
| `git push` | Workers commit locally; orchestrator pushes after review |
| Restart `orch serve` | Infrastructure is orchestrator-only |
| Restart daemon | Infrastructure is orchestrator-only |

**Why this order:** If agent dies after commit but before reporting Phase: Complete, orchestrator cannot detect completion. Reporting phase first ensures visibility even if agent dies before committing.

**Source:** `worker-base` skill, `2026-01-07-inv-workers-attempting-restart-orch-servers.md`

---

## 5. Workspace & Naming Conventions

Workers operate in isolated workspaces with predictable naming.

### Workspace Name Format

```
{project-prefix}-{type}-{task-description}-{date}-{unique-suffix}

Example: og-work-test-worker-naming-13jan-e072
         │   │    │                  │      │
         │   │    │                  │      └── Unique suffix (4 chars)
         │   │    │                  └── Date (DDmon format)
         │   │    └── Task description (slugified)
         │   └── Type (work, feat, inv, arch)
         └── Project prefix (og = orch-go)
```

### Workspace Location

```
.orch/workspace/{workspace-name}/
    ├── SPAWN_CONTEXT.md   # Task context, skill guidance
    ├── SYNTHESIS.md       # Knowledge externalization (full-tier)
    └── ...                # Other artifacts
```

**Source:** `2026-01-13-inv-test-worker-naming.md`

---

## 6. Server Separation

Critical distinction: project servers vs orchestration infrastructure.

### Project Servers (`orch servers`)

- What: Web frontends, APIs, dev servers for the project
- How: tmuxinator-managed, creates `workers-{project}` sessions
- Workers: CAN use `orch servers start/stop <project>`
- When: UI work requiring dev server restarts

### Orchestration Infrastructure (`orch serve`, daemon)

- What: Dashboard API (localhost:5188), autonomous agent daemon
- How: launchd-managed services
- Workers: NEVER touch these
- Who: Orchestrator-only infrastructure

```bash
# Workers CAN do this (project servers)
orch servers stop orch-go
orch servers start orch-go

# Workers NEVER do this (infrastructure)
launchctl kickstart ...
pkill orch serve
```

**Source:** `2026-01-07-inv-workers-attempting-restart-orch-servers.md`

---

## 7. Worker Metrics (vs Orchestrator Metrics)

Workers need different health signals than orchestrators.

### Orchestrator Metrics (NOT for workers)

| Metric | Purpose |
|--------|---------|
| `action_ratio` | Detect doing-without-asking |
| `frame_collapse` | Detect level collapse |
| `compensation_pattern` | Detect over-delegation |

### Worker Metrics

| Metric | Threshold | Purpose |
|--------|-----------|---------|
| `tool_failure_rate` | >= 3 consecutive | Detect stuck on tool errors |
| `context_usage` | Every 50 calls | Track token budget consumption |
| `time_in_phase` | Every 30 calls | Detect stalled progress |
| `commit_gap` | >= 30 min | Remind to checkpoint work |

**Emission Pattern:** Metrics emit at intervals (50 or 30 tool calls) or at thresholds to avoid flooding metrics file.

**Source:** `2026-01-17-inv-add-worker-specific-metrics-plugins.md`

---

## 8. Knowledge Externalization ("Leave it Better")

Before completing, workers must externalize any knowledge gained.

### The Prompt

> "Before completing, reflect: Did I discover anything that future agents or sessions should know?"

### Available Commands

```bash
kn decide "X"   --reason "Y"   # Choices made
kn tried "X"    --failed "Y"   # Failed approaches
kn constrain "X" --reason "Y"  # Discovered constraints
kn question "X"                # Open questions
```

### Escape Hatch

Not all sessions produce new knowledge. If nothing to externalize, explicitly note "no new knowledge" in completion comment.

**Source:** `2025-12-20-inv-update-all-worker-skills-include.md`

---

## 9. The worker-base Skill

Common patterns shared via skill dependencies.

### What worker-base Provides

- Authority delegation rules
- Beads progress tracking protocol
- Phase reporting instructions
- Status update patterns
- Exit/completion protocol

### How Skills Inherit

Skills declare dependency in frontmatter:
```yaml
dependencies: [worker-base]
```

At spawn time, `LoadSkillWithDependencies()` prepends worker-base content to skill.

**Note:** skillc doesn't support cross-directory dependencies at compile time. Resolution happens at runtime via orch-go's skill loader.

**Source:** `2025-12-25-inv-create-worker-base-skill-shared.md`

---

## 10. Tmux Interaction Patterns

Workers have limited tmux interaction by design.

### Headless Spawns (Default)

- No tmux window created
- Uses HTTP API to OpenCode server
- No tmux commands executed
- Highest concurrency (5+ agents)

### Tmux Spawns (Opt-in with `--tmux`)

- Creates window in `workers-{project}` session
- `select-window` focuses within session (doesn't switch sessions)
- `switch-client` only if `--attach` flag used
- For visual monitoring of critical work

### Known Behavior

If tmux client switches unexpectedly during headless spawn:
- **Not** caused by orch-go headless code (no tmux interaction)
- Check: tmux hooks, tmuxinator startup_window, external scripts

**Source:** `2026-01-08-inv-bug-worker-agents-cause-tmux.md`

---

## Troubleshooting

### Worker Not Detected

**Symptoms:** Worker metrics not appearing, orchestrator metrics polluting

**Causes:**
1. Light-tier spawn bypassing SPAWN_CONTEXT.md read
2. Plugin checking process.env instead of tool hooks
3. Workspace path not matching `.orch/workspace/` pattern

**Fix:** Ensure detection uses per-session tool hook patterns, not plugin init.

### Phase Not Reported

**Symptoms:** Orchestrator thinks agent is stuck, orch complete fails

**Causes:**
1. Agent didn't run `bd comment` within first 3 actions
2. Network/CLI error during comment
3. Wrong issue ID in bd comment

**Fix:** Workers must report Phase: Planning within first 3 tool calls.

### SYNTHESIS.md Missing

**Symptoms:** orch complete rejects work, asks for synthesis

**Causes:**
1. Light-tier spawn (doesn't require SYNTHESIS.md - this is OK)
2. Full-tier spawn forgot to create it
3. Created but not committed

**Fix:** Full-tier spawns must create and commit SYNTHESIS.md before /exit.

---

## Related Guides

- **decision-authority.md** - When to decide vs escalate
- **spawn.md** - How spawning works
- **agent-lifecycle.md** - Full agent lifecycle
- **completion.md** - Completion verification details
- **beads-integration.md** - Progress tracking via beads

---

## Provenance

This guide synthesizes findings from the worker investigation cluster:

| Investigation | Key Finding |
|---------------|-------------|
| `2025-12-20-inv-update-all-worker-skills-include` | Knowledge externalization phase |
| `2025-12-23-inv-set-orch-worker-environment-variable` | ORCH_WORKER=1 in all spawn paths |
| `2025-12-25-inv-create-worker-base-skill-shared` | worker-base skill with runtime dependencies |
| `2026-01-07-inv-workers-attempting-restart-orch-servers` | Project vs infrastructure server separation |
| `2026-01-08-inv-bug-worker-agents-cause-tmux` | Headless spawns don't touch tmux |
| `2026-01-10-inv-add-worker-filtering-coaching-ts` | Three-signal worker detection |
| `2026-01-10-inv-debug-worker-filtering-coaching-ts` | Plugin runs in server process constraint |
| `2026-01-13-inv-test-worker-naming` | Workspace naming convention |
| `2026-01-17-inv-add-worker-specific-metrics-plugins` | Worker-specific health metrics |
| `2026-01-17-inv-verify-worker-metrics-perform-10` | Light-tier spawn detection gap |
| `2026-01-07-inv-workers-attempting-restart-orch-servers` | Server separation clarity |
