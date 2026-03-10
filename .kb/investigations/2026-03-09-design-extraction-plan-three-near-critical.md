## Summary (D.E.K.N.)

**Delta:** Three near-critical files (kbcontext.go 1496, context.go 1495, stats_cmd.go 1490) each have 3-4 distinct responsibility domains that can be extracted into ~300-800 line files, bringing parent files to ~500-800 lines.

**Evidence:** Function analysis of all three files reveals clean domain boundaries: kbcontext.go has query/format/model-parsing/staleness, context.go has template/skill-processing/generation/embedded-templates, stats_cmd.go has types/aggregation/text-output.

**Knowledge:** All three files follow the same accumulation pattern—organic growth from feature additions without extraction checkpoints. The spawn package files share a package namespace (no import changes needed), while stats_cmd.go is in cmd/orch (also package-level visibility).

**Next:** Create 6 implementation issues for sequential extraction (3 files x 2 phases each: shared utilities first, then domain code). Sequence: stats_cmd.go first (lowest coupling), then kbcontext.go, then context.go (highest coupling).

**Authority:** implementation - Follows established extraction patterns from .kb/guides/code-extraction-patterns.md; no architectural decisions needed.

---

# Investigation: Extraction Plan for Three Near-Critical Hotspot Files

**Question:** How should we extract pkg/spawn/kbcontext.go (1496), pkg/spawn/context.go (1495), and cmd/orch/stats_cmd.go (1490) to bring them below the critical 1500-line threshold with room to grow?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — implementation issues created
**Status:** Complete
**Model:** code-extraction-patterns

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/guides/code-extraction-patterns.md | extends | Yes — patterns confirmed | None |
| daemon.go 1559→715 extraction (recent) | informs | Yes — successful reference | None |

---

## Findings

### Finding 1: kbcontext.go has 4 distinct responsibility domains

**Evidence:** Function-level analysis of 1496 lines reveals:

| Domain | Lines | Functions | Key Functions |
|--------|-------|-----------|---------------|
| Query & Search | ~250 | 10 | `RunKBContextCheck`, `RunKBContextCheckForDir`, `runKBContextQuery`, `filterToOrchEcosystem`, `filterToProjectGroup`, `resolveProjectAllowlist`, `mergeResults`, `applyPerCategoryLimits` |
| Parse & Classify | ~170 | 6 | `parseKBContextOutput`, `extractSource`, `ExtractKeywords`, `ExtractKeywordsWithContext`, `TaskIsScoped`, `FilterForScopedTask` |
| Format for Spawn | ~280 | 6 | `FormatContextForSpawn`, `FormatContextForSpawnWithLimit`, `FormatContextForSpawnWithLimitAndMeta`, `formatKBContextContent`, `filterByType`, `formatMatchesForDisplay` |
| Model Parsing & Staleness | ~430 | 14 | `formatModelMatchForSpawn`, `extractModelSectionsForSpawn`, `collectMarkdownHeadings`, `parseMarkdownHeading`, `extractSectionByHeading`, `truncateModelSection`, `extractCodeRefs`, `extractLastUpdated`, `checkModelStaleness`, `DetectCrossRepoModel`, `hasInjectedModelContent`, `hasValidFileExtension`, `normalizeHeading`, `indentBlock` |
| Types & Constants | ~100 | 0 | `KBContextMatch`, `KBContextResult`, `KBContextFormatResult`, `StalenessResult`, constants |

**Source:** `pkg/spawn/kbcontext.go:1-1496`, function listing via grep

**Significance:** The Model Parsing & Staleness domain is the largest (430 lines) and most self-contained — it only reads files and runs git commands. This is the primary extraction candidate. Query & Search is secondary.

---

### Finding 2: context.go has template, generation, and template-embedded-constants as distinct domains

**Evidence:** Function-level analysis of 1495 lines reveals:

| Domain | Lines | Functions | Key Functions |
|--------|-------|-----------|---------------|
| Go Template (SpawnContextTemplate) | ~335 | 0 | Large const string with Go template syntax |
| Skill Content Processing | ~150 | 4 | `ProcessSkillContentTemplate`, `StripBeadsInstructions`, `WriteSkillPromptFile`, regex patterns |
| Context Generation & Writing | ~200 | 4 | `GenerateContext`, `WriteContext`, `MinimalPrompt`, `GenerateInvestigationSlug` |
| Embedded Templates (SYNTHESIS, FAILURE) | ~335 | 6 | `DefaultSynthesisTemplate`, `DefaultFailureReportTemplate`, `EnsureSynthesisTemplate`, `EnsureFailureReportTemplate`, `WriteFailureReport`, `generateFailureReport` |
| Utility Functions | ~240 | 8 | `ParseScopeFromTask`, `ResolveScope`, `CreateScreenshotsDir`, `ValidateBeadsIDConsistency`, `extractProjectPrefix`, `DetectAreaFromTask`, `GenerateServerContext`, `GetClusterSummary`, `GenerateRegisteredProjectsContext`, `GetRegisteredProjects` |
| Types | ~50 | 0 | `contextData`, `skillContentData`, `RegisteredProject` |

