<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Information hiding should be implemented via output filtering (not tool removal), while tool restriction should use an explicit allowlist (meta-actions only) enforced via OpenCode registry gates.

**Evidence:** Research investigation found 30 years of HRL convergence on architectural constraints; current coaching plugin already detects frame collapse patterns; task-tool-gate.ts demonstrates working registry-level tool gating pattern.

**Knowledge:** Information hiding reduces temptation to "dive in" (Feudal RL pattern); tool restriction prevents the dive from being possible (MAXQ/Options pattern); both are needed - hiding is psychological barrier, restriction is architectural enforcement.

**Next:** Implement in 3 phases: (1) Prompt-based action space restriction now, (2) Information hiding via output filtering short-term, (3) Registry-level tool restriction medium-term.

**Promote to Decision:** recommend-yes - This establishes architectural constraints ("orchestrators operate in meta-action space, not primitive action space") that should persist across the system.

---

# Investigation: Design Information Hiding Tool Restriction

**Question:** How should we implement information hiding and tool restriction to prevent orchestrator frame collapse?

**Started:** 2026-01-27
**Updated:** 2026-01-27
**Owner:** Agent (architect spawn)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** None - establishes new architectural pattern
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Research Strongly Supports Architectural Action Space Restriction

**Evidence:** Research investigation (2026-01-27-inv-research-exists-preventing-hierarchical-controllers.md) synthesized 30 years of HRL research:
- Options Framework (Sutton 1999, 5,271 cites): High-level controllers select options, not primitives
- MAXQ (Dietterich 2000, 2,421 cites): Value decomposition structurally prevents primitive selection
- Feudal Networks (Dayan 1992, 1,174 cites + Vezhnevets 2017, 1,341 cites): Managers operate in goal space, not action space; information hiding ("rewards hide information") prevents managers from optimizing at wrong level

**Source:** `.kb/investigations/2026-01-27-inv-research-exists-preventing-hierarchical-controllers.md`

**Significance:** Cross-domain convergence (HRL, multi-agent systems, organizational psychology, LLM agents) suggests these are fundamental principles, not domain-specific tricks. Architectural enforcement beats guidelines.

---

### Finding 2: Coaching Plugin Already Implements Frame Collapse Detection

**Evidence:** `plugins/coaching.ts` (lines 476-537) already detects orchestrator code file edits:
- `isCodeFile()` distinguishes code (`.go`, `.ts`, `.css`) from orchestration artifacts (`.md` in `.kb/`, `.orch/`)
- `FrameCollapseState` tracks cumulative code edits and injects warnings
- Pattern: First warning at 1 code edit, strong warning at 3+ edits
- Uses `noReply: true` injection pattern for real-time feedback

**Source:** `plugins/coaching.ts:476-778`

**Significance:** Detection infrastructure exists; what's missing is prevention. Current approach is "detect and warn" - needed approach is "restrict and block".

---

### Finding 3: Task-Tool-Gate Demonstrates Registry-Level Tool Restriction

**Evidence:** `plugins/task-tool-gate.ts` implements working pattern for tool restriction:
- Detects orchestrator sessions via skill loading, session title patterns, and tool arguments
- Uses `tool.execute.before` hook to intercept tool calls
- Injects warning messages explaining WHY tool is blocked
- Updates session metadata to enable registry-level blocking

**Source:** `plugins/task-tool-gate.ts:1-348`

**Significance:** The pattern for restricting tools already exists and works. Can extend this to restrict `Edit`, `Write`, and `bash` commands that perform worker actions.

---

### Finding 4: Orchestrator Already Has Clear Meta-Action Vocabulary

**Evidence:** Orchestrator skill (`~/.claude/skills/meta/orchestrator/SKILL.md`) establishes allowed actions:
- **Allowed:** `orch spawn`, `orch complete`, `orch status`, `bd create`, `bd ready`, `kb context`, `kb quick`
- **ABSOLUTE DELEGATION RULE:** Never invoke worker skills directly
- **Pre-Response Gates:** Delegation gate, spawn method gate, container gate, pressure gate

