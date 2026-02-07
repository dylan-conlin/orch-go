## Summary (D.E.K.N.)

**Delta:** The orchestrator skill's 375-line core + 16 reference files architecture was optimized for 200K context windows. At 1M context, the progressive disclosure split is counterproductive - the cognitive overhead of deciding whether to load reference files costs more than the tokens saved, and critical guidance is fragmented across files that may never get loaded.

**Evidence:** Read all 376 lines of core SKILL.md, all 16 reference files (~1,200 lines total), 3 prior investigations (drift audit, value-add, 18% completion rate), principles.md, and orchestrator session lifecycle model. Classified every section as load-bearing, cargo cult, outdated, or missing by cross-referencing actual orchestrator behavior from the spawning session's evidence.

**Knowledge:** Five patterns are genuinely load-bearing (absolute delegation, skill selection, spawn mechanics, completion verification, COMPREHEND-TRIAGE-SYNTHESIZE operating model). The reference file split, token-conscious compression, and several "educational" sections are the wrong optimization for 1M context. The Strategic Orchestrator Model (COMPREHEND → TRIAGE → SYNTHESIZE) - the most important framing - is absent from the skill entirely.

**Next:** Implement proposed unified skill structure: single file, ~500-600 lines, organized around the strategic operating model, all content inline. Remove reference/ directory. This is an architectural decision requiring orchestrator/Dylan review.

**Authority:** architectural - Cross-component impact (skill structure affects all orchestrator sessions, daemon interaction, worker spawning). Multiple valid approaches exist (unified vs modular). Requires synthesis of prior investigations.

---

# Investigation: Redesign Orchestrator Skill for 1M Context Era

**Question:** Given 1M context windows, what should the orchestrator skill contain? What's load-bearing, what's cargo cult, what's outdated, and what's missing?

**Started:** 2026-02-05
**Updated:** 2026-02-05
**Owner:** og-arch-redesign-orchestrator-skill-05feb-f3d7
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A (this would create a new decision if accepted)
**Extracted-From:** N/A

## Prior Work

| Investigation                              | Relationship | Verified                                                                | Conflicts                                                   |
| ------------------------------------------ | ------------ | ----------------------------------------------------------------------- | ----------------------------------------------------------- |
| 2026-01-15 Orchestrator Skill Drift Audit  | extends      | Yes - read skill and confirmed 19 drift items still relevant            | None                                                        |
| 2026-01-17 Identify Orchestrator Value Add | extends      | Yes - read investigation, confirmed COMPREHEND→TRIAGE→SYNTHESIZE model  | Skill still missing this model                              |
| 2026-01-06 Diagnose 18% Completion Rate    | confirms     | Yes - orchestrators are coordination roles by design                    | None                                                        |
| 2026-01-13 Session Management Architecture | extends      | Partially - session handoff removed Jan 2026, skill references outdated | Skill still references SESSION_HANDOFF.md for orchestrators |

---

## Findings

### Finding 1: Five Patterns Are Genuinely Load-Bearing

**Evidence:** Cross-referencing the skill contents against failure modes documented in prior investigations, five patterns consistently prevent observable failures:

| Pattern                                            | Lines in Skill | Failure Without It                                   | Evidence Source                           |
| -------------------------------------------------- | -------------- | ---------------------------------------------------- | ----------------------------------------- |
| **Absolute Delegation Rule**                       | 56-106         | Orchestrator does implementation, blocking system    | Frame collapse in lifecycle model:136-158 |
| **Skill Selection Decision Trees**                 | 160-209        | Wrong skill spawned for task type                    | Skill inference errors in daemon model    |
| **Spawn Mechanics** (`kb context`, checklist)      | 213-254        | Agents hit known walls, lack critical context        | Spawning-best-practices reference         |
| **Completion Verification**                        | 258-270        | Work completes but isn't synthesized                 | Completion-verification reference         |
| **Core Responsibilities** (what to do vs delegate) | 300-315        | Orchestrator delegates everything or does everything | Core-responsibilities reference           |

