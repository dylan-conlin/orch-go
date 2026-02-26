# Investigation: Activate Defect-Class Metadata in kb reflect / Synthesis Pipeline

**Question:** How should Defect-Class metadata become active in the orch-go daemon pipeline so that recurring defect patterns are surfaced to agents and operators?

**Defect-Class:** configuration-drift

**Started:** 2026-02-26
**Updated:** 2026-02-26
**Owner:** og-arch-design-defect-class-26feb-eda1
**Phase:** Complete
**Next Step:** None — recommendations ready for implementation
**Status:** Complete

<!-- Lineage -->
**Patches-Decision:** N/A (new design)
**Extracted-From:** orch-go-1265
**Supersedes:** N/A
**Superseded-By:** N/A
**Actioned-By:** Pending

**Promote to Decision:** recommend-yes
**Authority:** architectural

---

## D.E.K.N. Summary

**Delta:** Identified 5 specific integration gaps where orch-go daemon silently drops defect-class results from `kb reflect`. Designed the activation path: add `DefectClass` to Go structs, fix the `createIssues` path to not filter to synthesis-only, and add defect-class to daemon periodic reflection and session start summaries.

**Evidence:** `kb reflect --type defect-class --format json` already returns `{"defect_class":[{"defect_class":"configuration-drift","count":5,...}]}`. But `pkg/daemon/reflect.go:76` (`kbReflectOutput` struct) has no `DefectClass` field, so this data is silently dropped by `json.Unmarshal`. The `RunReflectionWithOptions(true)` path at line 116-117 adds `--type synthesis` which excludes defect-class entirely.

**Knowledge:** The issue description contained two factual errors: (1) "No kb reflect logic consumes the field" — `findDefectClassCandidates()` in kb-cli/cmd/kb/reflect.go:1642 already works; (2) "Not in the investigation template" — it IS in the template at kb-cli/cmd/kb/create.go:80. The real gap is exclusively in orch-go's Go parsing structs and daemon integration.

**Next:** Implementation agent should modify `pkg/daemon/reflect.go` to add DefectClass types and update `RunReflectionWithOptions` to not narrow to `--type synthesis` when creating issues.

---

## Problem Framing

### Design Question

How should Defect-Class metadata flow through the orch-go pipeline so that:
1. The daemon detects and acts on recurring defect patterns
2. Agents see defect-class patterns at session startup
3. The dashboard can display defect-class information

### Success Criteria

- `kb reflect` defect-class results are parsed by orch-go (not silently dropped)
- Daemon periodic reflection creates architect issues for 3+ same-class clusters
- Session start hook surfaces defect-class patterns alongside synthesis opportunities
- No changes required to kb-cli (the detection logic already works)

### Constraints

- orch-go shells out to `kb` CLI — must parse its JSON output, not reimplement detection
- Must follow existing patterns (RunOpenReflection, RunModelDriftReflection)
- kb-cli's defect-class reflect type already works — don't duplicate logic

### Scope

**In scope:** orch-go daemon integration, Go struct additions, session start display
**Out of scope:** kb-cli changes, synthesis cross-referencing by defect-class, new defect classes

---

## Exploration (Fork Navigation)

### Fork 1: Scope of orch-go integration

**Options:**
- A: Parse-only (add DefectClass to Go structs, include in suggestions file)
- B: Parse + auto-issue creation (add to daemon periodic reflection)
- C: Parse + auto-issue + kb-cli synthesis cross-reference

**Substrate says:**
- Principle: "Defect Class Blindness" — synthesis must look across investigations. The daemon is where patterns get acted on.
- Principle: "Gate Over Remind" — if defect-class patterns need attention, surface them in the pipeline, don't rely on humans running `kb reflect` manually.
- Model: `kb-reflect-cluster-hygiene` — "reflect clusters are intentionally broad and lexical" / "Human/orchestrator semantic triage is required." Defect-class provides a non-lexical clustering dimension that complements synthesis.
- Existing pattern: `RunOpenReflection()` runs `kb reflect --type open --create-issue` separately in the daemon.

**RECOMMENDATION:** Option B

**Trade-off accepted:** Synthesis still won't cluster by defect-class (that's a kb-cli enhancement, separate issue). But defect-class detection already works standalone in kb-cli — making it visible in the orch-go daemon is the immediate gap.

**When this would change:** If defect-class patterns prove insufficient without synthesis cross-referencing, Option C would be warranted.

---

### Fork 2: How should the daemon call defect-class reflection?

