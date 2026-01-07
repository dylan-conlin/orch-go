<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Extended `orch resume` to support `--workspace` and `--session` flags, enabling resume of orchestrator sessions that don't have beads IDs.

**Evidence:** Implementation tested with unit tests (all pass), full test suite (all pass), and verified `orch resume --help` shows new flags.

**Knowledge:** Orchestrators use different context files (META_ORCHESTRATOR_CONTEXT.md, ORCHESTRATOR_CONTEXT.md) vs workers (SPAWN_CONTEXT.md), requiring detection logic for appropriate resume prompts.

**Next:** Close issue - feature complete with tests passing.

---

# Investigation: Extend Orch Resume Work Workspace

**Question:** How can we extend `orch resume` to work with orchestrator sessions that don't have beads IDs?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Original resume only supported beads ID lookup

**Evidence:** The original `runResume` function in `cmd/orch/resume.go` took a single argument (beads ID) and searched workspaces by matching the beads ID in the workspace name.

**Source:** `cmd/orch/resume.go:17-32` (original implementation)

**Significance:** This design worked for workers but failed for orchestrators which are spawned without beads tracking.

---

### Finding 2: Workspaces have three different context file types

**Evidence:** 
- Workers: `SPAWN_CONTEXT.md`
- Orchestrators: `ORCHESTRATOR_CONTEXT.md`
- Meta-orchestrators: `META_ORCHESTRATOR_CONTEXT.md`

Verified by examining workspace directories:
```
.orch/workspace/og-arch-dashboard-long-outcome-06jan-061c/SPAWN_CONTEXT.md (worker)
.orch/workspace/meta-orch-continue-meta-orch-06jan-2c9a/META_ORCHESTRATOR_CONTEXT.md (orchestrator)
```

**Source:** `.orch/workspace/*/` directory structure

**Significance:** Resume prompts need to reference the correct context file type for proper session resumption.

---

### Finding 3: Session ID stored in workspace .session_id file

**Evidence:** Workers have `.session_id` file in workspace containing OpenCode session ID. Prior investigation (`.kb/investigations/2026-01-06-inv-orchestrator-sessions-spawned-via-tmux.md`) documented that orchestrator sessions spawned via tmux may not have this file (depends on spawn mode).

**Source:** `pkg/spawn/session.go` - `ReadSessionID` function

**Significance:** `--workspace` flag can read `.session_id` from workspace, with fallback to API title matching.

---

## Synthesis

**Key Insights:**

1. **Multiple identification paths needed** - Beads ID (workers), workspace name (orchestrators), session ID (direct) all serve different use cases.

2. **Context file detection** - Resume prompts should reference the correct context file type, detected by checking which file exists in the workspace.

3. **Graceful fallbacks** - If `.session_id` file missing, try API-based session lookup by workspace name in title.

**Answer to Investigation Question:**

Extended `orch resume` with two new flags:
- `--workspace <name>`: Finds workspace, reads `.session_id`, generates context-aware resume prompt
- `--session <id>`: Sends resume prompt directly to session

Implementation detects context file type (META_ORCHESTRATOR_CONTEXT.md > ORCHESTRATOR_CONTEXT.md > SPAWN_CONTEXT.md) and generates appropriate resume prompts.

---

## Structured Uncertainty

**What's tested:**

- ✅ Unit tests for `GenerateResumePrompt`, `GenerateOrchestratorResumePrompt`, `GenerateSessionResumePrompt` (all pass)
- ✅ Full test suite passes (`go test ./...`)
- ✅ Build succeeds (`make install`)
- ✅ Command help shows new flags (`orch resume --help`)

**What's untested:**

- ⚠️ End-to-end resume of a real orchestrator session (would need running orchestrator to test)
- ⚠️ API fallback path when .session_id is missing (covered by code but not integration tested)

**What would change this:**

- If OpenCode API changes session listing format
- If workspace structure changes (different context file names)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Add --workspace and --session flags** - Extend cobra command with mutually exclusive identifier options.

**Why this approach:**
- Minimal change to existing beads ID flow
- Clear UX with --help documentation
- Context-aware resume prompts per session type

**Trade-offs accepted:**
- Three separate resume functions (one per identifier type) - slight code duplication
- Fallback complexity when .session_id missing

**Implementation sequence:**
1. ✅ Add flags to cobra command with validation
2. ✅ Implement `runResumeByWorkspace` with context detection
3. ✅ Implement `runResumeBySession` for direct resume
4. ✅ Add tests for new prompt generation functions

### Alternative Approaches Considered

**Option B: Single unified resume with auto-detection**
- **Pros:** Simpler UX (just one argument)
- **Cons:** Ambiguous when workspace name contains beads-like IDs
- **When to use instead:** If we want to simplify to just `orch resume <identifier>`

---

## References

**Files Examined:**
- `cmd/orch/resume.go` - Original and new implementation
- `cmd/orch/resume_test.go` - Tests
- `pkg/spawn/session.go` - Session ID read/write utilities
- `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Context

**Commands Run:**
```bash
# Run resume tests
go test ./cmd/orch/... -run "Resume" -v

# Full test suite
go test ./...

# Build and install
make install

# Verify help
orch resume --help
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Workspace/session mental model
- **Investigation:** `.kb/investigations/2026-01-06-inv-orchestrator-sessions-spawned-via-tmux.md` - Session ID capture issues
- **Issue:** `orch-go-xdcpc` - This implementation

---

## Investigation History

**2026-01-06 17:30:** Investigation started
- Initial question: How to resume orchestrators without beads IDs?
- Context: Spawned from issue orch-go-xdcpc

**2026-01-06 17:40:** Implementation complete
- Added --workspace and --session flags
- Added context file detection for appropriate prompts
- Tests passing, build verified

**2026-01-06 17:45:** Investigation completed
- Status: Complete
- Key outcome: orch resume now supports orchestrator sessions via --workspace flag
