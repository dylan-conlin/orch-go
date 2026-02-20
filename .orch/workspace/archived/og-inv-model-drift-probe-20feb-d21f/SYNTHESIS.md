# Session Synthesis

**Agent:** og-inv-model-drift-probe-20feb-d21f
**Issue:** orch-go-1102
**Outcome:** success

---

## Plain-Language Summary

The orchestration-cost-economics model has significant drift: 3 referenced files are deleted (`pkg/spawn/backend.go`, the Max subscription decision, `~/.anthropic/`), Flash is now banned for agent work (model still lists it as active), and the primary spawn path has inverted from "OpenCode API + pay-per-token" to "Claude backend + Max subscription" as the default. The model's core economic insight (Max beats API pricing) is more true than ever — it's now baked into the default config — but the implementation description is stale. Additionally, OpenAI/Codex is now a first-class provider (12 aliases) and the entire config resolution was refactored into a centralized `ResolvedSpawnSettings` system with provenance tracking, neither of which the model mentions.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — all claims verified against current code (`go build ./cmd/orch/` passes, file existence checks, code inspection).

---

## TLDR

Probed the orchestration-cost-economics model for drift caused by 3 deleted artifacts and found it **contradicts + extends**: the dual spawn architecture was refactored from monolithic `backend.go` into multi-file resolver, Flash is banned, OpenAI/Codex is first-class, and the primary economic path is now Max subscription by default. Probe recommends 7 model updates at HIGH/MEDIUM/LOW priority.

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestration-cost-economics/probes/2026-02-20-model-drift-stale-references-audit.md` - Probe documenting drift

### Files Modified
- None (probe-only session, no code changes)

---

## Evidence (What Was Observed)

- `pkg/spawn/backend.go` does not exist (verified via ls + git log — no history found)
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` deleted
- `~/.anthropic/` directory does not exist; auth is at `~/.local/share/opencode/auth.json`
- `pkg/model/model.go:19-22`: DefaultModel is `anthropic/claude-sonnet-4-5-20250929`
- `pkg/spawn/resolve.go:393-396`: Flash models explicitly banned via `validateModel()`
- `pkg/model/model.go:47-67`: OpenAI (gpt, gpt-5, o3) and Codex (codex, codex-mini, codex-max) have 12 aliases
- `pkg/spawn/resolve.go:103-189`: Centralized `Resolve()` with 7-level precedence and provenance tracking
- `go build ./cmd/orch/` — builds successfully

### Tests Run
```bash
go build ./cmd/orch/
# Success (no errors)
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Flash ban is a hard gate in `validateModel()`, not just a recommendation — any Flash model spawn fails immediately

### Externalized via `kn`
- Leave it Better: Straightforward investigation, no new knowledge to externalize beyond the probe itself (which documents all findings).

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (probe file created with all 4 required sections)
- [x] Probe status set to Complete
- [x] Ready for `orch complete orch-go-1102`

### Follow-up Work
The model itself needs updating based on the probe's recommended changes table (7 items). This is separate work — the probe documents the drift, the model update is a follow-up task.

---

## Unexplored Questions

- Whether the `she-llac.com` credit formula section is still accurate (external dependency, can't verify from code)
- Whether cost tracking was ever implemented (the model says "Not Yet Implemented" as of Jan 2026)
- Current DeepSeek V3 pricing — model claims $0.25/$0.38/MTok from Jan 2026

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** anthropic/claude-sonnet-4-5-20250929
**Workspace:** `.orch/workspace/og-inv-model-drift-probe-20feb-d21f/`
**Probe:** `.kb/models/orchestration-cost-economics/probes/2026-02-20-model-drift-stale-references-audit.md`
**Beads:** `bd show orch-go-1102`
