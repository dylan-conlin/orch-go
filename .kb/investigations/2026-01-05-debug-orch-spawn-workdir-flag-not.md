<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `GenerateWorkspaceName` was hardcoded to use `og-` prefix instead of deriving prefix from project name.

**Evidence:** Code inspection showed line 185 in config.go: `return fmt.Sprintf("og-%s-%s-%s", prefix, slug, date)` - no project parameter.

**Knowledge:** The `--workdir` flag correctly set `projectDir` and `projectName`, but workspace naming was disconnected from project.

**Next:** Fix implemented and tested. Ready for orch complete.

---

# Investigation: Orch Spawn Workdir Flag Not Respected

**Question:** Why does `orch spawn --workdir /path/to/project` produce workspace names with `og-` prefix instead of the target project's prefix?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: --workdir flag correctly sets projectDir

**Evidence:** In `spawn_cmd.go:531-551`, the `spawnWorkdir` variable is correctly resolved to an absolute path and validated:
```go
if spawnWorkdir != "" {
    projectDir, err = filepath.Abs(spawnWorkdir)
    // ... validation ...
}
```

**Source:** `cmd/orch/spawn_cmd.go:531-551`

**Significance:** The --workdir flag IS being respected for the working directory. The bug is elsewhere.

---

### Finding 2: projectName is derived from projectDir

**Evidence:** At line 554: `projectName := filepath.Base(projectDir)`. This correctly extracts the project name from the directory path.

**Source:** `cmd/orch/spawn_cmd.go:554`

**Significance:** The project name IS available and correctly derived. The issue is that it's not being used in workspace naming.

---

### Finding 3: GenerateWorkspaceName uses hardcoded `og-` prefix

**Evidence:** The function signature was `GenerateWorkspaceName(skillName, task string)` and returned:
```go
return fmt.Sprintf("og-%s-%s-%s", prefix, slug, date)
```

The `og-` prefix was hardcoded regardless of which project was being spawned for.

**Source:** `pkg/spawn/config.go:160-190`

**Significance:** This is the root cause. The workspace name was generated using only skill and task, not the project name.

---

### Finding 4: The call site doesn't pass project name

**Evidence:** At line 562: `workspaceName := spawn.GenerateWorkspaceName(skillName, task)` - no projectName parameter.

**Source:** `cmd/orch/spawn_cmd.go:562`

**Significance:** Even though projectName was available at the call site, it wasn't being passed to the workspace name generator.

---

## Synthesis

**Key Insights:**

1. **Symptom vs Root Cause** - The symptom was "agents spawn in wrong directory" but the actual issue was workspace naming, not directory setting.

2. **Hardcoded prefix was a shortcut** - The original `og-` prefix was likely added when orch-go was the only project using this system. It was never generalized.

3. **Clean fix path** - The fix requires: (1) add project parameter to GenerateWorkspaceName, (2) add helper to derive project prefix, (3) update caller.

**Answer to Investigation Question:**

The `--workdir` flag was correctly setting the project directory and deriving the project name. However, `GenerateWorkspaceName()` was hardcoded to use `og-` prefix (orch-go) regardless of the actual project. The fix adds a `projectName` parameter and a `generateProjectPrefix()` helper function that creates appropriate prefixes:
- 2-part names (orch-go, price-watch) → first letter of each part (og, pw)
- 1-part names (myproject) → first two characters (my)
- 3+ part names → first two characters of each part

---

## Structured Uncertainty

**What's tested:**

- ✅ GenerateWorkspaceName now accepts projectName parameter (unit tests pass)
- ✅ generateProjectPrefix creates correct prefixes for various project name formats (unit tests pass)
- ✅ Build succeeds with updated function signature

**What's untested:**

- ⚠️ End-to-end spawn with --workdir flag (not run during this fix)
- ⚠️ Interaction with existing workspaces that have og- prefix

**What would change this:**

- If there are other callers of GenerateWorkspaceName that we didn't find
- If there's workspace name validation elsewhere that expects og- prefix

---

## Implementation Recommendations

**Purpose:** Fix has been implemented. No further recommendations needed.

### Implemented Approach ⭐

**Add project name to workspace naming** - Updated `GenerateWorkspaceName` to accept `projectName` parameter and derive prefix.

**Changes made:**
1. `pkg/spawn/config.go`: Added `projectName` parameter to `GenerateWorkspaceName`, added `generateProjectPrefix()` helper
2. `cmd/orch/spawn_cmd.go`: Updated call to pass `projectName`
3. `pkg/spawn/context_test.go`: Updated existing tests, added new tests for `generateProjectPrefix`

**Trade-offs accepted:**
- Existing workspaces with `og-` prefix will remain unchanged
- New workspace names will now have project-specific prefixes

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - Main spawn command implementation
- `pkg/spawn/config.go` - Workspace name generation
- `pkg/spawn/context_test.go` - Unit tests for workspace naming
- `pkg/tmux/tmux.go` - tmux window creation (verified it receives correct directory)

**Commands Run:**
```bash
# Build and test
go build ./...
go test ./pkg/spawn/... -run "TestGenerateWorkspaceName|TestGenerateProjectPrefix" -v
```

---

## Investigation History

**2026-01-05:** Investigation started
- Initial question: Why does --workdir produce og- prefix workspaces?
- Context: Orchestrator reported agents spawning in wrong directory for price-watch project

**2026-01-05:** Root cause identified
- Found hardcoded `og-` prefix in GenerateWorkspaceName
- Identified missing projectName parameter in function signature

**2026-01-05:** Fix implemented and tested
- Added projectName parameter to GenerateWorkspaceName
- Created generateProjectPrefix helper
- Updated tests
- Status: Complete
