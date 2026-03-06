# Model: kb reflect Cluster Hygiene

**Domain:** Knowledge system maintenance / synthesis triage
**Last Updated:** 2026-03-06
**Synthesized From:** Top synthesis clusters (`feature`, `agents`, `quick`) and 10 synthesis investigations (Jan-Feb 2026)

---

## Summary (30 seconds)

`kb reflect --type synthesis` clusters investigations by lexical similarity. This is a useful discovery signal, but it is not automatically a valid synthesis boundary. Effective triage requires a second step: classify each cluster by semantic cohesion, verify key claims against current code/behavior, then route to one of three dispositions: **converge** (create/update decision/model), **split** (separate mixed lineages), or **demote** (use probe/quick artifact instead of full investigation).

---

## Core Mechanism

### 1) Reflect emits lexical cluster signal (and defect-class metadata signal)

Input: investigation titles and metadata.

Output: lexical topic buckets like `feature (5)` or `quick (4)`, plus a parallel metadata-based signal: `--type defect-class` emits clusters by `Defect-Class:` frontmatter tag (e.g., `configuration-drift (5)` with escalation suggestion). These are two distinct clustering dimensions — lexical (filename-derived) and semantic/manual (frontmatter tag).

### 2) Orchestrator performs semantic triage

For each cluster:

1. Check whether items describe one mechanism or multiple unrelated threads.
2. Re-validate cluster claims against primary evidence (code/tests/runtime), not only old investigation conclusions.
3. Choose disposition.

### 3) Disposition routing

- **Converge:** Coherent cluster with shared mechanism -> consolidate into canonical decision/model.
- **Split:** Lexically similar but semantically mixed -> map items to existing lineages/decisions.
- **Demote:** One-off fact checks or tactical validation -> probe or `kb quick`, not investigation.

### 4) Closure normalization

Redundant investigations are closed by metadata, not deletion:

- add `Superseded-By`
- make D.E.K.N. `Next` non-actionable for that file
- keep archived file for provenance chain

### Critical Invariants

1. **Lexical cluster != conceptual model**
2. **Code/test evidence outranks archived claims**
3. **One canonical decision/model per mechanism**
4. **Redundant investigations must point to canonical artifact**

---

## Why This Fails

### Failure Mode 1: Lexical collision (MITIGATED 2026-02-28)

`feature` clusters can mix tiering behavior, cross-repo implementation tasks, and decision-gate debugging. Treating them as one topic creates noisy synthesis.

**Mitigation:** Two-pass subclustering added to `findSynthesisCandidates`. Clusters with 5+ investigations (`InvestigationSubclusterThreshold`) are split by qualifying word (the next meaningful word after the primary keyword in each filename). For example, a "context" mega-cluster splits into "context-spawn" (3 files) and "context-window" (3 files). Small clusters (<5) are unchanged to preserve legitimate grouping. Subclusters with <3 files merge back to parent.

**Functions added:** `extractQualifyingWord()`, `subclusterInvestigations()` in `reflect.go`

**Source:** orch-go-jlar

### Failure Mode 2: Time-drifted conclusions

Investigation findings can become stale after code changes (for example, fail-open to fail-closed gate behavior). Re-validation is required during consolidation.

### Failure Mode 3: Artifact overuse

Quick fact lookups become full investigations, increasing maintenance burden without adding durable understanding.

### Failure Mode 4: Incomplete closure metadata

### Failure Mode 6: Cross-repo visibility gap

When `kb reflect` runs in default mode (single project), 59.5% of all knowledge artifacts are invisible. Specifically:

| Tier | Artifacts | Visibility |
|------|-----------|------------|
| orch-go/.kb/ (default) | 1,331 (40.5%) | Always scanned |
| Other project .kb/ directories | 1,867 (56.8%) | Only with `--global` |
| ~/.kb/ (global store) | 89 (2.7%) | Never scanned — structurally invisible |

