## Summary (D.E.K.N.)

**Delta:** The orchestration system has evolved 6 distinct artifact types serving different lifecycles, but they suffer from scattered placement and unclear handoff protocols.

**Evidence:** Analyzed 100+ workspaces, 140+ investigations, session synthesis examples, SESSION_HANDOFF.md, skills, and beads. Found redundancy (investigations in both .kb/ and workspaces) and missing handoff patterns.

**Knowledge:** Artifacts fall into three temporal categories: ephemeral (workspaces), persistent (knowledge base), and operational (beads). The core tension is cohesion (keep related artifacts together) vs discoverability (standard locations for search).

**Next:** Implement tiered artifact model with explicit promotion paths and adopt SESSION_HANDOFF.md pattern for orchestrator sessions.

**Confidence:** High (85%) - comprehensive audit but limited testing of proposed changes.

---

# Investigation: Deep Pattern Analysis Across Orchestration Artifacts

**Question:** What is the coherent architecture for artifacts across the orchestration ecosystem - where should each type live, how do they relate, and what handoff protocols should exist?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None - ready for decision
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Six Distinct Artifact Types Exist

**Evidence:** Systematic audit identified these artifact types:

| Type | Location | Creator | Purpose | Example |
|------|----------|---------|---------|---------|
| SPAWN_CONTEXT.md | .orch/workspace/{name}/ | orch spawn | Agent initialization | Full skill + task context |
| SYNTHESIS.md | .orch/workspace/{name}/ | Worker agent | Session summary | D.E.K.N. + delta + next |
| Investigation | .kb/investigations/ | Worker agent | Deep research | Question → Evidence → Answer |
| Decision | .kb/decisions/ | Architect/Orchestrator | Architectural choices | Options → Recommendation → Status |
| Session Synthesis | .orch/workspace/ (ad-hoc) | Orchestrator | Daily summary | session-synthesis-21dec.md |
| SESSION_HANDOFF.md | .orch/ | Orchestrator | Cross-session context | skillc/.orch/SESSION_HANDOFF.md |

**Source:** 
- `.orch/workspace/*/SPAWN_CONTEXT.md` - 100+ files
- `.orch/workspace/*/SYNTHESIS.md` - 52 files
- `.kb/investigations/*.md` - 140+ files
- `.kb/decisions/*.md` - 1 file (orch-go)
- `session-synthesis-21dec.md`, `SESSION_HANDOFF.md`

**Significance:** Each artifact serves a specific lifecycle stage. The system has organically evolved these types, but placement and relationships are inconsistent.

---

### Finding 2: Temporal Categories Emerge

**Evidence:** Artifacts fall into three temporal categories:

**Ephemeral (session-bound):**
- SPAWN_CONTEXT.md - created at spawn, read-only thereafter
- SYNTHESIS.md - created at session end, summarizes session
- Agent workspaces - contain both, cleaned after completion

**Persistent (survive sessions):**
- Investigations - accumulated knowledge, discoverable via `kb context`
- Decisions - architectural choices that guide future work
- Session Handoff - explicit handoff for cross-session context

**Operational (work tracking):**
- Beads issues - work queue state
- Beads comments - phase tracking, agent lifecycle
- `.kn/entries.jsonl` - quick decisions, constraints

**Source:** Observed lifecycle patterns across 3 days of system usage (Dec 19-21).

**Significance:** The temporal dimension is the key distinction. Ephemeral artifacts exist for a single agent session; persistent artifacts exist for the project lifetime; operational artifacts track work-in-progress.

---

### Finding 3: Investigation Placement Tension

**Evidence:** Currently investigations appear in two places:

1. `.kb/investigations/` - discovered via `kb context`, standard location
2. `.orch/workspace/{name}/` - coupled with agent workspace

The SYNTHESIS.md template references investigation path: `.kb/investigations/YYYY-MM-DD-*.md`, confirming investigations should live in .kb/, not workspaces.

However, agents working in workspaces create investigations there for cohesion with their work artifacts.

**Source:**
- SYNTHESIS.md template line 102
- Audit of `.kb/investigations/` (140+ files)
- Workspace structure observation

**Significance:** Current practice favors standard location (.kb/) which enables `kb context` discovery. Workspace cohesion is handled via SYNTHESIS.md which points to the investigation.

---

### Finding 4: Orchestrator Sessions Lack Standardized Artifacts

**Evidence:** Two patterns observed for orchestrator session state:

1. **Ad-hoc synthesis:** `session-synthesis-21dec.md` in workspace root
   - No standard naming convention
   - Not discoverable by tools
   - Valuable content (48 commits, 70 issues, lessons learned)

2. **Structured handoff:** `skillc/.orch/SESSION_HANDOFF.md`
   - Clear sections: TLDR, Current State, Priority Work, Key Decisions, Questions
   - Session Start Checklist for resumption
   - Explicitly named for handoff purpose