**Source:**

- `~/.opencode/skill/orchestrator/SKILL.md:56-315`
- `.kb/models/orchestrator-session-lifecycle.md:136-158` (frame collapse)
- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md:60-98` (value-add analysis)

**Significance:** These five patterns are non-negotiable. Any redesign must preserve them. They prevent the highest-impact failure modes: frame collapse, misrouting, context-starved agents, unsynthesized work, and role confusion.

---

### Finding 2: The Progressive Disclosure Architecture Is Now Counterproductive

**Evidence:** The reference/ directory README.md explicitly states the optimization goal:

> "Original skill: 1,110 lines, ~5,889 words, ~7,600-8,800 tokens (estimated). After split: Core SKILL.md: 375 lines, ~1,681 words, ~2,185 tokens (73% reduction). Result: Core skill is well under 10K token goal."

At 1M context, 10K tokens is 1% of budget. The entire original 8,800 tokens would be 0.88%. The split creates three costs that exceed the token savings:

1. **Decision overhead:** Orchestrator must decide whether to load a reference file. This is a cognitive tax on every interaction.
2. **Fragmented critical information:** The skill-selection-details.md reference (509 lines) contains the two-tier issue quality model, beads tracking requirements, knowledge capture patterns, and auto-capture guidance. These are frequently needed but not always loaded.
3. **Drift accumulation:** The drift audit found 19 drift items. Fragmentation across 17 files makes drift harder to detect and fix.

The spawning session evidence shows 17% context usage after ~25 agent spawns and completions. The orchestrator never loaded a single reference file - it either had the knowledge from the core skill or reinvented it.

**Source:**

- `~/.opencode/skill/orchestrator/reference/README.md:57-66` (metrics)
- `~/.opencode/skill/orchestrator/reference/skill-selection-details.md` (509 lines of fragmented critical content)
- `.kb/investigations/2026-01-15-inv-orchestrator-skill-drift-audit.md` (19 drift items)
- Spawn context evidence: "17% context usage" after full productive session

**Significance:** The entire reference/ directory architecture should be eliminated. All content should be inline in a single SKILL.md. The token cost (~6K tokens total) is negligible at 1M context, and the cognitive/maintenance costs of progressive disclosure exceed the savings.

---

### Finding 3: Several Sections Are Cargo Cult (Agents Do Them Naturally)

**Evidence:** Multiple sections tell agents to do things that modern Opus-class models already do:

| Section                            | Lines                                 | What It Teaches                          | Why It's Cargo Cult                                                                                                                    |
| ---------------------------------- | ------------------------------------- | ---------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Context Detection                  | 18-32                                 | "Am I Orchestrator or Worker?"           | Infrastructure handles this via ORCHESTRATOR_CONTEXT.md vs SPAWN_CONTEXT.md. Agents never need to self-diagnose role.                  |
| Skill System Architecture          | 36-53                                 | Hybrid model (interactive vs spawned)    | Agents don't need to understand architecture to use it correctly. Infrastructure routes properly regardless.                           |
| Anti-Pattern interaction examples  | 144-148                               | Don't ask "Want me to complete them?"    | Modern Opus agents with clear role framing handle interaction style naturally. The "Mind-Reading Test" is redundant with good framing. |
| Amnesia-Resilient Design reference | 106 lines                             | Create artifacts for next Claude         | Templates and infrastructure enforce this. Agents already externalize state.                                                           |
| Knowledge Placement Guide          | ~100 lines in skill-selection-details | Where to put decisions vs investigations | `kb create` and `kn` commands enforce placement. This is reference material for humans, not operational guidance for agents.           |
| Auto-Capture User Corrections      | ~40 lines in skill-selection-details  | Capture corrections with `kn`            | Modern agents already do this naturally when they observe user corrections.                                                            |

**Source:**

- `~/.opencode/skill/orchestrator/SKILL.md:18-53` (context detection, skill system architecture)
- `~/.opencode/skill/orchestrator/SKILL.md:144-148` (anti-patterns)
- `~/.opencode/skill/orchestrator/reference/amnesia-resilient-design.md` (106 lines)
- `~/.opencode/skill/orchestrator/reference/skill-selection-details.md:376-498` (knowledge placement, auto-capture)

**Significance:** These sections consume ~300+ lines total without preventing failures. They explain "how the system works" rather than "what you should do." Removing them frees space for actually missing guidance.

---

### Finding 4: The Strategic Orchestrator Model Is Absent from the Skill

**Evidence:** The drift audit (H1, Jan 15) identified this: the Strategic Orchestrator Model decision (Jan 7) established COMPREHEND → TRIAGE → SYNTHESIZE as the orchestrator's operating model. The skill still doesn't reflect this. Instead, the skill organizes around mechanics (delegation, spawning, completion) rather than the strategic operating loop.

The value-add investigation (Jan 17) further established:

- Orchestrator judgment matters for: synthesis, goal refinement, frame correction, hotspot detection, triage decisions (the 20%)
- Routing execution is automated by daemon (the 80%)
- "Routing overhead" is workflow debt, not necessary orchestrator function

Neither finding is reflected in the current skill. The skill treats orchestrators as "delegation machines" rather than "strategic comprehenders."

**Source:**

- `.kb/investigations/2026-01-15-inv-orchestrator-skill-drift-audit.md:29-43` (H1: Strategic Orchestrator Model not reflected)
- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md:58-76` (Finding 1: Strategic model redefines division of labor)
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` (referenced but not read - claimed by investigations)

**Significance:** This is the single most important missing piece. The skill's identity framing shapes orchestrator behavior more than any specific instruction (per lifecycle model: "Framing is stronger than instructions"). Without COMPREHEND → TRIAGE → SYNTHESIZE, orchestrators default to tactical dispatch mode.

---

### Finding 5: Daemon-Orchestrator Division of Labor Isn't Codified

**Evidence:** The value-add investigation established clear roles:

| Work Type                                                            | Who Does It                 |
| -------------------------------------------------------------------- | --------------------------- |
| Poll beads, infer skill, spawn, monitor completion, close            | Daemon (automated)          |
| Synthesis, goal refinement, frame correction, hotspot detection      | Orchestrator (judgment)     |
| Triage decisions (type correctness, scope clarity, dependency check) | Orchestrator (gates daemon) |

The skill mentions daemon in passing (`orch daemon run`, `orch daemon preview`) but doesn't frame the relationship. An orchestrator reading the skill would think they need to actively manage spawning - which is daemon's job.

Additionally, the triage workflow (how orchestrator labels control daemon autonomy) is buried in work-pipeline.md reference file rather than being a core operating pattern.

**Source:**

- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md:102-135` (Finding 3: Daemon automates routing)
- `~/.opencode/skill/orchestrator/reference/work-pipeline.md:1-44`
- `~/.opencode/skill/orchestrator/SKILL.md` (no daemon relationship section)

