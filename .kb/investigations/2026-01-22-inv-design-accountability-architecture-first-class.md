## Summary (D.E.K.N.)

**Delta:** Decidability currently implicit in 5 locations (skills prose, spawn context, beads schema, registry, principles); propose hybrid architecture with structured authority in skills + governance claims in `.kb/governance/`.

**Evidence:** Examined beads issue schema (issues.jsonl), spawn context template (context.go:54-348), worker-base skill authority section, decision-authority guide, and principles.md governance principles.

**Knowledge:** Authority is currently encoded as prose instructions, not queryable data; queries needed include "who decides this?", "what authority does this agent have?", "where is governance debt?".

**Next:** Create `orch-go-3k0j5` for beads fork extensions; then implement structured authority in skills frontmatter as Phase 1.

**Promote to Decision:** recommend-yes - Establishes the architectural pattern for governance infrastructure across the system.

---

# Investigation: Design Accountability Architecture as First-Class Queryable Artifact

**Question:** What would decidability look like as an explicit, queryable structure? How should authority boundaries be visible the way agent status is visible in dashboard?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker agent (architect skill)
**Phase:** Complete
**Next Step:** None (design investigation complete, ready for decision)
**Status:** Complete

**Extracted-From:** Epic orch-go-kz7zr (Governance Infrastructure for Human-AI Systems)

---

## Findings

### Finding 1: Decidability Currently Lives Implicitly in 5 Locations

**Evidence:** Examined codebase for where authority/decidability is encoded:

| Location | What's Encoded | Format | Queryable? |
|----------|---------------|--------|------------|
| **Skills (SKILL.md)** | Authority delegation rules | Prose (markdown) | No |
| **Spawn context (SPAWN_CONTEXT.md)** | Same authority rules, embedded | Prose (template) | No |
| **Beads issues** | Priority, type, dependencies, labels | Structured (JSON) | Yes |
| **Agent registry** | Agent state (active/completed/abandoned) | Structured (JSON) | Yes |
| **Principles.md** | Governance philosophy | Prose (markdown) | No |

**Source:**
- `pkg/spawn/context.go:176-194` (AUTHORITY section in template)
- `~/.claude/skills/skills/src/shared/worker-base/SKILL.md` (Authority Delegation section)
- `.beads/issues.jsonl` (issue schema)
- `pkg/registry/registry.go:44-69` (Agent struct)
- `~/.kb/principles.md` (Authority is Scoping, Perspective is Structural, Escalation is Information Flow)

**Significance:** Authority information exists but is scattered across locations with different formats. Only beads and registry are queryable; skill authority and governance philosophy are prose that must be parsed by humans/LLMs.

---

### Finding 2: The Implicit Schema Shows What's Missing

**Evidence:** Current schema captures:

**What's Captured (structured):**
- Work items (beads issues with status, priority, dependencies)
- Agent state (registry with active/completed/abandoned)
- Gates (labels like `triage:ready`)

**What's Not Captured (missing or prose-only):**
- **Consequence bearer** - Always implicitly "Dylan", never explicit
- **Authority scope** - Which decisions this agent can make
- **Judgment type** - Agent vs orchestrator vs human decision
- **Escalation target** - Who to escalate to (always "orchestrator" implicitly)
- **Governance debt** - Areas without clear authority assignment

**Source:**
- Beads issue schema from `.beads/issues.jsonl` (examined 5 sample issues)
- Decision-authority guide `.kb/guides/decision-authority.md:1-70`
- Governance handoff `~/Downloads/governance-infrastructure-handoff.md`

**Significance:** The gap between "what's tracked" and "what governs" reveals what needs to become infrastructure. Authority information exists in prose but isn't queryable.

---

### Finding 3: Five Stakeholder Query Patterns Identified

**Evidence:** Analyzed who asks what questions about authority:

**1. Orchestrator (spawn-time queries):**
- "Is this work ready to spawn?" → `bd ready` (partial - checks gates, not authority)
- "What authority should I delegate?" → hardcoded in skill selection
- "Who bears consequences for this?" → not queryable (implicitly always Dylan)

**2. Worker agents (execution-time queries):**
- "Am I authorized to do this?" → must read prose in SPAWN_CONTEXT.md
- "Should I escalate this?" → must read prose in skill
- "Who do I escalate to?" → always "orchestrator" by convention

