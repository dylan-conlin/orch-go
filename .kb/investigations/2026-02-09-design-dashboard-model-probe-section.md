# Design: Dashboard Model/Probe Section

**Date:** 2026-02-09
**Status:** Complete
**Owner:** og-feat-architect-redesign-dashboard-08feb-3034
**Parent Issue:** orch-go-j8s0y

---

## Summary (D.E.K.N.)

**Delta:** Designed a dedicated Models and Probes section for Work Graph Artifacts that treats models as first-class entities, shows probe verdicts and lineage, and surfaces merge review as an explicit action queue.

**Evidence:** Current artifact feed API recursively scans `.kb/models` and classifies probes as generic `model` artifacts, while UI rows and side panel have no fields for verdict, lineage, merge state, or model health.

**Knowledge:** The probe system is structurally model-centric, but the dashboard remains file-centric; adding model-level aggregation and probe-aware status is the missing bridge between knowledge production and orchestrator decision-making.

**Next:** Implement in three slices: backend aggregation endpoint, new Svelte store + model/probe section component, then artifact side panel lineage actions.

**Promote to Decision:** recommend-yes (introduces durable dashboard information architecture and status semantics for the model/probe loop)

---

# Investigation: Architect redesign for dashboard artifact interface - model/probe section

**Question:** How should the Work Graph artifact interface represent models and probes so the orchestrator can see validation health, contradictions, and merge actions at a glance while preserving existing dashboard constraints (including 666px usability)?

**Started:** 2026-02-09
**Updated:** 2026-02-09
**Owner:** Architect spawn
**Phase:** Complete
**Next Step:** Implementation spawn against this design
**Status:** Complete

---

## Findings

### Finding 1: Current artifact ingestion flattens probes into generic model files

**Evidence:** `handleKBArtifacts` calls `scanKBArtifacts`, which scans `.kb/models` using recursive `scanArtifactDir(..., "model", ...)`, so files under `.kb/models/<model>/probes/*.md` are returned as type `model` rather than a distinct probe type.

**Source:** `cmd/orch/serve_kb_artifacts.go:248`, `cmd/orch/serve_kb_artifacts.go:251`, `cmd/orch/serve_kb_artifacts.go:280`, `cmd/orch/serve_kb_artifacts.go:289`.

**Significance:** The backend currently erases model/probe structure at ingestion time, so the frontend cannot present model health or lineage without a new aggregation layer.

---

### Finding 2: Artifact feed response has no probe-aware fields

**Evidence:** `KBArtifactsResponse` only returns `needs_decision`, `recent`, and `by_type`; `ArtifactFeedItem` has no verdict, parent model, tested claim, merge state, or review-needed flags.

**Source:** `cmd/orch/serve_kb_artifacts.go:20`, `cmd/orch/serve_kb_artifacts.go:29`; `web/src/lib/stores/kb-artifacts.ts:20`, `web/src/lib/stores/kb-artifacts.ts:23`.

**Significance:** Requirement (2), (3), (4), and (5) cannot be met by UI-only changes; status and lineage must be modeled in API payloads.

---

### Finding 3: Current UI treats artifacts uniformly and cannot visualize probe verdicts

**Evidence:** `ArtifactRow` only branches icon/badge by coarse `artifact.type` and generic `artifact.status`; there is no verdict iconography, no color mapping for confirm/extend/contradict, and no model-level grouping.

**Source:** `web/src/lib/components/artifact-row/artifact-row.svelte:25`, `web/src/lib/components/artifact-row/artifact-row.svelte:42`, `web/src/lib/components/artifact-feed/artifact-feed.svelte:154`.

**Significance:** The current feed is optimized for file browsing, not for the probe feedback loop.

---

### Finding 4: Merge workflow exists in CLI completion but is invisible in dashboard

**Evidence:** `processProbes` in completion gates prints probe merge summary and prompts in terminal; no dashboard endpoint exposes pending extends/contradicts as action queue items.

**Source:** `cmd/orch/complete_gates.go:633`, `cmd/orch/complete_gates.go:644`, `cmd/orch/complete_gates.go:654`.

**Significance:** Requirement (4) is currently satisfied only for orchestrator CLI users, not dashboard-first operation.

---

### Finding 5: Probe volume and model health are now large enough to require dedicated visualization

**What I Tested:**

```bash
python3 - <<'PY'
from pathlib import Path
import re
root = Path('/Users/dylanconlin/Documents/personal/orch-go/.kb/models')
models = [p for p in root.glob('*.md') if p.name not in {'README.md','_TEMPLATE.md','PHASE3_REVIEW.md','PHASE4_REVIEW.md'}]
probes = list(root.glob('*/probes/*.md'))
verdict = {'confirms':0,'extends':0,'contradicts':0}
for p in probes:
    m = re.search(r'\*\*Verdict:\*\*\s*(confirms|extends|contradicts)', p.read_text(errors='ignore'), re.I)
    if m:
        verdict[m.group(1).lower()] += 1
print('models', len(models))
print('probes', len(probes))
print('verdict', verdict)
PY
```

**Observed output:**

```text
models 26
probes 11
verdict {'confirms': 2, 'extends': 8, 'contradicts': 1}
```

**Significance:** The feedback loop is active and now includes contradictions, so the dashboard needs dedicated status and review affordances rather than generic list rows.

---

## Synthesis

The artifact feed succeeded for broad KB browsing, but it is now misaligned with the model-centric probe decision (`2026-02-08-model-centric-probes-replace-investigations.md`). The core mismatch is representational: models and probes have a structural parent/child relationship and operational workflow (verdict -> review -> merge), while current UI/API flatten everything into standalone files.

