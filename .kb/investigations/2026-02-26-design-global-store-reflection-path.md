# Design: ~/.kb/ Global Store Reflection Path for kb reflect

**Date:** 2026-02-26
**Status:** Active
**Type:** Architect design
**Beads:** orch-go-u6eg
**Promote to Decision:** recommend-yes
**Authority:** architectural

---

## Problem Statement

`kb reflect` has a structural blind spot: the global knowledge store at `~/.kb/` is **never scanned** by any reflection mode. This store contains 89 artifacts including 15 decisions, 7 investigations, 6 models, and 8 guides — plus the master `principles.md` (46 KB) and the project registry.

### Why It's Invisible

1. **Default mode** (`kb reflect`): calls `Reflect(projectDir)` → `findKBDir(projectDir)` → looks for `{projectDir}/.kb/`. The global store lives at `~/.kb/`, which is not `{anything}/.kb/`.

2. **Global mode** (`kb reflect --global`): calls `ReflectGlobal()` → `discoverProjects()` → scans `~/.kb/projects.json` and filesystem for repos containing `.kb/` directories. `~/.kb/` itself is not _inside_ a project — it IS the global store. No code path reads it.

3. **Structural asymmetry**: `~/.kb/` symlinks to `~/orch-knowledge/kb/` (note: `kb/`, not `.kb/`). The discovery logic looks for `.kb/` directories inside project roots. `orch-knowledge` does have `~/orch-knowledge/.kb/` (584 files, project-level), but the global `~/orch-knowledge/kb/` (89 files) is a separate, sibling directory.

### Impact (from probe evidence)

Per the cross-repo visibility probe (2026-02-26):
- **59.5%** of all knowledge artifacts are invisible in default mode
- **2.7%** (89 artifacts in `~/.kb/`) are invisible even in `--global` mode
- Global decisions (15) never checked for staleness
- Global investigations (7) never checked for synthesis or open actions
- Cross-repo citations to global decisions are never detected

---

## Design Options

### Option A: Treat `~/.kb/` as a Virtual Project in `ReflectGlobal()`

Add `~/.kb/` to the project list in `ReflectGlobal()` by creating a new function `reflectKBDir(kbDir, opts)` that accepts a kbDir directly (bypassing `findKBDir(projectDir)` resolution).

**Changes:**
1. Extract `reflectKBDir(kbDir string, opts ReflectOptions) (ReflectResult, error)` from `Reflect()` — same logic, but takes kbDir directly instead of resolving from projectDir
2. `Reflect(projectDir, opts)` becomes: `kbDir := findKBDir(projectDir); return reflectKBDir(kbDir, opts)`
3. `ReflectGlobal()` adds `~/.kb/` as an additional kbDir: `reflectKBDir(globalKBDir, opts)` merged into combined results

**Pros:** Minimal refactor, `~/.kb/` gets all existing reflection types for free
**Cons:** Some reflection types don't make sense for the global store (drift checks code-related constraints — `~/.kb/` has no code). Functions that need `projectDir` (for beads issue creation) need a fallback.

### Option B: Dedicated `reflectGlobalStore()` Function

Create a purpose-built function that runs only the reflection types meaningful for `~/.kb/`:
- **stale**: Check 15 global decisions for cross-project citations
- **synthesis**: Cluster 7 global investigations
- **open**: Check 7 global investigations for unimplemented Next: actions
- **investigation-promotion**: Check for recommend-yes investigations
- **investigation-authority**: Group by authority level

Skip types that need project context (drift, promote, skill-candidate, defect-class).

**Pros:** Clean separation, no false positives from inapplicable types
**Cons:** Duplicates logic, must be maintained in parallel with `Reflect()`

### Option C: Include `~` as a Project in `discoverProjects()`

Make `discoverProjects()` always include the home directory if `~/.kb/` exists.

**Pros:** Zero changes to reflection logic — just discovery
**Cons:** `~` is not a real project. Issue creation with `bd create` in home directory would fail. `projectDir` semantics break down.

---

## Recommended Approach: Option A (Virtual Project) with Type Filtering

Option A with a small enhancement: add an `isGlobalStore bool` parameter to `reflectKBDir` that skips inapplicable types and adjusts behavior.

### Architecture

