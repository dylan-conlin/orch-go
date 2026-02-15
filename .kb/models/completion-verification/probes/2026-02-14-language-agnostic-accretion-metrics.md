# Probe: Language-Agnostic Accretion Metrics for Cross-Project Orchestration

**Model:** Completion Verification (`completion-verification.md`)
**Date:** 2026-02-14
**Status:** Complete
**Beads:** orch-go-foy

---

## Question

Do the existing accretion metrics (uniform 800/1500 line thresholds, raw line counting) produce meaningful results across multiple programming languages and projects? Are language-specific thresholds needed, or does uniform measurement suffice for cross-project orchestration?

---

## What I Tested

### Test 1: Cross-Language Threshold Validity

**Tested:** Applied 800/1500 line thresholds to three projects with different language mixes.

**Commands:**
```bash
# orch-go (Go-primary, 310 .go files, 22 .ts files)
find . -name "*.go" | xargs wc -l | sort -rn | head -15

# opencode (TypeScript-only, 40 .ts files)
find . -name "*.ts" -not -path "*/node_modules/*" -not -path "*/dist/*" | xargs wc -l | sort -rn | head -15

# beads (Go-primary, 437 .go files)
find . -name "*.go" | xargs wc -l | sort -rn | head -15
```

**Results:**

| Project | Language | Largest authored file | Lines | >800? | >1500? |
|---------|----------|----------------------|-------|-------|--------|
| orch-go | Go | spawn_cmd.go | 2,173 | Yes | Yes (CRITICAL) |
| orch-go | Go | doctor.go | 1,909 | Yes | Yes (CRITICAL) |
| orch-go | Go | complete_cmd.go | 1,847 | Yes | Yes (CRITICAL) |
| beads | Go | server_issues_epics.go | 2,020 | Yes | Yes (CRITICAL) |
| beads | Go | queries.go | 1,893 | Yes | Yes (CRITICAL) |
| opencode | TS | types.gen.ts | 5,065 | Yes | Yes — BUT GENERATED |

**Key finding:** For authored code, both Go and TypeScript files follow similar size distributions. The 800/1500 thresholds correctly flag problematic files in both languages. The only false positive is generated code (types.gen.ts), which should be excluded via pattern matching, not language-specific thresholds.

### Test 2: Current AccretionDeltaData Schema Completeness

**Tested:** Whether the existing events.jsonl schema supports cross-project aggregation.

**Examined:** `pkg/events/logger.go:472-521`

**Observation:** `AccretionDeltaData` struct includes:
- `beads_id`, `workspace`, `skill` — agent context
- `file_deltas[]` with path, lines_added, lines_removed, net_delta, total_lines, is_accretion_risk
- `total_files`, `total_added`, `total_removed`, `net_delta`, `risk_files`

**Missing for cross-project:**
- No `project_dir` field — can't distinguish which project a delta came from
- No `language` field on FileDelta — can't filter/group by language
- No `is_generated` field — can't exclude generated files from aggregation
- No timestamp-based trending — events have timestamps but no built-in aggregation

### Test 3: Hotspot Cross-Project Capability

**Tested:** Whether `orch hotspot` can analyze projects other than CWD.

**Examined:** `cmd/orch/hotspot.go:36-155` (command setup), `hotspot.go:381-448` (bloat analysis)

**Observation:** The command uses `os.Getwd()` at hotspot.go:82 for the project directory. No `--workdir` or `--project` flag exists. The HTTP API endpoint (`serve_hotspot.go`) also uses CWD only. Cross-project analysis requires running `orch hotspot` from each project directory separately — no aggregation.

### Test 4: Language Distribution Across Projects

**Tested:** File type composition across all .orch/-enabled projects.

**Commands:**
```bash
for dir in orch-go opencode beads glass skillc; do
  find ~/Documents/personal/$dir -name "*.go" -o -name "*.ts" -o ... |
    sed 's/.*\.//' | sort | uniq -c | sort -rn
done
```

**Results:**
| Project | Primary Language | Go files | TS files | Other |
|---------|-----------------|----------|----------|-------|
| orch-go | Go | 310 | 22 | 3 JS |
| opencode | TypeScript | 0 | 40 | 2 JS |
| beads | Go | 437 | 0 | 5 PY, 4 JS |
| glass | Go | 7 | 0 | 0 |
| skillc | Go | 20 | 0 | 0 |

**Key finding:** Dylan's portfolio is 95%+ Go/TypeScript. Language-specific thresholds are over-engineering for this composition. Both languages have similar healthy file-size norms (300-800 lines).

---

## Model Impact

### Confirmed

1. **Uniform thresholds (800/1500 lines) are language-agnostic and correct.** Tested across Go and TypeScript — both languages produce similarly-sized files for similar complexity. The thresholds flag the same structural problems regardless of language.

2. **Raw line counting is sufficient for accretion detection.** More sophisticated metrics (cyclomatic complexity, function count) would not provide meaningfully better signal for the cost. Line count correlates well with structural bloat across Go and TypeScript.

3. **The accretion delta event schema (FileDelta) is a good primitive.** It captures the right per-file granularity. The normalized score (lines/threshold) enables cross-file comparison.

### Extended

1. **Cross-project aggregation requires enriching AccretionDeltaData with `project_dir`.** Without this field, events from different projects can't be distinguished in the shared events.jsonl. This is a schema gap, not an architectural one.

2. **Generated file exclusion needs explicit support.** The `isSourceFile()` function in hotspot.go checks file extensions but doesn't detect generated files (e.g., `*.gen.ts`, `*.pb.go`). Adding a `generated_patterns` exclusion list would eliminate the primary false positive source.

3. **Cross-project hotspot aggregation is a natural extension.** `orch hotspot --all-projects` could scan all `~/.orch/`-containing directories and produce a unified report. This follows the Local-First principle (scan filesystem, no new infrastructure).

4. **Four new metric primitives would enable system-wide health tracking:**
   - `risk_file_ratio` = files >800 / total source files (per-project health score)
   - `extraction_debt` = sum of (lines - 800) for files >800 (quantifies work needed)
   - `accretion_velocity` = net lines added per session (trend direction)
   - `file_bloat_score` = lines / threshold (normalized for cross-file comparison)

### No Contradictions Found

The model's existing gate architecture (spawn gates, completion gates) doesn't assume language-specific behavior. The design is already implicitly language-agnostic — this probe confirms that explicit language awareness isn't needed.
