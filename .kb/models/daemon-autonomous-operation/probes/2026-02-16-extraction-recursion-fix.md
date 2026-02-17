# Probe: Extraction Recursion Fix

**Model:** Daemon Autonomous Operation  
**Date:** 2026-02-16  
**Status:** Active

---

## Question

How should we prevent extraction issues from recursively triggering more extraction checks? The model's "Duplicate Spawns" failure mode doesn't explicitly cover this recursion pattern where extraction issues themselves become the target of further extraction.

**Bug description from issue orch-go-986:**

> GenerateExtractionTask() produces titles containing the critical file path. When daemon processes these extraction issues, InferTargetFilesFromIssue() parses the file path from the title, triggering recursive extraction. inferConcernFromIssue() strips the 'extract' prefix, creating progressively longer concatenated titles.

**Related probe:** `2026-02-16-duplicate-extraction-provenance-trace.md` identified this as **Mechanism 2: Recursive extraction from extraction issues (cascading)**.

---

## What I Tested

### 1. Code Analysis

**GenerateExtractionTask() (extraction.go:117-130):**
```go
return fmt.Sprintf(
    "Extract %s from %s into %s. Pure structural extraction — no behavior changes.",
    concern,
    criticalFile,
    targetPkg,
)
```
- Creates titles like: "Extract [concern] from cmd/orch/spawn_cmd.go into pkg/orch/. Pure structural extraction — no behavior changes."
- The file path is embedded in the title

**InferTargetFilesFromIssue() (extraction.go:14-52):**
```go
filePathRegex := regexp.MustCompile(`\b([a-zA-Z0-9_-]+/[a-zA-Z0-9_/-]+\.[a-zA-Z0-9]+)\b`)
```
- Pattern 1 matches file paths like "cmd/orch/spawn_cmd.go"
- Extracts file paths from BOTH issue title AND description
- No filtering for extraction issues

**CheckExtractionNeeded() call site (daemon.go:826):**
```go
extraction := CheckExtractionNeeded(issue, d.HotspotChecker)
```
- Called for EVERY issue in the Once() method
- No check for whether the issue itself is an extraction issue

### 2. Design Options Analysis

**Option 1: Skip extraction checks for issues with titles starting with "Extract"**
- Pros: Simple, matches the generated title format exactly, no DB/label changes needed
- Cons: String-based detection is brittle (what if title format changes?)
- Implementation: Add check at start of CheckExtractionNeeded()

**Option 2: Remove file paths from generated extraction task titles**
- Pros: Prevents title parsing entirely
- Cons: Loses important context (which file is being extracted), makes tracking harder
- Implementation: Modify GenerateExtractionTask() format

**Option 3: Add a label/flag to mark extraction issues**
- Pros: Robust, explicit marking
- Cons: Requires beads schema changes, more complex implementation
- Implementation: Add `category:extraction` label in DefaultCreateExtractionIssue()

**Option 4: Check issue type or description for extraction markers**
- Pros: More robust than title matching
- Cons: Still string-based, requires description conventions

---

## What I Observed

### Evidence from Prior Probe

The `2026-02-16-duplicate-extraction-provenance-trace.md` probe documented the cascading extraction chain:

```
p6k6 → creates 95uh → creates xy7n → creates cu0r
```

With title concatenation showing the recursion:
- **l8k2** (clean): "Extract spawn flags phase 1: --mode from cmd/orch/spawn_cmd.go into pkg/orch/. Pure structural extraction — no behavior changes."
- **95uh** (2x): "Extract spawn flags phase 1: --mode from cmd/orch/spawn_cmd.go into pkg/orch/. **pure structural extraction — no behavior changes. from cmd/orch/spawn_cmd.go into pkg/orch/.** Pure structural extraction — no behavior changes."
- **xy7n** (3x): Title repeated 3 times
- **cu0r** (4x): Title repeated 4 times

### Substrate Consultation

**Principles check (`~/.kb/principles.md`):**
- **Evidence hierarchy** - Code is truth; artifacts are hypotheses to verify
- **Coherence over patches** - If 5+ fixes hit the same area, recommend redesign not another patch

