# Session Synthesis

**Agent:** og-inv-measure-issue-quality-21mar-f27d
**Issue:** orch-go-j4ej7
**Outcome:** success

---

## Plain-Language Summary

Measured whether beads issues are self-contained enough for automated daemon routing. The answer is yes — 100% of issues have a type field, and the daemon's 4-tier inference pipeline (label → title → description → type) always produces a skill. Zero failures across 641 spawns. But this "100% success rate" is itself false confidence: the pipeline is structurally guaranteed to succeed because the final fallback (type → skill mapping) always works. The real question — whether the *inferred skill was correct* — has never been measured. 69% of inferences fall through to the coarsest possible signal (type-based), and no accuracy ground truth exists. The redesign needs neither a quality gate nor a template system for routing completeness. It needs an accuracy measurement for routing correctness — which is a different problem.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- Probe confirms measurement-honesty invariants #1 (metric that cannot go red) and #2 (absent negative signal ≠ positive signal) in the skill inference subsystem
- Corpus metrics: 4,036 issues across 20 projects, 100% have type field, 66% have descriptions, 15% have labels
- Inference pipeline: 12% label, 12% title, 5% description, 69% type-based fallback, 0% failure

---

## Delta (What Changed)

### Files Created
- `.kb/models/measurement-honesty/probes/2026-03-21-probe-issue-quality-baseline-inference-honesty.md` — Probe testing invariants #1 and #2 against daemon skill inference

### Files Modified
- `.kb/models/measurement-honesty/model.md` — Added false-confidence example (exhaustive fallback), evolution entry, probe reference

### Commits
- (pending)

---

## Evidence (What Was Observed)

### Data Sources Examined

1. **`~/.orch/events.jsonl`** (176,038 events): 17,199 `spawn.skill_inferred` events, 641 `daemon.spawn` events
2. **`.harness/events.jsonl`** (61 events): All `gate.fired` — no skill inference events here (different purpose)
3. **Beads issue corpus**: 20 projects, 4,036 issues in `.beads/issues.jsonl` files
4. **Daemon source code**: `pkg/daemon/skill_inference.go` (302 lines), `pkg/daemon/ooda.go`, `pkg/daemon/spawn_execution.go`

### Key Numbers

| Metric | Value | Source |
|--------|-------|--------|
| Total issues across all projects | 4,036 | beads corpus |
| Issues with `issue_type` field | 100% | beads corpus |
| Issues with description | 66% | beads corpus |
| Issues with labels | 15% | beads corpus |
| Issues with `skill:*` label | 2% | beads corpus |
| Daemon spawns | 641 | events.jsonl |
| Inference failures | 0 | events.jsonl |
| Inference via type fallback | 69% | events.jsonl |
| Inference via label | 12% | events.jsonl |
| Inference via title | 12% | events.jsonl |
| Inference via description | 5% | events.jsonl |

---

## Architectural Choices

No architectural choices — this was a measurement investigation, not implementation.

---

## Knowledge (What Was Learned)

### Constraints Discovered

1. **My initial analysis script used `type` instead of `issue_type`** — produced the false finding that "100% of issues have no type." The field name in beads JSON is `issue_type`. This is a footgun for anyone querying the corpus programmatically.

2. **Events are split across two files**: `.harness/events.jsonl` (gate events, project-local) and `~/.orch/events.jsonl` (daemon events, global). The daemon's skill inference events only go to the global file.

3. **The description heuristic (5% of inferences) is undertested** — it catches "investigate," "audit," "analyze" etc. in descriptions, but 34% of orch-go issues have no description at all, so the channel has a hard ceiling.

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Probe file created with all 4 required sections
- [x] Probe findings merged into parent model
- [x] SYNTHESIS.md created

---

## Unexplored Questions

- **Routing accuracy**: What % of daemon skill inferences actually produce the correct skill? Requires human-labeled ground truth (sample 50 spawns, label correct skill).
- **Title coverage**: Could the title keyword set be expanded to catch more of the 69% type-fallback cases? Many issue titles contain actionable words not in the current map.
- **Cross-project variation**: orch-go has 52% description rate, but some projects (beads-ui: 208 issues) may have very different profiles.

---

## Friction

No friction — smooth session.

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-measure-issue-quality-21mar-f27d/`
**Probe:** `.kb/models/measurement-honesty/probes/2026-03-21-probe-issue-quality-baseline-inference-honesty.md`
**Beads:** `bd show orch-go-j4ej7`