**Options:**
- A: Fix `RunReflectionWithOptions(createIssues=true)` to not narrow to `--type synthesis` (run all types with `--create-issue`)
- B: Add separate `RunDefectClassReflection()` (like RunOpenReflection)
- C: Keep the synthesis-only path; add defect-class as a separate periodic task

**Substrate says:**
- Model: `kb-reflect-cluster-hygiene` — defect-class and synthesis serve different purposes but both benefit from auto-issue creation
- The current code at `reflect.go:116-117` narrows to `--type synthesis` when `createIssues=true`, which means `kb reflect --format json` is called without `--create-issue` for the default path. When `createIssues=true`, only synthesis gets issue creation.
- `kb reflect --create-issue --format json` (no `--type` filter) would create issues for ALL supported types at their respective thresholds (synthesis ≥10, defect-class ≥3, open ≥3 days)

**RECOMMENDATION:** Option A — Remove the `--type synthesis` filter when `createIssues=true`. Let `kb reflect --create-issue --format json` handle all types. This is the simplest change and lets kb-cli control thresholds.

**Substrate trace:**
- Principle: "Infrastructure Over Instruction" — let the tool handle the logic, don't duplicate threshold decisions in orch-go
- Pattern: kb-cli already deduplicates issues (7-day cooldown, title search). No risk of spam.

**Trade-off accepted:** This changes the behavior of the existing `createIssues` flag to create issues for ALL reflect types, not just synthesis. But this is the correct behavior — the current synthesis-only filtering was an arbitrary restriction.

---

### Fork 3: Session start hook integration

**Options:**
- A: Add defect-class count to existing suggestion summary (minimal)
- B: Show detailed defect-class info (class name, count, investigations)
- C: Don't show in session start (defect-class is for daemon only)

**Substrate says:**
- Principle: "Surfacing Over Browsing" — make patterns visible where attention naturally goes
- The session start hook already shows "1 synthesis opportunities" — defect-class is equally important
- Agents seeing "1 defect-class pattern" would be informed but not overwhelmed

**RECOMMENDATION:** Option A — Add to Summary() and HasSuggestions(). Keep it minimal: "1 defect-class pattern" in the summary line, alongside existing synthesis/promote/stale counts.

---

### Fork 4: Defect-class taxonomy (fixed vs. extensible)

**Options:**
- A: Keep fixed taxonomy (7 classes in kb-cli code)
- B: Allow free-form but suggest from taxonomy
- C: Extensible via config file

**Substrate says:**
- Fixed taxonomy ensures meaningful clustering (free-form fragments)
- The 7 classes cover observed patterns (configuration-drift, unbounded-growth, integration-mismatch used in practice)
- Principle: "Evolve by distinction" — if new patterns emerge, they should be deliberately added to the validated set

**RECOMMENDATION:** Option A — No change. Keep fixed. Add new classes to kb-cli code when warranted.

---

### Fork 5: Should synthesis cross-reference defect-class? (Scope boundary)

**Options:**
- A: Add defect-class annotation to SynthesisCandidate in kb-cli
- B: Create new "defect-synthesis" reflect type combining both dimensions
- C: Defer to separate issue

**Substrate says:**
- Principle: "Premise Before Solution" — the immediate gap is orch-go not parsing defect-class, not kb-cli lacking cross-references
- The defect-class reflect type already provides cross-investigation pattern detection independent of topic
- Cross-referencing is valuable but is a separate concern in kb-cli (different repo)

**RECOMMENDATION:** Option C — Create follow-up issue for kb-cli synthesis cross-reference. The current defect-class detection is already correct; orch-go just needs to consume it.

---

## Synthesis

### Recommendations

⭐ **RECOMMENDED:** Activate defect-class in orch-go daemon via 3 targeted changes

**Why:** The detection logic already works in kb-cli. The only gap is orch-go's Go structs silently dropping the data. Three changes close the loop: (1) add DefectClass types to reflect structs, (2) remove the `--type synthesis` restriction on createIssues path, (3) include in Summary/HasSuggestions.

**Trade-off:** Synthesis still clusters by topic only. Defect-class provides a parallel, orthogonal pattern detection — they don't need to be merged to both be useful.

**Expected outcome:** Daemon creates architect issues when 3+ investigations share a defect class within 30 days. Session start shows defect-class patterns. Dashboard API exposes them.

### Implementation Specification

#### Change 1: Add DefectClass types to `pkg/daemon/reflect.go`

