# Session Synthesis

**Agent:** og-arch-evaluate-encoding-options-19jan-a9e0
**Issue:** orch-go-7hd6h
**Duration:** 2026-01-19 22:41 → 2026-01-19 23:00
**Outcome:** success

---

## TLDR

Evaluated three encoding options for question subtypes (factual/judgment/framing). Recommend **labels with convention** (`subtype:{factual|judgment|framing}`) - works today with no schema changes, follows existing patterns like `triage:ready`.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-19-inv-evaluate-encoding-options-question-subtypes.md` - Full analysis with recommendation

### Files Modified
- None

### Commits
- Pending (will be committed with this SYNTHESIS.md)

---

## Evidence (What Was Observed)

- Beads Issue struct has `Labels []string` field but no dedicated subtype field (`pkg/beads/types.go:148`)
- Daemon already filters by labels via `issue.HasLabel()` (`pkg/daemon/daemon.go:331`)
- BD CLI supports `--label` filtering for `bd ready` (verified via `bd ready --help`)
- Questions are excluded from default `bd ready` (correct - they're not Work nodes)
- `bd ready --type question --label subtype:factual` would work today

### Tests Run
```bash
# Verified label filtering support
bd ready --help
# Confirmed: --label and --label-any flags available

# Listed open questions
bd list --type question --status open
# Found 2 open questions, no subtypes labeled yet
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-19-inv-evaluate-encoding-options-question-subtypes.md` - Full evaluation with three options

### Decisions Made
- **Recommend labels over dedicated field** because:
  - Zero schema changes required
  - Follows existing pattern (`triage:ready`)
  - Labels can change as questions evolve (factual → framing)
  - Daemon infrastructure already supports label filtering

### Constraints Discovered
- `answered` status doesn't unblock dependencies (only `closed` does) - separate constraint, not addressed here
- No schema validation on labels (freeform) - acceptable given need for flexibility

### Externalized via `kn`
- N/A - Recommend promoting investigation to decision if adopted

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-7hd6h`

### Follow-up Work (optional)
**Issue:** Update decidability-graph.md to reference encoding convention
**Skill:** feature-impl
**Context:**
```
Add section to decidability-graph.md documenting the label convention:
subtype:factual, subtype:judgment, subtype:framing
Reference this investigation as the source.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should daemon auto-spawn investigations for factual questions? (would need IsSpawnableType to allow `question` with `subtype:factual`)
- How should dashboard display questions grouped by subtype?

**Areas worth exploring further:**
- Integration of subtype labels with `bd create --type question` workflow
- Whether subtype can be inferred at creation or must be added later

**What remains unclear:**
- Whether questions actually evolve subtypes in practice (model prediction, not validated)
- How to handle questions with unknown subtype at creation

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-evaluate-encoding-options-19jan-a9e0/`
**Investigation:** `.kb/investigations/2026-01-19-inv-evaluate-encoding-options-question-subtypes.md`
**Beads:** `bd show orch-go-7hd6h`
