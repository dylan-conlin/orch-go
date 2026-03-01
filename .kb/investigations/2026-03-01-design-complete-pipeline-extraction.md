# Design: complete_pipeline.go Extraction Plan

**Phase:** Complete
**Type:** Design (Architect)
**Agent:** og-arch-complete-pipeline-go-01mar-363e
**Issue:** orch-go-ago9
**Date:** 2026-03-01

---

## Design Question

How should `complete_pipeline.go` (970 lines, coupling-cluster 74, fix-density 8) be decomposed to stay below the 800-line extraction boundary while preserving the 4-phase pipeline architecture?

## Problem Framing

**Success Criteria:**
- `complete_pipeline.go` reduced to <500 lines
- All extracted files within 200-400 line range (well below 800-line gate)
- No import changes required (all in `package main`)
- All tests continue to pass
- Pipeline orchestration (`runComplete` in `complete_cmd.go`) unchanged
- No behavioral changes

**Constraints:**
- Technical: `package main` allows friction-free file splitting (no import cycles)
- Architectural: The 4-phase decomposition (resolve → verify → advise → lifecycle) was established Feb 28, 2026 and is working well
- Process: Files >1500 lines are blocked from feature additions; at 970 lines, pipeline is approaching the warning threshold (>800 lines + 50-line additions)

**Scope:**
- IN: Splitting `complete_pipeline.go` into multiple files
- IN: Moving types to their appropriate files
- OUT: Refactoring function internals (the functions themselves are well-structured)
- OUT: Changing the 4-phase pipeline architecture
- OUT: Modifying `complete_cmd.go` or `runComplete()`

## Exploration

### Fork 1: How many files to split into?

**Options:**
- A: **2 files** - Extract only `executeVerificationGates` (leaves pipeline at ~640 lines, still >800 with growth)
- B: **3 files** - Extract verification + lifecycle transition (pipeline at ~370 lines)
- C: **4 files** - Extract each phase to its own file (pipeline becomes empty shell or types-only, 4 new thin files)

**Substrate says:**
- Guide: Target ~300-800 lines per file
- Model: 800-line gate is the threshold for Context Noise
- Principle (Coherence over patches): Don't under-extract; leave enough room for future growth
- Decision (decompose into 4 phases, Feb 28): The 4-phase structure is settled

**Recommendation:** **Option B (3 files)**

Rationale:
- `executeVerificationGates` (274 lines) + `applySkipFilters` (43 lines) = ~330 lines → natural cohesive unit (both about verification)
- `executeLifecycleTransition` (264 lines) → natural standalone (about state transition + side effects)
- Remaining `complete_pipeline.go` (~370 lines) → types + resolve + advisories → still cohesive (the "glue" of the pipeline)
- Option A under-extracts (640 lines is too close to 800)
- Option C over-extracts (4 files of 130-270 lines each; `resolveCompletionTarget` at 136 lines is too thin for a standalone file)

**Trade-off accepted:** `resolveCompletionTarget` and `runCompletionAdvisories` remain in the same file despite being different phases. This is acceptable because they're both under 200 lines and are "glue" phases (resolution + advisory dispatching) vs. the heavier "work" phases (verification + lifecycle).

### Fork 2: What stays in `complete_pipeline.go`?

