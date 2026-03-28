## Summary (D.E.K.N.)

**Delta:** Implemented `orch research` command with three modes: summary (all models), detail (per-model claims), and spawn (create probe issue for a claim).

**Evidence:** 19 tests pass (pkg/research). Command tested against live .kb/models/ data — correctly parses 115 claims across 12 models from both claims.yaml and model.md tables, cross-references 96 tested claims via probe file scanning.

**Knowledge:** Two claim sources (claims.yaml structured + model.md markdown tables) can be unified with a simple fallback strategy: prefer YAML, augment with markdown. Probe files use consistent `**claim:** ID` and `**verdict:** verdict` frontmatter that's reliable enough for automated parsing.

**Next:** Wire claim status into `orch orient` output (step 4 of architect design). Create research skill for structured probe protocol (step 2 of architect design).

**Authority:** implementation - New command composing existing primitives (claims.yaml, probe files, bd create), no architectural changes.

---

# Investigation: Implement orch research Command — Claims Parser, Status Display, Spawn Mode

**Question:** How to implement `orch research` with claims parsing from heterogeneous sources, probe cross-referencing, and issue creation for untested claims?

**Started:** 2026-03-28
**Updated:** 2026-03-28
**Owner:** research agent (orch-go-scogn)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-28-inv-design-research-cycle-autoresearch-style.md | implements | yes | None — followed the three-component design exactly |

---

## Findings

### Finding 1: Two claim sources unify cleanly

**Evidence:** 9 models have claims.yaml (structured YAML with id, text, confidence, priority, evidence). 4 models have `## Claims (Testable)` markdown tables in model.md. Some models have both. The strategy: parse claims.yaml first, augment with model.md for HowToVerify field. Models without either are skipped.

**Source:** `pkg/research/status.go:LoadModelStatus()`, `pkg/claims/claims.go:LoadFile()`

**Significance:** No format standardization needed for V1. The fallback approach handles current format variation without breaking changes to existing models.

### Finding 2: Probe frontmatter is parseable with caveats

**Evidence:** Probe files use `**claim:** NI-01, NI-03` and `**verdict:** confirms` format consistently. However, some probes have non-standard claim references (`n/a`, `extends (no prior claim)`, `implicit`). The parser handles these by extracting only well-formed claim IDs (PREFIX-NN pattern) and ignoring freeform text.

**Source:** `grep -r '^\*\*claim' .kb/models/*/probes/*.md` — 20+ probes examined

**Significance:** Automated probe scanning works for ~90% of probes. The remaining 10% with non-standard claim references are skipped, which is the correct behavior (they don't reference specific testable claims).

### Finding 3: Existing pkg/claims provides the YAML foundation

**Evidence:** `pkg/claims/claims.go` already has `Claim`, `File`, `ScanAll()`, `LoadFile()` with full YAML parsing. Built the new `pkg/research/` package to add probe scanning and status aggregation on top of this existing foundation.

**Source:** `pkg/claims/claims.go:134-158` (ScanAll), `pkg/research/status.go:50-68` (LoadModelStatus using claims.LoadFile)

**Significance:** ~50% less code needed than estimated because the claims YAML parser already existed.

---

## Structured Uncertainty

**What's tested:**

- ✅ Claims parsing from claims.yaml (19 tests, verified against 9 real model files)
- ✅ Claims parsing from model.md markdown tables (verified against 4 real models)
- ✅ Probe scanning with claim/verdict extraction (verified against ~60 real probe files)
- ✅ Status aggregation (untested/confirmed/contradicted/extended/mixed)
- ✅ Model prefix matching and error handling
- ✅ Issue creation via bd create (code path verified, not e2e tested to avoid creating real issues)

**What's untested:**

- ⚠️ Whether spawn mode produces good probe agents (depends on research skill and agent behavior)
- ⚠️ Performance with very large model directories (only tested with current 12 models)
- ⚠️ Orient integration (not in scope for this issue)

**What would change this:**

- If probe frontmatter format changes, the parser would need updating
- If models adopt a third claim format (beyond YAML and markdown), the parser needs extension

---

## References

**Files Created:**
- `pkg/research/probes.go` — Probe file scanning and frontmatter parsing
- `pkg/research/markdown.go` — Markdown claims table parsing from model.md
- `pkg/research/status.go` — Status aggregation, model loading, claim finding
- `pkg/research/probes_test.go` — 8 test functions for probe parsing
- `pkg/research/markdown_test.go` — 4 test functions for markdown parsing
- `pkg/research/status_test.go` — 7 test functions for status aggregation
- `cmd/orch/research_cmd.go` — CLI command with 3 modes

**Files Modified:**
- `cmd/orch/main.go` — Added researchCmd to rootCmd

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-28-inv-design-research-cycle-autoresearch-style.md` — Architect design this implements

---

## Investigation History

**2026-03-28:** Investigation started
- Initial question: How to implement orch research with claims parsing and probe status
- Context: Step 1 of research cycle (from architect design orch-go-47ppm)

**2026-03-28:** Implementation completed
- Status: Complete
- Key outcome: `orch research` command with 3 modes, 19 passing tests, works against live data (115 claims, 12 models, 83% tested)