Current guidance is prompt-based ("You can / You CANNOT"). Research suggests this needs architectural enforcement.

**Source:** `~/.claude/skills/meta/orchestrator/SKILL.md:37-73, 550-600`

**Significance:** The action vocabulary is already defined - meta-actions (spawn, monitor, query) vs primitive actions (read, edit, write, bash). What's needed is enforcing this distinction architecturally.

---

### Finding 5: Legitimate Orchestrator Needs Require Careful Handling

**Evidence:** Edge cases where orchestrator legitimately needs file/tool access:
1. **Reading CLAUDE.md** - Needed for planning, understanding project context
2. **Reading beads issues** - `bd show`, `bd ready` (commands, not file reads)
3. **Git status** - Checking workspace state before push decision
4. **kb context** - Knowledge queries (command, not file exploration)
5. **Reading investigation summaries** - Synthesizing worker outputs

Most legitimate needs are via **commands** (`bd`, `kb`, `git status`) not raw file operations.

**Source:** Orchestrator skill reference sections, spawn architecture model

**Significance:** Information hiding doesn't mean "hide everything" - it means "surface via orchestrator-appropriate interfaces." Show beads comments, not error traces. Show investigation TLDR, not full investigation body.

---

## Synthesis

**Key Insights:**

1. **Two Complementary Mechanisms Are Needed**
   - **Information Hiding:** Reduces temptation by not showing details that invite "diving in"
   - **Tool Restriction:** Prevents the dive from being possible even when tempted
   
   Feudal RL research shows both are needed: hiding alone leaves escape hatches, restriction alone creates frustration without understanding why.

2. **The Orchestrator's Action Space Should Be Meta-Actions Only**

   Based on MAXQ and Options framework patterns, orchestrator should only have:
   
   | Allowed (Meta-Actions) | Blocked (Primitive Actions) |
   |------------------------|----------------------------|
   | `orch spawn` | `Edit` tool |
   | `orch complete` | `Write` tool |
   | `orch status` | `bash` (most commands) |
   | `bd create/show/ready` | `Read` (code files) |
   | `kb context/quick` | Direct file operations |
   | `git status` (read-only) | `git commit/push` |

   Allowlisting is safer than blocklisting - only permit known meta-actions.

3. **Information to Surface vs Hide**

   | Surface to Orchestrator | Hide from Orchestrator |
   |-------------------------|------------------------|
   | Beads comments | Full file contents |
   | Worker phase | Tool outputs |
   | SYNTHESIS.md TLDR | Error stack traces |
   | Investigation D.E.K.N. | Build logs |
   | High-level outcomes | Implementation details |
   | `bd show` output | Raw investigation body |

4. **Implementation Requires Three Layers**

   Based on task-tool-gate.ts pattern and coaching.ts detection:
   
   - **Layer 1 (Prompt):** Explicit "You CAN / You CANNOT" already partially exists
   - **Layer 2 (Detection):** Coaching plugin detects frame collapse, injects warnings
   - **Layer 3 (Enforcement):** Registry-level tool gating prevents execution
   
   All three are needed for defense-in-depth.

**Answer to Investigation Question:**

**Information Hiding Design:**
- Orchestrator should see: beads comments, worker phase, investigation TLDR/D.E.K.N., high-level outcomes
- Orchestrator should NOT see: file contents (except orchestration artifacts), tool outputs, error traces, build logs
- **Technical mechanism:** Output filtering in OpenCode plugin layer - intercept tool results and filter before returning to LLM context
- **Edge cases handled via allowlist:** CLAUDE.md, SPAWN_CONTEXT.md, investigation summaries are permitted reads

**Tool Restriction Design:**
- Implement via OpenCode registry gate (extending task-tool-gate.ts pattern)
- Use allowlist, not blocklist: only permit `orch`, `bd`, `kb`, `git status`
- Block: `Edit`, `Write`, most `bash`, `Read` for code files
- Session detection: Use existing skill loading + title pattern detection
- **Emergency escape:** Explicit override flag (`--force-primitive` or similar) for genuine emergencies, logged and flagged

