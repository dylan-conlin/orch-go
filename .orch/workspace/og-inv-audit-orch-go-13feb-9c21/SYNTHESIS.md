# Session Synthesis

**Agent:** og-inv-audit-orch-go-13feb-9c21
**Issue:** orch-go-b4l
**Duration:** 2026-02-13T → 2026-02-13T
**Outcome:** success

---

## TLDR

Audited how orch-go code implements the model/probe/investigation system across 17 Go source files. Found the forward path (models → spawn context → agent probe guidance) is fully implemented, but the reverse path (probe verdicts → model updates) is completely absent from completion/verification code. `DefaultProbeTemplate` exists as dead code.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-13-inv-audit-orch-go-implementation-model.md` - Phase 2 code audit investigation with 6 findings mapping Phase 1 documentation issues to actual code behavior

### Files Modified
- None (read-only audit)

### Commits
- Investigation file creation and SYNTHESIS.md

---

## Evidence (What Was Observed)

- `pkg/spawn/probes.go:160-203` defines `DefaultProbeTemplate` constant but no other Go code references it — dead code
- `pkg/spawn/kbcontext.go:744-792` builds model injection with Summary, Critical Invariants, Why This Fails sections + recent probes listing
- `pkg/spawn/kbcontext.go:779` writes "Your findings should confirm, contradict, or extend" — creates expectation never fulfilled by completion
- `cmd/orch/complete_cmd.go` has ZERO references to probe, model, or .kb/models — completion is probe-blind
- `pkg/verify/` (entire package: check.go, synthesis_parser.go, discovered_work.go, synthesis_opportunities.go) has ZERO probe references
- `pkg/daemon/skill_inference.go:27-42` maps issue types to skills with no model-existence check
- `pkg/skills/loader.go:51-88` uses first-match-wins strategy — explains duplicate investigation skill versions

### Tests Run
```bash
# No code was modified, so no tests needed
# Verification was via code reading and grep searches across 17 files
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-13-inv-audit-orch-go-implementation-model.md` - Full code audit with 6 findings

### Decisions Made
- None (audit only, recommendations deferred to orchestrator)

### Constraints Discovered
- Probe system is architecturally half-built: forward path complete, reverse path absent
- `DefaultProbeTemplate` is dead code — never used by any function, waiting for kb-cli integration per comment at probes.go:161
- Skill loader first-match-wins means duplicate skill files at different paths silently resolve to whichever is found first

### Externalized via `kn`
- N/A — findings captured in investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with 6 findings + code-to-doc mapping)
- [x] No tests needed (read-only audit)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-b4l`

### Recommended Follow-up Work

Three implementation items identified:
1. **Wire PROBE.md template** — Write `.orch/templates/PROBE.md` from `DefaultProbeTemplate` content, or have kb-cli use the constant
2. **Add probe verdict parsing** — Extend `pkg/verify/synthesis_parser.go` to extract Model Impact / Confirms/Contradicts/Extends
3. **Clean duplicate investigation skill** — Remove `~/.claude/skills/src/worker/investigation/` (old version without probe mode)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Whether `HasInjectedModels` flag (kbcontext.go:508) triggers any downstream behavior beyond telemetry tracking
- Whether the beads issue `orch-go-xxa6e` referenced in probes.go:161 for kb-cli integration was ever resolved
- Whether existing probes in `.kb/models/*/probes/` follow the 4-section structure defined in `DefaultProbeTemplate`

**What remains unclear:**
- Whether `os.ReadDir` ordering is deterministic across macOS APFS — affects which investigation skill version the loader picks
- Whether probe mode in the newer investigation skill actually works end-to-end (no agent has been observed creating a probe via the skill's probe mode)

---

## Verification Contract

**Verification Spec:** N/A (read-only investigation, no VERIFICATION_SPEC.yaml needed)

**Key Outcomes:**
- 6 findings documented with specific file:line references
- All 6 Phase 1 findings mapped to actual code behavior
- Gap analysis: forward path complete, reverse path absent, template dead code

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-audit-orch-go-13feb-9c21/`
**Investigation:** `.kb/investigations/2026-02-13-inv-audit-orch-go-implementation-model.md`
**Beads:** `bd show orch-go-b4l`
