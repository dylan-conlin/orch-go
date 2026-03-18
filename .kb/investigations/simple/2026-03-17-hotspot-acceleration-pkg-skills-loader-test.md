---
TLDR: pkg/skills/loader_test.go (710→399 lines) extracted section filter concern into filter.go (138 lines) + filter_test.go (317 lines). Growth was organic (single feature commit), not accretion. All 19 tests pass post-extraction.
Status: Complete
Question: Is pkg/skills/loader_test.go growth a real hotspot requiring extraction, and what's the optimal split?
---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Finding 1: File Structure Analysis

**What I tested:** Analyzed the 710-line test file against the 334-line production code to identify natural seam boundaries.

**What I observed:**

Production code (`loader.go`, 334 lines) has two distinct concerns:
1. **Skill loading/discovery** (lines 1-190): `Loader`, `FindSkillPath`, `LoadSkillContent`, `LoadSkillWithDependencies`, `stripFrontmatter`, `ParseSkillMetadata`
2. **Section filtering** (lines 192-334): `SectionFilter`, `FilterSkillSections`, `parseSectionAttrs`, `sectionMatches`, `LoadSkillFiltered`

Test code (`loader_test.go`, 710 lines) mirrors this split:
1. **Loader tests** (lines 10-356, ~346 lines): `TestFindSkillPath`, `TestLoadSkillContent`, `TestParseSkillMetadata*`, `TestLoadSkillWithDependencies*`
2. **Filter tests** (lines 358-710, ~352 lines): `TestFilterSkillSections_*`, `TestParseSectionAttrs`, `TestSectionFilter_IsEmpty`, `TestStripFrontmatter`

## Finding 2: Growth Source Analysis

**What I tested:** `git log --oneline --follow -20 -- pkg/skills/loader_test.go`

**What I observed:** Three commits, all feature additions:
- `2c71a00c8` — progressive skill disclosure (section filtering) — **this is the +311 line commit**
- `c0004de68` — dependency resolution
- `ef51a8908` — initial spawn command with skill loading

The growth is organic feature addition, not accretion. The section filtering feature was a single deliberate addition that brought its own test surface.

## Finding 3: Extraction Plan Assessment

**What I tested:** Mapped function-to-file relationships to evaluate clean split feasibility.

**What I observed:**

**Proposed split:**
| File | Contents | Lines (est.) |
|---|---|---|
| `loader.go` | Loader, FindSkillPath, LoadSkillContent, LoadSkillWithDependencies, ParseSkillMetadata, stripFrontmatter | ~191 |
| `filter.go` | SectionFilter, FilterSkillSections, parseSectionAttrs, sectionMatches, LoadSkillFiltered | ~143 |
| `loader_test.go` | TestFindSkillPath, TestLoadSkillContent, TestParseSkillMetadata*, TestLoadSkillWithDependencies*, TestStripFrontmatter | ~358 |
| `filter_test.go` | TestFilterSkillSections_*, TestParseSectionAttrs, TestSectionFilter_IsEmpty | ~310 |

**Cross-dependencies:** `LoadSkillFiltered` in filter.go calls `l.LoadSkillWithDependencies` (Loader method), so it stays as a method on `*Loader`. This is fine — it bridges the two files within the same package.

**Note on TestStripFrontmatter:** `stripFrontmatter` is used by the Loader's `LoadSkillWithDependencies`, so its test belongs in `loader_test.go` despite being at the end of the file.

## Test Performed

```bash
go test ./pkg/skills/ -v -count=1
```

All 19 tests pass. No test interdependencies — each test creates its own temp directory. Extraction will not break any tests.

## Conclusion

**Verdict: False positive hotspot — growth is organic, but extraction is still beneficial.**

The +311 lines came from a single feature addition (section filtering, commit `2c71a00c8`). The file isn't accreting randomly — it grew because a coherent feature was added to a file that already housed a different concern.

However, the file is at 710 lines and the production code has a clean seam. Extracting now (before either concern grows further) is the right preventive move.

**Recommended extraction:**
- `filter.go` + `filter_test.go` — section filtering concern (~143 + ~310 lines)
- `loader.go` + `loader_test.go` — skill loading concern (~191 + ~358 lines)
- Both files stay well under the 800-line advisory threshold
- No API changes needed — same package, same exports

**Routing:** Extraction was mechanical file splitting — no architectural decisions required. Performed in-session.

## Finding 4: Extraction Performed

**What I tested:** Extracted section filtering code and tests into separate files, then ran full test suite and build.

**What I observed:**

Post-extraction file sizes:
| File | Lines |
|---|---|
| `loader.go` | 199 |
| `loader_test.go` | 399 |
| `filter.go` | 138 |
| `filter_test.go` | 317 |

- `go test ./pkg/skills/ -v -count=1` — all 19 tests pass
- `go build ./...` — clean build, no breakage

`LoadSkillFiltered` remains a method on `*Loader` in `loader.go` (7 lines) — it bridges the loader and filter by calling both `LoadSkillWithDependencies` and `FilterSkillSections`. This keeps the API surface unchanged.

## D.E.K.N. Summary

- **Delta:** Extracted section filter concern from loader.go/loader_test.go into filter.go (138 lines) + filter_test.go (317 lines). loader_test.go reduced from 710 to 399 lines.
- **Evidence:** All 19 tests pass. Clean build. No API changes — same package, same exports.
- **Knowledge:** Growth was organic (single feature commit `2c71a00c8`, not accretion). The 710-line file had a clean seam — two non-overlapping concerns (loading vs filtering) sharing one file. Preventive extraction before either concern grows further.
- **Next:** Close. No follow-up needed.