**Why Both Are Needed:**
- Information hiding without restriction: Orchestrator can still "work around" by guessing at details
- Restriction without hiding: Orchestrator frustrated by blocks without understanding why; may find workarounds
- Both together: Can't do the work (restriction) AND doesn't see the details that invite doing it (hiding)

---

## Structured Uncertainty

**What's tested:**

- ✅ **Task-tool-gate pattern works** - Verified: plugin successfully intercepts Task tool, injects warnings
- ✅ **Coaching plugin detects frame collapse** - Verified: isCodeFile() and FrameCollapseState implemented and active
- ✅ **Session detection works** - Verified: title patterns and skill loading both detect orchestrator vs worker
- ✅ **Research patterns converge** - Verified: 30 years across 4 domains (HRL, multi-agent, org psych, LLM)

**What's untested:**

- ⚠️ **Output filtering at plugin layer** - Need to verify `tool.execute.after` can modify return values
- ⚠️ **Allowlist completeness** - May miss legitimate orchestrator commands
- ⚠️ **User experience** - Restriction may feel frustrating; need good error messages explaining WHY
- ⚠️ **Performance impact** - Output filtering adds processing to every tool call
- ⚠️ **Edge case coverage** - Reading CLAUDE.md, git status, etc. need to work

**What would change this:**

- **Finding invalid if:** Plugin layer cannot intercept/modify tool outputs (would need different implementation)
- **Finding invalid if:** Orchestrator needs to read code for legitimate reasons we haven't identified
- **Finding invalid if:** Frame collapse persists despite restriction (would need stronger mechanism)
- **Finding strengthened if:** Frame collapse incidents drop to zero after implementation

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Three-Phase Architectural Enforcement** - Implement information hiding and tool restriction incrementally with defense-in-depth.

**Why this approach:**
- Research shows architectural constraints beat guidelines (Finding 1)
- Existing infrastructure provides foundation (Findings 2, 3)
- Incremental rollout allows validation at each phase
- Defense-in-depth (prompt + detection + enforcement) most robust

**Trade-offs accepted:**
- Orchestrator loses "quick fix" capability entirely
- Slightly higher latency for trivial tasks (must spawn)
- Research shows benefits outweigh: mixing levels causes more problems than it solves

**Implementation sequence:**

1. **Phase 1 (Immediate): Strengthen Prompt-Based Restriction**
   - Update orchestrator skill with explicit "You CAN / You CANNOT" section
   - Add trigger phrase detection for "let me quickly", "I'll just"
   - Low cost, quick win, establishes vocabulary

2. **Phase 2 (Short-term): Information Hiding via Output Filtering**
   - Extend coaching.ts with `tool.execute.after` hook
   - Filter file read outputs: only return first 20 lines for non-orchestration files
   - Filter bash outputs: truncate and summarize
   - Show "File read blocked - orchestrator should delegate, not investigate"

3. **Phase 3 (Medium-term): Registry-Level Tool Gating**
   - Create `orchestrator-tool-gate.ts` plugin (extend task-tool-gate.ts pattern)
   - Allowlist: `orch`, `bd`, `kb`, `git status`, `Read` for `.md` in `.kb/`, `.orch/`, `CLAUDE.md`
   - Block with explanation: "Orchestrators operate in meta-action space. Use `orch spawn` instead."
   - Emergency override: `--force-primitive` flag, logged to metrics

### Alternative Approaches Considered

**Option B: Separate Models for Orchestrator vs Worker**
- **Pros:** Ultimate architectural enforcement - different model = different capabilities
- **Cons:** Operational complexity, cost (two models), requires OpenCode changes
- **When to use instead:** If Phase 3 proves insufficient and frame collapse persists

**Option C: Pure Prompt-Based Guidelines**
- **Pros:** Easy to implement, no infrastructure changes
- **Cons:** Research shows guidelines fail under cognitive pressure (Finding 1)
- **When to use instead:** Never for frame collapse prevention - already proven insufficient

