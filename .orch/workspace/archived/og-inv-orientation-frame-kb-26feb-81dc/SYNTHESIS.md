# Session Synthesis

**Agent:** og-inv-orientation-frame-kb-26feb-81dc
**Issue:** orch-go-05w8
**Outcome:** success

---

## Plain-Language Summary

`kb reflect` only sees the knowledge artifacts in the current project's `.kb/` directory. But Dylan's knowledge system spans 20+ repos and a global `~/.kb/` store. I inventoried every `.kb/` location and found **3,287 total .md artifacts** across all locations. Of those, only **1,331 (40.5%)** are visible to `kb reflect` in its default mode (orch-go). The remaining **1,956 artifacts (59.5%)** live in other repos or the global store. Even with the existing `--global` flag (which the daemon never uses), 89 artifacts in `~/.kb/` are permanently invisible because that directory is structurally different from project `.kb/` directories — it's a symlink to `~/orch-knowledge/kb/`, not a project's `.kb/`.

The biggest blind spots: orch-knowledge (584 project artifacts), price-watch (711 work artifacts), orch-cli (221 legacy artifacts), and the global `~/.kb/` store (89 artifacts including the master `principles.md`, 6 global models, and 8 global guides).

## Verification Contract

See `VERIFICATION_SPEC.yaml` for commands run and evidence gathered.

---

## Delta (What Changed)

### Files Created
- `.kb/models/kb-reflect-cluster-hygiene/probes/2026-02-26-probe-cross-repo-knowledge-visibility-inventory.md` — Complete inventory and gap analysis as a model probe

### Files Modified
- None

---

## Evidence (What Was Observed)

### Complete Artifact Inventory

| Location | .md Files | Investigations | Decisions | Models | Guides | Other |
|----------|-----------|---------------|-----------|--------|--------|-------|
| **orch-go/.kb/** | 1,331 | 1,067 | 60 | 174 | 34 | 6 |
| **orch-knowledge/.kb/** | 584 | 490 | 85 | 0 | 3 | 6 |
| **~/.kb/ (global)** | 89 | 7 | 15 | 6 | 8 | 53 |
| **price-watch/.kb/** | 711 | 625 | 21 | 58 | 1 | 6 |
| **orch-cli/.kb/** | 221 | 215 | 5 | 0 | 1 | 0 |
| **kb-cli/.kb/** | 36 | 35 | 1 | 0 | 0 | 0 |
| **beads-ui-svelte/.kb/** | 71 | 70 | 1 | 0 | 0 | 0 |
| **skillc/.kb/** | 32 | 30 | 2 | 0 | 0 | 0 |
| **Other personal (11 repos)** | 111 | ~100 | ~5 | 0 | 0 | ~6 |
| **Work repos (6 repos)** | 101 | ~90 | ~5 | 0 | 0 | ~6 |
| **TOTAL** | **3,287** | | | | | |

### Key Findings

1. **`findKBDir()` is strictly `{projectDir}/.kb/`** — no fallback, no global awareness (kb-cli/cmd/kb/create.go:799-805)
2. **`--global` flag exists** and uses `discoverProjects()` which checks registered projects + directory walk (kb-cli/cmd/kb/search.go:105-170)
3. **Daemon never passes `--global`** to `kb reflect` (pkg/daemon/reflect.go:130)
4. **`~/.kb/` is a symlink** to `~/orch-knowledge/kb/` — structurally different from project `.kb/` dirs, never scanned
5. **2 unregistered repos** (badge-tracker, scs-explorer) at depth 4 under ~/Documents would be missed even by --global
6. **`projects.json` has 18 registered projects** covering most but not all repos with .kb/ directories

### Visibility Breakdown
- Default mode: 40.5% visible
- With --global: 97.3% visible (all project .kb/ dirs)
- Permanently invisible: 2.7% (~/.kb/ global store)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/kb-reflect-cluster-hygiene/probes/2026-02-26-probe-cross-repo-knowledge-visibility-inventory.md` — Extends model with Failure Mode 6 (cross-repo visibility gap)

### Constraints Discovered
- `~/.kb/` (global) is structurally different from project `.kb/` — it has principles, cross-project models, templates, and values that don't fit the project-level reflection paradigm
- The daemon's single-project reflect means synthesis clusters are always project-scoped, missing cross-repo investigation patterns

### Three-Tier Knowledge Architecture (discovered)
1. **Project .kb/** — investigations, decisions, models scoped to one repo
2. **Global ~/.kb/** — principles, cross-project models, guides, templates
3. **Cross-project patterns** — same topic (e.g., "opencode") appears in orch-go, orch-cli, orch-knowledge, and opencode repos with different semantic contexts

---

## Next (What Should Happen)

**Recommendation:** close + spawn follow-up (architect)

### Discovered Work

Three issues to consider:

1. **Daemon should use `--global` for synthesis detection** — Cross-repo investigation clusters (e.g., "opencode" appears in 4+ repos) are invisible without it. Low-risk change: add `--global` to daemon's reflect invocation.

2. **`~/.kb/` needs a reflection path** — 89 artifacts including principles.md, 6 global models, and 8 global guides have no reflection coverage. This is a design question: should `kb reflect` gain `~/.kb/` awareness, or should a separate global-reflection tool handle it?

3. **Register missing repos** — `badge-tracker` and `scs-explorer` have .kb/ directories but aren't in `projects.json`, making them invisible even to `--global`.

### Routing Recommendation

Issue #1 (daemon --global) could be a straightforward task.
Issue #2 (~/.kb/ reflection) needs **architect** review — the global store has different semantics and may need different reflection types than project-level stores.
Issue #3 (register repos) is a trivial manual step.

---

## Unexplored Questions

- **Cross-repo duplication**: Are there semantically duplicate investigations across repos? The filename-based check found no exact matches, but semantic duplicates (same finding, different titles) likely exist between orch-go and orch-knowledge (the legacy migration boundary).
- **orch-cli legacy value**: 221 artifacts from the Python-era predecessor. How much of this is still relevant vs fully superseded by orch-go knowledge?
- **price-watch knowledge isolation**: 711 artifacts in a work project that never cross-pollinates with the personal knowledge system. Is this by design or an unintended silo?

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-orientation-frame-kb-26feb-81dc/`
**Probe:** `.kb/models/kb-reflect-cluster-hygiene/probes/2026-02-26-probe-cross-repo-knowledge-visibility-inventory.md`
**Beads:** `bd show orch-go-05w8`
