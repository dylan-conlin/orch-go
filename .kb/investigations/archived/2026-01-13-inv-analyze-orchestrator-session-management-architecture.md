<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch spawn orchestrator` and `orch session start/end` are COMPLEMENTARY mechanisms solving different orchestration problems - hierarchical delegation (spawn autonomous orchestrator agents) vs temporal continuity (resume human orchestration sessions after breaks).

**Evidence:** Read orchestrator_context.go (spawned agent lifecycle: ORCHESTRATOR_CONTEXT.md → SESSION_HANDOFF.md → wait for external completion) vs session.go (interactive lifecycle: session.json → orch session end → SESSION_HANDOFF.md with symlink). Completion protocols differ: spawned agents wait for `orch complete`, interactive sessions self-complete via `orch session end`.

**Knowledge:** The 2026-01-04 investigation analyzed interactive sessions only, leaving spawned orchestrator pattern undocumented. Both mechanisms are architecturally sound but lack usage guidance. SESSION_HANDOFF.md serves different purposes in each context (progressive agent documentation vs reflective human handoff).

**Next:** Keep both mechanisms. Add decision tree to orchestrator skill clarifying when to use each. Update session-resume-protocol.md to specify "interactive sessions only." Create spawned-orchestrator-pattern.md guide with examples.

**Promote to Decision:** recommend-no (tactical guidance gap, not architectural decision worth preserving as formal decision record)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Analyze Orchestrator Session Management Architecture

**Question:** Are `orch spawn orchestrator` and `orch session start/end` redundant or complementary? When should each be used? Do they solve the multi-session problem differently?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** og-arch agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Two distinct mechanisms exist with different lifecycles

**Evidence:**

**Mechanism 1: `orch spawn orchestrator`** (spawned orchestrator agent)
- Creates ORCHESTRATOR_CONTEXT.md at `.orch/workspace/{name}/ORCHESTRATOR_CONTEXT.md`
- Pre-creates SESSION_HANDOFF.md with metadata for progressive filling
- Agent spawned WITH skill context (orchestrator skill embedded)
- Completion: Writes SESSION_HANDOFF.md and WAITS for level above to run `orch complete`
- No `/exit` or `orch session end` - completion is external
- Tracked via workspace, NOT beads issues (no `.beads_id` file)
- Tier: "orchestrator" (special verification rules)

**Mechanism 2: `orch session start/end`** (interactive orchestrator session)
- Creates session.json at `~/.orch/session.json` with goal, start time, spawns
- Creates timestamped directory at `{project}/.orch/session/{timestamp}/`
- Session end creates SESSION_HANDOFF.md and updates `latest` symlink
- Completion: Orchestrator runs `orch session end` themselves
- Tracked via session state, not workspace artifacts
- Session resume auto-injects latest handoff via hooks

**Source:**
- `pkg/spawn/orchestrator_context.go:19-127` (ORCHESTRATOR_CONTEXT template)
- `cmd/orch/session.go:68-143` (session start command)
- `cmd/orch/session.go:447-542` (session end command)
- `.kb/guides/session-resume-protocol.md` (session resume documentation)

**Significance:** These are NOT the same mechanism with different names - they have fundamentally different lifecycles (agent-based vs session-based) and different completion protocols (external vs self-directed).

---

### Finding 2: They solve different orchestration problems

**Evidence:**

**`orch spawn orchestrator` solves:** Hierarchical orchestration
- Meta-orchestrator spawns orchestrator agent to accomplish specific goal
- Orchestrator agent works autonomously toward goal, spawning workers
- Produces SESSION_HANDOFF.md as completion signal
- Meta-orchestrator reviews handoff and completes the orchestrator via `orch complete`
- Use case: "I need an orchestrator to handle epic X while I handle epic Y"

**`orch session start/end` solves:** Human-driven interactive orchestration
- Dylan (the human) IS the orchestrator
- Session tracking provides continuity across conversation breaks
- Session resume auto-injects prior context via hooks
- Use case: "I'm orchestrating work, need to take a break, resume tomorrow"

**Source:**
- `pkg/spawn/orchestrator_context.go:28-35` ("You are a spawned orchestrator...different from interactive sessions")
- `.kb/guides/session-resume-protocol.md:29-41` ("Dylan's core need: start any session by saying 'let's resume'")
- `pkg/spawn/orchestrator_context.go:73-88` (completion protocol - wait for level above)

**Significance:** These mechanisms address different needs in the orchestration hierarchy. One enables **delegation** (spawn an orchestrator to handle a goal), the other enables **continuity** (resume an interrupted human orchestration session).

---

### Finding 3: Both produce SESSION_HANDOFF.md but with different templates and purposes

**Evidence:**

**Spawned orchestrator handoff** (progressive):
- Pre-created with metadata at workspace creation
- Template at `pkg/spawn/orchestrator_context.go:358-522` (PreFilledSessionHandoffTemplate)
- Designed for filling AS YOU WORK (progressive documentation)
- Sections: TLDR, Spawns table, Evidence, Knowledge, Friction, Focus Progress, Next, Unexplored Questions
- Purpose: Signal completion to level above with synthesis of session's work

**Interactive session handoff** (reflective):
- Created at session end via `orch session end`
- Template at `cmd/orch/session.go:684-725` (inline template)
- Designed for END-OF-SESSION reflection
- Sections: Summary, What Was Accomplished, Active Work, Pending Work, Recommendations, Context
- Purpose: Resume context for next interactive session

**Source:**
- `pkg/spawn/orchestrator_context.go:358-522` (spawned orchestrator template)
- `cmd/orch/session.go:666-752` (createSessionHandoffDirectory function)
- Comparison: Progressive vs reflective documentation patterns

**Significance:** Even though both produce SESSION_HANDOFF.md, the templates differ to serve their contexts - spawned orchestrators document progressively during work (agent mindset), interactive sessions reflect at the end (human mindset).

---

### Finding 4: Prior investigation concluded "orchestrators are already spawnable" but didn't address hierarchical orchestration

**Evidence:**

The 2026-01-04 investigation (`.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md`) concluded:
- "Orchestrators ARE already structurally spawnable" (line 140)
- "orch session start/end IS the orchestrator spawn mechanism" (line 183)
- Recommended incremental enhancement of existing `orch session` commands

However, that investigation focused on INTERACTIVE orchestrator sessions (Dylan as orchestrator), not SPAWNED orchestrator agents (meta-orchestrator spawning orchestrator).

Current codebase has BOTH:
- Interactive orchestrator sessions (what 2026-01-04 analyzed)
- Spawned orchestrator agents (via `IsOrchestrator` flag, ORCHESTRATOR_CONTEXT.md)

**Source:**
- `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md:7-14` (D.E.K.N. summary)
- `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md:140-185` (conclusion about orch session)
- `pkg/spawn/config.go:134-140` (IsOrchestrator flag and behavior)
- `pkg/spawn/orchestrator_context.go` (entire file - spawned orchestrator infrastructure)

**Significance:** The 2026-01-04 investigation was correct within its scope (interactive sessions), but the codebase has evolved to support a SECOND orchestration pattern (spawned agents) that wasn't addressed in that investigation. This creates potential confusion about "which orchestrator mechanism to use when."

---

## Synthesis

**Key Insights:**

1. **These mechanisms are complementary, not redundant** - `orch spawn orchestrator` enables hierarchical orchestration (meta-orchestrator spawning orchestrator agents), while `orch session start/end` enables human-driven interactive orchestration with continuity across breaks. They solve different problems in the orchestration hierarchy.

2. **Different lifecycles require different completion protocols** - Spawned orchestrators wait for external completion (`orch complete` by level above) after writing SESSION_HANDOFF.md, while interactive sessions self-complete via `orch session end`. This reflects agent vs human agency.

3. **SESSION_HANDOFF.md serves different purposes in each context** - Progressive documentation (fill as you work) for spawned agents signals completion readiness, while reflective documentation (fill at end) for interactive sessions enables resume context. Same artifact name, different purposes.

4. **Prior investigations didn't capture full picture** - The 2026-01-04 investigation correctly analyzed interactive sessions but didn't address the spawned orchestrator pattern that exists in the codebase. This gap leaves guidance incomplete.

**Answer to Investigation Question:**

**Are they redundant or complementary?** COMPLEMENTARY. They address different orchestration patterns:
- `orch spawn orchestrator`: Hierarchical delegation (meta-orchestrator → orchestrator → workers)
- `orch session start/end`: Human-driven continuity (Dylan orchestrates, sessions resume across breaks)

**When should each be used?**

| Use Case | Mechanism | Why |
|----------|-----------|-----|
| **Meta-orchestrator needs to delegate epic to autonomous orchestrator** | `orch spawn orchestrator` | Creates orchestrator agent with goal, workspace, skill context. Meta-orchestrator can work on other epics while spawned orchestrator handles this one. |
| **Dylan is orchestrating and needs to take a break** | `orch session start/end` | Session tracking + resume hooks provide continuity. Next session auto-injects prior handoff. |
| **Need multiple orchestrators working on different epics concurrently** | `orch spawn orchestrator` (multiple) | Each spawned orchestrator has its own workspace and goal. |
| **Single human orchestrator managing multi-session work** | `orch session start/end` | One session.json tracks focus, spawns, duration across conversation breaks. |

**Do they solve multi-session problem differently?**

YES, they solve DIFFERENT multi-session problems:

1. **`orch spawn orchestrator`** solves: "How do I spawn multiple concurrent orchestrators to handle different epics?"
   - Answer: Spawn multiple orchestrator agents, each with ORCHESTRATOR_CONTEXT.md and workspace
   - Each orchestrator is autonomous, produces SESSION_HANDOFF.md when done
   - Meta-orchestrator reviews handoffs and completes them via `orch complete`

2. **`orch session start/end`** solves: "How do I maintain continuity when I (human orchestrator) take breaks?"
   - Answer: Session tracking via session.json + automatic resume via hooks
   - SESSION_HANDOFF.md at session end enables next session to resume context
   - One orchestrator (human), multiple sessions over time

**Critical distinction:** Hierarchical (spawn multiple) vs temporal (resume after break).

---

## Structured Uncertainty

**What's tested:**

- ✅ Both mechanisms exist in codebase (verified: read pkg/spawn/orchestrator_context.go and cmd/orch/session.go)
- ✅ ORCHESTRATOR_CONTEXT.md template differs from SPAWN_CONTEXT.md (verified: compared templates at lines 19-127 vs lines 38-286 in context.go)
- ✅ SESSION_HANDOFF.md templates differ between spawned and interactive (verified: compared PreFilledSessionHandoffTemplate vs createSessionHandoffDirectory)
- ✅ Completion protocols differ (verified: ORCHESTRATOR_CONTEXT says "WAIT for level above", session.go runs self-completion)
- ✅ Prior investigation didn't address spawned orchestrators (verified: read 2026-01-04 investigation - only mentions interactive sessions)

**What's untested:**

- ⚠️ Whether spawned orchestrators are actually used in practice (assumed based on infrastructure existence, not observed in use)
- ⚠️ Whether the distinction is clear to users (hypothesized confusion, not validated with user feedback)
- ⚠️ Whether both mechanisms will continue to evolve independently (assumption about future, not tested)

**What would change this:**

- If no spawned orchestrator workspaces exist in `.orch/workspace/` → mechanism exists but isn't used
- If users consistently choose wrong mechanism → guidance is insufficient or mechanisms aren't distinct enough
- If future changes unify the mechanisms → they weren't truly complementary, just implementation duplication

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Keep both mechanisms and add explicit guidance to orchestrator skill** - Document when to use spawned orchestrator vs interactive session.

**Why this approach:**
- Mechanisms serve genuinely different needs (hierarchical vs temporal orchestration)
- Both are already implemented and working
- Gap is guidance/discoverability, not architecture
- Unifying them would lose one of the orchestration patterns (either delegation or continuity)

**Trade-offs accepted:**
- Increased complexity (two mechanisms instead of one)
- Potential for user confusion about which to use
- Both mechanisms need maintenance and evolution
- Worth it because they solve fundamentally different problems

**Implementation sequence:**
1. **Add decision tree to orchestrator skill** - "Am I spawning orchestrators or managing my own session?" with clear use cases
2. **Update session-resume-protocol.md** - Clarify this is for interactive sessions only, not spawned orchestrators
3. **Create spawned-orchestrator-pattern.md guide** - Document the hierarchical orchestration pattern with examples
4. **Add usage examples to orch spawn --help** - Show both `orch spawn orchestrator` and `orch session start` patterns

### Alternative Approaches Considered

**Option B: Unify mechanisms into single `orch orchestrator` command**
- **Pros:** Simpler conceptual model, one mechanism to learn
- **Cons:** Loses hierarchical orchestration capability (can't spawn autonomous orchestrator agents). Would force meta-orchestrator to micromanage spawned orchestrators or give up delegation entirely.
- **When to use instead:** If hierarchical orchestration proves unnecessary in practice (no evidence of this yet)

**Option C: Deprecate `orch session start/end` in favor of spawned orchestrators only**
- **Pros:** Forces all orchestration through spawn mechanism, more uniform
- **Cons:** Human orchestrators (Dylan) would need to spawn themselves, losing natural session continuity. Resume protocol wouldn't work (spawned orchestrators wait for external completion, but Dylan can't complete himself).
- **When to use instead:** Never - human orchestrators need different lifecycle than spawned agents

**Option D: Make spawned orchestrators use session.json instead of workspaces**
- **Pros:** Could potentially unify state tracking
- **Cons:** Session.json is global (one per machine), workspaces are per-spawn (enables multiple concurrent orchestrators). Unification would prevent concurrent orchestrator agents.
- **When to use instead:** If concurrent orchestrator spawns aren't needed (conflicts with hierarchical orchestration goal)

**Rationale for recommendation:** Option A (keep separate + add guidance) preserves both orchestration patterns while addressing the real gap (discoverability and usage clarity). Alternatives either lose capability or create architectural contradictions.

---

### Implementation Details

**What to implement first:**
- Add decision tree to orchestrator skill ("Spawning Orchestrators vs Managing Sessions" section)
- Add comparison table showing both mechanisms side-by-side with use cases
- Update ORCHESTRATOR_CONTEXT.md template to clarify "you are a SPAWNED orchestrator, different from interactive sessions"

**Things to watch out for:**
- ⚠️ Don't conflate the two mechanisms in documentation - they're complementary, not synonyms
- ⚠️ Spawned orchestrators MUST NOT use `orch session start/end` (they have ORCHESTRATOR_CONTEXT.md, not session.json)
- ⚠️ Interactive sessions MUST NOT use workspace-based artifacts (they have session.json + timestamped directories)
- ⚠️ Both produce SESSION_HANDOFF.md but with different templates - don't mix them up

**Areas needing further investigation:**
- Are there workspaces with .orchestrator marker? (validates spawned orchestrators are used in practice)
- How often does Dylan use `orch session start/end`? (validates interactive session pattern is used)
- Are there cases where neither mechanism fits? (might reveal missing orchestration pattern)

**Success criteria:**
- ✅ Orchestrator skill has clear "which mechanism to use" decision tree
- ✅ Session resume protocol guide clarifies scope (interactive sessions only)
- ✅ Spawned orchestrator pattern documented with examples
- ✅ Users can answer "should I spawn or start a session?" without confusion

---

## References

**Files Examined:**
- `pkg/spawn/orchestrator_context.go` - ORCHESTRATOR_CONTEXT.md template for spawned orchestrators
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template for comparison
- `pkg/spawn/config.go` - IsOrchestrator and IsMetaOrchestrator flags
- `cmd/orch/session.go` - Session start/end/resume commands
- `.kb/guides/session-resume-protocol.md` - Session resume documentation
- `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md` - Prior investigation

**Commands Run:**
```bash
# None - this was purely codebase analysis
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md` - Prior analysis of orchestrator spawnability (focused on interactive sessions only)
- **Guide:** `.kb/guides/session-resume-protocol.md` - Session resume mechanics (applies to interactive sessions, not spawned orchestrators)

---

## Investigation History

**2026-01-13 13:45:** Investigation started
- Initial question: Are `orch spawn orchestrator` and `orch session start/end` redundant or complementary?
- Context: Task spawned to analyze orchestrator session management architecture

**2026-01-13 14:00:** Key finding emerged
- Discovered two distinct mechanisms with different lifecycles
- Spawned orchestrators (agent-based) vs interactive sessions (human-driven)

**2026-01-13 14:15:** Synthesis completed
- Status: Complete
- Key outcome: Mechanisms are COMPLEMENTARY (hierarchical vs temporal orchestration) - recommend keeping both with improved guidance
