# Session Synthesis

**Agent:** og-feat-synthesize-daemon-investigations-06jan-520a
**Issue:** orch-go-ws7k6
**Duration:** 2026-01-06 16:45 → 2026-01-06 17:00
**Outcome:** success

---

## TLDR

Synthesized 31 daemon investigations (Dec 2025 - Jan 2026) into a comprehensive guide at `.kb/guides/daemon.md`, expanding it from 208 lines to ~400 lines covering architecture, capacity management, completion detection, dependency handling, cross-project operation, and troubleshooting patterns.

---

## Delta (What Changed)

### Files Modified
- `.kb/guides/daemon.md` - Comprehensive rewrite synthesizing all daemon knowledge into single authoritative reference

### Commits
- (pending) - docs: synthesize 31 daemon investigations into comprehensive guide

---

## Evidence (What Was Observed)

### Investigations Analyzed (31 total)

**Dec 20, 2025 (Initial Implementation)**
- `inv-orch-add-daemon-command.md` - Core pkg/daemon package with NextIssue, Preview, Once, Run methods
- `inv-add-dry-run-flag-daemon.md` - --dry-run flag reusing Preview() method

**Dec 21-22, 2025 (Hook & Concurrency)**
- `inv-daemon-hook-integration-kb-reflect.md` - SessionStart hook for suggestions
- `inv-add-concurrency-control-daemon-worker.md` - WorkerPool pattern for slot management
- `inv-phase-daemon-hook-integration-add.md` - Deferred reflection on exit

**Dec 24, 2025 (Bug Fixes)**
- `inv-daemon-autostart-race-condition-causing.md` - Race between startlock and flock
- `inv-daemon-finds-triage-ready-issues.md` - Fixed hardcoded message vs result.Message
- `inv-daemon-selects-issues-triage-ready.md` - Confirmed prior fix was applied
- `inv-daemon-uses-bd-list-status.md` - Identified bd ready vs bd list mismatch

**Dec 25, 2025 (Completion & Beads Migration)**
- `inv-add-daemon-completion-polling-close.md` - ListCompletedAgents, ProcessCompletion
- `inv-migrate-daemon-listreadyissues-use-new.md` - RPC client with CLI fallback

**Dec 26, 2025 (Capacity & Launchd)**
- `inv-daemon-capacity-count-goes-stale.md` - Pool reconciliation with OpenCode
- `inv-daemon-needs-restart-after-make.md` - make install-restart target
- `inv-document-daemon-launchd-setup-orchestrator.md` - Plist documentation

**Jan 3-4, 2026 (Skip & Visibility)**
- `inv-test-verify-daemon-skip-functionality.md` - 14 skip-related tests verified
- `debug-daemon-show-rejection-reasons-preview.md` - RejectedIssue struct for visibility
- `inv-design-analyze-pkg-daemon-daemon.md` - 7 responsibility domains identified

**Jan 6, 2026 (Recent)**
- `inv-cross-project-daemon-single-daemon.md` - kb projects list for multi-repo polling
- `inv-daemon-blocks-child-tasks-parent.md` - Parent-child dependency fix
- `inv-add-periodic-kb-reflect-daemon.md` - Configurable periodic reflection
- `inv-daemon-doesn-see-issues-newly.md` - --limit 0 fix for bd ready

### Key Patterns Identified

1. **Evolution pattern:** Started simple (single repo, basic polling) → grew complex (multi-project, reconciliation, completion detection)

2. **Debugging pattern:** Many investigations were triggered by invisible daemon behavior → led to rejection reason visibility

3. **Capacity pattern:** Initial naive counting → WorkerPool → reconciliation with external source of truth

4. **Completion pattern:** SSE tried → false positives → beads polling adopted

---

## Knowledge (What Was Learned)

### Key Architectural Insights

1. **Daemon architecture is now modular** - pkg/daemon has 13 files with single responsibilities (from Jan 4 analysis)

2. **Two completion mechanisms exist:**
   - `completion.go` - SSE-based tracking (legacy, unreliable)
   - `completion_processing.go` - Beads polling (preferred)

3. **Cross-project is feasible** - Uses kb projects list, maintains single capacity pool

4. **Parent-child deps != blocking deps** - Different unblock semantics (in_progress vs closed)

### Decisions Confirmed

| Decision | Rationale |
|----------|-----------|
| Skill from type not labels | Type is required, labels optional |
| Beads polling over SSE | SSE idle has false positives |
| RPC-first with CLI fallback | Performance + reliability |
| WorkerPool with reconciliation | Prevents stale capacity |
| --limit 0 default | bd ready default of 10 misses issues |

### Externalized via `kn`

- No new kn entries needed - all decisions were already captured in investigations

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (guide updated)
- [x] No tests needed (documentation task)
- [x] SYNTHESIS.md created
- [x] Ready for `orch complete orch-go-ws7k6`

---

## Unexplored Questions

**Questions that emerged during this session:**

1. **Should pkg/daemon be further modularized?** - Jan 4 investigation recommended P0/P1/P2 extraction phases but implementation hasn't happened

2. **Is cross-project daemon being used?** - Design exists but unclear if enabled in production

3. **Completion.go vs completion_processing.go** - Should old SSE code be removed entirely?

**Areas worth exploring further:**

- Daemon metrics/observability (spawn rates, rejection rates, completion rates)
- Daemon health checks beyond `orch doctor`

**What remains unclear:**

- When to deprecate legacy completion.go SSE tracking

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude
**Workspace:** `.orch/workspace/og-feat-synthesize-daemon-investigations-06jan-520a/`
**Beads:** `bd show orch-go-ws7k6`
