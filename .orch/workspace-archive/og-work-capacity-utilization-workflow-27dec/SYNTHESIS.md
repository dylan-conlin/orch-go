# Session Synthesis

**Agent:** og-work-capacity-utilization-workflow-27dec
**Issue:** orch-go-t0fz
**Duration:** 2025-12-27
**Outcome:** success

---

## TLDR

Designed a "Triage Batch Workflow" for capacity-aware issue labeling. Key finding: skill selection judgment is preserved through correct issue typing at creation time—skill:* labels are not implemented and not needed. No code changes required, only documentation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-inv-capacity-utilization-workflow-orchestrator-proactively.md` - Full investigation with Triage Batch Workflow design

### Files Modified
- None (design session, no code changes)

### Commits
- None yet (investigation file created, not committed)

---

## Evidence (What Was Observed)

- Daemon's `InferSkill()` only uses issue type, not labels (pkg/daemon/daemon.go:302-316)
- Skill label support (`skill:*`) mentioned in orchestrator skill but not implemented in code
- Type→skill mapping is deterministic: bug→systematic-debugging, feature/task→feature-impl, investigation→investigation
- WorkerPool provides capacity tracking via `AvailableSlots()` and `Reconcile()` (pkg/daemon/daemon.go:200-266)
- Current triage model is binary: `triage:ready` (daemon picks up) vs `triage:review` (needs review)

### Tests Run
```bash
# Searched for skill label implementation
grep -r "skill:" pkg/daemon/*.go cmd/orch/*.go
# Result: No matches - skill labels not implemented

# Verified InferSkill logic
grep -r "InferSkill" pkg/daemon/*.go
# Result: Found in daemon.go:302-316, only uses issueType
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-inv-capacity-utilization-workflow-orchestrator-proactively.md` - Triage Batch Workflow design

### Decisions Made
- Skill selection judgment is preserved at issue creation time (choosing correct type), not labeling time
- Binary triage:ready/triage:review model is sufficient—no need for skill:* labels
- Batch-labeling trigger: when 3+ ready issues exist AND capacity permits
- Batch size limit: min(ready_issues, available_slots + 2) to prevent overwhelming daemon

### Constraints Discovered
- skill:* labels not implemented in daemon—only issue type→skill inference
- Orchestrator has no automated capacity check before labeling (gap)
- Issue type must be set correctly at creation time for correct skill inference

### Externalized via `kn`
- Not applicable for this design session (workflow is documented in investigation)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with workflow design)
- [x] Tests passing (N/A - design session, no code)
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-t0fz`

### Follow-up Work (Optional)
**Issue:** Add Triage Batch Workflow section to orchestrator skill
**Skill:** feature-impl
**Context:**
```
Add the Triage Batch Workflow documentation from the investigation file 
to ~/.claude/skills/meta/orchestrator/SKILL.md. This provides explicit 
triggers and thresholds for capacity-aware batch labeling.
```

**Issue:** Implement skill:* label support in daemon (low priority)
**Skill:** feature-impl
**Context:**
```
Add skill:* label parsing to daemon's issue selection. When present, 
use label skill instead of InferSkill(). Edge case handling for when 
type→skill inference is wrong.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How often does type→skill inference produce the wrong skill? Need data from real usage.
- Should `bd label` command automatically check capacity before labeling?
- Would an `orch triage` command that automates the batch workflow be useful?

**Areas worth exploring further:**
- Daemon metrics on skill selection accuracy
- Orchestrator productivity metrics (how much time spent on triage?)

**What remains unclear:**
- Maximum safe batch size before rate-limiting becomes an issue
- Whether orchestrators will actually follow the workflow (behavioral adoption)

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-work-capacity-utilization-workflow-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-inv-capacity-utilization-workflow-orchestrator-proactively.md`
**Beads:** `bd show orch-go-t0fz`
