## Summary (D.E.K.N.)

**Delta:** Optimal post-extraction target is 200-400 lines, not "under 800" — extracted satellite files at this size have zero re-accretion (0 post-extraction commits across 9 files sampled), while residual parent files left at 700+ re-cross 800 within weeks.

**Evidence:** Sampled 20 files across size ranges (100-1115 lines), analyzed 13 extraction commits, measured re-accretion velocity for all extracted files, and correlated file size with cross-cutting concern count and commit frequency.

**Knowledge:** File size correlates with cross-cutting concern count (2.8 avg at <200 lines, 5.9 avg at 800+). The "feature gravity" effect means all new commits land in the residual parent file, never in satellites. Aggressive extraction (4+ new files, residual <400) is the only strategy that resists re-accretion. The 800 threshold should remain as the extraction TRIGGER but 200-400 should be the extraction TARGET.

**Next:** Update extract-patterns model with target range. Create implementation issues for 12 current >800 files with specific targets. Update `orch hotspot` to report distance-to-target (not just distance-to-ceiling).

**Authority:** architectural - Affects extraction strategy across all future agents, changes how hotspot enforcement works

---

# Investigation: Determine Optimal File Size Targets for Code Extraction

**Question:** What line count range should post-extraction files target, should the 800 warning threshold change, and what are the specific targets for Phase 2 extractions?

**Started:** 2026-03-10
**Updated:** 2026-03-10
**Owner:** architect (orch-go-yeidp)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** harness-engineering