The global `~/.kb/` store (which is a symlink to `~/orch-knowledge/kb/`, NOT `~/orch-knowledge/.kb/`) contains the master `principles.md`, 6 global models, and 8 global guides. `discoverProjects()` finds repos with `.kb/` directories — `~/.kb/` is not inside any project, so no code path reads it.

The daemon never uses `--global` (runs single-project mode only), so even the 56.8% of cross-project artifacts are invisible to automatic reflection.

**Practical impact:** Synthesis clustering misses related investigations across repos (same "opencode" topic spans orch-go, orch-cli, orch-knowledge, opencode repos). Staleness detection misses cross-repo citations. The `kb reflect --type stale` check for uncited decisions can't see citations to `~/.kb/decisions/`.

**Fix design (validated):** `Reflect()` cleanly separates into `reflectKBDir(kbDir, projectDir, opts)`. Adding global store is one extra `reflectKBDir()` call after the project loop. For `kb context`, a new `GetGlobalStoreContext()` function is needed (the existing `GetContext()` cannot be called with `~` because `findKBDir()` expects `{projectDir}/.kb/`). The spawn system benefits automatically from kb-cli-level changes — no orch-go code changes needed.

### Failure Mode 7: Producer-consumer drift (reflect emits data consumer doesn't parse)

`kb reflect --type defect-class` emits valid data, but the orch-go pipeline silently drops it:
- `kbReflectOutput` struct lacks a `DefectClass` field → `json.Unmarshal` ignores it
- `createIssues=true` path narrows to `--type synthesis` → defect-class issue creation never happens
- `HasSuggestions()`, `TotalCount()`, `Summary()` exclude defect-class → never shown at session start
- `reflect-suggestions.json` never includes defect-class → dashboard API can't serve it

This is a clean example of configuration drift between kb-cli capabilities and orch-go consumption — kb-cli gained defect-class support but orch-go never added Go structs for it. **Fix:** Add `DefectClass` field to `kbReflectOutput` and `ReflectSuggestions`; remove `--type synthesis` restriction from `createIssues` path; update `HasSuggestions`/`TotalCount`/`Summary`.

Archived files without clear `Superseded-By` pointers remain discoverable but ambiguous, causing repeated re-triage.

### Failure Mode 5: Scans archived/synthesized directories (FIXED 2026-02-25)

~~kb reflect scans `.kb/investigations/archived/` and `.kb/investigations/synthesized/` directories, creating false positives for already-processed clusters.~~

**Fixed:** All 7 investigation-walking functions now use `isArchivedOrSynthesizedDir()` + `filepath.SkipDir` to exclude both directories. Additionally, age calculation was fixed to use filename date prefix instead of file modification time.

**Fix commit:** `015e6d9` in kb-cli (orch-go-1251)

**Source:** `2026-01-17-inv-synthesize-extract-investigation-cluster-13.md:59-69`, `2026-02-14-inv-synthesize-synthesize-investigations-10-synthesis.md`

---

## Constraints

### Why not auto-consolidate directly from reflect output?

**Constraint:** reflect clusters are intentionally broad and lexical.

**Implication:** Human/orchestrator semantic triage is required before creating decisions/models.

**This enables:** Better quality synthesis boundaries and less decision noise.
**This constrains:** Fully automated promotion from cluster -> decision.

### Why preserve redundant investigations instead of deleting?

**Constraint:** Provenance chain must remain inspectable.

**Implication:** Closure happens via metadata pointers (`Superseded-By`) and canonical references.

**This enables:** Auditable understanding evolution.
**This constrains:** Lightweight cleanup by file deletion alone.

---

## Evolution

**2026-02-08:** Top-cluster consolidation codified as a triage model with converge/split/demote routing and closure normalization.

**2026-02-14:** Added Failure Mode 5 (scans archived/synthesized directories) discovered from synthesis investigations analysis. kb reflect needs to exclude these directories to prevent false positives for already-processed clusters.

