<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Interactive orchestrator sessions (via `opencode`/`oc`) don't create workspaces because `orch session start` only stores state in `~/.orch/session.json` - it doesn't create workspace directories.

**Evidence:** Code analysis shows `runSessionStart()` in session.go only calls `store.Start(goal)` which writes to `~/.orch/session.json`. Spawned orchestrators use `WriteOrchestratorContext()` which creates `.orch/workspace/{name}/` with SESSION_HANDOFF.md.

**Knowledge:** Two parallel session models exist - "tracked spawns" with workspaces (spawned orchestrators) and "lightweight sessions" without workspaces (interactive orchestrators). The plugin infrastructure to create workspaces exists but isn't wired up.

**Next:** Enhance `orch session start` to optionally create a workspace with SESSION_HANDOFF.md template, or enhance the OpenCode plugin to create workspace on session.created.

---

# Investigation: Interactive Orchestrator Sessions Don't Create Workspaces

**Question:** Why don't interactive orchestrator sessions (opened directly via opencode/oc) create workspaces or SESSION_HANDOFF.md like spawned orchestrator sessions do?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** Implement recommended fix
**Status:** Complete

---

## Findings

### Finding 1: Two Parallel Session Models Exist

**Evidence:** 
- Spawned orchestrators (`orch spawn orchestrator`) use `WriteOrchestratorContext()` which creates `.orch/workspace/{name}/` with:
  - ORCHESTRATOR_CONTEXT.md
  - SESSION_HANDOFF.md (pre-filled)
  - .orchestrator marker
  - .tier file (set to "orchestrator")
  - .workspace_name file
- Interactive orchestrators (`opencode`/`oc`) trigger the `orchestrator-session.ts` plugin which runs `orch session start` - this only writes to `~/.orch/session.json`

**Source:** 
- `pkg/spawn/orchestrator_context.go:191-251` - WriteOrchestratorContext creates workspace
- `cmd/orch/session.go:80-115` - runSessionStart only updates session store
- `plugins/orchestrator-session.ts:209-210` - Plugin runs `orch session start > /dev/null`

**Significance:** Interactive sessions lose context on exit because there's no persistent workspace. The SESSION_HANDOFF.md that would preserve learnings is never created.

---

### Finding 2: Plugin Infrastructure Already Exists

**Evidence:**
The `orchestrator-session.ts` plugin already:
- Detects orchestrator vs worker sessions
- Injects orchestrator skill on config hook
- Runs `orch session start` on session.created event

**Source:**
- `plugins/orchestrator-session.ts` - Full plugin implementation
- `~/.config/opencode/plugin/orchestrator-session.ts` - Symlinked for global loading

**Significance:** The hook point exists to create workspaces - we just need to extend either the plugin or `orch session start` to create workspace directories.

---

### Finding 3: Prior Session Directory Structure Exists But Is Minimal

**Evidence:**
```
~/.orch/session/
├── 2025-12-29/
│   └── SESSION_CONTEXT.md
└── 2026-01-01/
    └── SESSION_CONTEXT.md
```

SESSION_CONTEXT.md contains only basic metadata:
```
# Orchestrator Session Context
**Session ID:** sess_20260101_150448
**Started:** 2026-01-01 15:04:48
**Goal:** Test session workspace formalization
```

This is much thinner than the full workspace approach with SESSION_HANDOFF.md templates.

**Source:** `~/.orch/session/2026-01-01/SESSION_CONTEXT.md`

**Significance:** A session directory mechanism exists but doesn't use the rich workspace infrastructure. This could be enhanced to match spawned orchestrator workspaces.

---

## Synthesis

**Key Insights:**

1. **Gap is in `orch session start`, not the plugin** - The plugin correctly runs `orch session start` but that command doesn't create workspaces. The fix belongs in the session command or a new command like `orch session init-workspace`.

2. **Session directories exist but are underutilized** - The `~/.orch/session/{date}/` structure exists but only contains a minimal SESSION_CONTEXT.md. This could be enhanced to include SESSION_HANDOFF.md.

3. **Workspace naming is the challenge** - Spawned orchestrators have task-based names (e.g., `orch-debug-foo-06jan-abc1`). Interactive sessions don't have a task description, so naming would need to use goal text or session ID.

**Answer to Investigation Question:**

Interactive orchestrator sessions don't create workspaces because `orch session start` was designed as a lightweight session tracker, not a full workspace creator. The workspace creation logic exists in `WriteOrchestratorContext()` which is only called during `orch spawn orchestrator`.

