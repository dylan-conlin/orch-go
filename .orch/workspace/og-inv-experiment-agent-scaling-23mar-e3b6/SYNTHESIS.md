# Session Synthesis

**Agent:** og-inv-experiment-agent-scaling-23mar-e3b6
**Issue:** orch-go-0p844
**Duration:** 2026-03-23 09:30 -> 2026-03-23 10:30
**Outcome:** success

---

## Plain-Language Summary

Structural placement (telling agents "put your code after function X") works perfectly at N=2 agents but we had no data for more agents. This experiment tested 4 and 6 agents competing for only 3 insertion points in the same file. The result: placement degrades gracefully, not catastrophically. Pairs of agents in separate regions still merge 100% of the time, but pairs forced to share a region (because there aren't enough regions to go around) always conflict. A second, previously invisible problem emerged: when agents need different imports (e.g., one needs `math`, another needs `unicode`), they both restructure the same import block, causing conflicts even when their functions are in different parts of the file. The practical takeaway for daemon design: route agents to different files whenever possible (N=1 per file), and when same-file coordination is unavoidable, ensure insertion-points >= agents AND import compatibility.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

**Key outcomes:**
- 130 agent invocations completed (all 4/4 individual scores)
- 210 pairwise merge checks + 25 N-way merge checks
- Degradation curve: 100% (N=2) -> 70% (N=4) -> 67% (N=6) pairwise with placement
- Two conflict mechanisms identified and verified via diff analysis
- Probe merged into coordination model

---

## Delta (What Changed)

### Files Created
- `pkg/scaling/scaling.go` - 3-function target file (Normalize, Clamp, Wrap) for constrained insertion points
- `pkg/scaling/scaling_test.go` - Tests for scaling package
- `experiments/coordination-demo/redesign/prompts/scaling/agent-{a,b,c,d,e,f}.md` - 6 task prompts
- `experiments/coordination-demo/redesign/run-scaling.sh` - N-agent scaling experiment harness
- `.kb/models/coordination/probes/2026-03-23-probe-agent-scaling-limited-insertion-points.md` - Probe

### Files Modified
- `.kb/models/coordination/model.md` - Merged scaling findings into Claim 2, updated open questions

### Results Directories
- `experiments/coordination-demo/redesign/results/scaling-n4-20260323-093936/` - N=4, 3 conditions
- `experiments/coordination-demo/redesign/results/scaling-n6-20260323-095838/` - N=6, no-placement
- `experiments/coordination-demo/redesign/results/scaling-n6-20260323-100553/` - N=6, even-placement

---

## Evidence (What Was Observed)

### Finding 1: Graceful Degradation Curve
Pairwise success with placement: 100% (N=2) -> 70% (N=4) -> 67% (N=6). Not cliff-edge.

### Finding 2: Deterministic Conflict Structure
At N=6 with even-placement, every pair is 5/5 SUCCESS or 0/5 CONFLICT. Zero stochastic variance. Conflict structure fully determined by region assignment + import compatibility.

### Finding 3: Import Block as Hidden Gravitational Point
Import conflicts were invisible at N=2 (both tasks needed same imports). At N>2 with diverse tasks, agents restructure the import block differently, causing conflicts between agents in separate regions. This is a new conflict mechanism not predicted by the model.

### Finding 4: Sub-Placement Ineffectiveness
"Place first after X" vs "place last before Y" produced identical results to simple "place after X" (21/30 both). Agents ignore sub-region directives and converge on the anchor function.

### Finding 5: N-Way Merge Always Fails
0/25 N-way merges succeeded across all conditions. With any conflict pair present, sequential merge fails.

### Tests Run
```bash
go test ./pkg/scaling/ -v  # PASS — baseline package compiles and tests pass
# 130 agent invocations all scored 4/4
# 210 pairwise + 25 N-way merge checks
```

---

## Architectural Choices

No architectural choices — task was a controlled experiment, not implementation.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Import block is a shared modification point that function-placement constraints cannot address
- Sub-region placement instructions are ignored by agents (gravitational pull is absolute)
- N-way merge has multiplicative failure: one bad pair kills the whole merge
- Conflict structure is deterministic and predictable from region assignments + import sets

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Probe file created with all 4 required sections
- [x] Probe findings merged into parent model
- [x] SYNTHESIS.md created
- [x] Ready for `orch complete orch-go-0p844`

---

## Unexplored Questions

- Can import-block conflicts be solved by a pre-merge import normalization step (e.g., `goimports` on merged output)?
- Does this finding generalize to languages with per-line imports (Python, JS)?
- Would sequential (not parallel) agent execution with intermediate commits solve N-way merge?
- Is the 67% pairwise rate at N=6 a floor, or does it continue degrading at N=10+?

---

## Friction

- `go build ./...` picked up stale committed Go files in experiment result directories, causing false build failures. Fixed by targeting builds to `./pkg/scaling/`. Infrastructure issue in experiment harness design.
- `local` keyword used outside a function in bash caused non-fatal error. Fixed.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-experiment-agent-scaling-23mar-e3b6/`
**Probe:** `.kb/models/coordination/probes/2026-03-23-probe-agent-scaling-limited-insertion-points.md`
**Beads:** `bd show orch-go-0p844`
