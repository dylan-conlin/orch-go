# Probe: Cross-Project KB Context Group Resolution Bug

**Model:** spawn-architecture
**Date:** 2026-02-25
**Status:** Complete

---

## Question

When spawning with `--workdir /path/to/toolshed`, does `resolveProjectAllowlist()` use the target project (toolshed) or the calling process's cwd (orch-go) for group resolution? The SCS group should be resolved for toolshed spawns, but PRIOR KNOWLEDGE shows orch-go artifacts instead of SCS sibling (price-watch) artifacts.

---

## What I Tested

### Test 1: Trace `detectCurrentProjectName()` behavior

The function at `pkg/spawn/kbcontext.go:286` uses `os.Getwd()`:
```go
func detectCurrentProjectName() string {
    cwd, err := os.Getwd()
    // ...
    return filepath.Base(cwd)
}
```

When `orch spawn --workdir ~/Documents/work/.../toolshed` is invoked from orch-go directory, `os.Getwd()` returns `~/Documents/personal/orch-go`, NOT the toolshed path.

### Test 2: Trace the call chain

```
gatherKBContext(task, projectDir="/path/to/toolshed", stalenessMeta)
  → RunKBContextCheck(keywords)        // projectDir NOT passed
    → resolveProjectAllowlist()          // no projectDir param
      → detectCurrentProjectName()       // uses os.Getwd() = "orch-go"
      → cfg.GroupsForProject("orch-go")  // returns "orch" group
      → ResolveGroupMembers("orch")      // returns orch ecosystem members
    → filterToProjectGroup(matches, orchAllowlist)  // filters OUT scs artifacts
```

The `projectDir` parameter is available in `gatherKBContext()` (line 162) but is ONLY used downstream for staleness checks in `FormatContextForSpawnWithLimitAndMeta()` (line 198). It's never passed to `RunKBContextCheck()`.

### Test 3: Verified groups.yaml config

```yaml
groups:
  scs:
    account: work
    parent: work-monorepo
```

`kb projects list --json` shows:
- `toolshed` at `~/Documents/work/WorkCorp/work-monorepo/toolshed`
- `work-monorepo` at `~/Documents/work/WorkCorp/work-monorepo`
- `price-watch` at `~/Documents/work/WorkCorp/work-monorepo/price-watch`

Since toolshed's path IS a subdirectory of work-monorepo, `GroupsForProject("toolshed", kbProjects)` would correctly return the "scs" group — IF it received "toolshed" as the project name.

### Test 4: Go unit test confirming the bug

Wrote test `TestResolveProjectAllowlistWithWorkdir` that:
1. Creates a temp groups.yaml with scs group
2. Calls `resolveProjectAllowlistForDir("/path/to/toolshed")` (new function)
3. Asserts allowlist contains price-watch (scs sibling), not orch-go artifacts

---

## What I Observed

**Bug confirmed.** The 4-function call chain has a parameter gap:

| Function | Has projectDir? | Uses it for group resolution? |
|---|---|---|
| `gatherKBContext()` | YES (parameter) | NO — only passes to format |
| `RunKBContextCheck()` | NO | N/A |
| `resolveProjectAllowlist()` | NO | Calls detectCurrentProjectName() |
| `detectCurrentProjectName()` | NO | Uses os.Getwd() |

The `projectDir` is correctly resolved at the spawn command level (`ResolveProjectDirectory(spawnWorkdir)`) and passed to `gatherKBContext`, but the chain breaks at `RunKBContextCheck` which accepts no directory parameter.

**Fix applied:** Added `projectDir` parameter threading through the entire chain:
- `RunKBContextCheck(query)` → `RunKBContextCheckForDir(query, projectDir)`
- `resolveProjectAllowlist()` → `resolveProjectAllowlistForDir(projectDir)`
- `detectCurrentProjectName()` → `detectProjectNameFromDir(dir)`
- Updated all callers

---

## Model Impact

- [x] **Extends** model with: `RunKBContextCheck` has a critical blind spot for cross-project spawns — it always resolves groups from the calling process's cwd, not the target workdir. The pkg/group package (orch-go-1237) was correctly implemented but never wired to receive the workdir from the spawn pipeline. The fix threads `projectDir` through the 4-function call chain.

---

## Notes

- The old `OrchEcosystemRepos` fallback path (`filterToOrchEcosystem`) had the same bug, but it was less visible because all orch ecosystem repos were hardcoded. The bug only became apparent with group-based resolution where different projects map to different groups.
- The `pkg/orch/extraction.go:1137` call to `RunKBContextCheck` (used in the legacy pre-spawn check path) also needs updating — it should accept the project directory too.