**Options:**
- A: Types + resolve + advisories (the current plan)
- B: Types only (make it a pure types file, extract all phases)
- C: Just the pipeline orchestrator (but that's already in `complete_cmd.go`)

**Substrate says:**
- Guide: "Keep all related code together" - types should be near their primary consumers
- Model: Cohesive extraction units share a single infrastructure substrate

**Recommendation:** **Option A**

The pipeline file should retain the pipeline types (`CompletionTarget`, `VerificationOutcome`, `AdvisoryResults`) because these are the typed interfaces between phases. The two smaller phases (resolve + advisories) stay because they don't warrant standalone files and benefit from proximity to the types they use.

### Fork 3: File naming convention

**Options:**
- A: `complete_verification.go` and `complete_lifecycle.go`
- B: `complete_gates.go` and `complete_transition.go`
- C: `complete_verify.go` and `complete_close.go`

**Substrate says:**
- Existing pattern: `complete_checklist.go`, `complete_hotspot.go`, `complete_postlifecycle.go`
- Guide: Use descriptive names that indicate responsibility

**Recommendation:** **Option A** - `complete_verification.go` and `complete_lifecycle.go`

These names align with the function names (`executeVerificationGates` → verification, `executeLifecycleTransition` → lifecycle) and follow the existing `complete_*` naming pattern.

## Synthesis

### Concrete Extraction Plan

#### Step 1: Create `complete_verification.go` (~330 lines)

Contents to extract from `complete_pipeline.go`:
- `VerificationOutcome` type (lines 40-47)
- `executeVerificationGates()` function (lines 191-464)
- `applySkipFilters()` function (lines 928-970)

Imports needed:
```go
import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/dylan-conlin/orch-go/pkg/checkpoint"
    "github.com/dylan-conlin/orch-go/pkg/events"
    "github.com/dylan-conlin/orch-go/pkg/spawn"
    "github.com/dylan-conlin/orch-go/pkg/state"
    "github.com/dylan-conlin/orch-go/pkg/verify"
    "golang.org/x/term"
)
```

#### Step 2: Create `complete_lifecycle.go` (~270 lines)

Contents to extract from `complete_pipeline.go`:
- `executeLifecycleTransition()` function (lines 663-926)

Imports needed:
```go
import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/dylan-conlin/orch-go/pkg/activity"
    "github.com/dylan-conlin/orch-go/pkg/agent"
    "github.com/dylan-conlin/orch-go/pkg/daemon"
    "github.com/dylan-conlin/orch-go/pkg/events"
    "github.com/dylan-conlin/orch-go/pkg/spawn"
    "github.com/dylan-conlin/orch-go/pkg/verify"
)
```

#### Step 3: Clean up `complete_pipeline.go` (~370 lines)

Remaining contents:
- `CompletionTarget` type (lines 27-38)
- `AdvisoryResults` type (lines 49-52)
- `resolveCompletionTarget()` function (lines 54-189)
- `runCompletionAdvisories()` function (lines 466-661)

Remove unused imports after extraction:
- `"github.com/dylan-conlin/orch-go/pkg/activity"`
- `"github.com/dylan-conlin/orch-go/pkg/agent"`
- `"github.com/dylan-conlin/orch-go/pkg/daemon"`
- `"github.com/dylan-conlin/orch-go/pkg/events"` (if unused by advisories... actually events is used by verification only in this file)
- `"github.com/dylan-conlin/orch-go/pkg/state"`
- `"golang.org/x/term"`
- `"bufio"`

#### Step 4: Verify

```bash
go build ./cmd/orch/
go vet ./cmd/orch/
go test ./cmd/orch/...
```

### Post-Extraction File Sizes

| File | Before | After | Change |
|------|--------|-------|--------|
| `complete_pipeline.go` | 970 | ~370 | -600 |
| `complete_verification.go` | (new) | ~330 | +330 |
| `complete_lifecycle.go` | (new) | ~270 | +270 |

### Complete Subsystem After Extraction

```
complete_cmd.go:            285 lines (CLI + orchestrator)
complete_pipeline.go:       370 lines (types + resolve + advisories) ← reduced
complete_verification.go:   330 lines (gate execution + skip filters) ← NEW
complete_lifecycle.go:       270 lines (lifecycle transition) ← NEW
complete_postlifecycle.go:  541 lines (post-lifecycle helpers)
complete_checklist.go:      206 lines (checklist + changelog UI)
complete_architect.go:      164 lines (auto-create impl issue)
complete_hotspot.go:        184 lines (hotspot advisory)
complete_model_impact.go:   238 lines (model impact advisory)
complete_synthesis.go:      120 lines (synthesis checkpoint)
complete_trust.go:           58 lines (trust calibration)
complete_cleanup.go:         43 lines (tmux cleanup)
complete_actions.go:         52 lines (dead code - archiveWorkspace)
                           -----
Total:                    2,861 lines across 13 files (same total, 2 new files)
```

All files within the 200-600 line range. No file exceeds the 800-line gate.

### Risks

1. **Test breakage:** No tests directly test the pipeline functions (they test helper functions like `archiveWorkspace`, `extractProjectFromBeadsID`, etc.). Risk is minimal—extraction is a pure file-split with no logic changes.

2. **Parallel agent conflicts:** Other agents may be modifying `complete_pipeline.go` concurrently. The extraction should be done as a single atomic commit.

3. **`complete_postlifecycle.go` at 541 lines:** This is the next candidate for growth concern, but it's well below 800 lines and stable (helpers not phases).

## Recommendations

**RECOMMENDED:** Option B (3-file split: verification + lifecycle extraction)

- **Why:** Brings `complete_pipeline.go` from 970 → ~370 lines (62% reduction), well below the 800-line gate, with room for growth. Each extracted file is a cohesive unit matching the existing pipeline phase architecture.
- **Trade-off:** 2 additional files in the `complete_*` namespace (13 total). This is acceptable given the subsystem already has 11 files.
- **Expected outcome:** File stops appearing in hotspot reports. Future modifications to verification gates or lifecycle transition don't increase `complete_pipeline.go` line count.
- **Implementation effort:** Straightforward file-split. ~30 minutes of feature-impl work. Zero behavioral changes.

**Alternative: Option A (extract verification only)**
- Pros: Simpler change (1 new file)
- Cons: Leaves `complete_pipeline.go` at ~640 lines, which will grow past 800 with the next feature addition
- When to choose: If minimizing file count is a priority

**Alternative: Option C (extract all 4 phases)**
- Pros: Each phase gets its own file, maximum separation
- Cons: `resolveCompletionTarget` at 136 lines is too thin for a standalone file; over-engineering
- When to choose: If the resolve or advisory phases are expected to grow significantly

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
This is a straightforward extraction following established patterns. Promotion to decision is NOT recommended—this is routine refactoring, not an architectural choice.

**Suggested implementation issue:**
```
bd create "Extract complete_pipeline.go into verification + lifecycle files" --type task --priority 3
```
Skill: `feature-impl` (straightforward file split)