**Significance:** Without explicit daemon-orchestrator framing, orchestrators over-index on spawning mechanics (which daemon handles) and under-index on triage quality and synthesis (which only they can do).

---

### Finding 6: Session Lifecycle Guidance Is Outdated

**Evidence:** The orchestrator session lifecycle model (updated Jan 29) documents:

- Session handoff machinery was removed in Jan 2026
- Orchestrators now produce SYNTHESIS.md (same as workers)
- Context continuity relies on kb/beads capture during work, not session handoffs
- `orch session start/end` commands were deprecated and removed

The skill still references:

- SESSION_HANDOFF.md as orchestrator artifact (lines implied by core responsibilities)
- `orch session start/end` workflow patterns
- Window-scoped handoff directories

**Source:**

- `.kb/models/orchestrator-session-lifecycle.md:312-324` (Phase 7: Session Handoff Machinery Removal)
- `~/.opencode/skill/orchestrator/reference/strategic-alignment.md:34` (references session tracking)

**Significance:** Outdated session lifecycle guidance creates confusion and wasted effort. Orchestrators may try to use removed commands or create artifacts that no longer have supporting infrastructure.

---

## Synthesis

**Key Insights:**

1. **The skill was optimized for the wrong constraint.** Token pressure drove a 17-file progressive disclosure architecture. At 1M context, the entire original skill (~8,800 tokens) is 0.88% of budget. The split's cognitive and maintenance costs exceed its token savings. All content should be inline.