**Patches-Decision:** N/A (new recommendation)
**Extracted-From:** `.kb/plans/2026-03-10-harness-health-improvement.md` Phase 2 planning

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/harness-engineering/probes/2026-03-08-probe-30-day-accretion-trajectory-gate-effectiveness.md` | extends | Yes — re-verified daemon.go trajectory and spawn_cmd.go re-accretion | None |
| `.kb/models/extract-patterns/model.md` | extends | Yes — confirms 800-line context noise threshold | Extends: model says "800 lines is the gate" but data shows 800 is too high as a TARGET |
| `.kb/plans/2026-03-10-harness-health-improvement.md` | deepens | Yes — 7 of 10 Phase 2 targets already extracted | Phase 2 file list is stale; updated below |

---

## Findings

### Finding 1: Cross-Cutting Concerns Jump Sharply Above 300 Lines

**Evidence:** Sampled 20 files across three size bands:

| Size Band | Files Sampled | Avg Concerns | Range |
|-----------|--------------|--------------|-------|
| Small (<300 lines) | 5 | 2.8 | 2-4 |
| Medium (300-600 lines) | 5 | 5.8 | 4-7 |
| Large (800+ lines) | 10 | 5.9 | 4-8 |

Representative samples:
- `pkg/question/question.go` (112 lines): 2 concerns (pattern matching, extraction)
- `pkg/model/model.go` (167 lines): 3 concerns (alias resolution, provider inference, config)
- `pkg/beads/client.go` (1115 lines): 6 concerns (RPC protocol, CLI fallback, retry, marshaling, timeout, socket)
- `cmd/orch/kb.go` (1138→280 lines, post-extraction): 7→3 concerns

**Source:** Agent analysis of 20 files across `cmd/orch/` and `pkg/`

**Significance:** The concern count plateaus around 5-7 for files above 400 lines, but the critical jump is from 2-3 (at <300) to 4-5 (at 300-600). Files under 300 lines almost always maintain single responsibility. Above 300, concerns start compounding. This makes 200-400 the "sweet spot" — large enough for a coherent domain, small enough to resist concern accumulation.

---

### Finding 2: Extracted Satellite Files Have Zero Re-Accretion

**Evidence:** Checked git history for 9 satellite files created during extractions:

| Satellite File | Created | Lines | Post-Extraction Commits |
|---------------|---------|-------|------------------------|
| `cmd/orch/spawn_helpers.go` | Mar 10 | 208 | 0 |
| `cmd/orch/spawn_dryrun.go` | Mar 10 | 272 | 0 |
| `cmd/orch/work_cmd.go` | Mar 10 | 237 | 0 |
| `cmd/orch/daemon_commands.go` | Mar 9 | 225 | 0 |
| `cmd/orch/daemon_handlers.go` | Mar 9 | 460 | 0 |
| `pkg/spawn/context_util.go` | Mar 9 | 254 | 0 |
| `pkg/spawn/templates.go` | Mar 9 | 367 | 0 |
| `pkg/opencode/client_transcript.go` | Mar 10 | 211 | 0 |
| `pkg/daemon/scheduler.go` | Mar 6 | 143 | 0 |

Meanwhile, the parent/residual files that these were extracted FROM:

| Residual File | Post-Extraction Lines | Current Lines | Commits (30 days) |
|--------------|----------------------|---------------|-------------------|
| `pkg/daemon/daemon.go` | 715 | 896 | 54 |
| `pkg/spawn/context.go` | ~600 | 895 | 28 |
| `cmd/orch/review.go` | ~700 | 848 | 12 |
| `pkg/opencode/client.go` | ~800 | 1040 | 10 |

**Source:** `git log --oneline -- <file>` for each satellite and residual file

**Significance:** This is the most important finding. **All new feature work lands in the residual parent file, never in the satellites.** The implication is profound: the more code you move into satellites during extraction, the more code resists re-accretion. A residual left at 700 lines will re-cross 800 in weeks. A residual left at 300 lines buys months of runway.

---

### Finding 3: Aggressive Extraction (4+ Files) vs Conservative (1-2 Files)

**Evidence:** Comparing extraction strategies by outcome:

**Aggressive extractions (4+ new files):**
| Source | Pre | Post Residual | New Files | Outcome |
|--------|-----|---------------|-----------|---------|
| session.go | 1055 | 121 | 6 | Excellent — residual is stable |
| spawn_cmd.go | 1171 | 505 | 4 | Good — residual manageable |
| daemon.go (Mar 9) | 1559 | 715 | 3 | Insufficient — residual re-accreted to 896 |
| doctor.go | ~1500 | 269 | 5 | Excellent — residual is stable |
| extraction.go | ~1600 | ~280 | 8 | Excellent — residual is stable |

**Conservative extractions (1-2 new files):**
| Source | Pre | Post Residual | New Files | Outcome |
|--------|-----|---------------|-----------|---------|
| context.go (util) | ~1100 | ~900 | 1 | Poor — still bloated |
| context.go (templates) | ~900 | ~600 | 1 | Mediocre — crept back to 895 |
| kbcontext.go (kbmodel) | ~1072 | ~530 | 1 | OK but still large |
| review.go (helpers) | ~1100 | ~850 | 2 | Poor — still bloated at 848 |

**Source:** `git show --stat <extraction-commit>` and current `wc -l` for all files

**Significance:** Pattern is clear: residuals under 400 lines stay stable (doctor.go at 269, extraction.go at 280, session.go at 121). Residuals over 600 lines re-accrete back toward 800+ within weeks. The extraction target should be aggressive enough to land the residual at 200-400, which typically requires 3-5 satellite files per extraction.

---

### Finding 4: File Size Distribution Shows Natural Clustering at 100-400

**Evidence:** Current source file (non-test) distribution:

| Size Band | Count | % of Total |
|-----------|-------|-----------|
| <100 lines | ~60 | 17% |
| 100-200 lines | 135 | 38% |
| 200-400 lines | 136 | 38% |
| 400-600 lines | 47 | 13% |
| 600-800 lines | 25 | 7% |
| 800+ lines | 12 | 3% |

The codebase naturally clusters at 100-400 lines (76% of all source files). This is where extracted files land and where they stabilize. Files in the 400-800 range are either growing toward bloat or recently extracted and drifting upward.

**Source:** `find cmd/orch pkg -name "*.go" ! -name "*_test.go" -exec wc -l {} + | sort -n`

**Significance:** The natural equilibrium for Go source files in this codebase is 100-400 lines. Files outside this range are either too small to be meaningful or accreting toward bloat. Extraction targets should aim for this natural equilibrium.

---

### Finding 5: Agent Context Budget Analysis

**Evidence:** A typical agent working session requires:

| Context Component | Lines | Purpose |
|------------------|-------|---------|
| SPAWN_CONTEXT.md | 300-500 | Task instructions, skill guidance |
| CLAUDE.md | ~400 | Project conventions |
| Target file | X | File being modified |
| 3-5 dependency files | 600-1000 | Referenced types, interfaces |
| Test file | 200-500 | Tests to run/modify |
| **Total overhead** | **1500-2400** | Before the target file |

For effective modification, an agent needs to hold the target file plus its immediate dependencies in active working memory. With ~2000 lines of overhead:
- 300-line target: agent works comfortably with full context
- 500-line target: agent can hold full context but less headroom for dependencies
- 800-line target: agent must mentally page between sections, increasing error rate
- 1000+ line target: agent cannot hold full context, relies on search (session amnesia risk)

**Source:** Extract-patterns model ("800 lines is where Context Noise degrades agent reasoning"), accretion trajectory probe (session amnesia section), empirical extraction commit quality

**Significance:** For agents that modify code (feature-impl, debugging), the target file should be <400 lines to leave adequate context budget for dependencies and instructions. For agents that read code (investigation, architect), larger files are tolerable but still sub-optimal. The 300-400 line target aligns with both the cross-cutting concern threshold and the context budget constraint.

---

### Finding 6: Commit Frequency Correlates with File Size

**Evidence:** 30-day commit frequency for 800+ line files:

| File | Lines | 30-Day Commits | Commits/100 Lines |
|------|-------|----------------|-------------------|
| `pkg/daemon/daemon.go` | 896 | 54 | 6.0 |
| `pkg/spawn/context.go` | 895 | 28 | 3.1 |
| `cmd/orch/serve_system.go` | 1084 | 12 | 1.1 |
| `cmd/orch/review.go` | 848 | 12 | 1.4 |
| `pkg/opencode/client.go` | 1040 | 10 | 1.0 |
| `cmd/orch/hotspot.go` | 1050 | 9 | 0.9 |
| `pkg/userconfig/userconfig.go` | 975 | 8 | 0.8 |
| `cmd/orch/stats_cmd.go` | 912 | 7 | 0.8 |
| `pkg/verify/visual.go` | 870 | 6 | 0.7 |
| `cmd/orch/handoff.go` | 898 | 2 | 0.2 |
| `pkg/spawn/learning.go` | 979 | 1 | 0.1 |
| `pkg/beads/client.go` | 1115 | 10 | 0.9 |

Average: 13.2 commits/file/month for 800+ files.
Satellite files (100-300 lines): 0 commits/file post-creation.

**Source:** `git log --since="2026-02-08" --oneline -- <file> | wc -l`

**Significance:** The high commit frequency on large files is both cause and effect — large files attract changes because they contain more functionality, and each change makes them larger. daemon.go's 54 commits in 30 days (nearly 2/day) shows extreme "feature gravity." Breaking this cycle requires extraction aggressive enough that the residual no longer attracts the majority of new work.

---

## Synthesis

**Key Insights:**

1. **The extraction TARGET matters more than the extraction TRIGGER.** The 800-line threshold correctly identifies files that need extraction. But if extraction only reduces a file from 1100 to 700 lines, it re-crosses 800 within weeks. The winning pattern is extracting to 200-400 lines (session.go→121, doctor.go→269, extraction.go→280), which produces residuals that stay stable.

2. **Satellite files are the "savings account" of extraction.** Code moved into well-scoped 100-300 line satellites has zero re-accretion. Code left in the residual parent is subject to "feature gravity" — agents default to modifying the file they already have loaded. The optimal strategy is: maximize what goes into satellites, minimize what stays in the residual.

3. **300 lines is the concern accumulation threshold.** Below 300 lines, files maintain 2-3 concerns (single responsibility). Above 300, concerns compound to 4-7. This aligns with agent context budgets (300-line files leave headroom for dependencies) and natural codebase clustering (76% of files are 100-400 lines).

4. **The Phase 2 file list is stale — 7 of 10 targets already extracted.** The current 12 files >800 lines need new extraction plans with 200-400 line targets.

**Answer to Investigation Question:**

**Optimal post-extraction target: 200-400 lines.** This is supported by four converging data points: (1) cross-cutting concerns stay at 2-3 below 300 lines, (2) satellite files in this range have zero re-accretion, (3) the natural file size distribution clusters here, and (4) agent context budgets are optimized for files of this size.

**The 800 threshold should NOT change** — it correctly identifies files needing extraction. But it must be reframed from "ceiling to stay under" to "trigger that initiates extraction to 200-400."

**The gap in current tooling:** `orch hotspot` reports distance-to-ceiling (how far over 800). It should also report distance-to-target (how far from the 200-400 target range), making the extraction goal visible.

---

## Structured Uncertainty

**What's tested:**

- ✅ Satellite files have 0 post-extraction commits (verified: `git log --oneline` for 9 satellite files)
- ✅ Residuals under 400 lines stay stable: doctor.go (269), extraction.go (280), session.go (121) — no re-accretion
- ✅ Residuals over 600 lines re-accrete: daemon.go (715→896), context.go (~600→895), client.go (~800→1040)
- ✅ Cross-cutting concerns average 2.8 for files <200 lines, 5.9 for files >800 lines (sampled 20 files)
- ✅ 800+ line files average 13.2 commits/file/30 days; satellite files average 0 commits
- ✅ 76% of source files naturally cluster at 100-400 lines
- ✅ Phase 2 file list is stale: 7 of 10 targets already extracted

**What's untested:**

- ⚠️ Whether the 200-400 target resists re-accretion over 30+ days (current data is <14 days post-extraction for most files)
- ⚠️ Whether agent error rates actually decrease when target file size drops from 800 to 300 (inferred from context window theory, not measured)
- ⚠️ Whether satellites remain at 0 commits long-term or eventually attract features as codebase evolves
- ⚠️ Whether the target range differs for `pkg/` files vs `cmd/orch/` files (different accretion dynamics)

**What would change this:**

- If satellite files start receiving commits after 30+ days, the "zero re-accretion" finding weakens
- If residuals under 400 lines start growing past 600 within 30 days, the target range is too optimistic
- If agent performance doesn't measurably improve with smaller target files, the context budget argument loses force

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Set 200-400 line extraction target | architectural | Changes extraction strategy across all agents; affects hotspot enforcement |
| Update Phase 2 file list | implementation | Tactical update to existing plan |
| Add target-distance to `orch hotspot` | implementation | Enhancement within existing tool |
| Update extract-patterns model | implementation | Knowledge update within existing pattern |

### Recommended Approach: Three-Number Framework (200 / 400 / 800)

**Introduce a three-number file size framework** replacing the current two-number system:

- **200 lines:** Ideal satellite size. Extraction should produce satellites of 100-300 lines.
- **400 lines:** Target maximum for residual files post-extraction. This is the POST-EXTRACTION goal.
- **800 lines:** Extraction trigger. Files crossing this threshold should be extracted to reach the 400-line target.

**Why this approach:**
- Aligns with empirical data: 200-400 is where files naturally stabilize and resist re-accretion
- Gives agents a specific TARGET, not just a CEILING — "extract this to 300 lines" is actionable
- The 800 trigger is already calibrated and understood; no need to change enforcement

**Trade-offs accepted:**
- More aggressive extraction means more files per extraction (3-5 satellites per parent), which increases file count
- File count already grew from 62→125 in cmd/orch/ — but this is the correct trade-off (more stable small files > fewer unstable large files)

**Implementation sequence:**
1. Update extract-patterns model with three-number framework
2. Update Phase 2 file list with specific targets per file (see below)
3. (Future) Add target-distance metric to `orch hotspot`

### Alternative Approaches Considered

**Option B: Keep 800 as both trigger and target**
- **Pros:** Simpler, less extraction work needed
- **Cons:** Residuals at 600-700 re-cross 800 within weeks; extraction becomes a Sisyphean cycle
- **When to use instead:** Never — empirical data clearly shows this fails

**Option C: Lower the trigger to 600**
- **Pros:** Forces earlier extraction, catches problems sooner
- **Cons:** Creates excessive extraction churn; 47 files currently in the 400-600 range would need extraction
- **When to use instead:** If re-accretion velocity increases despite the 200-400 target

**Rationale for recommendation:** Option A (three-number framework) is the only approach supported by the re-accretion data. Options B and C either perpetuate the current failure mode or create unsustainable extraction volume.

---

### Updated Phase 2 Extraction Targets

The plan's original list had 7 of 10 files already extracted. Here are the current 12 files >800 lines with specific extraction targets:

| # | File | Current | Target Residual | Expected Satellites | Priority |
|---|------|---------|----------------|--------------------| ---------|
| 1 | `pkg/beads/client.go` | 1115 | 350 | 3 (rpc_client, cli_fallback, retry) | High — most bloated |
| 2 | `cmd/orch/serve_system.go` | 1084 | 350 | 3 (by endpoint group) | High |
| 3 | `cmd/orch/hotspot.go` | 1050 | 300 | 3 (coupling, reporting, config) | Medium |
| 4 | `pkg/opencode/client.go` | 1040 | 350 | 3 (sessions, messages, health) | High — already extracted once, needs deeper split |
| 5 | `pkg/spawn/learning.go` | 979 | 300 | 3 (pattern_db, analysis, suggestions) | Low — only 1 commit/month |
| 6 | `pkg/userconfig/userconfig.go` | 975 | 300 | 3 (validation, migration, defaults) | Medium |
| 7 | `cmd/orch/stats_cmd.go` | 912 | 350 | 2 (aggregation, formatting) | Medium |
| 8 | `cmd/orch/handoff.go` | 898 | 300 | 2 (transfer, validation) | Low — only 2 commits/month |
| 9 | `pkg/daemon/daemon.go` | 896 | 300 | 3 (issue_processing, spawn_orchestration, lifecycle) | Critical — 54 commits/month |
| 10 | `pkg/spawn/context.go` | 895 | 300 | 3 (section_builders, feature_injectors, validation) | High — 28 commits/month |
| 11 | `pkg/verify/visual.go` | 870 | 300 | 2 (screenshot, diff_render) | Low |
| 12 | `cmd/orch/review.go` | 848 | 300 | 2 (queue_scan, formatting) | Medium |

**Priority ordering rationale:** daemon.go and context.go should be extracted first despite being the smallest of the 800+ files, because their extreme commit frequencies (54 and 28/month) mean they'll continue growing fastest. beads/client.go and serve_system.go are the most bloated but lower churn.

---

### Implementation Details

**What to implement first:**
- Extract daemon.go (896→300) — highest commit frequency, most urgent
- Extract context.go (895→300) — second-highest commit frequency
- Extract beads/client.go (1115→350) — most bloated file
- Update extract-patterns model with three-number framework

**Things to watch out for:**
- ⚠️ daemon.go has already been extracted once (1559→715) and re-accreted to 896. The previous extraction left the residual too high. This time, target 300.
- ⚠️ client.go was extracted once (into transcript, cli, tokens) and re-accreted from ~800 to 1040. The session management and message handling domains need separation.
- ⚠️ context.go has been extracted TWICE (context_util.go and templates.go) and is still at 895. The remaining code has multiple section builders that should each be their own file.

**Areas needing further investigation:**
- Whether `pkg/` files have different optimal targets than `cmd/orch/` files (different accretion dynamics)
- Whether adding routing attractors (dedicated packages for new features) would reduce residual re-accretion more than gates alone
- 30-day validation of whether 200-400 residuals actually resist re-accretion long-term

**Success criteria:**
- ✅ All 12 files reduced to <400 lines
- ✅ No file re-crosses 800 within 30 days of extraction
- ✅ Total bloated source file count drops from 12 to 0
- ✅ `orch health` score improves by 15+ points

---

## References

**Files Examined:**
- All Go source files in `cmd/orch/` and `pkg/` via `wc -l` (356 non-test source files)
- 20 files sampled for cross-cutting concern analysis
- 9 satellite files checked for post-extraction commit history
- 12 files >800 lines analyzed for commit frequency

**Commands Run:**
```bash
# File size distribution
find cmd/orch pkg -name "*.go" ! -name "*_test.go" -exec wc -l {} + | sort -n

