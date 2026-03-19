# Probe: Cross-Repo Knowledge Visibility Inventory

**Model:** kb-reflect-cluster-hygiene
**Date:** 2026-02-26
**Status:** Complete

---

## Question

The kb-reflect-cluster-hygiene model describes how `kb reflect` clusters and triages investigations. But what percentage of the total knowledge base is invisible to `kb reflect` when run in its default (single-project) mode? The model's mechanisms assume all relevant investigations are visible — if a large fraction lives in other repos or global locations, the clustering, staleness, and synthesis signals are incomplete.

---

## What I Tested

### 1. Confirmed `findKBDir` is strictly project-scoped

```bash
# kb-cli/cmd/kb/create.go:799-805
func findKBDir(projectDir string) (string, error) {
    kbDir := filepath.Join(projectDir, ".kb")
    if _, err := os.Stat(kbDir); err == nil {
        return kbDir, nil
    }
    return "", fmt.Errorf("no .kb directory found in %s", projectDir)
}
```

**Result:** `findKBDir` only checks `{projectDir}/.kb/`. No fallback, no global scan.

### 2. Tested `--global` flag behavior

```bash
kb reflect --global --type synthesis --format json
```

**Result:** `--global` flag exists and uses `discoverProjects()` which:
- Checks registered projects from `~/.kb/projects.json` (18 registered)
- Scans `~/Documents`, `~/Projects`, `~/repos`, `~/src`, `~/code` up to 3 levels deep
- Does NOT scan `~/.kb/` (global) as it's not a `.kb/` directory inside a project

### 3. Verified daemon never uses `--global`

```bash
# pkg/daemon/reflect.go:130
args := []string{"reflect", "--format", "json"}
# No --global flag ever appended
```

**Result:** Daemon always runs `kb reflect` in single-project mode (orch-go only).

### 4. Inventoried all .kb/ locations

Enumerated every .kb/ directory across the system:

| Location | .md Files | Category | Age Range | Visible to orch-go `kb reflect`? |
|----------|-----------|----------|-----------|----------------------------------|
| **orch-go/.kb/** | 1,331 | project | Dec 2025 – Feb 2026 | **YES** (default) |
| **orch-knowledge/.kb/** | 584 | project | Dec 2025 – Feb 2026 | No (--global only) |
| **~/.kb/** (→ orch-knowledge/kb/) | 89 | global | Dec 2025 – Feb 2026 | **NEVER** (not a project .kb/) |
| **orch-cli/.kb/** | 221 | project (legacy) | Dec 2025 | No (--global only) |
| **price-watch/.kb/** | 711 | project (work) | Dec 2025 – Feb 2026 | No (--global only) |
| **kb-cli/.kb/** | 36 | project | Dec 2025 – Feb 2026 | No (--global only) |
| **beads-ui-svelte/.kb/** | 71 | project | Dec 2025 | No (--global only) |
| **skillc/.kb/** | 32 | project | Dec 2025 – Feb 2026 | No (--global only) |
| Other personal repos (11) | 111 | project | Dec 2025 – Feb 2026 | No (--global only) |
| Other work repos (6) | 101 | project (work) | Dec 2025 – Feb 2026 | No (--global only) |
| **TOTAL** | **3,287** | | | |

### 5. Identified blind spots

**A. `~/.kb/` (global) — 89 artifacts, NEVER scanned by any mode**

`~/.kb/` is a symlink to `~/orch-knowledge/kb/` (note: NOT `~/orch-knowledge/.kb/`). It contains:
- `principles.md` (46 KB — the master principles file)
- `values.md` (2.6 KB)
- 6 global models (signal-to-design-loop, meta-orchestrator, control-plane-bootstrap, verifiability-first, planning-as-decision-navigation, human-ai-interaction-frames)
- 15 global decisions
- 8 global guides
- 7 global investigations
- 11 templates
- 1 context file (org-journal.md)
- `projects.json` (project registry)
- `.principlec/` (37 principle source files)

`discoverProjects()` finds repos with `.kb/` directories. `~/.kb/` itself is not inside any project — it IS the global knowledge store. No code path in `kb reflect` ever reads it directly.

**B. 2 unregistered repos with .kb/ directories**

`badge-tracker` and `work-explorer` have `.kb/` directories but are not in `projects.json`. `discoverProjects()` would only find them if they're within 3 levels of `~/Documents` — at depth 4 under `~/Documents/work/WorkCorp/work-monorepo/`, they'd be missed by the directory walk too.

**C. Daemon never uses --global**

Even though `--global` exists and works, the daemon's reflect pipeline (which runs periodically) only reflects on the current project.

---

## What I Observed

### Visibility breakdown

```
orch-go/.kb/ (default mode):    1,331 artifacts (40.5%)
Other repos via --global:       1,867 artifacts (56.8%)
~/.kb/ (global, NEVER scanned):    89 artifacts ( 2.7%)
                                ─────
Total knowledge base:           3,287 artifacts
```

**59.5% of all knowledge artifacts are invisible to `kb reflect` in its default mode.**

Even with `--global`, 89 artifacts (2.7%) in `~/.kb/` are permanently invisible — including the master `principles.md`, 6 global models, and 8 global guides.

### Notable findings

1. **orch-knowledge has TWO knowledge stores**: `~/orch-knowledge/.kb/` (project-level, 584 files) and `~/orch-knowledge/kb/` (global, 89 files, symlinked as `~/.kb/`). Only the former is discoverable by `kb reflect --global`.

2. **price-watch has 711 artifacts** — more than half the size of orch-go's knowledge base. It has 625 investigations and 58 model probe files. This is a substantial knowledge corpus that shares no cross-pollination with orch-go's reflect pipeline.

3. **orch-cli/.kb/ has 221 legacy artifacts** from the Python-era predecessor. These represent historical knowledge that may contain superseded but unreferenced investigations.

4. **`kb reflect --type stale`** checking for uncited decisions would miss cross-repo citations (e.g., an orch-go investigation citing a global decision in `~/.kb/decisions/`).

---

## Model Impact

- [x] **Extends** model with: new failure mode — **Failure Mode 6: Cross-repo visibility gap**. The model assumes all relevant investigations are in the scan scope. With 59.5% of artifacts outside default scope and 2.7% permanently invisible, the clustering, staleness, and synthesis signals can be incomplete. Specifically:
  - Synthesis clustering misses related investigations across repos (e.g., same "opencode" topic appears in orch-go, orch-cli, orch-knowledge, and opencode repos)
  - Staleness detection misses cross-repo citations to decisions
  - The global `~/.kb/` store (principles, models, guides) is structurally invisible — no code path reads it

- [x] **Confirms** invariant: "Lexical cluster != conceptual model" (Invariant 1). Cross-repo evidence shows the same topic name (e.g., "opencode", "agent") spans multiple repos with different semantic contexts, reinforcing why lexical clustering requires semantic triage.

---

## Notes

### Structural observation

The knowledge system has three tiers that `kb reflect` handles differently:
1. **Project .kb/** — fully scanned in default mode
2. **All project .kb/ directories** — scanned with `--global` flag (but daemon never uses it)
3. **Global ~/.kb/** — never scanned (structurally different: principles, cross-project models)

### Open question

Should `kb reflect` gain awareness of `~/.kb/` (global artifacts), or should a separate tool handle global-level reflection? The global store has different semantics (principles, cross-project models) vs project-level stores (investigations, decisions).

### Discovered work candidates

1. Daemon should consider running `kb reflect --global` periodically (currently single-project only)
2. `~/.kb/` needs its own reflection path or explicit inclusion in `discoverProjects()`
3. `badge-tracker` and `work-explorer` should be registered in `projects.json`
