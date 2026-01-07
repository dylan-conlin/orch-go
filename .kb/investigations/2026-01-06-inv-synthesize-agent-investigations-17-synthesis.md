## Summary (D.E.K.N.)

**Delta:** Synthesized 17 agent-related investigations into authoritative guide at `.kb/guides/agent-lifecycle.md` covering state management, display, cross-project visibility, and UI patterns.

**Evidence:** Read and analyzed all 17 investigations covering: registry architecture, inter-agent communication, multi-agent synthesis, dashboard UI (cards, detail panels, real-time updates), cross-project visibility, and state consolidation.

**Knowledge:** Agent state exists in 4 layers (tmux, OpenCode memory, OpenCode disk, beads). Beads is source of truth for lifecycle. Dual-mode (tmux + HTTP) serves distinct needs. Display state should be centralized in agents.ts. Cards need stable sort and reserved space patterns.

**Next:** Close - guide updated and comprehensive.

---

# Investigation: Synthesize 17 Agent Investigations

**Question:** What reusable patterns emerge from 17 agent-related investigations, and how can they be consolidated into a single authoritative guide?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Investigations Synthesized

| Date | Investigation | Key Contribution |
|------|---------------|------------------|
| 2025-12-20 | inv-orch-add-agent-registry-persistent | Registry implementation with file locking, merge logic |
| 2025-12-21 | inv-deep-dive-inter-agent-communication | Four-layer state model, dual-mode architecture |
| 2025-12-21 | inv-design-single-agent-review-command | Pre-completion review pattern |
| 2025-12-21 | inv-multi-agent-synthesis | Workspace isolation prevents conflicts, SYNTHESIS.md pattern |
| 2025-12-22 | inv-design-beginner-agent-learning-environment | Onboarding patterns (not directly lifecycle) |
| 2025-12-22 | inv-real-time-agent-activity-display | SSE message.part handling, activity state |
| 2025-12-24 | inv-agent-card-should-show-processing | is_processing from SSE session.status events |
| 2025-12-24 | inv-design-agent-card-click-interaction | Slide-out panel design, state-aware content |
| 2025-12-24 | inv-fix-nanm-runtime-display-agent | Guard against null/undefined timestamps |
| 2025-12-24 | inv-implement-agent-card-slide-out | SSR compatibility, click handler patterns |
| 2025-12-24 | inv-improve-active-agent-titles-show | Collapsed preview in section headers |
| 2025-12-25 | debug-agent-cards-dashboard-grow-shrink | Reserved space pattern for consistent card height |
| 2025-12-25 | inv-agent-card-has-excess-whitespace | Remove redundant TLDR display |
| 2025-12-25 | inv-cross-project-agent-visibility-fetch | PROJECT_DIR extraction for cross-project beads queries |
| 2025-12-25 | inv-regression-agent-cards-jostling-first | Stable sort using spawned_at (no regression found) |
| 2025-12-26 | inv-design-proper-cross-project-agent | Multi-project workspace aggregation |
| 2026-01-04 | inv-phase-consolidate-agent-status-model | computeDisplayState centralization |

---

## Findings

### Finding 1: Agent state exists in four independent layers

**Evidence:** Investigation `2025-12-21-inv-deep-dive-inter-agent-communication` identified that agent state spans tmux windows, OpenCode in-memory, OpenCode on-disk, and beads comments. The registry was a fifth layer attempting to cache all four, which caused drift.

**Source:** .kb/investigations/2025-12-21-inv-deep-dive-inter-agent-communication.md (lines 40-55)

**Significance:** This is the foundational architectural understanding. The solution is to query authoritative sources directly rather than maintaining a caching layer.

---

### Finding 2: Beads is the single source of truth for lifecycle

**Evidence:** Multiple investigations converge on this: session idle ≠ completion, Phase: Complete comment is the only reliable signal, OpenCode sessions persist indefinitely and don't indicate completion.

**Source:** 
- inv-deep-dive-inter-agent-communication (Finding 4)
- inv-multi-agent-synthesis (kn entry kn-bef2d9)
- inv-design-single-agent-review-command

**Significance:** This eliminates confusion about what "done" means. Agents must explicitly report Phase: Complete.

---

### Finding 3: UI requires stable sort and reserved space patterns