**Source:** `pkg/spawn/context.go:1-1495`, function listing via grep

**Significance:** The embedded templates (DefaultSynthesisTemplate + DefaultFailureReportTemplate + their helpers) total ~335 lines and are completely independent of the generation logic. The utility functions (~240 lines) serving area detection, server context, and registered projects are loosely coupled.

---

### Finding 3: stats_cmd.go has clean type/aggregation/output separation

**Evidence:** Function-level analysis of 1490 lines reveals:

| Domain | Lines | Functions | Key Functions |
|--------|-------|-----------|---------------|
| Types & Constants | ~260 | 3 | 17 struct types (`StatsReport`, `StatsSummary`, `SkillStatsSummary`, `VerificationStats`, etc.), `SkillCategory` const, `coordinationSkills` map, `getSkillCategory` |
| Command Setup & Entry | ~80 | 3 | `statsCmd` cobra command, `init()`, `runStats()`, `getEventsPath()`, `parseEvents()` |
| Aggregation Logic | ~790 | 1 | `aggregateStats()` — single 790-line function doing all event correlation |
| Text Output | ~355 | 1 | `outputStatsText()` — single 355-line function rendering all text output |
| JSON Output | ~10 | 1 | `outputStatsJSON()` |

**Source:** `cmd/orch/stats_cmd.go:1-1490`, function listing via grep

**Significance:** `aggregateStats()` at 790 lines is by far the single largest function across all three files. It handles 12+ event types in a single switch statement. The text output function is similarly monolithic at 355 lines. Both are extraction candidates, and `aggregateStats()` should ideally be decomposed further.

---

## Synthesis

**Key Insights:**

1. **Model parsing is the best kbcontext.go extraction** — The model parsing + staleness checking domain (~430 lines) is self-contained: it reads files, parses markdown, checks git history. Zero coupling to the query/format logic. Extracting to `kbmodel.go` brings kbcontext.go to ~1066 lines.

2. **Embedded templates are dead weight in context.go** — `DefaultSynthesisTemplate` (122 lines) and `DefaultFailureReportTemplate` (83 lines) plus their 6 helper functions are string constants with simple file I/O. Extracting to `templates.go` brings context.go to ~1160 lines. A second extraction of utility functions to `context_util.go` brings it to ~920 lines.

3. **stats_cmd.go needs structural decomposition, not just file splitting** — The 790-line `aggregateStats()` function should be broken into domain-specific aggregators. Extracting types to `stats_types.go` (~260 lines) and text output to `stats_output.go` (~365 lines) brings stats_cmd.go to ~865 lines. But the aggregation function itself should be refactored in the output file or a separate follow-up.

**Answer to Investigation Question:**

All three files can be extracted using the established patterns from `.kb/guides/code-extraction-patterns.md`. The work is sequenced by coupling risk (lowest first):

1. **stats_cmd.go** (lowest risk) — Types have zero logic, output is pure formatting
2. **kbcontext.go** (medium risk) — Model parsing is self-contained
3. **context.go** (highest risk) — Templates + utilities touch the most shared state

---

## Structured Uncertainty

**What's tested:**

- ✅ Line counts verified via `wc -l` (1496, 1495, 1490)
- ✅ Function boundaries verified via `grep -n '^func '` on all three files
- ✅ Full file content read and analyzed for coupling
- ✅ Existing spawn package file structure checked (43 files)
- ✅ Test file sizes checked (context_test.go 2960, kbcontext_test.go 1527, stats_test.go 1218)

**What's untested:**

- ⚠️ Exact line counts after extraction (estimates based on function analysis)
- ⚠️ Whether context_test.go can be cleanly split (2960 lines, may need extraction too)
- ⚠️ Whether aggregateStats decomposition into sub-functions compiles cleanly

**What would change this:**

- If context.go's utility functions have hidden coupling to template data, the context_util.go extraction would need different boundaries
- If kbcontext_test.go has shared test helpers, test extraction may require shared_test.go

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Extract all three files following established patterns | implementation | Stays within .kb/guides/code-extraction-patterns.md established workflow |
| Sequence: stats → kbcontext → context | implementation | Risk ordering within standard refactoring |

### Recommended Approach: Parallel Three-File Extraction

**Why this approach:**
- All three files have clean domain boundaries (no ambiguous code)
- Follows proven extraction patterns (13 prior extractions documented)
- Each extraction is independent — can be parallelized as separate issues

**Trade-offs accepted:**
- Test files may also need extraction (deferred to follow-up)
- `aggregateStats()` 790-line function not decomposed (separate refactoring task)

**Implementation sequence:**

#### Issue 1: Extract stats_cmd.go types → stats_types.go (~260 lines)
- Move all 17 struct types + SkillCategory const + coordinationSkills map + getSkillCategory
- **Result:** stats_cmd.go 1490 → ~1230 lines
- **Risk:** None — pure type definitions