The second pattern (SESSION_HANDOFF.md) is more amnesia-resilient.

**Source:**
- `.orch/workspace/session-synthesis-21dec.md` (142 lines)
- `skillc/.orch/SESSION_HANDOFF.md` (237 lines)

**Significance:** Orchestrator sessions need standardized handoff, just like worker agents have SYNTHESIS.md. The skillc pattern is more mature.

---

### Finding 5: Beads Comments as State Machine

**Evidence:** Beads comments track agent lifecycle phases:

```
Phase: Planning → Phase: Implementing → Phase: Complete
```

Plus metadata comments:
- `agent_metadata: {...}` - spawn context
- `investigation_path: /path/to/file.md` - artifact location
- `BLOCKED: ...` / `QUESTION: ...` - escalation triggers

**Source:** `.beads/issues.jsonl` samples showing comment progression.

**Significance:** Beads comments are the operational "heartbeat" of agent work. They're the primary mechanism for orchestrator monitoring and verification via `orch complete`.

---

### Finding 6: Skill Embedding vs Discovery

**Evidence:** Two skill loading patterns:

1. **Embedded at spawn:** Full SKILL.md content injected into SPAWN_CONTEXT.md
   - 100+ lines of skill guidance per agent
   - Immediate availability, no discovery needed
   - Spawn context grows large (646 lines in current example)

2. **Dynamic discovery:** Skill tool loads skills on demand
   - Not available in spawned agent contexts
   - Works for orchestrator interactive sessions

**Source:**
- SPAWN_CONTEXT.md size and structure
- `.kb/decisions/2025-11-22-skill-system-hybrid-architecture.md` (referenced in orchestrator skill)

**Significance:** The hybrid approach is intentional - spawned agents get embedded skills, interactive sessions get dynamic discovery. This is correct by design.

---

## Synthesis

**Key Insights:**

1. **Three-tier temporal model** - Artifacts serve ephemeral (session), persistent (project), or operational (work-in-progress) purposes. Understanding this explains where each type should live.

2. **Discoverability trumps cohesion** - The system has standardized on `.kb/` for investigations/decisions because `kb context` needs to find them. SYNTHESIS.md bridges workspace cohesion by pointing to the investigation.

3. **Orchestrator sessions need handoff** - Worker agents have SYNTHESIS.md, but orchestrator sessions lack equivalent. The skillc SESSION_HANDOFF.md pattern should be adopted.

4. **Beads is the heartbeat** - Phase comments are the operational interface between agents and orchestrator. They enable verification without reading artifacts.

5. **Promotion paths exist but are implicit** - Investigations → Decisions, kn entries → principles, workspace findings → kb artifacts. These should be more explicit.

**Answer to Investigation Question:**

The coherent artifact architecture follows these principles:

**Location by purpose:**
- **Ephemeral (agent session):** `.orch/workspace/{name}/` - SPAWN_CONTEXT.md, SYNTHESIS.md
- **Persistent (project knowledge):** `.kb/` - investigations, decisions, guides
- **Operational (work tracking):** `.beads/` - issues, comments, phase tracking
- **Orchestrator handoff:** `.orch/SESSION_HANDOFF.md` - cross-session context

**Relationships via reference:**
- SYNTHESIS.md points to investigation path
- Investigation references workspace
- Beads comments reference investigation_path
- Decisions reference investigations they promote

**Handoff protocol:**
- Workers: Create SYNTHESIS.md, call `bd comment "Phase: Complete"`, `/exit`
- Orchestrator: Update SESSION_HANDOFF.md before context exhaustion or session end

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Comprehensive artifact audit across 3 days of intensive system usage. Clear patterns emerged. However, proposed changes (SESSION_HANDOFF.md standard, explicit promotion paths) haven't been tested in practice.

**What's certain:**

- ✅ Six artifact types exist with distinct purposes
- ✅ Temporal categories explain placement decisions
- ✅ Beads comments are the operational interface
- ✅ `.kb/` standardization enables discovery

**What's uncertain:**

- ⚠️ Whether SESSION_HANDOFF.md scales to long-running orchestrator sessions
- ⚠️ Whether explicit promotion paths add value vs current implicit flow
- ⚠️ Whether workspace cleanup should preserve or archive SYNTHESIS.md

**What would increase confidence to Very High (95%+):**

- Test SESSION_HANDOFF.md pattern across 5+ orchestrator sessions
- Validate promotion path workflow (investigation → decision)
- Observe workspace cleanup behavior with archival

---

## Implementation Recommendations

**Purpose:** Bridge from findings to actionable architecture using directive guidance pattern.

### Recommended Approach ⭐

**Tiered Artifact Model with Explicit Handoffs** - Standardize artifact placement by temporal category and add orchestrator handoff protocol.

