## Summary (D.E.K.N.)

**Delta:** High patch density signals (5+ fixes in same area, 10+ conditions in logic, scattered duplicate logic) indicate missing coherent model that should trigger architect before more patches.

**Evidence:** Dashboard status logic had 10+ conditions from incremental agent patches; 135/360 commits (37%) in last 2 weeks were fixes; 454 investigations in .kb/, many clustered on dashboard/status topics; we only invoked architect skill after weeks of pain.

**Knowledge:** The system lacks a detection mechanism - neither daemon (skill inference), issue-creation (triage), nor kb reflect (pattern detection) currently surface "patch density hotspots." Detection should happen at spawn time via git/code analysis, with architect as the recommended skill instead of feature-impl/systematic-debugging.

**Next:** Implement `orch hotspot` command that analyzes git history and code complexity to surface areas needing architect intervention; integrate into daemon preview and spawn recommendations.

---

# Investigation: Patch Density Architect Escalation

**Question:** How do we detect "high patch density" and proactively escalate to architect before complexity compounds?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Design Session Agent
**Phase:** Complete
**Next Step:** None - Epic created with child tasks
**Status:** Complete

---

## Findings

### Finding 1: The dashboard status problem exemplifies accumulated incoherence

**Evidence:** 
- Dashboard status logic in `serve_agents.go` had 10+ conditions scattered across 350+ lines
- Each condition was added incrementally by agents solving local bugs without stepping back to design
- Prior investigation (2026-01-04-design-dashboard-agent-status-model.md) documented:
  - Line 609 optimization caused idle agents to skip Phase: Complete checks
  - SYNTHESIS.md check appeared in two places (lines 862-868 and 909-930)
  - "The complexity comes from incremental additions without a coherent model"
- Resolution required architect to design "Priority Cascade Model" with explicit priority order

**Source:** 
- `.kb/investigations/2026-01-04-design-dashboard-agent-status-model.md`
- `cmd/orch/serve_agents.go` (1403 lines)
- Git history: 4 fix commits to this file in recent weeks before architect intervention

**Significance:** This is the canonical example of the failure mode. Local patches were correct but globally incoherent. The architect skill was only invoked after weeks of completion bugs.

---

### Finding 2: No current mechanism detects "patch density hotspots"

**Evidence:**
Current touchpoints where detection could happen:
1. **Daemon skill inference** (`pkg/daemon/daemon.go:507-521`) - Only maps issue type to skill:
   - bug → systematic-debugging
   - feature/task → feature-impl
   - investigation → investigation
   - No analysis of target code area or history

2. **Issue creation skill** - Creates beads issues from symptoms, doesn't analyze code history

3. **kb reflect** - Has pattern detection for:
   - synthesis: 3+ investigations on same topic
   - promote: kn entries worth promoting
   - stale: uncited decisions
   - drift: constraints contradicted by code
   - skill-candidate: kn entry clusters
   - **But NOT:** code/git hotspot analysis

4. **Spawn context generation** - Only gathers kb context, not code churn analysis

**Source:** 
- `pkg/daemon/daemon.go:507-566` (InferSkill functions)
- `kb reflect --help` output
- `pkg/spawn/context.go`

**Significance:** The system currently has no way to detect that a code area is accumulating patches and should be escalated to architect. All detection relies on human judgment or post-hoc kb reflect analysis of investigations (not commits/code).

---

### Finding 3: Git and code analysis can surface hotspot signals

**Evidence:**
Potential detection signals identified:
1. **Git history signals:**
   - Fix commit frequency per file (135/360 = 37% fix commits in last 2 weeks)
   - Same file modified in 5+ recent commits
   - "fix:" prefix density per area

2. **Code complexity signals:**
   - High condition count (10+ conditions in status logic)
   - Duplicate logic patterns (SYNTHESIS.md check in two places)
   - Scattered assignments (10+ locations setting `status`)
   - Long functions (350+ lines handling related logic)