```
kb reflect --global
    │
    ├── discoverProjects()          ← existing project-level discovery
    │   └── for each project:
    │       └── reflectKBDir(project/.kb/, opts, isGlobal=false)
    │
    └── reflectGlobalStore()        ← NEW: global store inclusion
        └── reflectKBDir(~/.kb/, opts, isGlobal=true)
            ├── synthesis    ✅ (7 investigations)
            ├── stale        ✅ (15 decisions) + cross-project citations
            ├── open         ✅ (7 investigations)
            ├── inv-promotion ✅ (7 investigations)
            ├── inv-authority ✅ (7 investigations)
            ├── promote      ⬚ skip (no code project to promote into)
            ├── drift        ⬚ skip (no code to check against)
            ├── skill-candidate ⬚ skip (kn is global, already handled)
            ├── refine       ⬚ skip (already reads ~/.kb/principles.md)
            └── defect-class ⬚ skip (no code project context)
```

### Implementation Details

#### 1. Extract `reflectKBDir()` from `Reflect()`

```go
// reflectKBDir runs reflection on a specific kbDir.
// projectDir is the parent project (empty for global store).
// isGlobalStore controls which types are run.
func reflectKBDir(kbDir string, projectDir string, opts ReflectOptions) (ReflectResult, error) {
    isGlobalStore := projectDir == ""

    // ... existing type dispatch, but:
    // - skip drift, promote, skill-candidate, defect-class when isGlobalStore
    // - for issue creation, use projectDir if available, else skip auto-creation
}

func Reflect(projectDir string, opts ReflectOptions) (ReflectResult, error) {
    kbDir, err := findKBDir(projectDir)
    if err != nil {
        return ReflectResult{}, err
    }
    return reflectKBDir(kbDir, projectDir, opts)
}
```

#### 2. Add Global Store Inclusion in `ReflectGlobal()`

```go
func ReflectGlobal(opts ReflectOptions) (ReflectResult, error) {
    projects := discoverProjects()

    // ... existing project iteration ...

    // Include global store
    homeDir, _ := os.UserHomeDir()
    globalKBDir := filepath.Join(homeDir, ".kb")
    if _, err := os.Stat(globalKBDir); err == nil {
        globalResults, err := reflectKBDir(globalKBDir, "", opts)
        if err == nil {
            // Merge with combined results (same dedup logic)
            mergeResults(&combined, globalResults, seen, seenStale)
        }
    }

    return combined, nil
}
```

#### 3. Cross-Project Citation Check for Global Stale Detection

The most valuable enhancement: when checking if global decisions are stale, search for citations across **all** discovered projects.

Current `findStaleCandidates(kbDir)` only checks citations within the same kbDir. For global decisions, citations live in project-level investigations.

```go
func findStaleCandidates(kbDir string, limit int) ([]StaleCandidate, error) {
    // existing logic for decisions in kbDir...
}

// NEW: enhanced version for global store
func findStaleCandidatesWithCrossProject(kbDir string, projectDirs []string, limit int) ([]StaleCandidate, error) {
    // 1. Find decisions in kbDir (same as before)
    // 2. For each decision, search citations in:
    //    a. kbDir/investigations/ (same as before)
    //    b. ALSO: each projectDir/.kb/investigations/ (new)
    // 3. Decision is stale only if ZERO citations across all locations
}
```

This is critical because global decisions (e.g., "accountability-architecture-beads-first") are almost exclusively cited from project-level investigations, not from `~/.kb/investigations/`.

#### 4. Output Annotation

Global store results should be distinguishable in output:

```json
{
  "stale": [
    {
      "path": "~/.kb/decisions/2026-01-04-orchestrator-vs-multi-agent.md",
      "age_days": 53,
      "source": "global",
      "suggestion": "Global decision with no citations across 18 projects"
    }
  ]
}
```

Add a `Source string` field to result types to distinguish global vs project-level findings.

---

## Cross-Citation Architecture

The most significant design question is how to handle cross-store citations efficiently.

### Current: Single-Store Citation Check

```
findStaleCandidates(kbDir)
    ├── list decisions in kbDir/decisions/
    └── grep for citations in kbDir/investigations/
```

### Proposed: Multi-Store Citation Check (Global Store Only)

```
findStaleCandidatesWithCrossProject(~/.kb/, projectDirs)
    ├── list decisions in ~/.kb/decisions/
    └── for each decision:
        ├── grep ~/.kb/investigations/         (local)
        └── for each projectDir:               (cross-project)
            └── grep projectDir/.kb/investigations/
```

