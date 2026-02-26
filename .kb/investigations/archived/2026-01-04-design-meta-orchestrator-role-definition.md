<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Meta-orchestrator is Dylan's role - distinct from orchestrator agents. Three-tier hierarchy (meta → orchestrator → worker) is already implicit; making it explicit clarifies responsibilities without requiring new infrastructure.

**Evidence:** Prior investigations (Dec 24, Jan 4) established: (1) orchestrators ARE structurally spawnable, (2) meta-orchestration is 80% ready via existing tools (--workdir, kb --global), (3) SESSION_CONTEXT.md ↔ SESSION_HANDOFF.md provides orchestrator lifecycle parallel to SPAWN_CONTEXT.md ↔ SYNTHESIS.md.

**Knowledge:** Meta-orchestrator responsibilities are strategic and human-level: focus decisions, cross-project prioritization, handoff review, system evolution. Orchestrator responsibilities are tactical execution within delegated focus. Key distinction: meta-orchestrator decides WHICH focus; orchestrator decides HOW to execute.

**Next:** Add "Meta-Orchestrator Role" section to orchestrator skill (not a separate skill). Document the three-tier hierarchy, responsibilities, and interaction patterns.

---

# Investigation: Meta-Orchestrator Role Definition

**Question:** What does the meta-orchestrator do (vs orchestrator, vs worker), and how should this role be documented?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Design-session agent
**Phase:** Complete
**Next Step:** None - ready for orchestrator review
**Status:** Complete

<!-- Lineage -->
**Extracted-From:** Prior investigations on orchestrator spawning and meta-orchestration maturity
**Related:** `orch-go-kmoy` (Meta-orchestrator architecture investigation - completed same day)

---

## Findings

### Finding 1: Three-Tier Hierarchy Already Exists Implicitly

**Evidence:** The orchestrator skill (1400+ lines) and prior investigations reveal an implicit hierarchy:

| Tier | Role | Example | Scope | Key Artifact |
|------|------|---------|-------|--------------|
| **Meta-orchestrator** | Dylan | Human making strategic decisions | Cross-project, multi-session | Epic progress, handoffs |
| **Orchestrator** | Claude agent with orchestrator skill | Managing spawns within focus | Single project, single focus | SESSION_HANDOFF.md |
| **Worker** | Spawned agent with task skill | Implementing single issue | Single issue | SYNTHESIS.md, code |

This isn't a design proposal - it's documenting observed behavior. Dylan already:
- Decides which epic to focus on (strategic)
- Reviews orchestrator handoffs (verification)
- Makes cross-project prioritization decisions (coordination)
- Evolves the system itself (meta-level)

**Source:** Orchestrator skill sections: "Focus-Based Session Model", "Orchestrator Core Responsibilities", "Strategic Alignment"

**Significance:** Making this hierarchy explicit clarifies when Dylan is in meta-orchestrator mode vs when the orchestrator agent should act autonomously.

---

### Finding 2: Meta-Orchestrator Responsibilities Are Distinct from Orchestrator

**Evidence:** Analysis of current patterns reveals clear separation:

**Meta-orchestrator (Dylan) does:**
1. **Strategic focus decisions** - Which epic? Which project? What's the goal this week?
2. **Cross-session continuity** - Reviewing handoffs, resuming work, context synthesis
3. **System evolution** - Deciding tooling changes, process improvements, skill updates
4. **Cross-project prioritization** - When multiple projects compete, which wins?
5. **Human-level decisions** - Architectural choices that require stakeholder judgment
6. **Pattern recognition across orchestrators** - What's working? What's not?

**Orchestrator agent does (from existing skill):**
1. **Tactical execution within focus** - Spawn workers, complete workers, synthesize
2. **Triage and labeling** - Review issues, set priorities within scope
3. **Cross-agent synthesis** - Combine findings from workers
4. **Backlog management** - `bd ready`, `bd blocked`, issue prioritization
5. **Session-level decisions** - Which skill? What context? How to verify?

