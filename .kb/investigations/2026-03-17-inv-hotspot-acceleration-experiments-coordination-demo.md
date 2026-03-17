## Summary (D.E.K.N.)

**Delta:** `experiments/coordination-demo/.../display_test.go` hotspot alert (+220 lines/30d) is a false positive — the file is a static experiment result artifact created in a single commit, and the acceleration detector has no path exclusions for `experiments/`.

**Evidence:** Git log shows 1 commit creating the file (307bbcd0e, 2026-03-12). The `experiments/` directory contains 79 `.go` files exceeding the 200-line threshold, all from one-time experiment data capture — not ongoing code growth.

**Knowledge:** The `HotspotAccelerationDetector` in `pkg/daemon/trigger_detectors_phase2.go:314` only filters by `.go` suffix. It applies no path exclusions — unlike `shouldCountFile()` in `cmd/orch/hotspot_analysis.go` which excludes test files, generated files, build dirs, etc. This creates a class of false positives from `experiments/`, `.orch/`, `.beads/`, and other non-production directories.

**Next:** Create issue to add path exclusions to `defaultHotspotAccelerationSource.ListFastGrowingFiles()`. At minimum: `experiments/`, `skipBloatDirs` entries, and `_test.go` files.

**Authority:** architectural - Fix spans daemon detection system (pkg/daemon) and would align its filtering with the existing hotspot CLI (cmd/orch) — cross-component consistency decision.

---

# Investigation: Hotspot Acceleration — experiments/coordination-demo display_test.go

**Question:** Is `experiments/coordination-demo/redesign/results/20260310-174045/messaging/complex/trial-7/agent-b/display_test.go` (+220 lines/30d, now 220 lines) at risk of becoming a critical hotspot requiring extraction?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** Agent orch-go-hqmln
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-thread-thread-test.md | Same pattern — creation-churn false positive | yes | - |
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-control.md | Same pattern — creation-churn false positive | yes | - |

---

## Findings

### Finding 1: File is a static experiment result artifact, not production code

**Evidence:** The file lives at `experiments/coordination-demo/redesign/results/20260310-174045/messaging/complex/trial-7/agent-b/display_test.go`. This is output captured from a coordination demo experiment where two agents simultaneously implemented features in `pkg/display/`. The `prompt.md` in the same directory shows the task prompt given to "agent-b" for the experiment.

**Source:** `experiments/coordination-demo/redesign/results/20260310-174045/messaging/complex/trial-7/agent-b/prompt.md`, directory listing of sibling files (stdout.log, stderr.log, commits.txt, etc.)

**Significance:** Experiment result files are write-once artifacts — they capture what an agent produced during a trial. They never grow incrementally. The +220 lines is file birth, not accretion.

---

### Finding 2: 220 lines added in a single commit, zero subsequent changes

**Evidence:** `git log --diff-filter=A` shows exactly one commit:
```
307bbcd0e 2026-03-12 session: enforcement audit framework, daemon Limit fix, deny hooks audit
```

`git log --diff-filter=M` returns empty — the file has never been modified after creation.

**Source:** `git log --diff-filter=A -- "experiments/.../display_test.go"`, `git log --diff-filter=M -- "experiments/.../display_test.go"`

**Significance:** The +220 lines/30d metric equals the file's total existence. This is the same false-positive pattern documented in the thread_test.go and control_cmd.go investigations.

---

### Finding 3: The acceleration detector has NO path exclusions — 79 false positives from experiments/ alone

**Evidence:** `defaultHotspotAccelerationSource.ListFastGrowingFiles()` at `pkg/daemon/trigger_detectors_phase2.go:314-347` only checks:
1. `added < threshold` (line 329-331)
2. `!strings.HasSuffix(path, ".go")` (line 332-334)
3. File exists and has lines (line 336-338)

It does NOT use `shouldCountFile()`, `skipBloatDirs`, `containsSkippedDir()`, or any path-based exclusion. By contrast, `analyzeFixCommits()` in `cmd/orch/hotspot_analysis.go:57` calls `shouldCountFile()` which excludes test files, generated files, build directories, etc.

Counting `.go` files in `experiments/` with 200+ lines added in last 30 days: **79 files**. Total line additions in `experiments/` in the window: **116,931 lines**. There are 20 copies of `display_test.go` across 10 trials x 2 agents, all potential false positives.

**Source:** `pkg/daemon/trigger_detectors_phase2.go:314-347`, `cmd/orch/hotspot_analysis.go:55-61`, `git log --since="30 days ago" --numstat -- "experiments/"`