3. **Investigation clustering:**
   - Multiple investigations on same topic (20+ dashboard/status/complete investigations)
   - kb reflect already detects 3+ investigations → synthesis

**Source:**
- Git log analysis: `git log --oneline --since="2 weeks ago" -- "*.go" | grep -E "fix" | wc -l` = 135
- Investigation count: `ls .kb/investigations/ | wc -l` = 454
- Investigation clustering: `ls .kb/investigations/ | grep -E "dashboard|status|complete" | wc -l` = 20+

**Significance:** The signals exist but aren't surfaced to decision-makers (orchestrator, daemon). A hotspot analysis tool could detect these patterns automatically.

---

## Synthesis

**Key Insights:**

1. **The detection gap is at spawn/triage time** - By the time we have 10+ conditions or 5+ fix commits, it's too late. The daemon and orchestrator need signals BEFORE spawning another feature-impl/systematic-debugging agent on a hotspot area.

2. **Git history is the most reliable signal** - Code analysis requires parsing and understanding, but git history is simple: "5+ fix commits to same file in last N weeks" is a strong signal. Combined with investigation clustering from kb reflect, this provides multi-signal detection.

3. **The intervention point is skill selection** - The daemon currently infers `bug → systematic-debugging`, `feature → feature-impl`. When spawning on a detected hotspot, it should recommend `architect` instead, with context like "High patch density detected: 6 fix commits, 4 investigations on this area - recommend architect to establish coherent model before more patches."

**Answer to Investigation Question:**

To detect "high patch density" and proactively escalate to architect:

1. **Create `orch hotspot` command** that analyzes:
   - Git commit history (fix commit frequency per file/area)
   - Investigation clustering (from kb reflect synthesis detection)
   - Optionally: code complexity (condition count, function length)

2. **Integrate into spawn workflow**:
   - When spawning on an issue, check if target files/areas are hotspots
   - If hotspot detected, recommend architect skill with explanation
   - Daemon preview shows hotspot warnings for triage:ready issues

3. **Surface in dashboard**:
   - Hotspot indicator on issues targeting high-churn areas
   - "Needs architect" badge when pattern exceeds threshold

---

## Structured Uncertainty

**What's tested:**

- ✅ Dashboard status had 10+ conditions from incremental patches (verified: read investigation and code)
- ✅ 135/360 commits in last 2 weeks were "fix:" prefix (verified: git log)
- ✅ 20+ investigations exist on dashboard/status/complete topics (verified: ls | grep)
- ✅ Current daemon skill inference has no code analysis (verified: read InferSkill functions)

**What's untested:**

- ⚠️ Whether git history signals reliably predict architect need (hypothesis, not validated)
- ⚠️ Appropriate thresholds (5+ fixes? 3+ investigations? unknown)
- ⚠️ Implementation complexity of git analysis at spawn time (not prototyped)
- ⚠️ Whether orchestrators/daemon would act on hotspot warnings (behavioral assumption)

**What would change this:**

- If git history is too noisy (too many false positives), may need more sophisticated analysis
- If implementation is complex, may prefer simpler heuristics first
- If orchestrators ignore warnings, may need gates instead of warnings

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Multi-Signal Hotspot Detection** - Create `orch hotspot` command that combines git history + investigation clustering to surface areas needing architect intervention.

**Why this approach:**
- Git history is readily available and simple to analyze
- Investigation clustering already exists in kb reflect
- Combining signals reduces false positives
- CLI tool enables integration at multiple touchpoints (daemon, spawn, dashboard)

**Trade-offs accepted:**
- Adds analysis step to spawn workflow (minor latency)
- Requires tuning thresholds (start conservative, iterate)
- Won't catch all cases (some hotspots have no prior investigations)

**Implementation sequence:**
1. **Phase 1: CLI tool (`orch hotspot`)** 
   - Analyze git history for fix commit density per file
   - Query kb reflect for investigation clustering
   - Output: List of files/areas with hotspot scores

2. **Phase 2: Spawn integration**
   - In `orch spawn --issue`, check hotspot status of target area
   - If hotspot detected, print warning with architect recommendation
   - Add `--hotspot-check` flag (default: on)