2. **Framing is the skill's most important function.** The orchestrator session lifecycle model establishes that "framing is stronger than instructions" - ORCHESTRATOR_CONTEXT.md sets behavioral mode through framing, not directive guidance. The skill's most impactful content is identity framing (COMPREHEND → TRIAGE → SYNTHESIZE), not procedural instructions. Yet the current skill leads with mechanics.

3. **The skill teaches "how the system works" when it should teach "what to do."** ~300+ lines explain context detection, skill system architecture, amnesia-resilient design, and knowledge placement. Agents don't need to understand architecture - they need clear operational guidance. Remove explanatory content, keep actionable content.

4. **The daemon-orchestrator relationship is the new central organizing concept.** With daemon handling 80% of routing, the orchestrator skill should be organized around what the orchestrator uniquely does: comprehend, triage (with judgment), and synthesize. The current mechanics-first organization (delegate → spawn → complete) implicitly treats orchestrators as manual dispatchers.

5. **Prior investigations found the answers but they haven't reached the skill.** The strategic orchestrator model (Jan 7), daemon value-add analysis (Jan 17), and drift audit (Jan 15) all produced findings that should be in the skill. The skill has accumulated incident-driven patches without integrating strategic decisions.

**Answer to Investigation Question:**

The orchestrator skill should be redesigned as a single unified file (~500-600 lines, ~4K tokens = 0.4% of 1M context) organized around the Strategic Orchestrator Model (COMPREHEND → TRIAGE → SYNTHESIZE) rather than the current mechanics-first organization (delegate → spawn → complete).

**What to keep (load-bearing):**

- Absolute Delegation Rule (with frame collapse awareness)
- Skill selection decision trees
- Spawn mechanics (kb context, critical path checklist, methods)
- Completion verification workflows
- Core responsibilities distinction
- Red flags and quick decision trees

**What to remove (cargo cult + outdated):**