**Significance:** This is not a one-off false positive — the entire `experiments/` directory is a systematic source of noise. The detector also lacks exclusions for `.orch/`, `.beads/`, `.claude/`, `node_modules`, and other non-production paths that the bloat detector already handles.

---

## Synthesis

**Key Insights:**

1. **False positive — experiment artifact, not growing code.** The flagged file is a static result from a coordination demo experiment. It was created in one commit and will never change.

2. **Systematic gap in the acceleration detector.** The `HotspotAccelerationDetector` was built with minimal filtering (`.go` suffix only), while the existing `orch hotspot` CLI has mature exclusions via `shouldCountFile()` and `skipBloatDirs`. The detector needs to reuse or replicate those exclusions.

3. **79 false positives currently latent.** If the daemon runs this detector across the full repo, `experiments/` alone would generate 79 spurious investigation issues. Combined with other false-positive patterns (file creation = "growth"), this could flood the backlog.

**Answer to Investigation Question:**

No. This file is not at risk of becoming a hotspot. It's a static experiment artifact that was created in a single commit and will never grow. The hotspot acceleration alert is a false positive caused by the detector's lack of path exclusions. No extraction or architect follow-up is needed for this file. The detector itself needs a fix.

---

## Structured Uncertainty

**What's tested:**

- File has exactly 1 commit creating it, 0 modifications (verified: `git log --diff-filter=A` and `--diff-filter=M`)
- 79 `.go` files in `experiments/` exceed 200-line threshold (verified: `git log --numstat` with awk filter)
- Acceleration detector has no path exclusions (verified: read `pkg/daemon/trigger_detectors_phase2.go:314-347`)

**What's untested:**

- Whether adding `experiments/` to `skipBloatDirs` would cause side effects in other hotspot detectors
- How many non-experiments false positives exist (`.orch/`, `.claude/`, vendor, etc.)

**What would change this:**

- If the file were modified after creation (it hasn't been)
- If `experiments/` contained active production code (it doesn't — it's purely captured artifacts)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add path exclusions to HotspotAccelerationDetector | architectural | Cross-component consistency between daemon detector and hotspot CLI; affects what issues the daemon auto-creates |

### Recommended Approach: Add path exclusions to ListFastGrowingFiles

**Why this approach:**
- Aligns daemon detector with existing `orch hotspot` exclusion patterns
- Eliminates 79+ false positives from `experiments/` alone
- Minimal code change — reuse existing `skipBloatDirs` and `shouldCountFile` patterns

**Trade-offs accepted:**
- May miss genuinely growing files in excluded directories (acceptable — those aren't production code)

**Implementation sequence:**
1. Add `"experiments"` to `skipBloatDirs` in `cmd/orch/hotspot.go` (affects both CLI and provides canonical exclusion list)
2. In `defaultHotspotAccelerationSource.ListFastGrowingFiles()`, add path filtering using `containsSkippedDir()` and optionally `_test.go` exclusion
3. Add test cases for the path exclusions

### Alternative: Exclude experiments/ in git log query

**Pros:** Simpler, no code sharing needed
**Cons:** Doesn't address other non-production directories; diverges from existing exclusion system
**When to use instead:** If the detector is intentionally separate from the hotspot CLI

---

## References

**Files Examined:**
- `experiments/coordination-demo/redesign/results/20260310-174045/messaging/complex/trial-7/agent-b/display_test.go` — The flagged file (220 lines, static experiment artifact)
- `experiments/coordination-demo/redesign/results/20260310-174045/messaging/complex/trial-7/agent-b/prompt.md` — Experiment task prompt confirming artifact nature
- `pkg/daemon/trigger_detectors_phase2.go:314-347` — Acceleration detector source, no path exclusions
- `cmd/orch/hotspot.go:34-54` — `skipBloatDirs` and `defaultExclusions` (mature exclusion system)
- `cmd/orch/hotspot_analysis.go:120-161` — `shouldCountFile()` with comprehensive filtering

**Commands Run:**
```bash
# Verify file creation history
git log --diff-filter=A -- "experiments/.../display_test.go"

# Check for any post-creation modifications
git log --diff-filter=M -- "experiments/.../display_test.go"

# Count false positives from experiments/
git log --since="30 days ago" --numstat -- "experiments/" | awk '$1 != "-" && $1+0 >= 200 && $3 ~ /\.go$/ {count++} END {print count}'
```