**Evidence:** Card jostling was fixed by using spawned_at (immutable) for Active/Recent sections. Card height variance was fixed by always rendering section containers with placeholders.

**Source:**
- inv-regression-agent-cards-jostling-first (verified fix exists)
- debug-agent-cards-dashboard-grow-shrink (reserved space pattern)

**Significance:** These are fundamental UI patterns that prevent visual instability in grid layouts with dynamic data.

---

### Finding 4: Cross-project visibility requires PROJECT_DIR awareness

**Evidence:** When agents are spawned with `--workdir`, their workspaces go to the target project. The solution is to discover projects from OpenCode session directories and aggregate workspace caches.

**Source:**
- inv-cross-project-agent-visibility-fetch (implementation)
- inv-design-proper-cross-project-agent (architecture)

**Significance:** Without this, dashboard shows "Waiting for activity..." for cross-project agents.

---

### Finding 5: Display state computation should be centralized

**Evidence:** The `2026-01-04-inv-phase-consolidate-agent-status-model` moved getDisplayState from agent-card.svelte to agents.ts as computeDisplayState.

**Source:** inv-phase-consolidate-agent-status-model

**Significance:** Centralizing domain logic in the store enables reuse across components and maintains consistency.

---

## Synthesis

**Key Patterns Extracted:**

1. **Four-Layer State Model** - Agent state spans tmux, OpenCode (memory + disk), and beads. Beads is authoritative for lifecycle.

2. **Dual-Mode Architecture** - tmux for visual access, HTTP API for programmatic access. Both needed, serving different purposes.

3. **Reserved Space Pattern** - Always render UI containers with placeholders to prevent layout jitter.

4. **Stable Sort Pattern** - Use immutable fields (spawned_at) for grid layouts to prevent card jostling.

5. **Cross-Project Aggregation** - Discover projects from OpenCode sessions, build multi-project workspace caches.

6. **Centralized Display State** - computeDisplayState in agents.ts, not in components.

**Answer to Investigation Question:**

The 17 investigations reveal a coherent evolution of the agent lifecycle system:
- Architecture was clarified (four layers, beads as source of truth)
- UI patterns were established (stable sort, reserved space, SSE handling)
- Cross-project visibility was designed and implemented
- Domain logic was centralized (display state computation)

All key findings have been synthesized into `.kb/guides/agent-lifecycle.md` as the single authoritative reference.

---

## Structured Uncertainty

**What's tested:**
- ✅ Four-layer state model is accurate (comprehensive code analysis in deep-dive investigation)
- ✅ Stable sort fix exists and works (verified in regression investigation)
- ✅ Reserved space pattern prevents height jitter (visual verification)

**What's untested:**
- ⚠️ Multi-project workspace aggregation performance with many projects (not benchmarked)
- ⚠️ Edge cases in cross-project visibility (projects deleted, permissions issues)

**What would change this:**
- Finding that beads comments are unreliable would invalidate lifecycle detection
- Performance issues with multi-project scanning would require optimization

---

## Implementation Recommendations

**Recommendation:** Close - guide updated and comprehensive

The existing guide at `.kb/guides/agent-lifecycle.md` has been updated with all patterns from the 17 investigations. No additional implementation needed.

**Future maintainers should:**
- Update the guide when new lifecycle patterns are discovered
- Add to "Key Decisions (Settled)" section when questions are resolved
- Refer agents to this guide before spawning lifecycle investigations

---

## References

**Investigations Analyzed:** 17 investigations in `.kb/investigations/` (see table above)

**Guide Updated:** `.kb/guides/agent-lifecycle.md`

**Commands Run:**
```bash
kb chronicle "agent"  # Timeline of agent-related artifacts
bd comments add orch-go-0c3zy "Phase: Synthesis - ..."  # Progress tracking
```

---

## Investigation History

**2026-01-06 16:30:** Investigation started
- Initial question: Synthesize 17 agent investigations into guide
- Context: Topic "agent" accumulated 17 investigations needing consolidation

**2026-01-06 17:00:** Completed reading all 17 investigations
- Identified five key patterns across investigations
- Noted existing guide at agent-lifecycle.md

**2026-01-06 17:30:** Investigation completed
- Updated agent-lifecycle.md with comprehensive synthesis
- All patterns documented with code examples
- Status: Complete