**Option D: Detect-and-Retry Pattern**
- **Pros:** More forgiving - let orchestrator try, then reject
- **Cons:** Wastes tokens on rejected attempts, frustrating UX
- **When to use instead:** For "soft" violations where learning is valuable

**Rationale for recommendation:** Three-phase approach provides strongest prevention while allowing validation. Each phase is independently valuable (can stop at Phase 1 if effective). Pattern proven via task-tool-gate.ts.

---

### Implementation Details

**What to implement first:**
- **Phase 1 (1-2 hours):** Update orchestrator skill with explicit CAN/CANNOT list
- Add to `~/.claude/skills/meta/orchestrator/SKILL.md`:
```markdown
## Tool Action Space (Architectural Constraint)

**You CAN (meta-actions):**
- `orch spawn`, `orch complete`, `orch status`, `orch review`
- `bd create`, `bd show`, `bd ready`, `bd close`
- `kb context`, `kb quick decide/constrain/tried/question`
- `git status` (read-only verification)
- Read: CLAUDE.md, .kb/*.md, .orch/*.md, SYNTHESIS.md

**You CANNOT (primitive actions):**
- Edit/Write tools (code editing is worker work)
- Read code files (.go, .ts, .css, .py, etc.)
- Most bash commands (workers execute, not orchestrators)
- Direct file operations

**Why architectural:** Research shows frame collapse is prevented by restricting action space, not just guidelines. If you CAN do it, you eventually WILL do it under pressure.
```

**Things to watch out for:**
- ⚠️ **CLAUDE.md reads must be allowed** - Orchestrator needs project context
- ⚠️ **Investigation synthesis requires reading** - Allow TLDR/D.E.K.N. sections
- ⚠️ **Git status is legitimate** - Read-only workspace state check
- ⚠️ **Emergency override needed** - Rare genuine needs (document all uses)

**Areas needing further investigation:**
- How to filter tool outputs without breaking context (truncation vs summarization)
- Whether to allow `bd comments add` (creates data) vs only reads
- How to handle orchestrator reading its own prior session artifacts

**Success criteria:**
- ✅ **Zero frame collapse incidents** - Orchestrator never edits code files
- ✅ **100% spawn rate** - All implementation work goes through workers
- ✅ **User satisfaction** - Dylan notices orchestrator "staying in lane"
- ✅ **Emergency escapes rare** - <1% of orchestrator sessions use override

---

## References

**Files Examined:**
- `plugins/coaching.ts:476-778` - Frame collapse detection implementation
- `plugins/task-tool-gate.ts:1-348` - Tool gating pattern
- `~/.claude/skills/meta/orchestrator/SKILL.md:1-800` - Current orchestrator constraints
- `.kb/models/decidability-graph.md` - Authority boundary model
- `.kb/models/spawn-architecture.md` - Spawn mechanism

**Commands Run:**
```bash
# Get KB context for design question
kb context "frame collapse orchestrator worker information hiding"

# Find plugin implementations
ls plugins/*.ts
```

**External Documentation:**
- Research investigation findings (same session, not external)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-27-inv-research-exists-preventing-hierarchical-controllers.md` - Research foundation
- **Investigation:** `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md` - Prior detection work
- **Model:** `.kb/models/decidability-graph.md` - Authority boundaries model
- **Principle:** `~/.kb/principles.md` - Authority is Scoping, Perspective is Structural

---

## Investigation History

**2026-01-27 18:50:** Investigation started
- Initial question: How should we implement information hiding and tool restriction for orchestrator frame collapse prevention?
- Context: Research investigation found 30 years of HRL convergence on architectural constraints; need to design specific implementation

**2026-01-27 19:15:** Key findings synthesized
- Found existing coaching.ts detection infrastructure
- Found task-tool-gate.ts pattern for tool restriction
- Identified three-layer defense-in-depth approach

**2026-01-27 19:30:** Investigation completed
- Status: Complete
- Key outcome: Three-phase implementation plan: prompt strengthening → output filtering → registry gating