**Why this approach:**
- Maintains existing patterns (investigations in .kb/, workspaces ephemeral)
- Adds missing orchestrator handoff (SESSION_HANDOFF.md)
- Makes promotion paths explicit (investigation → decision)
- Minimal disruption to working system

**Trade-offs accepted:**
- Orchestrator sessions require more discipline (update SESSION_HANDOFF.md)
- Some redundancy between SYNTHESIS.md and SESSION_HANDOFF.md
- Not collapsing artifact types (maintains current distinctions)

**Implementation sequence:**
1. **Adopt SESSION_HANDOFF.md** - Create template in `.orch/templates/SESSION_HANDOFF.md`, update orchestrator skill to reference it
2. **Document promotion paths** - Add to `.kb/guides/artifact-organization.md` with explicit "when to promote" criteria
3. **Consider SYNTHESIS.md archival** - Option to copy to `.orch/archive/` before workspace cleanup (separate issue)

### Alternative Approaches Considered

**Option B: Collapse workspaces into .kb/**
- **Pros:** Single location for all persistent artifacts
- **Cons:** Loses ephemeral/persistent distinction; clutters .kb/ with session-specific data
- **When to use instead:** Never - the distinction is valuable

**Option C: Investigations in workspaces**
- **Pros:** Perfect cohesion - all agent work in one place
- **Cons:** Breaks `kb context` discovery; investigations disappear with workspace cleanup
- **When to use instead:** Never - discoverability is essential

**Option D: No orchestrator handoff formalization**
- **Pros:** No new discipline required
- **Cons:** Perpetuates session amnesia for orchestrator context
- **When to use instead:** If orchestrator sessions are always short and context reloads are cheap

**Rationale for recommendation:** Option A preserves what's working (existing artifact types, locations, discovery patterns) while filling the gap (orchestrator handoff) identified in findings.

---

### Implementation Details

**What to implement first:**
- Create `.orch/templates/SESSION_HANDOFF.md` based on skillc pattern
- Update orchestrator skill to mention SESSION_HANDOFF.md for cross-session context
- Document the tiered artifact model in CLAUDE.md or a guide

**Things to watch out for:**
- ⚠️ SESSION_HANDOFF.md can grow stale - needs explicit update discipline
- ⚠️ Multiple orchestrator sessions in same day might conflict on single handoff file
- ⚠️ Cross-project orchestration (multiple repos) might need per-project handoffs

**Areas needing further investigation:**
- How to handle session synthesis when orchestrator session ends mid-day vs end-of-day
- Whether to version or timestamp SESSION_HANDOFF.md updates
- Integration with `orch review` for orchestrator work

**Success criteria:**
- ✅ Next Claude instance can resume orchestrator work via SESSION_HANDOFF.md
- ✅ `kb context` finds all investigations regardless of which agent created them
- ✅ Workspace cleanup doesn't lose knowledge (SYNTHESIS.md captured findings before deletion)

---

## References

**Files Examined:**
- `.orch/workspace/*/SPAWN_CONTEXT.md` - Agent initialization context structure
- `.orch/workspace/*/SYNTHESIS.md` - Session summary format
- `.orch/templates/SYNTHESIS.md` - Canonical template
- `.kb/investigations/*.md` - 140+ investigation examples
- `.kb/decisions/2025-12-21-single-agent-review-command.md` - Decision format
- `session-synthesis-21dec.md` - Ad-hoc orchestrator synthesis
- `skillc/.orch/SESSION_HANDOFF.md` - Mature handoff pattern
- `~/.kb/principles.md` - Foundational principles
- `~/.kb/guides/orch-ecosystem.md` - Ecosystem architecture
- `~/.claude/skills/` - Skills structure
- `.beads/issues.jsonl` - Operational tracking format

**Commands Run:**
```bash
# Count workspaces and artifacts
ls .orch/workspace/*/SPAWN_CONTEXT.md | wc -l  # 100+
ls .orch/workspace/*/SYNTHESIS.md | wc -l      # 52

# Count investigations
ls .kb/investigations/*.md | wc -l             # 140+

# Examine beads structure
head -50 .beads/issues.jsonl
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-single-agent-review-command.md` - Shows decision format
- **Investigation:** Multiple in `.kb/investigations/` - Shows investigation format
- **Workspace:** `.orch/workspace/og-feat-implement-port-allocation-21dec/` - Shows workspace lifecycle

---

## Investigation History

**2025-12-21 10:00:** Investigation started
- Initial question: What is the coherent architecture for artifacts across the orchestration ecosystem?
- Context: Need to understand artifact relationships and propose improvements

**2025-12-21 10:15:** Completed artifact type audit
- Identified 6 distinct artifact types
- Discovered temporal categorization (ephemeral/persistent/operational)

**2025-12-21 10:30:** Analyzed handoff patterns
- Found SESSION_HANDOFF.md in skillc
- Identified gap in orchestrator session artifacts

**2025-12-21 11:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Tiered artifact model with explicit orchestrator handoff recommended