**Key test:** "Is this about WHICH focus or HOW to execute?"
- WHICH = meta-orchestrator (Dylan)
- HOW = orchestrator (Claude)

**Source:** 
- Orchestrator skill "⛔ ABSOLUTE DELEGATION RULE" (orchestrators delegate, never implement)
- Orchestrator skill "Orchestrator Core Responsibilities (Never Delegate)"
- Prior investigation: `2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md`

**Significance:** The distinction prevents role confusion. Orchestrators shouldn't make strategic focus decisions; Dylan shouldn't be doing tactical triage.

---

### Finding 3: Interaction Patterns Already Have Implicit Structure

**Evidence:** Current patterns show how meta-orchestrator and orchestrator interact:

**Session Lifecycle:**
```
Meta-orchestrator: orch session start "Goal"
    ↓
Orchestrator: Execute within focus (spawn, complete, synthesize)
    ↓
Meta-orchestrator: Review SESSION_HANDOFF.md
    ↓
Meta-orchestrator: orch session end (or continue with new goal)
```

**Decision Escalation:**
```
Worker encounters ambiguity → Escalates via SYNTHESIS.md "Unexplored Questions"
    ↓
Orchestrator reviews → Either decides (tactical) or escalates (strategic)
    ↓
If strategic → Meta-orchestrator decides (Dylan in conversation)
```

**Focus Shifts:**
```
Meta-orchestrator: "Let's shift to X" (strategic redirect)
    ↓
Orchestrator: orch focus "X" (update tracking)
    ↓
Orchestrator: Continues with new focus
```

**Source:** 
- Orchestrator skill "Session Reflection", "Orchestrator Autonomy"
- Prior investigation maturity assessment Dec 24

**Significance:** The patterns exist but aren't documented as "meta-orchestrator ↔ orchestrator interaction." Making them explicit reduces confusion.

---

### Finding 4: Guardrails Differ by Tier

**Evidence:** Each tier has different failure modes and guardrails:

**Meta-orchestrator guardrails:**
- ⚠️ Don't micromanage - Let orchestrator make tactical decisions
- ⚠️ Don't compensate - Per "Pressure Over Compensation" principle, let system gaps surface
- ⚠️ Don't skip handoff review - SESSION_HANDOFF.md is the handoff, not conversation
- ⚠️ Beware becoming bottleneck - If every spawn needs approval, system is too dependent

**Orchestrator guardrails (existing):**
- ⛔ ABSOLUTE DELEGATION RULE - Never do spawnable work
- ⚠️ Don't make strategic decisions - Escalate WHICH focus questions
- ⚠️ Don't skip verification - `orch complete` gates on evidence

**Worker guardrails (existing):**
- ⛔ Never push/deploy - Orchestrator-exclusive
- ⚠️ Surface before circumvent - Don't work around constraints silently
- ⚠️ Phase reporting - `bd comment` for visibility

**Source:** 
- Orchestrator skill guardrails throughout
- "Pressure Over Compensation" principle (`~/.kb/principles.md`)

**Significance:** Meta-orchestrator guardrails are NOT documented. The orchestrator skill has extensive guardrails, but Dylan's constraints as meta-orchestrator are implicit.

---

### Finding 5: Meta-Orchestrator is NOT an Orchestrator Session

**Evidence:** Prior investigation (orch-go-kmoy) found:

> "Meta-orchestrator IS Dylan (initially) - No automation needed. Dylan makes strategic decisions, spawns orchestrator sessions, reviews handoffs."

Key distinction:
- **Orchestrator session** = Claude agent with orchestrator skill, produces SESSION_HANDOFF.md
- **Meta-orchestrator** = Dylan, interacts with orchestrator, reviews handoffs, sets focus

The temptation is to make meta-orchestrator a "spawn level above orchestrator." This is wrong because:
1. Dylan provides strategic judgment that AI currently can't
2. Meta-orchestrator is interactive, not autonomous
3. Spawning meta-orchestrator would require... what spawns meta-meta-orchestrator?