**Performance concern:** 15 decisions × 18 projects × file walks. Mitigations:
- Only for global store (project-level stale detection unchanged)
- File walk already happens during project reflection — cache investigation file lists
- 15 decisions is small; grep is fast
- Run only when `--global` flag is set (already the case)

**Alternative: Grep-based citation check.** Instead of walking investigation files, use a single `grep -rl "decision-filename" projectDir/.kb/` per decision. Faster than opening and parsing each file.

---

## What This Does NOT Change

1. **Default mode** (`kb reflect` without `--global`): unchanged, still project-scoped
2. **Project-level reflection**: unchanged, all 10 types still work as before
3. **Daemon integration**: `pkg/daemon/reflect.go` already passes `--global` — global store results will automatically flow through
4. **Issue creation**: Global store findings skip auto-issue creation (no project context for `bd create`)

---

## Implementation Guidance

### Phase 1: Core Refactor (kb-cli)

1. Extract `reflectKBDir(kbDir, projectDir string, opts)` from `Reflect()`
2. `Reflect()` becomes a thin wrapper: resolve kbDir, call `reflectKBDir()`
3. Add `isGlobalStore` guard that skips inapplicable types
4. Add `Source` field to result types

### Phase 2: Global Store Inclusion (kb-cli)

1. In `ReflectGlobal()`, include `~/.kb/` after project iteration
2. Merge global results with existing dedup logic
3. Test with `kb reflect --global --format json` — verify global artifacts appear

### Phase 3: Cross-Project Citations (kb-cli)

1. Add `findStaleCandidatesWithCrossProject()` for global store
2. When checking global decisions, search citations across all projects
3. This is the highest-value enhancement — global decisions that nobody cites are genuinely stale

### Phase 4: orch-go Consumption (orch-go)

1. No changes needed in `pkg/daemon/reflect.go` — it already parses all types
2. `Source` field in suggestions will flow through naturally (additive JSON field)
3. Session start hook will show global reflection items automatically

---

## Edge Cases

### `~/.kb/` Is a Symlink

`~/.kb/` → `~/orch-knowledge/kb/`. The code should follow the symlink (Go's `os.Stat` follows by default). The resolved path should be used for dedup to avoid double-counting if `orch-knowledge` is also discovered as a project.

### `orch-knowledge` Has Both `.kb/` and `kb/`

`~/orch-knowledge/.kb/` (project-level, 584 files) and `~/orch-knowledge/kb/` (global, 89 files) are siblings. `discoverProjects()` discovers `orch-knowledge` via its `.kb/` directory. The global store `~/orch-knowledge/kb/` (accessed via `~/.kb/` symlink) must be treated as a separate artifact source, not confused with the project-level `.kb/`.

**Dedup rule:** Resolve `~/.kb/` to its real path. If the real path is inside a discovered project's directory tree but NOT the project's `.kb/` directory, treat it as the global store (not a duplicate of the project).

### Issue Auto-Creation

`createIssue` requires a `projectDir` for `bd create` to work (beads is project-scoped). For global store results:
- **Skip auto-creation** — global findings should be surfaced to the orchestrator who decides which project to route them to
- Add a `"auto_issue": false, "reason": "global store — route manually"` annotation

---

## Risks

1. **Performance**: Cross-project citation check adds N project walks. Mitigated by grep-based approach and small decision count (15).
2. **False positives on stale**: A global decision could be actively used in conversations/principles without being cited in any investigation file. The "stale" signal is weaker for foundational decisions.
3. **Maintenance**: `reflectKBDir` must be updated when new reflection types are added. Mitigated by having a single function (not two parallel implementations as in Option B).

---

## Success Criteria

1. `kb reflect --global --format json` includes `~/.kb/` artifacts in results
2. Global decisions appear in stale analysis with cross-project citation checking
3. Global investigations appear in synthesis, open, and investigation-promotion analysis
4. Global results are annotated with `"source": "global"` for distinguishability
5. No regression in project-level reflection behavior
6. Performance: `kb reflect --global` completes within 5 seconds (current baseline ~2s)

---

## Next

Implement Phase 1-3 in kb-cli as a `feature-impl` task. Phase 4 (orch-go consumption) requires no code changes — the daemon already passes `--global` and parses all types.
