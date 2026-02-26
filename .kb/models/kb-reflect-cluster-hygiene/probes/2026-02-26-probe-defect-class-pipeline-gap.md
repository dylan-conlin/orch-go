# Probe: Defect-Class Data Dropped by orch-go Pipeline

**Model:** kb-reflect-cluster-hygiene
**Date:** 2026-02-26
**Status:** Complete
**Beads:** orch-go-1265

---

## Question

Does the kb reflect cluster hygiene model's claim that "reflect emits lexical cluster signal" fully account for defect-class as a non-lexical clustering dimension? Specifically: does the orch-go pipeline correctly consume defect-class results from kb reflect?

## What I Tested

### Test 1: Does kb reflect emit defect-class data?

```bash
kb reflect --type defect-class --format json
```

**Result:**
```json
{
  "defect_class": [
    {
      "defect_class": "configuration-drift",
      "count": 5,
      "window_days": 30,
      "investigations": [
        "2026-02-14-inv-fix-spawn-template-default-remove.md",
        "2026-02-14-inv-fix-spawn-template-remove-default.md",
        "2026-02-14-inv-probe-vs-investigation-routing-invisible.md",
        "2026-02-20-audit-verification-infrastructure-end-to-end.md",
        "2026-02-24-design-orchestrator-skill-behavioral-compliance.md"
      ],
      "suggestion": "Coherence Over Patches gate: escalate to architect for systemic remediation"
    }
  ]
}
```

**Observation:** kb reflect correctly detects defect-class patterns. The `configuration-drift` class has 5 investigations in 30 days, well above the threshold of 3.

### Test 2: Does orch-go parse defect-class data?

Inspected `pkg/daemon/reflect.go:75-83`:

```go
type kbReflectOutput struct {
    Synthesis  []SynthesisSuggestion `json:"synthesis,omitempty"`
    Promote    []PromoteSuggestion   `json:"promote,omitempty"`
    Stale      []StaleSuggestion     `json:"stale,omitempty"`
    Drift      []DriftSuggestion     `json:"drift,omitempty"`
    ModelDrift []json.RawMessage     `json:"model_drift,omitempty"`
    Refine     []kbRefineOutput      `json:"refine,omitempty"`
}
```

**Observation:** No `DefectClass` field. When `json.Unmarshal` processes kb reflect output containing `"defect_class":[...]`, the data is silently dropped because Go's json decoder ignores unknown fields by default.

### Test 3: Does createIssues path include defect-class?

Inspected `pkg/daemon/reflect.go:114-118`:

```go
func RunReflectionWithOptions(createIssues bool) (*ReflectSuggestions, error) {
    args := []string{"reflect", "--format", "json"}
    if createIssues {
        args = append(args, "--type", "synthesis", "--create-issue")
    }
```

**Observation:** When `createIssues=true`, the command narrows to `--type synthesis`, which completely excludes defect-class. Even if the Go struct had the field, the CLI command wouldn't return it.

### Test 4: Does investigation template include Defect-Class?

Inspected `kb-cli/cmd/kb/create.go:80`:

```
**Defect-Class:** {{defect_class}}
```

**Observation:** The field IS in the template. The issue description's claim that it's "Not in the investigation template" is incorrect.

### Test 5: Summary/HasSuggestions coverage

Inspected `pkg/daemon/reflect.go:212-224`:

```go
func (s *ReflectSuggestions) HasSuggestions() bool {
    return len(s.Synthesis) > 0 || len(s.Promote) > 0 || len(s.Stale) > 0 ||
        len(s.Drift) > 0 || len(s.ModelDrift) > 0 || len(s.Refine) > 0
}
```

**Observation:** Even if DefectClass were parsed, it wouldn't be included in HasSuggestions, TotalCount, or Summary. The session start hook would never report it.

## What I Observed

The orch-go pipeline has a **complete blind spot** for defect-class data:

1. **Parse gap:** `kbReflectOutput` and `ReflectSuggestions` lack DefectClass field → data silently dropped
2. **Command gap:** `createIssues=true` narrows to `--type synthesis` → defect-class issue creation never happens
3. **Display gap:** `HasSuggestions()`, `TotalCount()`, `Summary()` exclude defect-class → never shown at session start
4. **Pipeline gap:** `reflect-suggestions.json` never includes defect-class → dashboard API can't serve it

This is a clean example of **configuration drift between kb-cli capabilities and orch-go consumption** — the detection tool evolved (gained defect-class support) but the consumption layer didn't track the change.

## Model Impact

### Extends: Critical Invariant 1

The model states: "Lexical cluster != conceptual model"

This probe reveals a corollary: **Defect-class provides a non-lexical clustering dimension that the model doesn't account for.** Synthesis clusters by filename-derived topic (lexical). Defect-class clusters by metadata tag (semantic/manual). The model's core mechanism section only describes lexical clustering → semantic triage. It should also acknowledge metadata-based clustering (defect-class) as a parallel signal.

### Extends: Failure Mode coverage

A new failure mode should be documented: **"Reflect emits data that the consumer doesn't parse."** This is distinct from the existing failure modes (lexical collision, time-drifted conclusions, artifact overuse, incomplete closure, archived scanning). It's a producer-consumer version drift where kb-cli gained new reflect types that orch-go doesn't have Go structs for.

### Confirms: Constraint about auto-consolidation

The model's constraint "reflect clusters are intentionally broad and lexical" confirms that defect-class (which is precise and semantic) should remain a separate reflect type, not merged into synthesis clustering. The two serve different purposes: synthesis = "consolidate understanding"; defect-class = "escalate mechanism."

## Recommendations

1. Add `DefectClass` field to `kbReflectOutput` and `ReflectSuggestions` in `pkg/daemon/reflect.go`
2. Remove `--type synthesis` restriction from `createIssues=true` path
3. Update `HasSuggestions()`, `TotalCount()`, `Summary()` to include DefectClass
4. Update the model's Core Mechanism to acknowledge metadata-based clustering as a parallel signal to lexical clustering
5. Add Failure Mode 6 to the model: "Reflect emits data consumer doesn't parse (producer-consumer drift)"
