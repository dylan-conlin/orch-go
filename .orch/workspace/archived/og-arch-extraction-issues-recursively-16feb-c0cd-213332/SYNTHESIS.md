# Session Synthesis

**Agent:** og-arch-extraction-issues-recursively-16feb-c0cd  
**Issue:** orch-go-986  
**Duration:** 2026-02-16 20:47 → 2026-02-16 21:15  
**Outcome:** success

---

## Plain-Language Summary

**What was built:** A guard to prevent extraction issues from recursively triggering more extraction checks. The daemon was creating cascading chains of duplicate extraction issues because it would parse file paths from extraction task titles (e.g., "Extract X from cmd/orch/spawn_cmd.go...") and trigger another extraction if that file was still >1500 lines.

**Why it matters:** This bug caused the daemon to create 7+ duplicate extraction issues in a single cascade (documented in probe `2026-02-16-duplicate-extraction-provenance-trace.md`), wasting agent slots and creating noise. The fix is a simple title prefix check that prevents extraction issues from being subject to extraction checks.

**How to verify:** The fix works when extraction issues (titles starting with "Extract ") do NOT trigger new extraction checks, while regular issues mentioning the same file still DO trigger extraction. Test coverage added.

---

## TLDR

Fixed extraction recursion bug by adding a guard to `CheckExtractionNeeded()` that skips issues with titles starting with "Extract ". This prevents the daemon from parsing file paths from extraction task titles and creating cascading chains of duplicate extraction issues.

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/extraction.go` - Added guard at start of `CheckExtractionNeeded()` to skip extraction issues (lines 215-221)
- `pkg/daemon/extraction_test.go` - Added test case "extraction issues skipped to prevent recursion" (lines 396-407)

### Files Created
- `.kb/models/daemon-autonomous-operation/probes/2026-02-16-extraction-recursion-fix.md` - Probe documenting bug analysis, design options, and fix verification

### Commits
- `6c5c9d8d` - probe: extraction recursion bug analysis and fix verification
- `f525ec80` - fix: prevent extraction issues from triggering recursive extractions

---

## Evidence (What Was Observed)

### Code Analysis

**Bug mechanism identified:**
1. `GenerateExtractionTask()` (extraction.go:117-130) creates titles with embedded file paths:
   - Format: "Extract [concern] from [file] into [pkg]. Pure structural extraction..."
   - Example: "Extract spawn flags from cmd/orch/spawn_cmd.go into pkg/orch/..."

2. `InferTargetFilesFromIssue()` (extraction.go:14-52) uses regex to parse file paths from issue titles:
   - Pattern 1: `\b([a-zA-Z0-9_-]+/[a-zA-Z0-9_/-]+\.[a-zA-Z0-9]+)\b`
   - Matches paths like "cmd/orch/spawn_cmd.go" from ANY issue title

3. `CheckExtractionNeeded()` (daemon.go:826) runs on ALL issues, including extraction issues:
   - No guard to skip extraction issues
   - If target file still >1500 lines → creates another extraction issue
   - Results in cascading chains

**Prior evidence from probe `2026-02-16-duplicate-extraction-provenance-trace.md`:**
- Cascading chain: p6k6 → 95uh → xy7n → cu0r
- Title concatenation showing recursion:
  - l8k2: "Extract spawn flags phase 1: --mode from cmd/orch/spawn_cmd.go into pkg/orch/. Pure structural extraction — no behavior changes."
  - 95uh (2x): Same text repeated twice
  - xy7n (3x): Same text repeated three times
  - cu0r (4x): Same text repeated four times

### Tests Run

```bash
# Unit tests for CheckExtractionNeeded
$ go test ./pkg/daemon -run TestCheckExtractionNeeded -v
=== RUN   TestCheckExtractionNeeded
=== RUN   TestCheckExtractionNeeded/extraction_issues_skipped_to_prevent_recursion
--- PASS: TestCheckExtractionNeeded (0.00s)
    --- PASS: TestCheckExtractionNeeded/extraction_issues_skipped_to_prevent_recursion (0.00s)
PASS

# Reproduction verification
Regular issue ("Add feature to cmd/orch/spawn_cmd.go"):
  OLD: triggers extraction ✅
  NEW: triggers extraction ✅

Extraction issue ("Extract spawn flags from cmd/orch/spawn_cmd.go..."):
  OLD: triggers extraction (BUG: recursion!)
  NEW: does NOT trigger extraction ✅ (FIXED)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/daemon-autonomous-operation/probes/2026-02-16-extraction-recursion-fix.md` - Documents bug analysis, design options, and fix verification

### Decisions Made
- **Use title prefix check ("Extract ") rather than label-based marking:**
  - **Why:** Simplest implementation (1 line), no beads schema changes, matches existing `GenerateExtractionTask()` format exactly
  - **Trade-off:** String-based detection is less robust than label-based marking, but avoids schema changes and complexity
  - **When this would change:** If we need to support user-created issues that legitimately start with "Extract " and should still trigger extraction checks (then use label-based marking)

### Constraints Discovered
- Extraction issue titles MUST contain file paths for context/tracking
- Daemon cannot distinguish extraction issues from feature issues without metadata (type, labels, or title convention)
- The title prefix "Extract " is a de-facto convention established by `GenerateExtractionTask()`

### Model Impact
- **Extends** the "Daemon Autonomous Operation" model's "Duplicate Spawns" failure mode
- Should add new failure mode: "Extraction Recursion"
- Confirms that extraction system needs metadata to distinguish extraction work from feature work

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (fix implemented, tested, committed)
- [x] Tests passing (TestCheckExtractionNeeded passes with new test case)
- [x] Probe file has `Status: Complete`
- [x] SYNTHESIS.md created with Plain-Language Summary
- [x] Ready for `orch complete orch-go-986`

---

## Verification Contract

See: `VERIFICATION_SPEC.yaml`

**Key verification:**
- Extraction issues (titles starting with "Extract ") do NOT trigger new extraction checks
- Regular issues mentioning the same file still DO trigger extraction
- Test case "extraction issues skipped to prevent recursion" passes

---

## Unexplored Questions

**Potential improvements not in scope:**

1. **Should extraction issues use a label instead of title prefix for marking?**
   - Pros: More robust, explicit metadata
   - Cons: Requires beads schema changes, more complex
   - Worth exploring if title prefix detection becomes fragile

2. **Should the daemon have content-aware deduplication?**
   - Related probe: `2026-02-16-daemon-dedup-fundamentally-broken-content-aware-fix.md`
   - This would catch duplicate extraction issues even without the recursion fix
   - Orthogonal to this fix (defense in depth)

3. **Should extraction task titles omit file paths?**
   - Would prevent title parsing entirely
   - But loses important context for tracking/debugging
   - Trade-off: clarity vs. safety

---

## Session Metadata

**Skill:** architect  
**Model:** claude-sonnet-4-5-20250929  
**Workspace:** `.orch/workspace/og-arch-extraction-issues-recursively-16feb-c0cd/`  
**Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-16-extraction-recursion-fix.md`  
**Beads:** `bd show orch-go-986`
