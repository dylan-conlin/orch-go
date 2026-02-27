# Session Synthesis

**Agent:** og-debug-cross-repo-claude-26feb-b39d
**Issue:** orch-go-nw73
**Outcome:** success

---

## Plain-Language Summary

Cross-repo Claude-mode agents (e.g., spawned from orch-go to work in kb-cli) couldn't report Phase: Complete because `bd comment` couldn't find the beads issue. The beads issue lives in orch-go's `.beads/` database, but the agent runs `bd` from kb-cli's directory, where the issue doesn't exist. The fix injects `BEADS_DIR` environment variable into the Claude CLI launch command for cross-repo spawns, transparently redirecting all `bd` commands to the correct beads database. This is the same proven pattern used for `CLAUDE_CONFIG_DIR` account isolation.

## TLDR

Cross-repo claude-mode agents failed Phase: Complete reporting because `bd comment` silently fails when run from a different project directory. Fixed by injecting `BEADS_DIR` env var into the Claude launch command when the agent's working directory differs from the beads issue's project.

---

## Delta (What Changed)

### Files Modified
- `pkg/spawn/config.go` - Added `BeadsDir` field to Config struct
- `pkg/spawn/claude.go` - BuildClaudeLaunchCommand now accepts and injects `BEADS_DIR`; SpawnClaude passes cfg.BeadsDir
- `pkg/spawn/claude_test.go` - Added 3 test cases for BEADS_DIR injection (cross-repo, empty, combined with configDir)
- `pkg/orch/extraction.go` - Added `BeadsDir` to SpawnContext, wired through BuildSpawnConfig
- `cmd/orch/spawn_cmd.go` - Added cross-repo detection in step 4b: compares source vs target .beads/ paths

---

## Evidence (What Was Observed)

- `bd show orch-go-9v7c` from kb-cli directory: "no issue found" — confirms root cause
- `BEADS_DIR=/path/to/.beads bd comment orch-go-nw73 "test"` from kb-cli: succeeds — confirms fix mechanism
- Both failed agents (orch-go-9v7c, orch-go-1266) had beads IDs prefixed with "orch-go" but worked in different project directories

### Root Cause Chain
1. `orch spawn --workdir ~/kb-cli` creates beads issue in orch-go's .beads/ database
2. Claude agent is launched in kb-cli directory via tmux
3. Agent tries `bd comment orch-go-XXXX "Phase: Planning..."` from kb-cli
4. `bd` looks for .beads/ in current directory (kb-cli), doesn't find the orch-go issue
5. `bd comment` fails silently or with error
6. Agent never successfully reports Phase: Planning or Phase: Complete
7. Orchestrator sees no phase comments → agent appears stuck/dead

### Tests Run
```bash
go test ./pkg/spawn/ -run TestBuildClaudeLaunchCommand -v -count=1
# PASS: all 14 test cases (including 3 new cross-repo tests)

go test ./pkg/spawn/ -count=1
# PASS: all spawn package tests

go test ./cmd/orch/ -count=1 -timeout 60s
# PASS: all cmd/orch tests

go build ./cmd/orch/
# SUCCESS: no errors

go vet ./cmd/orch/
# SUCCESS: no issues
```

---

## Architectural Choices

### BEADS_DIR env var vs --db flag in spawn context template
- **What I chose:** Environment variable injection in Claude launch command
- **What I rejected:** Modifying all `bd comment {{.BeadsID}}` references in spawn context and skill templates to use `bd --db /path comment`
- **Why:** Env var is transparent — it affects ALL `bd` commands the agent runs, not just the ones explicitly in the template. Agent doesn't need to know about cross-repo mechanics. Same proven pattern as `CLAUDE_CONFIG_DIR` for account isolation.
- **Risk accepted:** If the agent somehow clears environment variables, bd would revert to local .beads/. This is extremely unlikely in practice.

---

## Verification Contract

See: `VERIFICATION_SPEC.yaml`

Key outcomes:
- BuildClaudeLaunchCommand correctly injects BEADS_DIR for cross-repo spawns
- BEADS_DIR env var enables cross-project bd operations (manually verified)
- All existing tests continue to pass (no regressions)

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `bd` commands are project-scoped by default — they use `.beads/` in the current working directory
- `BEADS_DIR` env var overrides the beads directory discovery, enabling cross-project operations
- `BD_DB` env var only works within a project that already has `.beads/` — it doesn't work from /tmp

### Decisions Made
- Use `BEADS_DIR` over `BD_DB` because it works more reliably across directories
- Detection logic: compare `filepath.Join(sourceDir, ".beads")` vs `filepath.Join(projectDir, ".beads")` when `spawnWorkdir` is set

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (14/14 BuildClaudeLaunchCommand, all spawn + cmd/orch tests)
- [x] Build succeeds
- [x] Installed to ~/bin/orch
- [x] SYNTHESIS.md created
- [x] VERIFICATION_SPEC.yaml created

No discovered work — this was a targeted fix for a specific cross-repo beads communication failure.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-cross-repo-claude-26feb-b39d/`
**Beads:** `bd show orch-go-nw73`
