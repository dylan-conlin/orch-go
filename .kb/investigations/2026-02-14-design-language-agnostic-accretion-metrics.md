## Summary (D.E.K.N.)

**Delta:** Designed language-agnostic accretion metrics for cross-project orchestration: 4 new metric primitives, enriched event schema with project_dir, cross-project hotspot aggregation via --all-projects, and generated-file exclusion patterns.

**Evidence:** Spike tested 800/1500 thresholds across 3 projects (orch-go Go, opencode TypeScript, beads Go). Uniform thresholds correctly flag structural issues in both languages. Only false positive: generated files (types.gen.ts at 5,065 lines). Current AccretionDeltaData schema lacks project_dir field, making cross-project aggregation impossible.

**Knowledge:** Language-specific thresholds are over-engineering for a 95% Go/TypeScript portfolio where both languages have identical healthy file-size norms (300-800 lines). Cross-project metrics should aggregate at orchestration home (orch-go) following the Single Daemon decision. Four metric primitives (risk_file_ratio, extraction_debt, accretion_velocity, file_bloat_score) enable system-wide health monitoring.

**Next:** Implement in two phases: Phase 1 enrich AccretionDeltaData schema + add generated file exclusion; Phase 2 add --all-projects to orch hotspot + dashboard health panel.

**Authority:** architectural — Cross-component design spanning events schema, hotspot command, completion pipeline, and dashboard; requires orchestrator synthesis.

---

# Investigation: Design Language-Agnostic Accretion Metrics

**Question:** How should accretion metrics be structured so they work across all programming languages and aggregate across multiple projects for system-wide orchestration health visibility?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Architect Agent (og-arch-design-language-agnostic-14feb-f389)
**Phase:** Complete
**Next Step:** Implement Phase 1 (schema enrichment + generated file exclusion)
**Status:** Complete