- Context detection section (infrastructure handles it)
- Skill system architecture explanation (agents don't need to know)
- Progressive disclosure reference file system (eliminate reference/ directory entirely)
- Interaction anti-patterns (modern agents handle this naturally)
- Amnesia-resilient design reference (templates enforce this)
- Knowledge placement guide (commands enforce this)
- Session handoff references (machinery removed)

**What to add (missing):**

- COMPREHEND → TRIAGE → SYNTHESIZE operating model as identity framing
- Daemon-orchestrator division of labor
- Triage as judgment bottleneck (labels gate daemon autonomy)
- Session lifecycle (checkpoint thresholds, SYNTHESIS.md artifact, kb/beads capture)
- Friction identification as meta-capability

---

## Structured Uncertainty

**What's tested:**

- ✅ Current skill is 375 lines core + 16 reference files (verified: read all files, counted)
- ✅ Reference README states 10K token goal as design constraint (verified: read README.md:57-66)
- ✅ 19 drift items exist between skill and models/guides (verified: read drift audit investigation)
- ✅ Strategic Orchestrator Model decision exists but isn't in skill (verified: drift audit H1, read skill completely)
- ✅ Session handoff machinery was removed Jan 2026 (verified: lifecycle model Phase 7)
- ✅ Daemon automates poll-spawn-complete cycle (verified: value-add investigation Finding 3)
- ✅ Orchestrator value is synthesis/judgment/comprehension not routing (verified: value-add investigation)

**What's untested:**

- ⚠️ Whether removing cargo cult sections actually improves orchestrator performance (hypothesis: less noise = better signal, not benchmarked)
- ⚠️ Whether the proposed ~500-600 line unified structure is the right size (could be too long or too short, not validated)
- ⚠️ Whether the COMPREHEND→TRIAGE→SYNTHESIZE framing as lead section improves behavior over mechanics-first ordering (hypothesis based on "framing stronger than instructions" model, not A/B tested)
- ⚠️ Whether 17% context usage from the spawning session is representative (single session, may not generalize)
- ⚠️ Whether knowledge placement and auto-capture guidance is truly unnecessary (claim that agents do it naturally, not validated across multiple sessions)

**What would change this:**

- If removing knowledge placement guidance leads to agents not using kn/kb → guidance needed after all
- If 500-600 lines turns out to be too many for agents to internalize → need compression, not progressive disclosure
- If the COMPREHEND→TRIAGE→SYNTHESIZE framing doesn't change orchestrator behavior → framing hypothesis is wrong
- If future context windows shrink (unlikely) → progressive disclosure becomes relevant again

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation                               | Authority      | Rationale                                                                               |
| -------------------------------------------- | -------------- | --------------------------------------------------------------------------------------- |
| Restructure skill as single unified file     | architectural  | Cross-component: affects all orchestrator sessions, skill loading, maintenance patterns |
| Organize around COMPREHEND→TRIAGE→SYNTHESIZE | architectural  | Integrates prior strategic decisions into operational tool                              |
| Remove reference/ directory                  | implementation | Mechanical change, reversible, within skill maintenance scope                           |
| Add daemon-orchestrator framing              | architectural  | New conceptual model in the skill, changes how orchestrators understand their role      |

### Recommended Approach ⭐

**Unified Single-File Skill Organized Around Strategic Operating Model** - Replace 17-file progressive disclosure architecture with a single ~500-600 line SKILL.md organized around COMPREHEND → TRIAGE → SYNTHESIZE.

**Why this approach:**

- Eliminates progressive disclosure overhead (cognitive + maintenance) that was optimizing for 200K constraints
- Integrates strategic decisions (Jan 7 model, Jan 17 value-add) that are currently missing
- Aligns skill structure with how orchestrators actually operate (comprehend patterns, triage work, synthesize findings)
- At 1M context, ~4K tokens is 0.4% - negligible cost for complete guidance

**Trade-offs accepted:**

- Slightly larger always-loaded token cost (~4K vs ~2.2K current core)
- Loses ability to load reference details "on demand" (but evidence shows this rarely happened)
- Requires rebuilding skill from source templates if using skillc
- One-time migration effort to consolidate and rewrite

**Implementation sequence:**

1. **Write proposed new SKILL.md** - Single unified file with all content inline, organized around COMPREHEND → TRIAGE → SYNTHESIZE
2. **Validate against recent sessions** - Run the new skill through 2-3 orchestrator sessions and check if behavior improves
3. **Remove reference/ directory** - Once unified skill validated, delete reference files
4. **Update skillc source** - If skill is auto-generated, update templates in orch-knowledge

### Alternative Approaches Considered

**Option B: Keep Progressive Disclosure, Fix Content**

- **Pros:** Preserves existing architecture, smaller change, less risk
- **Cons:** Optimizing for wrong constraint (tokens not scarce at 1M). Maintenance burden of 17 files persists. Drift will re-accumulate. Doesn't address structural organization problem (mechanics-first vs model-first).
- **When to use instead:** If context windows shrink below 200K again (unlikely)

**Option C: Minimal Skill + Heavy Infrastructure**

- **Pros:** Smallest possible skill footprint. Move all guidance into infrastructure (plugins, hooks, gates)
- **Cons:** Per principles, "Infrastructure Over Instruction" is the goal, but not all guidance can be infrastructuralized. Strategic framing (COMPREHEND→TRIAGE→SYNTHESIZE), skill selection judgment, and synthesis patterns require reasoning, not gates.
- **When to use instead:** When more guidance can be mechanized into plugins/hooks. Long-term aspiration.

**Rationale for recommendation:** Option A directly addresses all three problems: wrong optimization target (tokens → effectiveness), missing strategic framing (COMPREHEND→TRIAGE→SYNTHESIZE), and cargo cult content. The 1M context era makes the tradeoff obvious - include everything, optimize for clarity.

---

### Implementation Details

**What to implement first:**

- Draft the new unified SKILL.md with proposed section structure (see below)
- Get Dylan's review on the structural change before implementation

**Proposed new section structure:**

```
# Orchestrator Skill

## Identity: Strategic Comprehension
- COMPREHEND → TRIAGE → SYNTHESIZE operating model
- You comprehend patterns, you don't dispatch tasks
- Daemon handles routine routing; you handle judgment

## The One Rule: Absolute Delegation
- Never do implementation work (code, investigation, debugging)
- What you do directly: synthesis, triage, knowledge integration
- Frame collapse: how it happens, how to detect it
- The test: "If you're about to read code → STOP, that's an investigation"

## Operating Loop

### Comprehend
- Reading SYNTHESIS.md from completed agents
- Cross-agent pattern recognition
- Goal refinement with Dylan
- Friction identification (process improvements)

### Triage
- Two-tier issue quality (known cause → bd create, unknown → issue-creation)
- Triage labels gate daemon autonomy (triage:ready vs triage:review)
- Hotspot detection (5+ bugs → architect, not more debugging)
- When manual spawn is needed vs when daemon handles it

### Synthesize
- Cross-agent synthesis (combining findings)
- Knowledge integration (promoting to decisions, creating issues)
- Interactive synthesis with Dylan
- Conflict resolution (contradictory findings)

## Skill Selection (What to Spawn)
- Decision trees for build/fix/understand/scope/design
- Design triage (feature requests)
- Bug triage (broken things)
- All skill details inline (no reference file needed)

## Spawning (How to Spawn)
- Pre-spawn: `kb context` (mandatory)
- Critical path context checklist
- Methods in order of preference
- Phase/model/MCP selection

## Completion (How to Close)
- `orch complete` and `orch review` workflows
- Verification requirements
- Integration audit for epics
- Uncommitted changes: present to Dylan

## Autonomy (How to Interact)
- Always Act: complete agents, synthesize, monitor
- Propose-and-Act: single-action operations
- Actually Ask: genuine ambiguity, tradeoffs, irreversible
- The Mind-Reading Test

## Session Management
- Checkpoint thresholds: 2h/3h/4h (duration as proxy)
- SYNTHESIS.md for spawned orchestrators
- kb/beads capture for context continuity
- Session end: write synthesis and wait (don't /exit)

## Tools & Commands
- orch, bd, kb/kn quick reference
- Strategic: focus, drift, next
- Always use `orch send`, never raw tmux

## Red Flags
- Orchestrator: reading code, making edits, debugging
- Agent: claims complete without evidence, pushing remote
- Verification: UI without browser check
- System: issue closure ≠ feature works
```

**Things to watch out for:**

- ⚠️ Constraint from spawn context: "Auto-generated skills require template edits" - if orchestrator skill uses skillc, need to edit source templates, not SKILL.md directly
- ⚠️ Constraint: "orch-knowledge repo is at ~/orch-knowledge" - skill sources may live there
- ⚠️ The drift audit found skill is generated from `~/orch-knowledge/skills/src/meta/orchestrator/` - changes must go to source files
- ⚠️ Don't over-compress - at 1M context, clarity > brevity
- ⚠️ Preserve the Absolute Delegation Rule emphasis (proven load-bearing through multiple failure incidents)

**Areas needing further investigation:**

- How exactly is the skill generated? (skillc vs manual, source template structure)
- Should the skill reference principles.md or inline key principles?
- Is there a way to validate skill effectiveness empirically (A/B with old vs new)?

**Success criteria:**

- ✅ Single file SKILL.md with all content inline (no reference/ directory)
- ✅ COMPREHEND→TRIAGE→SYNTHESIZE as leading identity section
- ✅ Daemon-orchestrator relationship explicitly documented
- ✅ All 19 drift items from Jan 15 audit resolved in new version
- ✅ Session lifecycle reflects current state (no session handoff references)
- ✅ Orchestrator sessions show improved behavior (fewer frame collapses, better triage quality) - qualitative validation over 1 week

---

## References

**Files Examined:**

- `~/.opencode/skill/orchestrator/SKILL.md` (376 lines) - Current core skill
- `~/.opencode/skill/orchestrator/reference/*.md` (16 files, ~1,200 lines total) - All reference files
- `.kb/investigations/2026-01-15-inv-orchestrator-skill-drift-audit.md` (408 lines) - 19 drift items
- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md` (409 lines) - Value-add analysis
- `.kb/investigations/2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md` (288 lines) - Completion rate analysis
- `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` (343 lines) - Session management
- `.kb/models/orchestrator-session-lifecycle.md` (393 lines) - Lifecycle model with 7 evolution phases
- `~/.kb/principles.md` (935 lines) - System principles

**Commands Run:**

```bash
# Verify project directory
pwd  # /Users/dylanconlin/Documents/personal/orch-go

# Create investigation file
kb create investigation redesign-orchestrator-skill-1m-context
```

**Related Artifacts:**

- **Decision:** `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Establishes COMPREHEND→TRIAGE→SYNTHESIZE
- **Decision:** `.kb/decisions/2026-01-19-remove-session-handoff-machinery.md` - Session handoff removal
- **Investigation:** `.kb/investigations/2026-01-15-inv-orchestrator-skill-drift-audit.md` - 19 drift items
- **Investigation:** `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md` - Orchestrator value analysis
- **Model:** `.kb/models/orchestrator-session-lifecycle.md` - Orchestrator lifecycle with evolution phases

---

## Investigation History

**[2026-02-05 10:00]:** Investigation started

- Initial question: What should the orchestrator skill contain for 1M context era?
- Context: Skill was written at 200K, split into 17 files, accumulated 24K tokens of guidance. Spawning session showed 17% context usage after ~25 agent spawns.

**[2026-02-05 10:15]:** Read core SKILL.md and all 16 reference files

- Total content: ~1,600 lines across 17 files
- Core skill: 375 lines focused on mechanics (delegation, spawning, completion)
- Reference files: ranging from 16 lines (backlog-ownership) to 509 lines (skill-selection-details)

**[2026-02-05 10:30]:** Read 4 prior investigations and lifecycle model

- Drift audit: 19 drift items, most significant is missing Strategic Orchestrator Model
- Value-add: Daemon handles 80% routing, orchestrator value is synthesis/judgment (20%)
- 18% completion: By design (coordination role, not task)
- Session management: Handoff machinery removed Jan 2026

**[2026-02-05 10:45]:** Classification complete

- 5 load-bearing patterns identified
- ~300+ lines of cargo cult content identified
- Progressive disclosure architecture identified as outdated optimization
- COMPREHEND→TRIAGE→SYNTHESIZE and daemon relationship identified as critical missing pieces

**[2026-02-05 11:00]:** Investigation completed

- Status: Complete
- Key outcome: Propose unified single-file skill (~500-600 lines) organized around strategic operating model, eliminating reference/ directory and cargo cult content while adding missing strategic framing