#### Issue 2: Extract stats_cmd.go text output → stats_output.go (~365 lines)
- Move `outputStatsText()` + `outputStatsJSON()` + `truncateSkill()`
- **Result:** stats_cmd.go ~1230 → ~865 lines
- **Risk:** Low — output functions only read StatsReport, don't modify it

#### Issue 3: Extract kbcontext.go model parsing → kbmodel.go (~430 lines)
- Move: `formatModelMatchForSpawn`, `extractModelSectionsForSpawn`, `collectMarkdownHeadings`, `parseMarkdownHeading`, `normalizeHeading`, `extractSectionByHeading`, `truncateModelSection`, `indentBlock`, `extractCodeRefs`, `hasValidFileExtension`, `extractLastUpdated`, `checkModelStaleness`, `hasInjectedModelContent`, `DetectCrossRepoModel`, `modelSpawnSections` type, `markdownHeading` type, `maxModelSectionChars` const
- **Result:** kbcontext.go 1496 → ~1066 lines
- **Risk:** Low — model parsing only uses `os`, `strings`, `filepath`, `context`, `exec` (no spawn-internal deps)

#### Issue 4: Extract context.go embedded templates → templates.go (~335 lines)
- Move: `DefaultSynthesisTemplate`, `DefaultFailureReportTemplate`, `EnsureSynthesisTemplate`, `EnsureFailureReportTemplate`, `WriteFailureReport`, `generateFailureReport`
- **Result:** context.go 1495 → ~1160 lines
- **Risk:** Low — templates are string constants with simple file I/O

#### Issue 5: Extract context.go utilities → context_util.go (~240 lines)
- Move: `DetectAreaFromTask`, `GenerateServerContext`, `GetClusterSummary`, `GenerateRegisteredProjectsContext`, `GetRegisteredProjects`, `RegisteredProject` type, `ValidateBeadsIDConsistency`, `extractProjectPrefix`
- **Result:** context.go ~1160 → ~920 lines
- **Risk:** Medium — `DetectAreaFromTask` uses beads client, `GenerateServerContext` uses config+tmux packages

### Final Size Estimates

| File | Before | After | Reduction |
|------|--------|-------|-----------|
| stats_cmd.go | 1490 | ~865 | ~625 (-42%) |
| kbcontext.go | 1496 | ~1066 | ~430 (-29%) |
| context.go | 1495 | ~920 | ~575 (-38%) |

### New Files Created

| File | Est. Lines | Content |
|------|-----------|---------|
| cmd/orch/stats_types.go | ~260 | Stats report type definitions |
| cmd/orch/stats_output.go | ~365 | Text and JSON output formatting |
| pkg/spawn/kbmodel.go | ~430 | Model parsing, markdown extraction, staleness checking |
| pkg/spawn/templates.go | ~335 | Embedded template constants and template file management |
| pkg/spawn/context_util.go | ~240 | Area detection, server context, registered projects, beads ID validation |

### Things to watch out for:
- ⚠️ `context_test.go` at 2960 lines will likely need extraction after context.go is split
- ⚠️ `kbcontext_test.go` at 1527 lines is already above threshold and may need splitting alongside kbmodel.go extraction
- ⚠️ `aggregateStats()` at 790 lines deserves its own refactoring issue (decompose the switch statement into per-event-type handlers)

---

## References

**Files Examined:**
- `pkg/spawn/kbcontext.go` (1496 lines) - Full read, function analysis
- `pkg/spawn/context.go` (1495 lines) - Full read, function analysis
- `cmd/orch/stats_cmd.go` (1490 lines) - Full read, function analysis
- `pkg/spawn/kbcontext_test.go` (1527 lines) - Size check
- `pkg/spawn/context_test.go` (2960 lines) - Size check
- `cmd/orch/stats_test.go` (1218 lines) - Size check, function listing
- `.kb/guides/code-extraction-patterns.md` - Established extraction workflow

**Commands Run:**
```bash
wc -l pkg/spawn/kbcontext.go pkg/spawn/context.go cmd/orch/stats_cmd.go
grep -n '^func ' pkg/spawn/kbcontext.go
grep -n '^func ' pkg/spawn/context.go
grep -n '^func ' cmd/orch/stats_cmd.go
wc -l pkg/spawn/*.go | sort -n
```

**Related Artifacts:**
- **Guide:** `.kb/guides/code-extraction-patterns.md` - Established extraction workflow
- **Recent extraction:** daemon.go 1559→715 - Successful reference case

---

## Investigation History

**2026-03-09:** Investigation started
- Initial question: Design extraction plan for three near-critical hotspot files
- Context: Files are 4-10 lines from the 1500-line critical threshold

**2026-03-09:** Function-level analysis complete
- All three files analyzed for responsibility boundaries
- Clean extraction domains identified in each file

**2026-03-09:** Investigation completed
- Status: Complete
- Key outcome: 5 extraction targets identified across 3 files, reducing each by 29-42%
