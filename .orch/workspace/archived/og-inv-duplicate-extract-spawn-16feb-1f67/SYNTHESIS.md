# Session Synthesis

**Agent:** og-inv-duplicate-extract-spawn-16feb-1f67
**Issue:** orch-go-982
**Duration:** 2026-02-16T15:27 → 2026-02-16T15:50
**Outcome:** success

---

## Plain-Language Summary

The 9 duplicate "Extract spawn flags phase 1: --mode" issues were all created by the **daemon's extraction logic** (`pkg/daemon/extraction.go`), not by an orchestrator session or manual action. The original trigger was `orch-go-hcrg` (a phase 1 implementation task created by an orchestrator after design issue `orch-go-82eg` closed). When the daemon picked up hcrg, it detected that `cmd/orch/spawn_cmd.go` (2202 lines) exceeds the 1500-line CRITICAL threshold and auto-created an extraction issue. The problem: two amplification bugs caused unbounded issue creation — (1) every time an extraction closes, hcrg becomes unblocked and the daemon creates ANOTHER extraction because spawn_cmd.go is still >1500 lines, and (2) extraction issues themselves mention spawn_cmd.go in their titles, causing them to recursively trigger more extraction when the daemon processes them. The existing fixes (content-aware dedup and bd create title dedup) address symptoms but the extraction logic itself still lacks a convergence condition.

## Verification Contract

See probe: `.kb/models/daemon-autonomous-operation/probes/2026-02-16-duplicate-extraction-provenance-trace.md`

Key verifiable claims:
- `orch-go-hcrg` is the parent of all extraction duplicates (verify: `grep depends_on_id .beads/issues.jsonl` shows hcrg depends on l8k2, kzqq, pg9l, p6k6, a2li, ahx8)
- spawn_cmd.go remains 2202 lines (verify: `wc -l cmd/orch/spawn_cmd.go`)
- Title concatenation in cu0r, xy7n, 95uh, m8u7 proves recursive self-triggering (verify: read titles in issues.jsonl)

---

## Delta (What Changed)

### Files Created
- `.kb/models/daemon-autonomous-operation/probes/2026-02-16-duplicate-extraction-provenance-trace.md` - Probe documenting full provenance trace
- `.orch/workspace/og-inv-duplicate-extract-spawn-16feb-1f67/SYNTHESIS.md` - This file

### Files Modified
- None (investigation only — no code changes)

### Commits
- Pending (probe + synthesis)

---

## Evidence (What Was Observed)

- `orch-go-hcrg` has 8 extraction dependencies, one for each duplicate extraction issue (verified via issues.jsonl dependency records)
- hcrg was created Feb 15 23:03:45 as follow-up from orch-go-82eg, mentions "Files: cmd/orch/spawn_cmd.go" in description
- First extraction l8k2 created 4 minutes later (23:07:26) — daemon picked up hcrg, detected critical hotspot, created extraction
- l8k2 did actual work (commit 41f5a781 at 23:10), but spawn_cmd.go only dropped one flag — still 2202 lines
- Each subsequent extraction found work already done, closed, unblocked hcrg, triggering another extraction
- Title concatenation in 95uh (2x), xy7n (3x), cu0r (4x) proves recursive mechanism: `inferConcernFromIssue()` strips "extract " prefix from parent title and wraps in new template
- Inter-extraction dependencies (p6k6→95uh→xy7n→cu0r) prove cascading chain

### Tests Run
```bash
# Verified critical hotspot trigger
$ wc -l cmd/orch/spawn_cmd.go
    2202 cmd/orch/spawn_cmd.go  # Still >1500 threshold
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- **Extraction logic has no convergence condition**: It will create extraction issues indefinitely as long as a file exceeds 1500 lines, regardless of whether extraction was already attempted
- **Extraction titles are self-referential**: They contain the critical filename, causing recursive triggering via `InferTargetFilesFromIssue()`
- **Title dedup (bj0v) is necessary but not sufficient**: Stops duplicate issue creation but doesn't address the root cause of the daemon wanting to create them

### Externalized via `kb`
- To be recorded after commit

---

## Next (What Should Happen)

**Recommendation:** close + create follow-up issue

### Follow-up Issues Needed

1. **Extraction convergence condition**: The extraction logic needs to check whether extraction was already attempted for a given (parent_issue, critical_file) pair. Without this, any issue mentioning a >1500-line file will generate extraction issues on every poll cycle after each extraction closes.

2. **Extraction title should NOT include target filename**: `GenerateExtractionTask()` should generate titles that don't contain parseable file paths, preventing `InferTargetFilesFromIssue()` from recursively matching them. Alternative: exclude extraction issues from extraction checking entirely (e.g., skip issues whose titles start with "Extract ").

---

## Unexplored Questions

- **Is hcrg still open?** If so, it will continue generating extraction issues for spawn_cmd.go unless one of the dedup fixes catches it. Verified: hcrg is still status:open with `triage:ready` label.
- **How many other issues mention >1500-line files?** This bug affects any issue whose title/description mentions a CRITICAL hotspot file, not just spawn_cmd.go.
- **Does the daemon still run extraction checks?** If the daemon is running now, hcrg could generate more duplicates unless the bd create dedup (bj0v) catches them.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-duplicate-extract-spawn-16feb-1f67/`
**Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-16-duplicate-extraction-provenance-trace.md`
**Beads:** `bd show orch-go-982`