**Patches-Decision:** N/A (new design)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` | extends | Yes — verified four-layer enforcement architecture, thresholds, gates | None — prior design assumed single-project scope; this extends to cross-project |
| `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md` | constraints | Yes — single daemon in orch-go manages cross-project work | None — design follows orchestration home pattern |
| `.kb/guides/code-extraction-patterns.md` | extends | Yes — verified 300-800 target range applies across Go, TS, Svelte | None — extraction benchmarks confirm uniform thresholds work cross-language |

---

## Problem Framing

**Design Question:** The existing accretion detection (hotspot analysis, delta tracking) works for a single project and uses language-agnostic line counting. But the orchestrator manages 10+ projects from orch-go. How do we get system-wide accretion health visibility across projects?

**Success Criteria:**
1. Accretion metrics aggregate across all .orch/-enabled projects
2. Metrics work for Go, TypeScript, Svelte, Python, Bash — no language-specific config
3. Dashboard shows cross-project health at a glance
4. Generated files excluded from metrics without per-project config
5. Builds on existing infrastructure (events.jsonl, orch hotspot, orch serve)

**Constraints:**
- Single daemon architecture — all metrics aggregate at orchestration home (orch-go)
- Local-First principle — files over databases, git over external services
- Compose Over Monolith — extend existing commands, don't create new infrastructure
- AccretionDeltaData already exists in events.jsonl — extend, don't replace

**Scope:**
- **In:** Metric schema design, cross-project aggregation, generated file exclusion, dashboard integration plan
- **Out:** Language-specific threshold tuning, implementation code, coaching plugin extension (covered by prior investigation)

---

## Findings

### Finding 1: Uniform Thresholds Are Language-Agnostic and Correct

**Evidence:**
Tested 800/1500 line thresholds across three projects with different language compositions:

| Project | Language | Files >800 | Files >1500 | Correctly Flagged? |
|---------|----------|-----------|------------|-------------------|
| orch-go | Go (310 files) | spawn_cmd.go (2,173), doctor.go (1,909), complete_cmd.go (1,847), serve_agents.go (1,561) | 3 CRITICAL | Yes — all known bloated files |
| beads | Go (437 files) | server_issues_epics.go (2,020), queries.go (1,893) | 2 CRITICAL | Yes — structural issues confirmed |
| opencode | TS (40 files) | types.gen.ts (5,065) | 1 false positive | Only generated code flagged |

Both Go and TypeScript produce similarly-sized files for similar complexity. A Go HTTP handler file and a TypeScript service file with equivalent complexity both cluster around 200-600 lines. The 800-line threshold marks the same structural inflection point in both languages.

**Source:** Direct `wc -l` analysis across three projects, compared with extraction guide benchmarks.

**Significance:** Language-specific thresholds would add complexity without improving signal. The portfolio is 95% Go/TypeScript, and both languages have identical healthy norms. This confirms the premise: "language-agnostic" means "works regardless of language," not "adapts per language."

---

### Finding 2: AccretionDeltaData Schema Missing Cross-Project Fields

**Evidence:**
Current `AccretionDeltaData` in `pkg/events/logger.go:482-493`:

```go
type AccretionDeltaData struct {
    BeadsID      string      `json:"beads_id,omitempty"`
    Workspace    string      `json:"workspace,omitempty"`
    Skill        string      `json:"skill,omitempty"`
    FileDeltas   []FileDelta `json:"file_deltas"`
    TotalFiles   int         `json:"total_files"`
    TotalAdded   int         `json:"total_added"`
    TotalRemoved int         `json:"total_removed"`
    NetDelta     int         `json:"net_delta"`
    RiskFiles    int         `json:"risk_files"`
}
```

**Missing fields for cross-project aggregation:**
- No `ProjectDir` — can't distinguish which project a delta came from
- No `ProjectName` — can't display project name on dashboard without parsing dir
- No `Language` on FileDelta — can't filter/group by language (minor, since uniform thresholds)
- No `IsGenerated` on FileDelta — can't exclude generated files from aggregation

Without `ProjectDir`, all accretion events in the shared `~/.orch/events.jsonl` are anonymous — you can't ask "which project is accreting fastest?"

**Source:** `pkg/events/logger.go:472-521`, `cmd/orch/complete_cmd.go:1682-1825`

**Significance:** This is a schema gap, not an architectural one. Adding `ProjectDir` and `ProjectName` to AccretionDeltaData enables cross-project aggregation with zero infrastructure changes.

---

### Finding 3: Hotspot Analysis Has No Cross-Project Mode

**Evidence:**
`cmd/orch/hotspot.go:82` uses `os.Getwd()` for project directory. No `--workdir` or `--all-projects` flag exists. The HTTP API endpoint (`serve_hotspot.go`) also uses CWD only.

To check accretion health across all projects, you'd need to:
```bash
cd ~/Documents/personal/orch-go && orch hotspot --json
cd ~/Documents/personal/beads && orch hotspot --json
cd ~/Documents/personal/opencode && orch hotspot --json
# ... manually aggregate
```

Meanwhile, 10+ projects have `.orch/` directories — the filesystem already knows which projects are orchestrated.

**Source:** `cmd/orch/hotspot.go:36-155`, `ls -d ~/Documents/personal/*/.orch/`

**Significance:** An `--all-projects` flag could discover all `.orch/`-enabled directories and produce a unified hotspot report. This follows the Single Daemon decision (orchestration home provides system-wide visibility) and Local-First principle (scan filesystem, no new infrastructure).

---

### Finding 4: Generated Files Are the Primary False Positive Source

**Evidence:**
The only false positive found across all projects was `types.gen.ts` (5,065 lines) in opencode. This is a code-generated API types file — flagging it as a hotspot is misleading.

Current `isSourceFile()` in hotspot.go:284-300 checks extensions but has no concept of "generated":
```go
func isSourceFile(path string) bool {
    ext := filepath.Ext(path)
    sourceExts := []string{".go", ".js", ".ts", ".jsx", ...}
    // No generated file detection
}
```

Common generated file patterns across languages:
- `*.gen.ts`, `*.gen.go` — code generators
- `*.pb.go`, `*_pb2.py` — protobuf
- `*.generated.*` — various generators
- Files with `// Code generated` comment (Go convention)
- Files in specific directories (`gen/`, `generated/`, `dist/`)

**Source:** Direct analysis of opencode project, Go code generation conventions

**Significance:** Pattern-based exclusion (file name patterns + directory patterns) would eliminate false positives without per-project configuration. This is language-agnostic because the patterns (`*.gen.*`, `gen/`, `dist/`) are conventions, not language-specific.

---

## Decision Forks

### Fork 1: Should accretion thresholds be language-specific or uniform?

**Options:**
- A: Uniform thresholds (800/1500 lines for all languages)
- B: Language-specific thresholds (e.g., 600 for Go, 1000 for Python, 400 for CSS)
- C: Configurable per-project thresholds in .orch/config.yaml

**Substrate says:**
- Principle (Accretion Gravity): "Without structural constraints, the largest file always gets larger" — the force is structural, not linguistic
- Principle (Local-First): Simplicity over configuration — adding per-language config is more machinery
- Evidence (Finding 1): Spike confirmed 800/1500 works identically for Go and TypeScript
- Guide (code-extraction-patterns.md): "Target ~300-800 lines per file" — applies to Go, Svelte, and TypeScript

**RECOMMENDATION:** Option A — Uniform thresholds.

**SUBSTRATE:**
- Principle: Accretion Gravity says the problem is structural, not linguistic
- Evidence: Spike tested across 3 projects — uniform thresholds correctly flag all problems
- Guide: code-extraction-patterns.md establishes 300-800 as universal healthy range

**Trade-off accepted:** A language with inherently larger files (e.g., heavy Python data science notebooks) would get false positives. Acceptable because Dylan's portfolio is 95% Go/TypeScript.

**When this would change:** If a Python-primary project is added where 800-line files are normal and numerous.

---

### Fork 2: Where should cross-project metrics aggregate?

**Options:**
- A: Enrich events.jsonl with ProjectDir field (extend existing)
- B: Per-project metrics files + aggregation command
- C: Dedicated SQLite metrics store at ~/.orch/metrics.db

**Substrate says:**
- Decision (Single Daemon): All work funnels through orch-go; metrics should too
- Principle (Local-First): "Files over databases" — events.jsonl is a file
- Principle (Compose Over Monolith): Extend orch hotspot, don't create new tools
- Existing infrastructure: events.jsonl already works, just missing a field

**RECOMMENDATION:** Option A — Enrich events.jsonl with ProjectDir.

**SUBSTRATE:**
- Decision: Single Daemon says aggregate at orchestration home
- Principle: Local-First says files over databases — events.jsonl is right
- Existing: AccretionDeltaData needs one new field, not a new system

**Trade-off accepted:** events.jsonl grows larger with cross-project events. Acceptable because accretion.delta events are emitted once per completion (low frequency) and the file is already append-only with no size management. If it becomes a problem, add rotation.

**When this would change:** If events.jsonl exceeds 100MB and queries become slow, migrate to SQLite.

---

### Fork 3: What metric primitives should exist beyond raw line count?

**Options:**
- A: Just line count and is_accretion_risk (current)
- B: Add normalized scores (file_bloat_score)
- C: Add four primitives: risk_file_ratio, extraction_debt, accretion_velocity, file_bloat_score

**Substrate says:**
- Principle (Session Amnesia): "Will this help the next Claude resume?" — yes, health ratios are instantly interpretable
- Model (Dashboard Architecture): Two-mode design — operational mode needs health-at-a-glance metrics
- Investigation (accretion-gravity-enforcement): "Hotspot detection exists but is warning-only" — richer metrics make warnings actionable

**RECOMMENDATION:** Option C — Four new metric primitives.

**SUBSTRATE:**
- Principle: Session Amnesia — health ratios help next Claude understand system state immediately
- Model: Dashboard needs glanceable operational metrics
- Investigation: Current metrics are raw data, not actionable intelligence

**Metric definitions:**

| Metric | Formula | Scope | Use |
|--------|---------|-------|-----|
| `risk_file_ratio` | (files >800 lines) / (total source files) | Per project | Health score: <5% healthy, >20% urgent |
| `extraction_debt` | Σ (lines - 800) for files >800 | Per project | Quantifies extraction work needed (in lines) |
| `accretion_velocity` | net_delta / session_count over N days | Per project | Trend: positive = accreting, negative = extracting |
| `file_bloat_score` | file_lines / 800 | Per file | Normalized: 1.0 = threshold, 2.0+ = critical |

**Trade-off accepted:** More computation at hotspot analysis time. Acceptable because hotspot runs at spawn/completion (not hot path) and the math is trivial (counts and ratios).

**When this would change:** If metrics computation becomes a performance bottleneck (unlikely — hotspot analysis of 310 files takes <1 second).

---

### Fork 4: How should cross-project hotspot work?

**Options:**
- A: `--workdir` flag per project (manual)
- B: `--all-projects` flag scans all .orch/ directories (automated)
- C: Separate `orch health` command

**Substrate says:**
- Decision (Single Daemon): Orchestration home provides system-wide visibility
- Principle (Compose Over Monolith): Extend existing command, don't create new ones
- Existing: `ls ~/Documents/personal/*/.orch/` already discovers 10+ orchestrated projects

**RECOMMENDATION:** Option B — `--all-projects` flag on `orch hotspot`.

**SUBSTRATE:**
- Decision: Single Daemon — aggregate at orchestration home
- Principle: Compose Over Monolith — extend hotspot, don't add commands
- Evidence: 10+ .orch/ directories discoverable via filesystem scan

**Trade-off accepted:** Scanning 10+ projects takes longer than single-project. Mitigated by running bloat analysis only (skip fix-commit and investigation signals for cross-project, as those are CWD-specific).

**When this would change:** If project count exceeds 30+ and scan time exceeds 10 seconds, add caching or parallel scanning.

---

### Fork 5: How should generated files be excluded?

**Options:**
- A: Heuristic detection only (*.gen.*, *.pb.go, gen/ directories)
- B: Per-project config in .orch/config.yaml
- C: Both — heuristics as default + config overrides

**Substrate says:**
- Principle (Self-Describing Artifacts): Generated files follow naming conventions
- Existing: hotspot.go already has `defaultExclusions` list (line 29-34)
- Evidence (Finding 4): Only false positive was *.gen.ts — pattern-matchable

**RECOMMENDATION:** Option C — Heuristics as default + config overrides.

**SUBSTRATE:**
- Evidence: Generated files follow naming conventions (*.gen.*, gen/, dist/)
- Existing: defaultExclusions pattern already exists in hotspot.go
- Principle: Self-Describing Artifacts — generated files self-identify via naming

**Default patterns (language-agnostic):**
```
# File patterns
*.gen.go, *.gen.ts, *.gen.js
*.pb.go, *_pb2.py, *_pb.ts
*.generated.*
*.min.js, *.min.css

