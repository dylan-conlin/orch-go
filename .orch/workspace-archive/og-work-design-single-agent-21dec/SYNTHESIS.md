# Session Synthesis

**Agent:** og-work-design-single-agent-21dec
**Issue:** orch-go-3anf
**Duration:** 2025-12-21 ~12:40 → 2025-12-21 ~13:10
**Outcome:** success

---

## TLDR

Designed `orch complete <id> --preview` command to address the gap where orchestrators couldn't review agent work before completing. Recommend integrating preview into completion flow rather than separate command for better UX.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-design-single-agent-review-command.md` - Design investigation with findings and recommendations
- `.kb/decisions/2025-12-21-single-agent-review-command.md` - Decision artifact for orchestrator review

### Files Modified
- None (design session, no code changes)

### Commits
- (To be committed after this synthesis)

---

## Evidence (What Was Observed)

- Analyzed `cmd/orch/main.go:1560-1673` - Complete flow is verify-then-close with no preview step
- Analyzed `cmd/orch/review.go` - Batch-oriented, groups by project, shows SYNTHESIS cards
- Analyzed `pkg/verify/check.go:282-327` - Verification checks phase status and SYNTHESIS.md existence
- Analyzed `pkg/registry/registry.go:37-61` - Agent metadata includes workspace path, skill, timestamps
- Found SYNTHESIS.md examples in `.orch/workspace/` showing the D.E.K.N. structure available

### Current Gap
The orchestrator workflow has no "review what happened" step between agent completion and beads issue closure. Must manually:
1. Read SYNTHESIS.md
2. Run `bd show <id>` for comments
3. Check git log for commits
4. Then decide to complete

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-design-single-agent-review-command.md` - Full design investigation
- `.kb/decisions/2025-12-21-single-agent-review-command.md` - Decision document awaiting approval

### Decisions Made
- **--preview flag over separate command:** Integrates review into decision point, one command flow
- **Alias for discoverability:** `orch review <id>` can alias to `--preview --dry-run` for users who think "review first"
- **Reuse existing patterns:** printSynthesisCard in review.go shows how to display SYNTHESIS.md data

### Constraints Discovered
- Git operations should scope to agent workspace, not project root
- Beads comment fetch can fail - need graceful degradation
- Interactive prompt requires stdin (scriptable with --yes)

### Externalized via `kn`
- None needed - design captured in decision document

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + decision)
- [x] Design questions answered (separate vs integrated)
- [x] Ready for orchestrator review of decision
- [x] Ready for `orch complete orch-go-3anf`

### Implementation (follow-up if decision accepted)
Create beads issue for: "Implement `orch complete --preview` for single-agent review"

**Scope:**
1. Create `pkg/verify/review.go` with `AgentReview` struct and `GetAgentReview()`
2. Add `--preview` and `--yes` flags to completeCmd
3. Display logic reusing printSynthesisCard patterns
4. Optional: Add `orch review <id>` alias command

---

## Session Metadata

**Skill:** design-session
**Model:** anthropic/claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-work-design-single-agent-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-design-single-agent-review-command.md`
**Decision:** `.kb/decisions/2025-12-21-single-agent-review-command.md`
**Beads:** `bd show orch-go-3anf`
