# Session Synthesis

**Agent:** og-arch-determine-optimal-file-10mar-1f8d
**Issue:** orch-go-yeidp
**Duration:** 2026-03-10
**Outcome:** success

---

## Plain-Language Summary

Analyzed 13 extraction commits, 20 files for cross-cutting concern density, and 9 satellite files for post-extraction stability to determine optimal file size targets. The key finding is that extracted satellite files (100-300 lines) receive zero additional commits after creation — all new feature work lands in the residual parent file. This means the extraction TARGET matters more than the extraction TRIGGER. Residuals left at 600-700 lines re-cross 800 within weeks; residuals at 200-400 lines stay stable. Recommending a three-number framework: 200 (ideal satellite), 400 (max residual), 800 (extraction trigger). The Phase 2 file list in the harness health plan is stale (7 of 10 already extracted) and has been updated with 12 current >800 files and specific per-file targets.

---

## TLDR

Determined optimal post-extraction file size target is 200-400 lines (not "under 800"). Satellite files at this size have zero re-accretion while residuals above 600 re-cross 800 within weeks. Updated extract-patterns model with three-number framework and produced revised Phase 2 extraction targets for 12 current bloated files.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-10-design-determine-optimal-file-size-targets.md` - Full investigation with 6 findings, updated Phase 2 file list

### Files Modified
- `.kb/models/extract-patterns/model.md` - Updated "800-Line Gate" section to "Three-Number Framework (200/400/800)" with empirical evidence; added evolution entry for Mar 10 findings

### Commits
- (pending)

---

## Evidence (What Was Observed)

- 9 satellite files (spawn_helpers.go, spawn_dryrun.go, work_cmd.go, daemon_commands.go, daemon_handlers.go, context_util.go, templates.go, client_transcript.go, scheduler.go) have exactly 0 post-extraction commits — verified via `git log --oneline`
- 800+ line files average 13.2 commits/file/30 days; daemon.go alone has 54 commits/month
- Cross-cutting concern sampling (n=20): <200 lines = 2.8 avg concerns, 300-600 = 5.8, 800+ = 5.9
- Residuals under 400 remain stable: doctor.go (269), extraction.go (280), session.go (121)
- Residuals over 600 re-accrete: daemon.go (715→896), context.go (~600→895), client.go (~800→1040)
- 76% of source files cluster at 100-400 lines (135 + 136 files out of 356 total)
- Phase 2 plan list is stale: 7 of 10 original targets already extracted

### Tests Run
```bash
# No code changes to test — investigation-only deliverable
# Verified all file counts via wc -l across full codebase
# Verified all git histories via git log for extraction and satellite files
```

---

## Architectural Choices

### Three-Number Framework vs Lowering the Trigger
- **What I chose:** Keep 800 as trigger, add 400 as target, add 200 as satellite ideal
- **What I rejected:** Lowering trigger to 600 (would create excessive extraction churn — 47 files in 400-600 range)
- **Why:** The trigger is correctly calibrated; the missing piece was the TARGET. Data shows the failure mode is "extract to 700 and re-accrete" not "trigger too late at 800"
- **Risk accepted:** More aggressive extraction means more files per extraction (3-5 satellites), increasing total file count

### Priority by Commit Frequency vs by File Size
- **What I chose:** Prioritize daemon.go (896 lines, 54 commits/month) over beads/client.go (1115 lines, 10 commits/month)
- **What I rejected:** Prioritizing by absolute file size (extract biggest first)
- **Why:** High commit frequency means the file will re-accrete fastest if not extracted aggressively; the biggest file that nobody touches is a lower priority
- **Risk accepted:** The biggest files remain bloated longer, but the highest-churn files get the most benefit from extraction

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-10-design-determine-optimal-file-size-targets.md` - Complete analysis of optimal extraction targets

### Decisions Made
- Decision: Three-number framework (200/400/800) replaces two-number system (800/1500) because empirical data shows residuals >600 re-accrete while residuals <400 remain stable

### Constraints Discovered
- Satellite files resist re-accretion completely (0 commits post-extraction) — this is the fundamental mechanism that makes extraction work
- "Feature gravity" — agents default to modifying the file they already have loaded, so all new work lands in the residual parent
- The concern accumulation threshold is ~300 lines, above which files accumulate 4-7 cross-cutting concerns

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification specification.

Key outcomes:
- Investigation file complete with 6 findings and structured uncertainty
- Extract-patterns model updated with three-number framework
- Phase 2 file list updated with 12 current targets and per-file strategies

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Status:** Complete`
- [x] Extract-patterns model updated
- [ ] Ready for `orch complete orch-go-yeidp`

---

## Unexplored Questions

- Whether `pkg/` files have fundamentally different accretion dynamics than `cmd/orch/` files (same package main advantage doesn't apply)
- Whether routing attractors (dedicated sub-packages) would reduce residual re-accretion more effectively than extraction alone
- Whether the 200-400 target holds for non-Go codebases (Svelte, TypeScript)
- 30-day longitudinal validation of whether current extractions hold below 400

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-determine-optimal-file-10mar-1f8d/`
**Investigation:** `.kb/investigations/2026-03-10-design-determine-optimal-file-size-targets.md`
**Beads:** `bd show orch-go-yeidp`