**Source:** 
- `2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md`
- `2025-12-24-inv-meta-orchestration-maturity-assessment.md`

**Significance:** Meta-orchestrator documentation should be guidance for Dylan's role, not a skill file for an agent.

---

## Synthesis

### Key Insights

1. **Three-tier hierarchy is descriptive, not prescriptive** - Dylan already operates as meta-orchestrator. The role exists; documenting it prevents confusion about when Dylan is meta-orchestrating vs participating in orchestrator conversation.

2. **WHICH vs HOW is the core distinction** - Meta-orchestrator decides WHICH focus/project/direction. Orchestrator decides HOW to execute within that focus. This maps cleanly to strategic vs tactical.

3. **Meta-orchestrator guardrails are the gap** - The orchestrator skill has extensive guardrails. Dylan's meta-orchestrator guardrails are undocumented. Without them, Dylan risks:
   - Micromanaging (bypassing orchestrator judgment)
   - Compensating (providing context that should be in system)
   - Bottlenecking (requiring approval for every spawn)

4. **Meta-orchestrator is NOT spawnable** - Unlike orchestrators (which ARE structurally spawnable), meta-orchestrator is Dylan's human role. Trying to automate it creates recursion (what spawns meta-meta-orchestrator?).

5. **Documentation should live in orchestrator skill** - Not a separate skill, but a section that clarifies the relationship. Orchestrators need to know when to escalate to meta-orchestrator, and meta-orchestrator needs to know when to let orchestrator act autonomously.

### Answer to Investigation Question

**What does the meta-orchestrator do (vs orchestrator, vs worker)?**

| Responsibility | Meta-orchestrator (Dylan) | Orchestrator (Claude) | Worker (Spawned) |
|----------------|---------------------------|----------------------|------------------|
| **Strategic focus** | Decides which epic/project | Operates within focus | Within single issue |
| **Cross-session** | Reviews handoffs, resumes work | Produces SESSION_HANDOFF.md | Produces SYNTHESIS.md |
| **System evolution** | Decides tooling/process changes | Applies existing patterns | Follows skill guidance |
| **Work creation** | Creates epics, sets priorities | Spawns workers, triages issues | Implements assigned task |
| **Verification** | Reviews orchestrator handoffs | Completes workers, verifies | Reports phase, produces evidence |

**How should this role be documented?**

Add a **"Meta-Orchestrator Role"** section to the orchestrator skill, NOT a separate skill file. This section should include:

1. **The Three-Tier Hierarchy** - Clarify meta → orchestrator → worker
2. **Meta-Orchestrator Responsibilities** - Strategic focus, handoff review, system evolution
3. **Interaction Patterns** - How meta-orchestrator and orchestrator interact
4. **Meta-Orchestrator Guardrails** - Don't micromanage, don't compensate, don't bottleneck
5. **Escalation Triggers** - When orchestrator should escalate to meta-orchestrator

---

## Structured Uncertainty

**What's tested:**

- ✅ Three-tier hierarchy is observed (Dylan does operate as meta-orchestrator)
- ✅ Prior investigations confirm orchestrators are spawnable (gap is verification)
- ✅ Current patterns show meta ↔ orchestrator interaction
- ✅ Meta-orchestrator IS Dylan initially (prior investigation Dec 24)

**What's untested:**

- ⚠️ Whether explicit meta-orchestrator documentation changes behavior
- ⚠️ Whether guardrails prevent current failure modes (micromanaging, compensating)
- ⚠️ Whether orchestrators correctly escalate with clearer guidance

**What would change this:**

- If autonomous meta-orchestrator becomes viable → need skill file instead
- If the three tiers prove insufficient → may need more nuanced hierarchy
- If guardrails are ignored → need gates, not documentation

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add "Meta-Orchestrator Role" section to orchestrator skill**

This is a documentation enhancement, not a new system.

**Why this approach:**
- Meta-orchestrator is Dylan's role, not an agent skill
- Orchestrators need to know when to escalate
- Avoiding skill proliferation keeps system simple
- Aligns with "Share Patterns Not Tools" principle