**2026-02-25:** Failure Mode 5 FIXED (orch-go-1251). Also fixed age calculation across all reflect types to use filename date prefix instead of ModTime.

---

## Evolution

**2026-02-26:** Added Failure Modes 6 (cross-repo visibility gap) and 7 (producer-consumer drift). Core Mechanism updated to acknowledge defect-class as a parallel metadata-based clustering signal alongside lexical synthesis clusters. Three-tier visibility model established: project (default), cross-project (--global), global ~/.kb/ (never scanned).

---

## References

**Investigations:**
- `.kb/investigations/archived/2025-12-24-inv-feature-impl-agents-completing-without.md`
- `.kb/investigations/archived/2025-12-26-inv-feature-impl-agents-not-producing.md`
- `.kb/investigations/archived/2026-01-14-inv-feature-register-friction-guidance-links.md`
- `.kb/investigations/archived/2026-01-14-inv-feature-skillc-warns-load-bearing.md`
- `.kb/investigations/archived/2026-01-28-inv-implement-test-feature.md`
- `.kb/investigations/archived/2025-12-22-inv-40-agents-showing-as-active.md`
- `.kb/investigations/archived/2026-01-03-inv-agents-going-idle-without-phase.md`
- `.kb/investigations/archived/2026-01-08-inv-25-28-agents-not-completing.md`
- `.kb/investigations/archived/2026-02-04-inv-agents-own-declaration-via-bd.md`
- `.kb/investigations/archived/2026-01-10-inv-quick-test-default-port-orch.md`
- `.kb/investigations/archived/2026-01-19-inv-quick-test-read-claude-md.md`
- `.kb/investigations/archived/2026-01-21-inv-audit-kb-quick-entries-stale.md`
- `.kb/investigations/archived/2026-01-27-inv-quick-test-verify-coaching-plugin.md`

**Decisions informed by this model:**
- `.kb/decisions/2026-02-08-kb-reflect-cluster-disposition-feature-agents-quick.md`

**Primary evidence:**
- `pkg/spawn/config.go` - `feature-impl` tier default
- `cmd/orch/spawn_validation.go` - decision-gate fail-closed behavior
- `cmd/orch/status_statedb.go` - fallback status discovery over `workers-*` tmux sessions
- `cmd/orch/serve.go` - default serve port constant

### Merged Probes

| Probe | Date | Key Finding |
|-------|------|-------------|
| `2026-02-26-probe-defect-class-pipeline-gap.md` | 2026-02-26 | orch-go pipeline has complete blind spot for defect-class: missing Go struct field, `createIssues` narrows to synthesis only, `HasSuggestions` excludes it — silent data drop |
| `2026-02-26-probe-cross-repo-knowledge-visibility-inventory.md` | 2026-02-26 | 59.5% of 3,287 total artifacts invisible in default mode; `~/.kb/` (2.7%, global store) permanently invisible even with `--global`; daemon never uses `--global` |
| `2026-02-26-probe-cross-repo-global-store-design-validation.md` | 2026-02-26 | Option A (virtual project) for kb reflect feasible; `kb context` needs new `GetGlobalStoreContext()` function; `~/.kb/` symlink structure confirmed, no double-counting risk with orch-knowledge/.kb/ |

**Primary Evidence (Verify These):**
- `~/Documents/personal/kb-cli/` - kb reflect implementation (cluster algorithm)
- `.kb/investigations/archived/` - Archived investigations (should be excluded from kb reflect scans)
- `.kb/investigations/synthesized/` - Synthesized investigations (should be excluded from kb reflect scans)
- `.kb/decisions/2026-02-08-kb-reflect-cluster-disposition-feature-agents-quick.md` - Disposition routing decision
- `pkg/spawn/config.go` - Feature-impl tier configuration showing referenced behavior
