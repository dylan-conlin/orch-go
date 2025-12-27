<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-project gap capture requires extending gap-tracker.json with source project context, not changing orchestrator workflow. The existing `~/.orch/gap-tracker.json` already aggregates gaps from ANY project - no architectural change needed.

**Evidence:** Analyzed `pkg/spawn/learning.go:169-217` - GapTracker uses `~/.orch/gap-tracker.json` (global), not per-project. Missing field: `SourceProject` to identify where gap was discovered. Current `orch learn` works cross-project but can't differentiate gap origins.

**Knowledge:** Gaps are already captured globally. The missing piece is metadata (which project, what project's context was missing). Adding `SourceProject` and `TargetProject` fields to GapEvent enables routing gaps back to orch-go for improvement.

**Next:** Implement Option B (Enhance Existing Tracker with Project Context) - add SourceProject/TargetProject fields to GapEvent, filter in `orch learn` by `--from <project>` to surface orch-go-relevant gaps discovered elsewhere.

---

# Investigation: Design Cross-Project Gap Capture

**Question:** How should the orchestration system capture gaps in orch-go tooling discovered while orchestrating external projects, given the tension between Pressure Over Compensation (don't manually provide context) and the need to actually improve the system?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Promote recommendation to implementation (feature-impl)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Problem Framing

### The Core Tension

Dylan identified a persistent tension in the orchestration system:

1. **When orchestrating in external projects** (e.g., price-watch, specs-platform), gaps in the orchestration system itself (orch-go) are discovered
2. **But the orchestrator lacks orch-go context** - it's working in price-watch, not orch-go
3. **It can't spawn into orch-go** to fix tooling while orchestrating another project
4. **Manual compensation breaks Pressure Over Compensation** - if Dylan pastes the context, the system never learns

**Dylan's insight:** This tension has persisted throughout the system's existence, suggesting a missing abstraction, not just a missing feature.

### Success Criteria

A good solution must:
1. **Preserve pressure** - Gaps must create pressure on the system to improve, not be manually compensated
2. **Not break flow** - Orchestrator shouldn't have to context-switch to orch-go mid-work
3. **Be discoverable** - Gaps captured in price-watch must surface when working on orch-go later
4. **Be actionable** - Captured gaps should lead to beads issues or improvements in orch-go

### Constraints

- Gap tracker already exists at `~/.orch/gap-tracker.json` (global location)
- beads is per-repo by design - gaps in orch-go can't be tracked in price-watch's beads
- Orchestrator sessions have limited context when working in external projects
- Pressure Over Compensation principle must be respected

---

## Findings

### Finding 1: Gap Tracker is Already Global

**Evidence:** The gap tracker file is stored at `~/.orch/gap-tracker.json`, not per-project.

**Source:** `pkg/spawn/learning.go:142-147`
```go
func defaultTrackerPath() string {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return ""
    }
    return filepath.Join(homeDir, ".orch", "gap-tracker.json")
}
```

**Significance:** Gaps are already captured globally regardless of which project the spawn occurs in. The gap-tracker already aggregates across all projects. This means the infrastructure for cross-project gap capture exists - we just need to add project context metadata.

---

### Finding 2: GapEvent Currently Lacks Project Context

**Evidence:** Current GapEvent struct has no field indicating which project the gap was discovered in or which project the gap affects.

**Source:** `pkg/spawn/learning.go:27-55`
```go
type GapEvent struct {
    Timestamp      time.Time `json:"timestamp"`
    Query          string    `json:"query"`
    GapType        string    `json:"gap_type"`
    Severity       string    `json:"severity"`
    Skill          string    `json:"skill,omitempty"`
    Task           string    `json:"task,omitempty"`
    ContextQuality int       `json:"context_quality"`
    Resolution     string    `json:"resolution,omitempty"`
    ResolutionDetails string `json:"resolution_details,omitempty"`
}
```

**Significance:** Without project context, we cannot:
- Filter gaps by source project (e.g., "show me gaps discovered while working on price-watch")
- Route gaps to the correct project for resolution (e.g., "this is a gap in orch-go knowledge")
- Distinguish between "missing context in price-watch" vs "missing context about orch-go tooling"

---

### Finding 3: The Dogfooding Loop Works Within orch-go

**Evidence:** When working IN orch-go, the flow is complete: gaps → issues → fixes. The `orch learn` command surfaces recurring gaps and suggests `bd create` or `kn` commands.

**Source:** `cmd/orch/learn.go:311-341` (runLearnAct function)

**Significance:** The system works when there's no cross-project boundary. The issue is specifically when the gap is discovered in price-watch but needs to be addressed in orch-go. The closed loop breaks because beads is per-repo.

---

### Finding 4: kb context Already Works Cross-Project

**Evidence:** `kb context "topic" --global` searches across all registered projects.

**Source:** `~/.orch/ECOSYSTEM.md:56-57`
> **Cross-Repo Capability:** `kb context --global` searches across 17+ registered projects

**Significance:** Knowledge discovery already works cross-project. The issue is specifically about gap CAPTURE, not gap discovery. This suggests the solution should mirror kb's approach - capture locally (in tracker), query globally.

---

## Synthesis

**Key Insights:**

1. **The infrastructure exists** - Gap tracker is already global. No new storage mechanism needed.

2. **Missing piece is metadata** - GapEvent needs to know: (a) which project spawned the agent (source), and (b) which project's context was missing (target, often inferrable from the query).

3. **Routing, not restructuring** - The solution is about routing captured gaps to the right place for action, not changing where gaps are captured.

4. **Pressure is already applied** - Gaps create friction (context quality warnings, gap gating). The issue is that the pressure isn't being channeled to orch-go when the gap is about orch-go tooling.

**Answer to Investigation Question:**

The solution is to enhance the existing GapEvent struct with project context metadata (`SourceProject`, optionally `TargetProject`), then add filtering to `orch learn` to surface gaps by project. This maintains the global gap tracker while enabling project-specific views.

The key insight: **gaps about orch-go tooling discovered in price-watch are still captured** - they're just not easily filterable. Adding project context enables `orch learn --from price-watch` (what gaps did I hit while there?) or `orch learn --affecting orch-go` (what gaps suggest orch-go needs improvement?).

---

## Structured Uncertainty

**What's tested:**

- ✅ Gap tracker is global (verified: read gap-tracker.json path in learning.go:142-147)
- ✅ GapEvent struct lacks project fields (verified: read struct definition at learning.go:27-55)
- ✅ kb context works cross-project (verified: read ECOSYSTEM.md)
- ✅ orch learn act runs commands (verified: read learn.go runLearnAct)

**What's untested:**

- ⚠️ Whether orchestrators actually encounter orch-go gaps in external projects (assumed from Dylan's observation)
- ⚠️ Whether filtering by project will actually surface useful patterns (hypothesis)
- ⚠️ Whether TargetProject can be reliably inferred from query (needs implementation testing)

**What would change this:**

- Finding would be wrong if gaps about orch-go tooling are rare (the problem may not be worth solving)
- Finding would be wrong if gaps are too heterogeneous to filter usefully (project metadata might not help)
- Finding would be wrong if the real issue is orchestrator context, not gap capture (may need different solution)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Option B: Enhance Existing Tracker with Project Context**

Add `SourceProject` and `TargetProject` fields to GapEvent, then add filtering to `orch learn`.

**Why this approach:**
- Minimal change - extends existing infrastructure rather than creating new mechanisms
- Preserves Pressure Over Compensation - gaps still create pressure, now routable
- Enables both "where was I?" and "what needs fixing?" queries
- Aligns with kb's pattern (capture locally, query globally)

**Trade-offs accepted:**
- Orchestrator still can't spawn into orch-go mid-work (accepted - would break flow)
- Gaps need later processing via `orch learn` (accepted - this is the existing pattern)
- TargetProject inference may be imprecise (acceptable - fallback to manual annotation)

**Implementation sequence:**
1. **Extend GapEvent** - Add `SourceProject` (required, detected from cwd) and `TargetProject` (optional, inferred or empty)
2. **Update RecordGap** - Capture source project from current working directory
3. **Add --from/--affecting flags to orch learn** - Filter suggestions by project
4. **Update orch learn suggest** - Group by target project when known

### Alternative Approaches Considered

**Option A: orch gap Command (Lightweight Reporting)**
- **Mechanism:** Simple `orch gap "description"` command that records to orch-go's beads from any project
- **Pros:** Explicit capture, minimal infrastructure change, immediately actionable
- **Cons:** Manual capture breaks flow (orchestrator must remember to run command); doesn't leverage existing gap detection; creates duplicate issue tracking alongside gap tracker
- **When to use instead:** If gaps are rare and high-value enough to warrant explicit capture

**Option C: Always Orchestrate from orch-go**
- **Mechanism:** Orchestrator sessions always start in orch-go, use `--workdir` for external projects
- **Pros:** All gaps naturally route to orch-go's beads; no cross-project gap capture needed
- **Cons:** Beads issues live in orch-go for work done elsewhere (confusing); workspace artifacts scattered; doesn't solve the gap detection problem, just the routing problem
- **When to use instead:** If wanting to centralize ALL orchestration tracking, not just gaps

**Option D: Automatic Gap → Issue Creation**
- **Mechanism:** When a gap is detected, automatically create beads issue in orch-go (even from external projects)
- **Pros:** Zero manual intervention; immediate pressure applied
- **Cons:** Would create noise (every gap becomes an issue); requires cross-repo beads integration; may create issues for one-time gaps
- **When to use instead:** If most gaps are high-value and warrant immediate tracking

**Rationale for recommendation:** Option B builds on existing infrastructure (gap tracker is already global), doesn't require cross-repo beads integration (which violates beads' per-repo design), and enables the learning loop to work across projects without changing the orchestrator workflow.

---

### Implementation Details

**What to implement first:**
1. Add `SourceProject string` field to GapEvent struct
2. Update recordGapForLearning() to detect and set SourceProject from pwd
3. Add `--from <project>` flag to `orch learn` to filter by source project

**Things to watch out for:**
- ⚠️ Project detection should use directory name, not full path (for portability)
- ⚠️ GapEvent already has many fields - consider JSON backward compatibility
- ⚠️ orch learn suggestions already have limit of 5 - filtering may need pagination

**Areas needing further investigation:**
- TargetProject inference from query - may need NLP or heuristics
- Whether to add project context to the spawn context warning messages
- Whether gap gating should consider source/target project

**Success criteria:**
- ✅ Gaps captured in price-watch with SourceProject=price-watch
- ✅ `orch learn --from price-watch` shows gaps from that project
- ✅ Gaps that mention "orch" or "spawn" can be filtered for orch-go improvement
- ✅ No change to existing gap detection or recording behavior

---

## References

**Files Examined:**
- `pkg/spawn/learning.go` - GapTracker implementation, GapEvent struct
- `cmd/orch/learn.go` - orch learn command implementation
- `cmd/orch/main.go:4626-4661` - recordGapForLearning function
- `~/.orch/ECOSYSTEM.md` - Cross-repo patterns documentation
- `~/.kb/principles.md` - Pressure Over Compensation principle

**Commands Run:**
```bash
# Check current gap tracker
cat ~/.orch/gap-tracker.json | head -100

# Verify gap tracker path
grep -n "gap-tracker" pkg/spawn/learning.go
```

**External Documentation:**
- N/A (self-contained within orch ecosystem)

**Related Artifacts:**
- **Decision:** N/A (this will become a decision if recommendation accepted)
- **Investigation:** N/A
- **Workspace:** `.orch/workspace/og-arch-design-cross-project-27dec/`

---

## Investigation History

**2025-12-27 XX:XX:** Investigation started
- Initial question: How to capture orch-go gaps discovered while orchestrating external projects
- Context: Dylan observed persistent tension - gaps discovered in external projects have no path back to orch-go improvement

**2025-12-27 XX:XX:** Key discovery - gap tracker is global
- Found that gap-tracker.json lives in ~/.orch/, not per-project
- Realized the infrastructure exists, just missing project metadata

**2025-12-27 XX:XX:** Investigation completed
- Status: Complete
- Key outcome: Recommend extending GapEvent with SourceProject field and adding --from filter to orch learn
