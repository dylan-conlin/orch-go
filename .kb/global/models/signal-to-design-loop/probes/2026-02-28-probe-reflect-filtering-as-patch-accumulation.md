# Probe: kb reflect Filtering as Patch Accumulation Instance

**Model:** signal-to-design-loop
**Date:** 2026-02-28
**Status:** Complete
**Triggered by:** orch-go-z643 — Design artifact lifecycle state model for kb reflect filtering

---

## Question

Does the Signal-to-Design Loop model correctly predict that kb reflect's 40+ exclusion patches are an instance of Failure Mode 4 (Design Response Targets Instances, Not System)? Specifically:

**Model claims tested:**
- Claim (FM4): "Each signal gets a fix but the same class keeps appearing" — predicts that individual exclusion patches fix specific false-positive signals without addressing the systemic gap
- Claim (FM4 fix): "When a cluster has 3+ instances, the design response must target the CLASS, not the latest instance" — predicts the right response is a unified state model, not more patches
- Claim (Stage 3): "Requires explicit, machine-readable clustering key" — predicts that the current ad-hoc filtering (text matching, directory checks, field presence) fails because there's no unified lifecycle concept
- Claim (core): "Individual signals are noise. Clustered signals are design pressure. The difference is metadata that makes signals groupable." — tests whether the accumulated exclusion patches represent ungroupable signals

---

## What I Tested

### 1. Enumeration of All Exclusion Mechanisms in reflect.go

Analyzed the complete 2818-line reflect.go file and supporting code to catalog every filtering/exclusion point across all 11 reflect candidate types.

### 2. Classification of Exclusion Patches by Intent

Grouped the 40+ exclusion points into categories to see if they cluster around a shared concern:

| Category | Count | Examples |
|----------|-------|---------|
| Directory-based terminal state | 7 calls | `isArchivedOrSynthesizedDir()` in 7 `find*` functions |
| Entry status filtering | 4 checks | `Status == "" \|\| Status == "active"` in promote, refine, skill-candidate, drift |
| Investigation completion detection | 3 checks | Status: Complete, Next field completion markers, template placeholders |
| Relationship-based exclusion | 2 checks | Actioned-By present, Superseded-By present |
| Content quality filtering | 3 checks | Template brackets, next field too short, next field is dashes |
| Already-synthesized topic dedup | 2 checks | Guide/decision exists, model exists for topic |

### 3. Cross-Reference with Coherence Over Patches Decision

Checked whether the pattern matches the canonical example from `~/.kb/decisions/2026-01-04-coherence-over-patches.md`:
- Dashboard status logic: 10+ conditions, 350+ lines, each fix locally correct → globally incoherent
- reflect.go filtering: 40+ exclusion points, 2818 lines, each exclusion locally correct → globally ad-hoc

### 4. Tested Whether Current Filtering Can Be Expressed as State Query

For each reflect type, asked: "Could a single lifecycle state derivation replace the N exclusion checks?"

---

## What I Observed

### The model maps cleanly — this is a textbook FM4 instance

**Confirms: Failure Mode 4 predicts the exact pattern.** Each exclusion patch fixed a specific false-positive:
- Archived investigations surfacing? → Add directory check
- Completed investigations surfacing? → Add Status: Complete check
- Template placeholders surfacing? → Add bracket detection
- Superseded entries surfacing? → Add status != superseded check
- Actioned investigations surfacing? → Add Actioned-By field check

Each was correct locally. But "this artifact is no longer eligible" is expressed through 6 different mechanisms because there's no unified lifecycle concept.

**Confirms: The clustering key is missing.** The model predicts "clustering relies on natural language similarity instead of explicit metadata" for FM2. Here, the equivalent is: "lifecycle state relies on ad-hoc content scanning instead of explicit state derivation." Directory placement, Status field text, Next field content, and field presence all encode the SAME concept (lifecycle terminal state) through different mechanisms.

**Confirms: 3+ instances means target the class.** There are 7 separate calls to `isArchivedOrSynthesizedDir()`, 4 copies of status-active filtering, 2 copies of template detection, 2 copies of completion marker checking. The class is "lifecycle state filtering" and the system response should be a state model.

**Extends: This is also a Coherence Over Patches instance.** The Coherence Over Patches decision describes the dashboard status logic as the canonical example. reflect.go filtering is the same pattern:
- Patches: 40+ → exceeds the "10+ fixes: missing coherent model" threshold
- Each fix: locally correct (archived dirs SHOULD be skipped)
- Missing model: no unified concept of "artifact lifecycle state"

The Signal-to-Design Loop model explains WHY patches accumulate (no clustering key → no systemic response). The Coherence Over Patches principle identifies WHEN to stop patching (3+ fixes to same area). Together they predict both the mechanism and the trigger for this exact situation.

**Extends: The design response IS the state model.** The model says "response targets the SYSTEM, not individual instances." A `DeriveLifecycleState()` function that maps all current exclusion signals to a finite set of states would:
- Replace 7 directory checks with one state derivation
- Replace 4 status-active checks with one state query
- Replace scattered completion/template/actioned checks with state transitions
- Make future exclusion criteria additive to ONE function, not N reflect types

---

## Model Impact

- [x] **Confirms** invariant: FM4 — "each signal gets a fix but the same class keeps appearing" — 40+ exclusion patches each fix individual false-positive signals without addressing the systemic gap (missing lifecycle state model)
- [x] **Confirms** invariant: FM4 fix — "target the CLASS, not the latest instance" — a unified state derivation function would replace all scattered patches
- [x] **Confirms** invariant: Clustering (Stage 3) — "requires explicit, machine-readable clustering key" — current filtering uses ad-hoc text matching instead of explicit state values
- [x] **Extends** model with: Connection to Coherence Over Patches — this is a dual-model instance where Signal-to-Design Loop explains the mechanism and Coherence Over Patches identifies the trigger threshold

---

## Notes

The design response (artifact lifecycle state model) is documented in the companion investigation: `.kb/investigations/2026-02-28-design-artifact-lifecycle-state-model.md`

Key observation: the 40+ patches evolved organically as each reflect type was built. No single agent introduced "bad" filtering. The pattern is structural — without a lifecycle model, each new reflect type must independently re-derive "is this artifact eligible?" from raw signals. The state model makes the question answerable once.
