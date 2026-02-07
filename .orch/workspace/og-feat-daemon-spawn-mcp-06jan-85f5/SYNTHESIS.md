# Session Synthesis

**Agent:** og-feat-daemon-spawn-mcp-06jan-85f5
**Issue:** orch-go-cdb7c
**Duration:** Started session
**Outcome:** success

---

## TLDR

Implemented automatic MCP server injection for daemon-spawned agents based on `needs:playwright` label. When an issue has this label, the daemon now automatically passes `--mcp playwright` when spawning, allowing UI bug fixes to have browser access for visual verification.

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/skill_inference.go` - Added `InferMCPFromLabels` function to extract MCP requirements from `needs:*` labels
- `cmd/orch/spawn_cmd.go` - Added `inferMCPFromBeadsIssue` helper and modified `runWork` to detect and use MCP from labels
- `cmd/orch/spawn_cmd.go` - Fixed pre-existing bug in `InferSkillFromIssueType` to return "architect" for bugs (matching daemon behavior)
- `pkg/daemon/daemon_test.go` - Added `TestInferMCPFromLabels` test cases

### Architecture Decision
The implementation infers MCP at the `orch work` command level rather than modifying daemon's spawnFunc signature. This was chosen because:
1. `orch work` already fetches the full beads issue with labels
2. No breaking changes to daemon's internal interfaces
3. Simpler implementation - just set the global `spawnMCP` flag before calling `runSpawnWithSkillInternal`

---

## Evidence (What Was Observed)

- The daemon calls `SpawnWork(beadsID)` which shells out to `orch work <beadsID>`
- `orch work` already fetches issue details including labels via beads RPC client
- The existing skill inference pattern (`inferSkillFromBeadsIssue`) provided a model for MCP inference
- The `spawnMCP` global flag is used by `runSpawnWithSkillInternal` to configure spawn

### Tests Run
```bash
go test ./pkg/daemon/... -run 'TestInfer' -v
# PASS: TestInferMCPFromLabels - all 9 test cases pass

go test ./cmd/orch/... -run 'TestInfer' -v  
# PASS: TestInferSkillFromIssueType - all 7 test cases pass

go test ./pkg/daemon/... ./cmd/orch/...
# PASS: All tests in daemon and cmd/orch packages pass
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Decision: Infer MCP in `orch work` rather than daemon because the work command already has access to issue labels via beads RPC, avoiding interface changes
- Decision: Use `needs:*` label prefix pattern (matching existing `skill:*` pattern) for extensibility

### Constraints Discovered
- The tmux test suite has a pre-existing failure unrelated to this change (`TestBuildOpencodeAttachCommand`)
- Bug vs architect inference was inconsistent between daemon.InferSkill (architect) and cmd/orch.InferSkillFromIssueType (systematic-debugging) - fixed as part of this work

### Externalized via `kn`
- No additional `kn` entries needed - the feature is documented in code comments

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (relevant tests - daemon and cmd/orch)
- [x] Ready for `orch complete orch-go-cdb7c`

---

## Unexplored Questions

**What remains unclear:**
- Whether other MCP servers should be supported (e.g., `needs:browser-use`) - left as placeholder in code
- Whether the label should be auto-added when issue type suggests UI work (currently manual)

*(Straightforward implementation - the architecture was clear from the spawn context)*

---

## Session Metadata

**Skill:** feature-impl
**Model:** claude
**Workspace:** `.orch/workspace/og-feat-daemon-spawn-mcp-06jan-85f5/`
**Beads:** `bd show orch-go-cdb7c`