```go
// DefectClassSuggestion represents a recurring defect mechanism.
type DefectClassSuggestion struct {
    DefectClass    string   `json:"defect_class"`
    Count          int      `json:"count"`
    WindowDays     int      `json:"window_days"`
    Investigations []string `json:"investigations"`
    Suggestion     string   `json:"suggestion"`
    IssueCreated   bool     `json:"issue_created,omitempty"`
}
```

Add to both `ReflectSuggestions` and `kbReflectOutput`:
```go
DefectClass []DefectClassSuggestion `json:"defect_class,omitempty"`
```

Update `HasSuggestions()`:
```go
return len(s.Synthesis) > 0 || len(s.Promote) > 0 || len(s.Stale) > 0 ||
    len(s.Drift) > 0 || len(s.ModelDrift) > 0 || len(s.Refine) > 0 ||
    len(s.DefectClass) > 0
```

Update `TotalCount()`:
```go
return len(s.Synthesis) + len(s.Promote) + len(s.Stale) + len(s.Drift) +
    len(s.ModelDrift) + len(s.Refine) + len(s.DefectClass)
```

Update `Summary()`:
```go
if len(s.DefectClass) > 0 {
    parts = append(parts, fmt.Sprintf("%d defect-class patterns", len(s.DefectClass)))
}
```

#### Change 2: Fix `RunReflectionWithOptions` createIssues path

Current (line 114-118):
```go
func RunReflectionWithOptions(createIssues bool) (*ReflectSuggestions, error) {
    args := []string{"reflect", "--format", "json"}
    if createIssues {
        args = append(args, "--type", "synthesis", "--create-issue")
    }
```

Change to:
```go
func RunReflectionWithOptions(createIssues bool) (*ReflectSuggestions, error) {
    args := []string{"reflect", "--format", "json"}
    if createIssues {
        args = append(args, "--create-issue")
    }
```

This lets `kb reflect --create-issue --format json` handle ALL types at their respective thresholds. No `--type` filter means defect-class, synthesis, and open all get issue creation.

#### Change 3: Wire DefectClass through the pipeline

In `RunReflectionWithOptions`, after building suggestions:
```go
suggestions := &ReflectSuggestions{
    Timestamp:    time.Now().UTC(),
    Synthesis:    rawOutput.Synthesis,
    Promote:      rawOutput.Promote,
    Stale:        rawOutput.Stale,
    Drift:        rawOutput.Drift,
    ModelDrift:   rawOutput.ModelDrift,
    Refine:       refine,
    DefectClass:  rawOutput.DefectClass,  // ADD THIS
}
```

### File Targets

| File | Change | Lines Affected |
|------|--------|----------------|
| `pkg/daemon/reflect.go` | Add DefectClassSuggestion type, add to structs, update methods | ~20 lines added/modified |
| `pkg/daemon/reflect.go` | Remove `--type synthesis` from createIssues path | 1 line changed |

### Acceptance Criteria

- [ ] `kb reflect --format json` defect_class data is parsed into Go structs (not silently dropped)
- [ ] `RunReflectionWithOptions(true)` creates issues for defect-class (not just synthesis)
- [ ] `HasSuggestions()` returns true when defect-class patterns exist
- [ ] `Summary()` includes defect-class count
- [ ] `TotalCount()` includes defect-class items
- [ ] Existing tests continue to pass
- [ ] `reflect-suggestions.json` includes defect_class section

### Out of Scope

- kb-cli changes (synthesis cross-reference by defect-class)
- Taxonomy changes (keep fixed 7 classes)
- Dashboard UI changes (API will expose data; UI is separate work)
- Template changes (Defect-Class field already in investigation template)

---

## Decision Gate Guidance (if promoting to decision)

**Add `blocks:` frontmatter when:**
- This decision resolves a recurring gap (defect-class data silently dropped from pipeline)
- Future spawns modifying reflect.go should know about DefectClass field

**Suggested blocks keywords:**
- defect-class
- reflect pipeline
- daemon reflection

---

## Discovered Work

### Follow-up Issues

1. **kb-cli: Add defect-class annotation to SynthesisCandidate** — When synthesis groups investigations by topic, also extract and display defect-class tags from each investigation in the cluster. This gives the synthesis reviewer visibility into mechanism patterns alongside topic patterns. (Type: feature, Priority: 3, Repo: kb-cli)

2. **orch-go: Add defect-class to serve API** — The `/api/reflect` endpoint should include defect_class data when serving `reflect-suggestions.json`. Currently handled automatically since it reads the file, but should be verified after the reflect.go changes. (Type: task, Priority: 4)
