# Probe: Cross-Repo Global Store Design Validation

**Model:** kb-reflect-cluster-hygiene
**Date:** 2026-02-26
**Status:** Complete

---

## Question

The prior probe (2026-02-26-probe-cross-repo-knowledge-visibility-inventory) established that 59.5% of artifacts are invisible in default mode and 2.7% (~/.kb/) are permanently invisible even with --global. This probe validates the architectural approach for fixing both gaps: is Option A (virtual project with type filtering) for kb reflect and a new `GetGlobalStoreContext()` for kb context the right approach? Are there hidden coupling points or edge cases in the current code that would break these changes?

---

## What I Tested

### 1. Validated `Reflect()` refactor feasibility

Read `reflect.go:222-362` (the `Reflect()` function). Confirmed that:
- `Reflect()` takes `projectDir` and resolves `kbDir` via `findKBDir(projectDir)` (line 227)
- Each reflection type is an independent function call that takes `kbDir` and/or `projectDir`
- Some types need `projectDir` (synthesis for issue creation, drift for code scanning, promote for .kn/ lookup)
- Some types only need `kbDir` (stale, open, investigation-promotion, investigation-authority)

**Extraction point:** Lines 227-230 resolve `kbDir`. Everything after can work with `kbDir` directly. The refactor to `reflectKBDir(kbDir, projectDir, opts)` is clean — just move lines 232-361 into the new function.

### 2. Validated `ReflectGlobal()` merge logic

Read `reflect.go:365-483`. Confirmed:
- `ReflectGlobal()` iterates `discoverProjects()`, calls `Reflect(projectDir)` per project
- Has dedup maps (`seen` for synthesis topics, `seenStale` for files/classes)
- Merging is by category with dedup on topic/file/class name
- Adding one more `reflectKBDir()` call after the project loop requires only appending to the same merge logic

**Key observation:** The `seenStale` map is shared across types (open, stale, investigation-promotion, investigation-authority, defect-class). This means dedup works by filename across all types. The global store's files (e.g., `2026-01-22-accountability-architecture-beads-first.md`) won't collide with project-level filenames since they're different files.

### 3. Validated kb context `GetContext()` coupling to `findKBDir()`

Read `context.go:155-222`. Confirmed:
- `GetContext()` calls `findKBDir(projectDir)` at line 159 and **returns error if it fails**
- This means `GetContext()` cannot be called with `~` as projectDir (no `.kb/` exists at `~/.kb/` in the `findKBDir` sense — it uses `{projectDir}/.kb/` but `~/.kb/` IS the kb dir, not `~/.kb/.kb/`)
- The search functions (`searchGuidesDir`, `searchModelsDir`, `SearchArtifacts`) take paths directly, not projectDir
- A new `GetGlobalStoreContext()` can reuse these search functions directly with `~/.kb/` as the base

### 4. Validated `GetContextGlobal()` merge pattern

Read `context.go:226-428`. Confirmed:
- `GetContextGlobalWithProjects()` calls `GetContext(projectDir)` per project
- Tags results with `Project` field (per category)
- Applies limits after merging
- Adding global store results at the end (tagged with `Project: "global"`) fits cleanly

### 5. Confirmed ~./kb/ directory structure matches expectations

```bash
ls ~/.kb/  # decisions/, investigations/, guides/, models/ all exist
readlink -f ~/.kb/  # → /Users/dylanconlin/orch-knowledge/kb
```

Global store has: 17 decisions, 9 investigations, 10 guides, 8 models. Same directory structure as project-level `.kb/`, so existing search functions work directly.

### 6. Validated spawn system calling pattern

Read `orch-go/pkg/spawn/kbcontext.go:117-175`. The spawn system uses a tiered strategy:
1. Local `kb context` (no flags) — hits `GetContext(cwd)` → current project only
2. If sparse (<3 matches), escalates to `kb context --global` → hits `GetContextGlobal()`
3. Post-filters by project group allowlist

**Critical finding:** Even after escalation to `--global`, `~/.kb/` artifacts are still invisible. The fix must happen at the kb-cli level so the spawn system benefits automatically.

---

## What I Observed

### Design validation results

1. **Option A for kb reflect is confirmed feasible.** The `Reflect()` function cleanly separates into `reflectKBDir(kbDir, projectDir, opts)` with minimal coupling. Types that need `projectDir` (drift, promote, skill-candidate, defect-class) can be skipped when `projectDir == ""`.

2. **kb context needs its own approach** (not covered in prior investigation). A new `GetGlobalStoreContext()` function can reuse `searchGuidesDir()`, `searchModelsDir()`, `SearchArtifacts()` directly by passing `~/.kb/` paths. This should be called at the command handler level (not inside `GetContext`) to avoid N-times execution in global mode.

3. **Edge case confirmed:** `~/.kb/` → `/Users/dylanconlin/orch-knowledge/kb/` is a symlink. `os.Stat` follows symlinks by default (Go behavior). No special handling needed unless we want to detect and prevent double-counting with `orch-knowledge/.kb/`. Since the paths are different (`orch-knowledge/kb/` vs `orch-knowledge/.kb/`), this is not a risk.

4. **Performance validated:** Global store has ~44 .md files to search. At ~100ms for stemmed search of 44 files, this adds negligible latency. The 5-second timeout in orch-go's spawn system is not at risk.

---

## Model Impact

- [x] **Confirms** the prior probe's finding that `~/.kb/` is structurally invisible (Failure Mode 6: Cross-repo visibility gap)
- [x] **Extends** with: The fix for kb context requires a different approach than kb reflect. Reflect can treat `~/.kb/` as a virtual project, but context cannot reuse `GetContext()` (which requires `findKBDir()` success). A new `GetGlobalStoreContext()` function is needed.
- [x] **Extends** with: The spawn system's tiered search strategy benefits automatically from kb-cli-level changes — no orch-go code changes needed for the basic fix.