**Model check (Daemon Autonomous Operation):**
- The model's "Duplicate Spawns" failure mode focuses on poll timing and SpawnedIssueTracker TTL
- Does not explicitly cover extraction recursion as a distinct failure mode
- This probe should extend the model

---

## Model Impact

This probe **extends** the model's "Duplicate Spawns" failure mode.

### Recommended Model Update

Add a new failure mode to "Daemon Autonomous Operation" model:

**### 4. Extraction Recursion**

**What happens:** Extraction issues trigger more extraction checks, creating cascading chains of duplicate extraction issues.

**Root cause:** 
1. GenerateExtractionTask() embeds file path in title ("Extract X from file.go into pkg/")
2. InferTargetFilesFromIssue() parses file paths from issue titles
3. CheckExtractionNeeded() runs on ALL issues, including extraction issues
4. If the target file is still >1500 lines, a new extraction issue is created

**Why detection is hard:** The recursion appears as separate issues with valid provenance. Title concatenation signals the problem (repeated fragments), but only visible after multiple cycles.

**Fix:** Skip extraction checks for issues that are themselves extraction work. Add guard at CheckExtractionNeeded() entry: if title starts with "Extract" and matches extraction task format, return nil.

**Prevention:** Label-based marking (`category:extraction`) would be more robust but requires schema changes.

---

## Recommended Solution

**Option 1 (Recommended): Skip extraction checks for extraction issues based on title prefix**

Reasoning:
- Simplest implementation (1 line check)
- No beads schema changes needed
- Matches existing GenerateExtractionTask() format exactly
- Low risk - false positives unlikely (how many non-extraction issues start with "Extract X from Y into Z"?)

Implementation:
```go
// CheckExtractionNeeded determines if an issue targets a CRITICAL hotspot file (>1500 lines)
func CheckExtractionNeeded(issue *Issue, checker HotspotChecker) *ExtractionResult {
    if issue == nil || checker == nil {
        return nil
    }
    
    // Skip extraction checks for extraction issues themselves (prevents recursion)
    // Extraction issues have titles like: "Extract X from file.go into pkg/..."
    if strings.HasPrefix(issue.Title, "Extract ") {
        return nil
    }
    
    // ... rest of function
}
```

**Trade-off accepted:** String-based detection is less robust than label-based marking, but avoids schema changes and complexity.

**When this would change:** If we need to support user-created issues that legitimately start with "Extract " and should still trigger extraction checks. In that case, use Option 3 (label-based marking).

---

## Verification

### Implementation

**Added guard to CheckExtractionNeeded():**
```go
// Skip extraction checks for extraction issues themselves (prevents recursion).
// Extraction issues have titles like: "Extract X from file.go into pkg/..."
if strings.HasPrefix(issue.Title, "Extract ") {
    return nil
}
```

**Added test case:**
```go
{
    name: "extraction issues skipped to prevent recursion",
    issue: &Issue{
        Title:       "Extract spawn flags phase 1: --mode from cmd/orch/spawn_cmd.go into pkg/orch/. Pure structural extraction — no behavior changes.",
        Description: "Auto-generated extraction task",
    },
    hotspots: []HotspotWarning{
        {Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2200},
    },
    expected: false, // Should NOT trigger extraction even though file is >1500 lines
}
```

### Test Results

**Unit test:** `go test ./pkg/daemon -run TestCheckExtractionNeeded`
- ✅ All subtests pass, including new "extraction issues skipped to prevent recursion" test
- Result: Extraction issues return nil (no extraction triggered)

**Reproduction verification:**
```bash
Regular issue ("Add feature to cmd/orch/spawn_cmd.go"):
  OLD: triggers extraction ✅
  NEW: triggers extraction ✅

Extraction issue ("Extract spawn flags from cmd/orch/spawn_cmd.go..."):
  OLD: triggers extraction (BUG: recursion!)
  NEW: does NOT trigger extraction ✅ (FIXED)
```

**Conclusion:** Fix successfully prevents extraction recursion while maintaining normal extraction behavior for regular issues.

---

## Status

**Status:** Complete

The bug is fixed. Extraction issues are now excluded from extraction checks, preventing the recursive cascading described in the original bug report and confirmed in probe `2026-02-16-duplicate-extraction-provenance-trace.md`.