**3. Human oversight (governance queries):**
- "What has been decided without me?" → not queryable
- "What decisions are pending my input?" → partial (beads comments with QUESTION/BLOCKED)
- "Where is authority unclear?" → not queryable (governance debt)

**4. Dashboard/monitoring (visibility queries):**
- "What agents are active?" → `orch status` (works)
- "What authority do they have?" → not visible
- "What decisions are awaiting human?" → not visible

**5. Composability (future - cross-boundary queries):**
- "What authority claims does this system make?" → not queryable
- "What are the accountability contracts?" → not queryable

**Source:**
- Governance handoff doc sections on composability (lines 66-76)
- Epic orch-go-kz7zr (related issues orch-go-ag261, orch-go-8ke18, orch-go-xuejv)

**Significance:** These queries define the interface the accountability architecture must support. The current system answers ~20% of them (work state queries) but 0% of governance queries.

---

## Synthesis

**Key Insights:**

1. **Authority is prose, not data** - The decision-authority guide and skill authority sections contain rich information about what agents can decide, but it's all prose. No CLI can answer "who decides X?" without parsing natural language.

2. **Consequence bearer is implicitly "Dylan"** - The handoff doc names this as "Dylan-bears-consequences" as the current authority termination point. This is nowhere explicit in the schema. When authority extends beyond Dylan (shipping to others), the architecture has no way to represent this.

3. **The gap is structured authority, not authority documentation** - Authority documentation exists (`.kb/guides/decision-authority.md` is excellent). What's missing is making that documentation machine-queryable.

**Answer to Investigation Question:**

A decidability graph as a first-class queryable artifact would require:

1. **Structured authority in skills** - Convert prose authority sections to YAML frontmatter
2. **Authority scope in spawn context** - Inject structured authority at spawn time
3. **Governance claims file** - New `.kb/governance/authority.yaml` for system-wide claims
4. **Query interface** - `kb authority` and `orch governance` commands
5. **Dashboard visibility** - New panel for authority state

The design must support the 5 stakeholder query patterns identified above.

---

## Structured Uncertainty

**What's tested:**

- ✅ Beads schema fields verified (examined `issues.jsonl` with `jq`)
- ✅ Spawn context template verified (read `pkg/spawn/context.go`)
- ✅ Skill authority section format verified (read worker-base SKILL.md)
- ✅ Decision-authority guide exists and has structured tables (read `.kb/guides/decision-authority.md`)

**What's untested:**

- ⚠️ YAML frontmatter parsing in skills (skillc may need extension)
- ⚠️ Beads fork extensibility for `authority_scope` field (depends on beads architecture)
- ⚠️ kb CLI extension complexity (kb would need new `authority` subcommand)

**What would change this:**

- If beads can't be extended (fork too divergent), authority would need to live elsewhere
- If skillc can't parse frontmatter authority, would need separate authority files per skill
- If kb team rejects authority subcommand, would need standalone `orch authority` tool

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Hybrid Architecture ⭐

**Three-layer authority representation:**

1. **Skills (structured authority section)** - What agents can decide
2. **Spawn context (authority injection)** - What this specific agent can decide
3. **.kb/governance/ (system claims)** - Who bears consequences, escalation chains

**Why this approach:**
- Each part lives where it conceptually belongs (skill authority in skills, system governance in kb)
- Incremental adoption - can start with skills, add governance later
- Minimal tooling disruption - extends existing patterns rather than creating new ones

**Trade-offs accepted:**
- Two locations for authority (skills + governance) requires coordination
- Skills must be rebuilt when authority changes (existing pattern with skillc)

**Implementation sequence:**
1. **Phase 1: Structured authority in skills** - Add YAML `authority:` section to skill frontmatter
2. **Phase 2: Authority injection in spawn** - `pkg/spawn` reads and injects structured authority
3. **Phase 3: kb governance commands** - `kb authority` queries for system-wide governance
4. **Phase 4: Dashboard visibility** - Authority panel in web UI

### Alternative Approaches Considered

**Option B: All authority in beads (extend issue schema)**
- **Pros:** Single query interface, integrates with existing bd commands
- **Cons:** Couples governance to work items; beads is for issue tracking, not system governance
- **When to use instead:** If kb extension proves infeasible

**Option C: Standalone .authority/ directory**
- **Pros:** Clean separation, purpose-built for governance
- **Cons:** Yet another artifact location, new tooling needed
- **When to use instead:** If governance requirements become complex enough to warrant dedicated tooling