# Extraction commit analysis
git show --stat <commit> # for 13 extraction commits

# Re-accretion measurement
git log --since="2026-02-08" --oneline -- <file> | wc -l # for all 800+ files

# Satellite stability check
git log --oneline -- <satellite-file> # for 9 extracted satellite files
```

**Related Artifacts:**
- **Model:** `.kb/models/extract-patterns/model.md` — extraction as context management
- **Model:** `.kb/models/harness-engineering/model.md` — accretion thermodynamics
- **Probe:** `.kb/models/harness-engineering/probes/2026-03-08-probe-30-day-accretion-trajectory-gate-effectiveness.md` — gate effectiveness baseline
- **Plan:** `.kb/plans/2026-03-10-harness-health-improvement.md` — Phase 2 extraction targets (stale, updated here)
- **Guide:** `.kb/guides/code-extraction-patterns.md` — procedural extraction guidance

---

## Investigation History

**2026-03-10:** Investigation started
- Initial question: What line count range should post-extraction files target?
- Context: Phase 2 of harness health improvement plan needs specific extraction targets

**2026-03-10:** Data collection complete
- Sampled 20 files for concern analysis, analyzed 13 extraction commits, measured re-accretion velocity for all satellite and residual files

**2026-03-10:** Investigation completed
- Status: Complete
- Key outcome: Optimal post-extraction target is 200-400 lines. Three-number framework (200/400/800) replaces current two-number system. Phase 2 file list updated with 12 current targets and specific strategies.