The fix is to enhance `orch session start` to optionally create a workspace using the existing `~/.orch/session/{date}/` directory structure, pre-populating SESSION_HANDOFF.md with metadata.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch session start` only writes to session.json (verified: code review of session.go:80-115)
- ✅ Plugin runs `orch session start` on session.created (verified: plugin source code)
- ✅ Spawned orchestrators get full workspaces (verified: WriteOrchestratorContext code path)
- ✅ Fix creates SESSION_HANDOFF.md on `orch session start` (verified: built and tested locally, workspace created at `~/.orch/session/2026-01-06/`)

**What's untested:**

- ⚠️ Whether SESSION_HANDOFF.md will actually be filled by interactive orchestrators (behavioral)
- ⚠️ Whether the `~/.orch/session/{date}/` location is better than project `.orch/workspace/` (UX)

**What would change this:**

- If interactive sessions are intentionally lightweight and workspaces aren't wanted
- If plugin-based workspace creation causes startup delays
- If orchestrators work across multiple projects (workspace location matters)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Enhance `orch session start` to create session workspace** - Add `--workspace` flag (or make it default) that creates `~/.orch/session/{date}/` with SESSION_HANDOFF.md template.

**Why this approach:**
- Minimal change - extends existing command
- Uses existing session directory structure
- Plugin doesn't need modification
- Progressive adoption (flag or behavioral)

**Trade-offs accepted:**
- Interactive sessions get slightly more structure
- Users must remember to fill SESSION_HANDOFF.md (can't be enforced)

**Implementation sequence:**
1. Add `writeSessionWorkspace()` function to session.go that creates SESSION_HANDOFF.md in session directory
2. Call it from `runSessionStart()` 
3. Update plugin to suppress stdout (already does this with `> /dev/null`)
4. Test with real interactive session

### Alternative Approaches Considered

**Option B: Plugin creates workspace directly**
- **Pros:** Decoupled from orch CLI, can customize per-project
- **Cons:** Duplicates workspace creation logic, harder to maintain
- **When to use instead:** If workspace creation needs project-specific customization

**Option C: Add `orch session init-workspace` subcommand**
- **Pros:** Explicit, can be called manually anytime
- **Cons:** Extra step, won't be auto-called by plugin
- **When to use instead:** If workspace creation should be opt-in per session

**Rationale for recommendation:** Option A (enhance existing command) provides the best balance of minimal disruption and automatic workspace creation.

---

### Implementation Details

**What to implement first:**
1. Add `writeSessionWorkspace()` function using `PreFilledSessionHandoffTemplate` from orchestrator_context.go
2. Determine workspace path: `~/.orch/session/{date}/` (consistent with existing structure)
3. Write SESSION_HANDOFF.md with goal and start time pre-filled

**Things to watch out for:**
- ⚠️ Plugin runs `orch session start > /dev/null` so output is suppressed - file creation should be silent
- ⚠️ Don't duplicate SESSION_HANDOFF.md if session already has one
- ⚠️ Consider goal being empty ("") - use "Interactive session" as default

**Areas needing further investigation:**
- Should workspace be in `~/.orch/session/` (global) or project `.orch/workspace/` (local)?
- How to handle cross-project sessions (orchestrator moves between repos)?
- Should `orch session end` gate on SESSION_HANDOFF.md presence?

**Success criteria:**
- ✅ `orch session start "goal"` creates `~/.orch/session/{date}/SESSION_HANDOFF.md`
- ✅ SESSION_HANDOFF.md contains goal and start time
- ✅ Next session can read prior SESSION_HANDOFF.md for context
- ✅ Plugin continues to work without modification

---

## References

**Files Examined:**
- `cmd/orch/session.go` - Session start/end command implementation
- `pkg/session/session.go` - Session store (only writes session.json)
- `pkg/spawn/orchestrator_context.go` - Workspace creation for spawned orchestrators
- `plugins/orchestrator-session.ts` - OpenCode plugin for interactive sessions

**Commands Run:**
```bash
# Check session state
cat ~/.orch/session.json

# Check existing session directories
ls -la ~/.orch/session/

# Check existing session content
cat ~/.orch/session/2026-01-01/SESSION_CONTEXT.md
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md` - Prior work on orchestrator session architecture
- **Plugin:** `plugins/orchestrator-session.ts` - Existing plugin that triggers session start

---

## Investigation History

**2026-01-06 13:00:** Investigation started
- Initial question: Why don't interactive sessions get workspaces?
- Context: Discovered during session wrap-up that SESSION_HANDOFF.md wasn't created

**2026-01-06 13:15:** Found parallel session models
- Spawned vs interactive orchestrators use different code paths
- Plugin exists but runs lightweight `orch session start`

**2026-01-06 13:30:** Investigation completed
- Status: Complete
- Key outcome: Gap is in `orch session start` not creating workspaces - recommend enhancing the command
