# Session Synthesis

**Agent:** og-work-automated-reflection-daemon-06jan-111c
**Issue:** orch-go-lxux2
**Duration:** 2026-01-06 11:15 → 2026-01-06 12:00
**Outcome:** success

---

## TLDR

Designed the full automation loop for knowledge maintenance via kb reflect. The daemon should auto-create issues for two high-signal types (synthesis with 10+ investigations, open items >3 days old) while surfacing other types (promote, stale, drift, skill-candidate, refine) for orchestrator review only.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-automated-reflection-daemon-kb-reflect.md` - Comprehensive design for automated reflection

### Files Modified
- None - this was a design-session investigation

### Commits
- None yet - design artifact ready for implementation

---

## Evidence (What Was Observed)

- `kb reflect --help` shows 7 reflection types available (synthesis, open, promote, stale, drift, refine, skill-candidate)
- `kb reflect --type skill-candidate` returns 72 entries for "spawn" topic alone - high noise, low signal
- `kb reflect --type open` returns only 4 items, all with explicit Next: actions - high signal
- Current daemon already supports `ReflectEnabled`, `ReflectInterval`, `ReflectCreateIssues` config (daemon.go:37-47)
- `pkg/daemon/reflect.go:107-115` passes `--type synthesis --create-issue` when `createIssues=true`

### Tests Run
```bash
# Checked reflection types and signal quality
kb reflect --type synthesis --format json | head -50  # 39 dashboard, 36 spawn - topics are meaningful
kb reflect --type open --format json  # 4 items with explicit actions
kb reflect --type skill-candidate --format json | head -100  # 72 spawn entries - noisy
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-06-inv-automated-reflection-daemon-kb-reflect.md` - Full automation design with type-by-type analysis

### Decisions Made
- **Two-tier automation**: synthesis + open auto-create issues; all others surface-only
  - Rationale: Signal quality varies dramatically by type. Only high-signal patterns (explicit synthesis need, explicit forgotten commitments) should create issues automatically.
  
- **Open type threshold**: Any item >3 days old triggers issue creation
  - Rationale: Open items have self-declared Next: actions - they're already flagged as actionable by the agent who created them.

- **Surfacing-only types remain manual**: promote, stale, drift, skill-candidate, refine
  - Rationale: These require human judgment to triage. Auto-creating issues would generate noise.

### Constraints Discovered
- Skill-candidate uses keyword-based clustering, not semantic grouping - produces noisy results
- Issue deduplication needed - don't create duplicate issues for same open investigation

### Externalized via `kn`
- None - decisions captured in investigation artifact

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation artifact with full design)
- [x] Investigation file has all sections filled
- [x] Design is implementation-ready with file targets and sequence
- [x] Ready for `orch complete orch-go-lxux2`

### Implementation Roadmap (for future spawn)

**Phase 1: kb-cli changes**
- Add `--create-issue` support for open type in `cmd/reflect.go`
- Issue format: "Complete investigation: {title} - {age} days without action"
- Label: `triage:review`

**Phase 2: orch-go daemon extension**
- Add `ReflectOpenEnabled` and `ReflectOpenInterval` to daemon Config
- Add `RunOpenReflection` function to reflect.go
- Update daemon run loop to call both synthesis and open reflection

**Phase 3: Surfacing improvements**
- Add all type counts to `orch daemon status` output
- Update SessionStart hook to show surfacing-only type summaries

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch daemon reflect` have a `--type` flag for manual runs? (Currently runs all types)
- Should there be a `--clear-surfacing` flag to acknowledge surface-only suggestions without action?
- Could skill-candidate detection be improved with semantic clustering instead of keyword matching?

**Areas worth exploring further:**
- False positive rate measurement for auto-created issues after implementation
- Threshold tuning (10+ for synthesis, 3 days for open) based on real usage

**What remains unclear:**
- Whether open issue auto-close should happen when investigation status changes to Complete (likely yes)

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-work-automated-reflection-daemon-06jan-111c/`
**Investigation:** `.kb/investigations/2026-01-06-inv-automated-reflection-daemon-kb-reflect.md`
**Beads:** `bd show orch-go-lxux2`
