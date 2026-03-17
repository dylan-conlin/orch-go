---
Status: Complete
Question: Is pkg/orch/spawn_pipeline.go a genuine hotspot requiring extraction?
---

**TLDR:** spawn_pipeline.go is a false positive hotspot — the +506 lines/30d metric measures the extraction event itself (created Mar 1 from a larger file), not organic growth. File has been stable at ~463-480 lines since creation.

## D.E.K.N. Summary

- **Delta:** The hotspot alert is a false positive. spawn_pipeline.go was *created* on 2026-03-01 via `592e0984c refactor: extract extraction.go into 8 domain files` at 480 lines. Its net change over 10 subsequent commits is -17 lines (480→463). The +506 lines/30d metric counts the creation itself as growth.
- **Evidence:** `git log --follow` shows the file was created 2026-03-01 at 480 lines. Line-by-line commit history shows it peaked at 494 lines (2026-03-03) and was reduced to 463 by 2026-03-16. No single function exceeds 70 lines (BuildSpawnConfig is the largest at ~85 lines including the struct literal).
- **Knowledge:** The hotspot detection tool counts file creation as "growth" — this produces false positives for files born from extractions. The file is well-structured with clear responsibility boundaries documented in its header comment (lines 2-10).
- **Next:** No action needed. File is healthy at 463 lines, well below the 1500-line threshold. Recommend improving hotspot detection to distinguish creation-from-extraction vs organic growth.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Findings

### Finding 1: File was born from extraction, not grown organically

**What I tested:** `git log --format="%h %ad %s" --date=short --diff-filter=A -- pkg/orch/spawn_pipeline.go`

**What I observed:** File was created on 2026-03-01 by commit `592e0984c refactor: extract extraction.go into 8 domain files` at 480 lines. This was a planned extraction that split a larger file into 8 domain-specific files (spawn_pipeline, spawn_preflight, spawn_types, etc.), producing 1509 insertions from 1799 deletions — a net reduction.

### Finding 2: File has been stable/shrinking since creation

**What I tested:** Tracked line count across all 10 commits touching the file.

**What I observed:**
| Date | Hash | Lines |
|---|---|---|
| 2026-03-01 | 592e0984c | 480 (created) |
| 2026-03-02 | b350208dc | 481 (+1) |
| 2026-03-03 | 31e1f24e8 | 482 (+1) |
| 2026-03-03 | 8c20383df | 483 (+1) |
| 2026-03-03 | abac3214d | 494 (+11) |
| 2026-03-05 | d95bca8fb | 474 (-20) |
| 2026-03-11 | 0f397a147 | 477 (+3) |
| 2026-03-11 | e31b34ca0 | 479 (+2) |
| 2026-03-12 | 8b0603bae | 480 (+1) |
| 2026-03-16 | 185dd2f06 | 463 (-17) |

Net change: -17 lines. The file has been *shrinking* over its lifetime.

### Finding 3: File structure is healthy

**What I tested:** Read the full 463-line file and analyzed function responsibilities.

**What I observed:** Six exported functions, each with clear single responsibility:
- `CheckAndAutoSwitchAccount()` — account auto-switching (63 lines)
- `ResolveProjectDirectory()` — thin delegation (3 lines)
- `LoadSkillAndGenerateWorkspace()` — skill loading + workspace naming (43 lines)
- `ResolveAndValidateModel()` — model resolution + flash validation (35 lines)
- `ResolveSpawnSettings()` — settings resolution + warning display (34 lines)
- `ExtractBugReproInfo()` — bug repro extraction (13 lines)
- `BuildSpawnConfig()` — config struct construction (85 lines, mostly field mapping)
- `ValidateAndWriteContext()` — context validation + Phase 1 (66 lines)
- Two unexported helpers: `dirExists`, `checkWorkspaceExists`, `truncate`

No function exceeds 85 lines. No obvious extraction targets — each function has a coherent responsibility.

### Finding 4: pkg/orch already has good domain separation

**What I tested:** `wc -l pkg/orch/*.go | sort -rn`

**What I observed:** The package already has 21 files with the largest being spawn_modes.go at 532 lines. The extraction that created spawn_pipeline.go was part of a well-executed refactor that split responsibilities across: spawn_types.go (87), spawn_inference.go (130), spawn_preflight.go (173), spawn_kb_context.go (171), spawn_backend.go (117), spawn_beads.go (194), spawn_design.go (86). Total package: 3713 lines across 21 files — reasonable distribution.

## Test Performed

Git history analysis (primary evidence) confirms file was created from extraction and has been stable/shrinking. No code changes needed.

## Conclusion

**False positive.** The +506 lines/30d metric reflects the file's creation from extraction on 2026-03-01, not organic growth. Since creation, the file has a net change of -17 lines (shrinking). At 463 lines with clear function boundaries and good domain separation in the parent package, no extraction is needed.