**Rationale for recommendation:** Hybrid approach follows existing patterns (authority in skills, knowledge in kb) while making them queryable. Lower adoption friction than new artifact type.

---

### Implementation Details

**File Format Proposals:**

**1. Skill frontmatter extension:**
```yaml
---
name: worker-base
skill-type: foundation
authority:
  decides:
    - implementation-details
    - testing-strategy
    - documentation-structure
  escalates:
    - architectural-decisions
    - scope-boundaries
    - ambiguous-tradeoffs
  consequence_bearer: orchestrator
---
```

**2. Governance claims file (`.kb/governance/authority.yaml`):**
```yaml
# Authority claims for this system
consequence_bearer: dylan
authority_chain:
  - level: agent
    scope: implementation
    escalates_to: orchestrator
  - level: orchestrator
    scope: architectural
    escalates_to: human
  - level: human
    scope: strategic
    escalates_to: null  # termination point

decision_types:
  implementation:
    judgment: agent
    reversible: true
  architectural:
    judgment: orchestrator
    reversible: partially
  strategic:
    judgment: human
    reversible: false
```

**3. CLI interface:**
```bash
# Query authority (kb)
kb authority show                     # Show authority structure
kb authority who-decides "action"     # Who decides this type?
kb authority debt                     # Show governance debt

# Query authority (orch)
orch authority <agent-id>             # Agent's authority scope
orch governance status                # System governance health
orch governance debt                  # Early warning
```

**What to implement first:**
- Skill frontmatter `authority:` section (Phase 1)
- skillc extension to parse and validate authority
- Spawn context template to inject structured authority

**Things to watch out for:**
- ⚠️ skillc may need significant changes to parse frontmatter
- ⚠️ Existing skills will need migration to add authority section
- ⚠️ Dashboard authority panel requires new API endpoint

**Areas needing further investigation:**
- How does authority compose across trust boundaries? (see orch-go-v3jax)
- What observations would have helped during AI psychosis? (see orch-go-bslcv)
- Where does Dylan-bears-consequences authority end? (see orch-go-8ke18)

**Success criteria:**
- ✅ Can answer "who decides X?" with a CLI command
- ✅ Dashboard shows agent authority scope
- ✅ Governance debt is detectable and surfaced
- ✅ Authority claims are machine-readable

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - Spawn context template with AUTHORITY section
- `pkg/registry/registry.go` - Agent struct and state management
- `.beads/issues.jsonl` - Beads issue schema
- `~/.claude/skills/skills/src/shared/worker-base/SKILL.md` - Authority delegation prose
- `.kb/guides/decision-authority.md` - Decision authority guide
- `~/.kb/principles.md` - Governance principles

**Commands Run:**
```bash
# Examined beads issue schema
head -5 .beads/issues.jsonl | jq '.'

# Checked authority in skills
cat ~/.claude/skills/skills/src/shared/worker-base/SKILL.md | head -100

# Reviewed governance principles
cat ~/.kb/principles.md | head -200
```

**External Documentation:**
- `~/Downloads/governance-infrastructure-handoff.md` - Governance infrastructure handoff doc

**Related Artifacts:**
- **Epic:** orch-go-kz7zr - Governance Infrastructure for Human-AI Systems
- **Related issues:** orch-go-3k0j5 (beads fork extensions), orch-go-xuejv (principles audit), orch-go-ag261 (early warning), orch-go-8ke18 (authority edge)
- **Model:** `.kb/models/beads-integration-architecture.md` - How beads integrates with orch
- **Guide:** `.kb/guides/decision-authority.md` - Current authority documentation

---

## Investigation History

**2026-01-22 21:58:** Investigation started
- Initial question: What would decidability look like as first-class queryable artifact?
- Context: Part of Governance Infrastructure epic (orch-go-kz7zr)

**2026-01-22 22:30:** Exploration phase complete
- Found decidability implicit in 5 locations
- Identified 5 stakeholder query patterns
- Determined the gap is structured authority, not documentation

**2026-01-22 23:00:** Synthesis phase complete
- Proposed hybrid architecture (skills + governance)
- Defined file formats and CLI interface
- Identified 4 implementation phases

**2026-01-22 23:15:** Investigation completed
- Status: Complete
- Key outcome: Hybrid architecture recommended with structured authority in skills + governance claims in `.kb/governance/`