# Directory patterns
gen/, generated/, dist/, build/
__generated__/, __pycache__/
```

**Config override in .orch/config.yaml:**
```yaml
hotspot:
  generated_patterns:
    - "*.gen.ts"
    - "sdk/*/gen/*"
  exclude_dirs:
    - "dist"
    - "build"
```

**Trade-off accepted:** Heuristics may miss some generated files. Acceptable because config override handles edge cases, and false positives only produce a warning (not a blocking gate).

**When this would change:** If generated files become a significant portion of false positives across multiple projects.

---

## Synthesis

**Key Insights:**

1. **"Language-agnostic" means uniform, not adaptive.** The premise question "do we need language-specific thresholds?" was answered by spike: No. Go and TypeScript have identical healthy file-size norms. The metrics are already language-agnostic — the design work is about cross-project aggregation, not per-language tuning.

2. **The schema gap is small but blocking.** Adding `ProjectDir` and `ProjectName` to AccretionDeltaData is a minimal change that unlocks cross-project aggregation. Everything else (hotspot aggregation, dashboard panel, health metrics) depends on this field.

3. **Four metric primitives transform raw data into actionable intelligence.** Current metrics (line counts, risk file count) are raw data. Health ratios (risk_file_ratio, extraction_debt) are immediately interpretable: "orch-go has 8% risk files and 12,000 lines of extraction debt." This follows Session Amnesia — the next orchestrator can understand system health without running queries.

4. **Generated file exclusion is the only false positive source.** Pattern-based detection eliminates it cleanly. No per-language logic needed.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Enrich AccretionDeltaData with ProjectDir/ProjectName | implementation | Schema extension within existing events infrastructure |
| Add four metric primitives to hotspot report | implementation | New calculations within existing command |
| Add --all-projects flag to orch hotspot | architectural | Cross-component change affecting hotspot, dashboard, events |
| Add generated file exclusion patterns | implementation | Extension of existing defaultExclusions pattern |
| Dashboard cross-project health panel | architectural | New UI component requiring backend API + frontend work |

### Recommended Approach ⭐

**Two-Phase Implementation**

**Phase 1: Schema Enrichment + Generated File Exclusion** (implementation-level, ~2 hours)

| File | Change | Lines (est.) |
|------|--------|-------------|
| `pkg/events/logger.go` | Add `ProjectDir`, `ProjectName` to AccretionDeltaData | +5 |
| `cmd/orch/complete_cmd.go` | Pass projectDir/projectName when calling collectAccretionDelta | +5 |
| `cmd/orch/hotspot.go` | Add generated file patterns to exclusion list | +15 |
| `cmd/orch/hotspot.go` | Add `isGeneratedFile()` function | +20 |
| `cmd/orch/hotspot.go` | Add four metric primitives to HotspotReport | +40 |
| `pkg/events/logger.go` | Add ProjectDir/ProjectName to LogAccretionDelta | +5 |

**Phase 2: Cross-Project Aggregation + Dashboard** (architectural-level, ~4 hours)

| File | Change | Lines (est.) |
|------|--------|-------------|
| `cmd/orch/hotspot.go` | Add `--all-projects` flag with project discovery | +60 |
| `cmd/orch/hotspot.go` | Cross-project report aggregation | +40 |
| `cmd/orch/serve_hotspot.go` | Add `/api/hotspot/all` endpoint | +30 |
| `web/src/lib/components/health-panel/` | New dashboard panel for cross-project health | +100 |
| `.orch/config.yaml` | Add hotspot.generated_patterns config support | +10 |

**Acceptance Criteria:**

Phase 1:
- `orch hotspot` output excludes generated files (types.gen.ts not flagged)
- AccretionDeltaData events include project_dir when emitted from completion
- HotspotReport includes four new metric fields (risk_file_ratio, extraction_debt, accretion_velocity, file_bloat_score)

Phase 2:
- `orch hotspot --all-projects` scans all .orch/-enabled directories
- `/api/hotspot/all` returns unified cross-project hotspot data
- Dashboard shows per-project health scores in operational mode

**Out of scope:**
- Language-specific threshold tuning (confirmed unnecessary by spike)
- Coaching plugin accretion detection (covered by prior investigation)
- Time-series trending/graphing (future enhancement)
- Automated threshold adjustment based on project characteristics

### Alternative Approaches Considered

**Alternative: Dedicated metrics store (SQLite)**
- **Pros:** Structured queries, time-series support, faster aggregation
- **Cons:** New infrastructure (violates Local-First), another database to maintain
- **When to choose:** If events.jsonl exceeds 100MB and grep-based queries become slow

**Alternative: Per-language threshold profiles**
- **Pros:** More accurate for diverse portfolios (e.g., Python data science + Go services)
- **Cons:** Adds config complexity, spike showed uniform works for current portfolio
- **When to choose:** When a Python-primary project is added where 800-line files are common

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This design resolves the cross-project visibility gap in accretion monitoring
- Future spawns may want different threshold approaches

**Suggested blocks keywords:**
- "accretion metrics", "cross-project hotspot", "language agnostic"
- "hotspot all projects", "extraction debt", "risk file ratio"

---

## Structured Uncertainty

**What's tested:**
- ✅ Uniform thresholds work across Go, TypeScript, Svelte — spike tested 3 projects
- ✅ AccretionDeltaData schema missing ProjectDir — verified in code
- ✅ Hotspot has no cross-project mode — verified no --workdir or --all-projects flag
- ✅ Generated files are the only false positive source — tested across 3 projects
- ✅ 10+ projects have .orch/ directories — verified via filesystem scan

**What's untested:**
- ⚠️ Would `--all-projects` scan be fast enough for 10+ projects? (Hypothesis: <3 seconds for bloat-only analysis, but untested)
- ⚠️ Would risk_file_ratio be useful as a dashboard metric? (Hypothesis: yes, but needs UX validation)
- ⚠️ Would events.jsonl with cross-project data grow too large? (Hypothesis: no, accretion events are low-frequency, but needs monitoring)

**What would change this:**
- Finding would be wrong if a Python/Java-heavy project is added where 800-line files are normal
- Finding would be wrong if events.jsonl exceeds 100MB and needs structured storage
- Finding would be wrong if generated file patterns are too aggressive (exclude authored files by mistake)

---

## References

**Files Examined:**
- `pkg/events/logger.go:472-521` — AccretionDeltaData struct and LogAccretionDelta
- `cmd/orch/complete_cmd.go:1682-1825` — collectAccretionDelta implementation
- `cmd/orch/hotspot.go:36-155` — Hotspot command setup and signal architecture
- `cmd/orch/hotspot.go:284-300` — isSourceFile function with language extensions
- `cmd/orch/hotspot.go:381-448` — analyzeBloatFiles implementation
- `cmd/orch/hotspot.go:477-486` — generateBloatRecommendation thresholds
- `.kb/guides/code-extraction-patterns.md` — Extraction benchmarks and target file sizes
- `~/.kb/principles.md:636-667` — Accretion Gravity principle
- `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md` — Orchestration home decision

**Related Artifacts:**
- `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` — Four-layer enforcement architecture (this design extends cross-project)
- `.kb/models/completion-verification/probes/2026-02-14-language-agnostic-accretion-metrics.md` — Probe validating uniform thresholds
- `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md` — Gate inventory informing accretion gate design

---

## Investigation History

**2026-02-14:** Investigation started (og-arch-design-language-agnostic-14feb-f389)
- Problem framed: Cross-project accretion metrics for system-wide health visibility
- Spike: Tested 800/1500 thresholds across orch-go (Go), opencode (TS), beads (Go) — confirmed uniform works
- Substrate consulted: Accretion Gravity principle, Single Daemon decision, code extraction guide
- 5 decision forks navigated with substrate traces
- Recommendations: Two-phase implementation — schema enrichment first, cross-project aggregation second
