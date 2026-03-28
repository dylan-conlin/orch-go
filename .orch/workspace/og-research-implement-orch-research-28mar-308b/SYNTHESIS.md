# Session Synthesis

**Agent:** og-research-implement-orch-research-28mar-308b
**Issue:** orch-go-scogn
**Duration:** 2026-03-28 → 2026-03-28
**Outcome:** success

---

## Plain-Language Summary

Built `orch research` — a command that reads all model claims (from claims.yaml and model.md tables), scans probe files for test results, and shows which claims have been tested and which haven't. Three modes: summary view across all 12 models (115 claims total, 83% tested), detail view per model showing each claim with its probes, and spawn mode that creates a beads issue to probe a specific untested claim. This makes the research cycle visible: you can see at a glance what's been tested, what hasn't, and trigger a probe for any gap.

---

## TLDR

Implemented `orch research` command with claims parser, probe scanner, and issue creation. Parses 115 claims across 12 models from both claims.yaml and model.md, cross-references with probe files to show test status. 19 tests pass.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

Key outcomes:
- `go test ./pkg/research/...` — 19 tests pass
- `go build ./cmd/orch/` — builds clean
- `orch research` — displays all models with claim counts
- `orch research named-incompleteness` — shows 6 claims with probe details
- `orch research named-incompleteness NI-99` — correct error for missing claim

---

## Delta (What Changed)

### Files Created
- `pkg/research/probes.go` — Probe file scanning and frontmatter parsing
- `pkg/research/markdown.go` — Markdown claims table parsing from model.md
- `pkg/research/status.go` — Status aggregation, model finding, claim lookup
- `pkg/research/probes_test.go` — 8 test functions
- `pkg/research/markdown_test.go` — 4 test functions
- `pkg/research/status_test.go` — 7 test functions
- `cmd/orch/research_cmd.go` — CLI command with 3 modes

### Files Modified
- `cmd/orch/main.go` — Added `researchCmd` to `rootCmd.AddCommand()`

---

## Evidence (What Was Observed)

- 9 models have claims.yaml, 4 have markdown claims tables, some have both
- Probe files use `**claim:** ID` and `**verdict:** verdict` frontmatter consistently (~90%)
- ~10% of probes have non-standard claim refs (n/a, implicit) — parser correctly skips these
- Existing `pkg/claims/` package provided YAML foundation — ~50% less code than estimated

### Tests Run
```bash
go test ./pkg/research/... -v
# PASS: 19/19 tests (0.257s)

go build ./cmd/orch/
# BUILD OK

go run ./cmd/orch/ research
# Shows 12 models, 115 claims, 83% tested
```

---

## Architectural Choices

### Separate pkg/research/ instead of extending pkg/claims/
- **What I chose:** New package for probe scanning and status aggregation
- **What I rejected:** Adding probe logic to existing pkg/claims/
- **Why:** pkg/claims/ handles YAML read/write/mutation. Probe scanning and cross-referencing is a distinct concern. Keeps claims package focused.
- **Risk accepted:** Two packages that both deal with "claims" — could confuse future agents

### Fallback strategy (YAML then markdown) instead of format standardization
- **What I chose:** Try claims.yaml first, fall back to model.md tables
- **What I rejected:** Requiring all models to have claims.yaml
- **Why:** Architect design identified format standardization as optional. Current approach handles variation without breaking changes.
- **Risk accepted:** Models with only markdown claims lose confidence/priority metadata

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-28-inv-implement-orch-research-command-claims.md` — Implementation investigation

### Constraints Discovered
- Probe frontmatter has ~10% non-standard entries that can't be auto-linked to claims
- Some claims.yaml files use numeric priority (1, 2, 3) instead of string (core, supporting) — displayed as-is

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete (claims parser, status display, spawn mode)
- [x] Tests passing (19/19)
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-scogn`

Follow-up work (from architect design):
1. Orient integration — add claim status to `orch orient` output
2. Research skill — structured probe protocol for spawned agents

---

## Unexplored Questions

- Whether the spawn mode produces good probe agents (depends on agent behavior with the assembled context)
- Whether orient integration should show per-model detail or just aggregate counts

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** research (implementation, not web research)
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-research-implement-orch-research-28mar-308b/`
**Investigation:** `.kb/investigations/2026-03-28-inv-implement-orch-research-command-claims.md`
**Beads:** `bd show orch-go-scogn`