To satisfy the issue requirements, the dashboard needs a new section whose primary entity is the model, with probe data attached as state transitions and actions. This should not replace the generic artifact feed; it should sit inside Artifacts view as a dedicated subsection because model/probe work is a distinct cognitive workflow.

---

## Recommended Design

### 1) Information architecture (Artifacts view)

Add a new top section: **Models and Probes** above existing Needs Decision/Recent/Browse.

- Left column (desktop) / top stack (<=666px): model cards (one row per model)
- Right column (desktop) / bottom stack (<=666px): review queue of probes needing merge attention (extends + contradicts)
- Existing artifact sections remain below for non-model knowledge browsing

This preserves backwards compatibility and makes model/probe loop first-class without deleting current feed behavior.

### 2) Model status taxonomy (feedback loop health)

Each model gets one primary status and supporting badges:

- `needs_review`: has at least one unmerged probe with verdict `extends` or `contradicts`
- `stale`: no probes in last 30 days
- `well_validated`: 3+ probes in last 30 days, no unmerged contradictions
- `active`: probes in last 30 days but not meeting well-validated threshold

Badges shown on model row:

- `+N unmerged` (actionable)
- `last probe 12d ago` (freshness)
- `C/E/X` counts (confirms/extends/contradicts)

### 3) Probe verdict visuals

Map verdicts to color and icon in both model timeline and queue:

- confirms -> green check
- extends -> blue plus
- contradicts -> red alert

Use identical visual mapping in row chips, timeline bullets, and side-panel metadata to avoid context switching.

### 4) Merge workflow accessibility

Add a **Merge Review Queue** list keyed to probes with `extends|contradicts` that are not merged.

Each queue item shows:

- probe title/date
- verdict chip
- target model name
- extracted tested claim (first sentence from `## Question`)
- actions: `Review Probe`, `Open Model`, `Copy Merge Command`

`Copy Merge Command` is v1-safe and avoids adding write endpoints immediately. Suggested command pattern:

```bash
kb merge-probe .kb/models/<model>/probes/<probe>.md
```

(If `kb merge-probe` does not exist yet, copy fallback guidance text used in CLI merge prompt.)

### 5) Probe-model lineage interactions

Two-way drill-down:

- Click model -> expand probe timeline for that model (newest first)
- Click probe (timeline or queue) -> side panel opens with:
  - Probe summary + verdict
  - Tested claim excerpt
  - Link back to parent model
  - Nearby probes on same model (prev/next)

Side panel header should include lineage breadcrumb:

```text
Model: daemon-autonomous-operation  >  Probe: skill-inference-mapping-verification
```

---

## API Contract (new endpoint)

Add `GET /api/kb/model-probes?project_dir=<path>&stale_days=30`.

Response shape:

```json
{
  "summary": {
    "models_total": 26,
    "probes_total": 11,
    "needs_review": 5,
    "stale": 20,
    "well_validated": 5
  },
  "queue": [
    {
      "probe_path": ".kb/models/.../probes/....md",
      "model": "daemon-autonomous-operation",
      "verdict": "contradicts",
      "date": "2026-02-09",
      "claim": "The model's Skill Inference table claims...",
      "merged": false
    }
  ],
  "models": [
    {
      "name": "daemon-autonomous-operation",
      "path": ".kb/models/daemon-autonomous-operation.md",
      "last_updated": "2026-02-09",
      "status": "needs_review",
      "probe_counts": { "confirms": 0, "extends": 1, "contradicts": 1 },
      "unmerged_count": 2,
      "last_probe_at": "2026-02-09",
      "probes": []
    }
  ]
}
```

Parsing rules:

- verdict: parse `**Verdict:**` in `## Model Impact`
- claim: first non-empty line from `## Question`
- merged: true when probe slug appears in model under `## Merged Probes` or `## Recent Probes` (v1 heuristic)

---

## Implementation Sequence

1. Backend: add model/probe scanner and `GET /api/kb/model-probes` in `cmd/orch/serve_kb_artifacts.go` (or split into `serve_kb_models.go` if file size becomes unwieldy).
2. Frontend store: add `web/src/lib/stores/kb-model-probes.ts` with fetch + type definitions.
3. UI: add `web/src/lib/components/model-probe-section/*.svelte` and integrate in `artifact-feed.svelte` above current sections.
4. Side panel: extend `artifact-side-panel.svelte` to render lineage metadata when artifact is a probe.
5. Tests: add server parsing tests (verdict/claim/merged/stale) and one UI smoke test for queue rendering.

---

## 666px Constraint Check

To preserve half-screen workflow, the section uses a single-column stacked layout under `md`:

- Model list first (scan health quickly)
- Merge queue second (action queue)
- No horizontal tables; badges wrap and truncate
- Probe timeline collapsed by default per model

This keeps critical status visible without horizontal scrolling.

---

## Structured Uncertainty

**What is validated now:**

- Current ingestion and UI cannot represent probe workflow semantics (verified by code paths above).
- Current model/probe corpus has enough volume and verdict diversity to justify dedicated dashboard treatment.

**What remains to validate during implementation:**

- Best heuristic for `merged` detection until an explicit merge marker exists.
- Whether queue should include confirms when marked as unresolved by orchestrator policy.
- Whether review actions should remain copy-command only (v1) or support server-side merge actions (v2).

---

## References

- `.kb/decisions/2026-02-08-model-centric-probes-replace-investigations.md`
- `.kb/investigations/2026-02-08-inv-probe-system-lifecycle-audit-trace.md`
- `cmd/orch/serve_kb_artifacts.go`
- `cmd/orch/complete_gates.go`
- `web/src/lib/components/artifact-feed/artifact-feed.svelte`
- `web/src/lib/components/artifact-row/artifact-row.svelte`
- `web/src/lib/stores/kb-artifacts.ts`
