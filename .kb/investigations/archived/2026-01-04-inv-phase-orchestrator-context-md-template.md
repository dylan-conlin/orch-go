<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created ORCHESTRATOR_CONTEXT.md template for spawnable orchestrator sessions with session goal focus and SESSION_HANDOFF.md requirement.

**Evidence:** All 10 orchestrator-related tests pass; routing works correctly via Config.IsOrchestrator field.

**Knowledge:** Orchestrator spawns differ from workers: no beads tracking, SESSION_HANDOFF.md instead of SYNTHESIS.md, `orch session end` instead of `/exit`.

**Next:** Phase 3 can now proceed with complete verification for orchestrator spawns.

---

# Investigation: Phase 2 - ORCHESTRATOR_CONTEXT.md Template

**Question:** How should the ORCHESTRATOR_CONTEXT.md template be structured to support spawnable orchestrator sessions?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Feature-impl agent
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

**Extracted-From:** `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md`

---

## Findings

### Finding 1: Config.IsOrchestrator and SessionGoal fields enable routing

**Evidence:** Added `IsOrchestrator bool` and `SessionGoal string` to spawn.Config struct. The existing IsOrchestrator field comment was updated to reflect the actual implementation.

**Source:** `pkg/spawn/config.go:132-144`

**Significance:** Clean separation between worker and orchestrator spawn paths. WriteContext() routes to WriteOrchestratorContext() when IsOrchestrator=true.

---

### Finding 2: ORCHESTRATOR_CONTEXT.md template structure

**Evidence:** Created `pkg/spawn/orchestrator_context.go` with:
- OrchestratorContextTemplate (markdown template with placeholders)
- GenerateOrchestratorContext() - generates content from Config
- WriteOrchestratorContext() - writes file + marker files
- MinimalOrchestratorPrompt() - returns prompt pointing to ORCHESTRATOR_CONTEXT.md
- DefaultSessionHandoffTemplate - template for SESSION_HANDOFF.md
- EnsureSessionHandoffTemplate() - ensures template exists

**Source:** `pkg/spawn/orchestrator_context.go:1-222`

**Significance:** Complete infrastructure for orchestrator spawn context generation. Template includes session goal focus, `orch session end` completion protocol, and explicit "Do NOT use /exit" guidance.

---

### Finding 3: Routing integrated into existing context.go

**Evidence:** Modified `WriteContext()` to check `cfg.IsOrchestrator` and route to `WriteOrchestratorContext()`. Modified `MinimalPrompt()` to check the same field and return appropriate prompt.

**Source:** `pkg/spawn/context.go:451-456, 658-665`

**Significance:** Seamless integration with existing spawn infrastructure. Worker spawns are unaffected.

---

## Synthesis

**Key Insights:**

1. **Clean separation via IsOrchestrator flag** - Single boolean controls routing decision; no skill-type string parsing needed at this layer.

2. **Template mirrors worker structure with key differences** - Same progressive disclosure pattern but with orchestrator-specific content (session goal, orch session end, SESSION_HANDOFF.md).

3. **Marker file enables completion verification** - `.orchestrator` marker file written to workspace allows `orch complete` to detect orchestrator spawns and apply appropriate verification.

**Answer to Investigation Question:**

The ORCHESTRATOR_CONTEXT.md template is structured around session goal focus rather than task focus. It:
- Opens with session goal and skill metadata
- Explains the spawned orchestrator concept (different from interactive)
- Provides authority/escalation guidance appropriate for orchestrators
- Requires SESSION_HANDOFF.md as completion artifact
- Uses `orch session end` instead of `/exit`
- Explicitly warns against worker patterns (bd comment, /exit, Phase: Complete)

---

## Structured Uncertainty

**What's tested:**

- ✅ GenerateOrchestratorContext produces expected content (verified: TestGenerateOrchestratorContext)
- ✅ WriteContext routes to orchestrator template when IsOrchestrator=true (verified: TestWriteContext_RoutesToOrchestrator)
- ✅ Worker spawns are unaffected (verified: TestWriteContext_WorkerDoesNotRouteToOrchestrator)
- ✅ MinimalPrompt routes correctly (verified: TestMinimalPrompt_RoutesToOrchestrator)
- ✅ All spawn package tests pass (verified: go test ./pkg/spawn/...)

**What's untested:**

- ⚠️ Integration with spawn_cmd.go (Phase 1 handles detection, this phase handles template)
- ⚠️ orch complete verification for orchestrator spawns (Phase 3 scope)
- ⚠️ End-to-end spawning of orchestrator session (requires all phases)

**What would change this:**

- If orchestrators need beads tracking after all, template would need beads instructions
- If SESSION_HANDOFF.md structure doesn't work well, template guidance may need updates
- If orch session end protocol changes, template needs corresponding update

---

## Implementation Recommendations

### Recommended Approach ⭐

**Template routing via IsOrchestrator** - Check single boolean in WriteContext(), generate appropriate template.

**Why this approach:**
- Minimal code change (3 lines in WriteContext, 4 lines in MinimalPrompt)
- Clean separation of concerns (routing logic vs template content)
- Easy to test in isolation

**Trade-offs accepted:**
- Config.IsOrchestrator must be set by caller (spawn_cmd.go in Phase 1)
- Two separate template files (context.go and orchestrator_context.go) vs merged template

**Implementation sequence:**
1. ✅ Add SessionGoal field to Config
2. ✅ Create orchestrator_context.go with template and helpers
3. ✅ Modify context.go WriteContext/MinimalPrompt for routing
4. ✅ Write comprehensive tests
5. ✅ Verify no regressions

---

## References

**Files Created/Modified:**
- `pkg/spawn/config.go:132-144` - Added SessionGoal field, updated IsOrchestrator comment
- `pkg/spawn/orchestrator_context.go` - New file with template and functions
- `pkg/spawn/context.go:451-456, 658-665` - Routing logic
- `pkg/spawn/orchestrator_context_test.go` - New test file

**Commands Run:**
```bash
# Run orchestrator tests
go test ./pkg/spawn/... -v -run "Orchestrator"
# PASS: all 10 tests

# Run all spawn tests
go test ./pkg/spawn/...
# ok github.com/dylan-conlin/orch-go/pkg/spawn 0.039s
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md` - Design investigation

---

## Investigation History

**2026-01-04 10:00:** Implementation started
- Task: Phase 2 of spawnable orchestrator sessions
- Context: Building on infrastructure investigation findings

**2026-01-04 10:30:** Implementation complete
- Created orchestrator_context.go with full template
- Added routing in context.go
- All tests passing
- Status: Complete
- Key outcome: ORCHESTRATOR_CONTEXT.md template ready for use