3. **Phase 3: Daemon integration**
   - In `daemon preview`, show hotspot warnings
   - Consider auto-upgrading skill to architect for high-score hotspots

4. **Phase 4: Dashboard integration**
   - Show hotspot badges on issues
   - "Needs architect" indicator

### Alternative Approaches Considered

**Option B: kn constraint only (no tooling)**
- **Pros:** Zero implementation, immediate effect
- **Cons:** Relies on human memory to check; exactly what we did and failed at
- **When to use instead:** If tooling is blocked or low priority

**Option C: Gate-based (block spawns to hotspots without architect)**
- **Pros:** Enforces escalation, can't be ignored
- **Cons:** May be too restrictive; blocks legitimate quick fixes
- **When to use instead:** After Option A proves orchestrators ignore warnings

**Option D: Investigation-only trigger (use kb reflect)**
- **Pros:** Already exists; extend kb reflect with "hotspot" type
- **Cons:** Investigations lag behind commits; misses pure-code hotspots
- **When to use instead:** If git analysis is too complex

**Rationale for recommendation:** Option A provides detection at the right moment (spawn time) with actionable output (skill recommendation). It combines existing signals (kb reflect) with new analysis (git history) for robust detection. Options B-D are either too weak (B, D) or too strong (C) for initial rollout.

---

### Implementation Details

**What to implement first:**
- `orch hotspot` CLI command with git analysis
- Thresholds: 5+ fix commits in 4 weeks, 3+ investigations → "hotspot"
- Output format: JSON for integration, text for humans

**Things to watch out for:**
- ⚠️ Git history analysis needs to handle repos with different commit styles
- ⚠️ Investigation clustering query may be slow for large .kb/ dirs
- ⚠️ Threshold tuning will require iteration based on false positive/negative rates
- ⚠️ Cross-repo hotspots (issue in repo A, code in repo B) need special handling

**Areas needing further investigation:**
- Optimal thresholds (start with 5/3, tune based on results)
- Code complexity analysis (may add later if git signals insufficient)
- Integration with beads issue types (should hotspot issues auto-get "needs-architect" label?)

**Success criteria:**
- ✅ `orch hotspot` command exists and outputs hotspot analysis
- ✅ Spawn to hotspot area prints warning with architect recommendation
- ✅ Next dashboard status-like problem is caught BEFORE 5+ fix commits

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go:507-566` - InferSkill functions
- `cmd/orch/serve_agents.go` - The 1403-line file that exemplifies the problem
- `.kb/investigations/2026-01-04-design-dashboard-agent-status-model.md` - Prior architect work

**Commands Run:**
```bash
# Fix commit ratio
git log --oneline --since="2 weeks ago" -- "*.go" | wc -l  # 360
git log --oneline --since="2 weeks ago" -- "*.go" | grep -E "fix" | wc -l  # 135

# Investigation count
ls .kb/investigations/ | wc -l  # 454
ls .kb/investigations/ | grep -E "dashboard|status|complete" | wc -l  # 20+

# kb reflect capabilities
kb reflect --help
```

**Related Artifacts:**
- **Constraint:** `kn constrain` entry already exists: "High patch density in a single area signals missing coherent model - spawn architect before more patches"
- **Investigation:** `.kb/investigations/2026-01-04-design-dashboard-agent-status-model.md` - The architect work that resolved the dashboard problem

---

## Investigation History

**2026-01-04 11:45:** Investigation started
- Initial question: How to detect patch density and escalate to architect?
- Context: Dashboard status logic accumulated 10+ conditions from incremental patches

**2026-01-04 12:00:** Context gathering complete
- Found prior investigation on dashboard status model
- Confirmed no current detection mechanism in daemon/spawn
- Identified git history + kb reflect as signal sources

**2026-01-04 12:30:** Synthesis in progress
- Recommending multi-signal hotspot detection
- CLI tool + spawn/daemon integration