**Trade-offs accepted:**
- No spawnable meta-orchestrator (appropriate for now)
- Documentation rather than automation
- Relies on Dylan following guardrails (no gates yet)

**Implementation sequence:**
1. Add section to orchestrator skill: "Meta-Orchestrator Role (Dylan)"
2. Document three-tier hierarchy with table
3. Add meta-orchestrator guardrails
4. Add escalation triggers
5. Update orchestrator autonomy section to reference meta-orchestrator

### Alternative Approaches Considered

**Option B: Create separate meta-orchestrator skill file**
- **Pros:** Clean separation, could eventually be loaded by spawned agent
- **Cons:** Meta-orchestrator is Dylan, not an agent; skill file implies automation
- **When to use instead:** If autonomous meta-orchestrator becomes viable

**Option C: Create a guide in .kb/guides/**
- **Pros:** Guides are for reusable frameworks
- **Cons:** This is operational guidance, not a framework; won't be loaded by agents
- **When to use instead:** If pattern needs to be shared outside orchestration system

**Rationale for recommendation:** Meta-orchestrator is Dylan's operating mode, not an agent role. Adding it to the orchestrator skill provides context for orchestrators about who they're serving and when to escalate.

---

### Implementation Details

**What to implement first:**
- Three-tier hierarchy table (clear, actionable reference)
- Meta-orchestrator guardrails (prevents common failure modes)

**Things to watch out for:**
- ⚠️ Don't make guardrails feel like rules - they're anti-patterns to avoid
- ⚠️ Keep section concise - orchestrator skill is already 1400+ lines
- ⚠️ Ensure escalation triggers are clear, not vague

**Areas needing further investigation:**
- Whether SESSION_HANDOFF.md is being consistently produced
- Whether `orch session end` gates are sufficient for reflection
- Whether pattern analysis (`kb reflect --type orchestrator`) is needed

**Success criteria:**
- ✅ Orchestrators can identify when to escalate vs act autonomously
- ✅ Dylan has documented guardrails against micromanaging/compensating
- ✅ Three-tier hierarchy is clear to new readers of the skill

---

## References

**Files Examined:**
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Current orchestrator skill (1396 lines)
- `~/.kb/principles.md` - Foundational principles including "Pressure Over Compensation"
- `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md` - Same-day architecture investigation
- `.kb/investigations/2025-12-24-inv-meta-orchestration-maturity-assessment.md` - Maturity assessment

**Commands Run:**
```bash
# Check beads for related issues
bd list | head -20
bd show orch-go-kmoy

# Check for existing meta-orchestrator artifacts
ls ~/.kb/investigations/*meta* 2>/dev/null || echo "none"

# Review kb context
kb context "meta-orchestrator orchestrator role responsibilities"
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md` - Architecture recommendation (incremental enhancement)
- **Investigation:** `.kb/investigations/2025-12-24-inv-meta-orchestration-maturity-assessment.md` - Found meta-orchestration 80% ready
- **Principles:** `~/.kb/principles.md` - "Pressure Over Compensation" directly relevant to meta-orchestrator guardrails

---

## Investigation History

**2026-01-04 09:30:** Investigation started
- Initial question: What does the meta-orchestrator do?
- Context: Orchestrator skill well-defined, but Dylan's role as meta-orchestrator is implicit

**2026-01-04 09:45:** Context gathered
- Found prior investigations already addressed architecture
- Three-tier hierarchy identified as implicit pattern
- Recognized meta-orchestrator ≠ spawnable orchestrator

**2026-01-04 10:15:** Synthesis complete
- Key insight: WHICH vs HOW is the core distinction
- Recommendation: Add section to orchestrator skill, not separate skill
- Meta-orchestrator guardrails identified as key gap

**2026-01-04 10:30:** Investigation completed
- Status: Complete
- Key outcome: Meta-orchestrator is Dylan's role; document in orchestrator skill with three-tier hierarchy, responsibilities, and guardrails
