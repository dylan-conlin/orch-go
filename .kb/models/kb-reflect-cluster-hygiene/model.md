# Model: kb reflect Cluster Hygiene

**Domain:** Knowledge system maintenance / synthesis triage
**Last Updated:** 2026-02-28
**Synthesized From:** Top synthesis clusters (`feature`, `agents`, `quick`) and 10 synthesis investigations (Jan-Feb 2026)

---

## Summary (30 seconds)

`kb reflect --type synthesis` clusters investigations by lexical similarity. This is a useful discovery signal, but it is not automatically a valid synthesis boundary. Effective triage requires a second step: classify each cluster by semantic cohesion, verify key claims against current code/behavior, then route to one of three dispositions: **converge** (create/update decision/model), **split** (separate mixed lineages), or **demote** (use probe/quick artifact instead of full investigation).

---

## Core Mechanism

### 1) Reflect emits lexical cluster signal

Input: investigation titles and metadata.

Output: topic buckets like `feature (5)` or `quick (4)`.

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

**Primary Evidence (Verify These):**
- `~/Documents/personal/kb-cli/` - kb reflect implementation (cluster algorithm)
- `.kb/investigations/archived/` - Archived investigations (should be excluded from kb reflect scans)
- `.kb/investigations/synthesized/` - Synthesized investigations (should be excluded from kb reflect scans)
- `.kb/decisions/2026-02-08-kb-reflect-cluster-disposition-feature-agents-quick.md` - Disposition routing decision
- `pkg/spawn/config.go` - Feature-impl tier configuration showing referenced behavior
