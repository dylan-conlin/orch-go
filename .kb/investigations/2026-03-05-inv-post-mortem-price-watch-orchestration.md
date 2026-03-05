## Summary (D.E.K.N.)

**Delta:** The Agent tool is an ungated shadow spawn path that bypasses all orch spawn infrastructure (hooks, skill loading, workspace, completion pipeline), and "manually" is undefined in orchestrator vocabulary — enabling misinterpretation of "spawn manually" as "use Agent tool" instead of "run orch spawn CLI."

**Evidence:** Session transcript shows orchestrator used Agent tool with `isolation: "worktree"` for two workers, bypassing spawn hook, skill loading, and beads tracking; worktree edits were invisible to Docker containers; 3 failed orch spawn attempts when corrected.

**Knowledge:** Claude Code's built-in Agent tool creates a parallel spawn path that no hook can intercept. Known-bad patterns (worktree + Docker) recur because knowledge exists only as investigation files, not as gates. The spawn hook validates Bash commands only — tool-level spawn bypasses are invisible.

**Next:** Three recommendations: (1) Add "Agent tool is NOT a spawn mechanism" to orchestrator skill [implementation], (2) Add "manually spawn = orch spawn CLI" vocabulary entry [implementation], (3) Gate Agent tool usage in orchestrator context via hook [architectural].

**Authority:** architectural — Recommendations span orchestrator skill, hook infrastructure, and cross-tool interaction design.

---

# Investigation: Post-Mortem — Price-Watch Orchestration Session 2026-03-05

**Question:** Why did the orchestrator use Agent tool with worktree isolation instead of orch spawn, and what systemic patterns enabled this failure?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect agent (spawned for post-mortem)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A (no existing decision — this recommends new constraints)
**Extracted-From:** `~/Documents/work/SendCutSend/scs-special-projects/price-watch/2026-03-05-122852-pw-orch-session-for-pm.txt`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-02-27-inv-claude-code-worktree-agent-isolation.md` | Directly relevant — worktree + Docker known-bad | Yes — Feb 12 P0 revert confirmed | None — this incident confirms the finding |
| `2026-02-28-investigation-orchestrator-intent-spiral.md` | Pattern match — intent displacement across boundary | Yes — same mechanism ("manually" → wrong interpretation) | None |
| `2026-03-03-inv-experiment-post-hooks-behavioral-baseline.md` | Extends — hooks enforce but can't cover all tool surfaces | Yes | None |

---

## Incident Timeline

```
12:28  Session resumes. Orchestrator orients: 3 guardrail issues ready (pw-mfdv, pw-lx1k, pw-fx4x).
12:31  Dylan: "let's work through them. daemon is down so spawn manually"
12:33  Orchestrator uses Explore agent to gather code context (~1min)
12:35  Orchestrator launches 2 Agent tool calls with isolation: "worktree"
         Agent 1: pw-lx1k + pw-mfdv (persona rotation + daily cap)
         Agent 2: pw-fx4x (freshness windows)
12:40  Dylan: "why didn't you use orch spawn?"
         Orchestrator: "I took 'manually' to mean use Agent tool directly"
12:41  Dylan: "yes [use orch spawn]"
12:43  Dylan: "status?" — Agent 1 struggling with Docker volume mounts
12:45  Dylan asks for comprehension on worktree/Docker conflict
         Orchestrator explains: bind mounts → main repo, not worktree. Edits invisible.
12:47  Dylan: "let's just not use worktrees"
         Orchestrator acknowledges it was its decision to use isolation: "worktree"
12:48  Dylan: "yes" — abandon and re-spawn
12:49  orch spawn attempt 1: HOOK DENIED — missing --issue and --intent
12:50  orch spawn attempt 2: CLI ERROR — missing --reason for --bypass-triage
12:51  orch spawn attempt 3: SUCCESS — both agents spawned correctly
12:53  Worktrees cleaned up (--force required due to modified files)
```

**Total wasted time:** ~20 minutes (12:31→12:51), including two agent sessions that produced no usable work.

---

## Findings

### Finding 1: Agent Tool Is an Ungated Shadow Spawn Path

**Evidence:** The orchestrator used the Claude Code built-in `Agent` tool with `isolation: "worktree"` to spawn two worker agents. These agents:
- Had no SKILL.md loaded (no worker-base, no feature-impl methodology)
- Had no SPAWN_CONTEXT.md (no orientation frame, no intent type, no deliverables)
- Had no beads tracking (no bd comment phases, no completion pipeline)
- Had no hotspot enforcement
- Had no workspace in `.orch/workspace/`

The spawn hook (`~/.orch/hooks/gate-spawn-context-validation.py`) operates on `PreToolUse:Bash` matching `orch spawn` commands. The Agent tool is a different tool type entirely — the hook never fires.

**Source:** Session transcript lines 140-153 (Agent tool launch), `gate-spawn-context-validation.py` lines 50-51 (regex only matches `orch spawn` in Bash commands)

**Significance:** Every safety mechanism in the orch spawn pipeline — intent validation, skill loading, workspace creation, completion compatibility — is bypassed when the Agent tool is used for spawning. The Agent tool is Defect Class 1 (Filter Amnesia): filters exist on path A (`orch spawn`), missing on path B (Agent tool).

---

### Finding 2: "Manually" Is Undefined Orchestrator Vocabulary

**Evidence:** Dylan said "daemon is down so spawn manually." This has one obvious meaning to Dylan (run `orch spawn` CLI by hand) and a different plausible meaning to the orchestrator (handle spawning yourself using available tools, i.e., Agent tool).

The orchestrator skill defines these spawn-related entries in its Fast Path table:
- "Release work to daemon" → `bd create ... -l triage:ready`
- "Spawn directly (exception)" → `orch spawn SKILL "task"`

But no entry for: "Daemon unavailable, spawn via CLI" or "Manual spawn" or "Spawn without daemon."

The orchestrator's response reveals the misinterpretation explicitly: "I took 'manually' to mean use Agent tool directly rather than orch spawn" (line 159).

**Source:** Session transcript lines 92 (Dylan's instruction), 159-162 (orchestrator's explanation), orchestrator skill lines 56-61 (Fast Path table)

**Significance:** This is a variant of the Feb 28 intent spiral: intent doesn't survive translation when vocabulary is ambiguous. "Manually" crossed a semantic boundary — from Dylan's domain (CLI operations) to the orchestrator's domain (available tools). The orchestrator optimized within its own frame (Agent tool is available, worktrees prevent conflicts) without checking whether that frame matched Dylan's intent.

---

### Finding 3: Worktree + Docker Is Known-Bad But Ungated

**Evidence:** Worktree isolation was P0 reverted on Feb 12, 2026 during the entropy spiral. A full investigation on Feb 27, 2026 (`2026-02-27-inv-claude-code-worktree-agent-isolation.md`) documented the Docker bind mount incompatibility. Despite this documented history, the orchestrator used `isolation: "worktree"` for Docker-dependent work.

The price-watch project uses Docker Compose with bind mounts:
```yaml
volumes:
  - ./backend:/app          # Rails container
  - ./backend:/app/backend  # Sidekiq container
```

Worktree edits go to `.claude/worktrees/<name>/backend/` — a completely separate path. Docker containers see only the main repo files.

**Source:** Session transcript lines 197-226 (worktree/Docker explanation), `2026-02-27-inv-claude-code-worktree-agent-isolation.md` lines 22-40 (Feb failure documentation)

**Significance:** This is a "Gate Over Remind" principle violation. The knowledge exists as an investigation file (reminder) but not as a hook or constraint (gate). The orchestrator either didn't have this knowledge in context or didn't connect it to the decision. For Docker-dependent projects, worktree isolation is fundamentally incompatible — but nothing prevents re-attempting the pattern.

---

### Finding 4: Spawn Flag Contract Not Internalized

**Evidence:** When the orchestrator finally attempted `orch spawn`, it failed 3 times:

1. **Attempt 1** (line 244): `orch spawn feature-impl "task..."` → Hook denied: missing `--issue` and `--intent`
2. **Attempt 2** (line 266): Added `--issue` and `--intent` → CLI error: `--reason` required with `--bypass-triage`
3. **Attempt 3** (line 285): Added `--reason` → Success

Each failure revealed a different missing requirement. The orchestrator doesn't have a complete mental model of the `orch spawn` flag contract — it discovers requirements iteratively through errors.

**Source:** Session transcript lines 244-303 (three spawn attempts), `gate-spawn-context-validation.py` (hook), orch CLI source (--reason requirement)

**Significance:** The spawn command's required flags (`--issue`, `--intent`, `--reason` for bypass-triage) are documented in the hook's error messages but not consolidated in the orchestrator skill. The skill's spawning section (lines 166-206) covers model selection and spawn modes but not the full flag contract. This creates a friction cascade where each attempt discovers one more requirement.

---

## Synthesis

**Key Insights:**

1. **The Agent tool is structurally analogous to an unmonitored back door.** Every gate, hook, and safety mechanism in the orch system operates on the `orch spawn` CLI path. The Agent tool is a parallel path with zero coverage. This isn't a configuration error — it's an architectural blind spot. The hooks can only intercept tools they know about, and the Agent tool is a Claude Code built-in that no custom hook currently targets.

2. **Vocabulary precision is a structural requirement, not a stylistic preference.** The Feb 28 intent spiral showed that intent displaces across the spawn boundary. This incident shows it also displaces across the instruction boundary — "manually" crossed from Dylan's frame (CLI operation) to the orchestrator's frame (built-in tools) without either party noticing until the consequences appeared. Defined vocabulary prevents this class of misinterpretation.

3. **Known-bad patterns recur when knowledge is passive (investigation files) rather than active (gates/hooks).** The worktree + Docker incompatibility was thoroughly documented. But investigations are consulted, not enforced. The "Gate Over Remind" principle predicts this: if an agent CAN do the wrong thing, it eventually WILL — especially under time pressure or novel context (cross-project operation).

4. **Cross-project orchestration operates with thinner context.** The orchestrator was in price-watch, not orch-go. Its knowledge of orch-go-specific learnings (worktree failures, spawn patterns) was limited to what's in the orchestrator skill — which had no mention of Agent tool risks or Docker/worktree incompatibility.

**Answer to Investigation Question:**

The orchestrator used Agent tool instead of orch spawn because: (a) the orchestrator skill doesn't explicitly define Agent tool as NOT a spawn mechanism, (b) "manually" is undefined vocabulary that the orchestrator interpreted within its own frame, and (c) the Agent tool's `isolation: "worktree"` feature seemed like a reasonable solution for parallel agent file conflicts — without knowledge that Docker + worktree is incompatible.

The systemic enablers are: (1) Agent tool as ungated spawn bypass (Class 1 Filter Amnesia), (2) vocabulary gaps enabling intent displacement, (3) known-bad patterns accessible because knowledge is passive not gated, and (4) cross-project context thinning.

---

## Structured Uncertainty

**What's tested:**

- ✅ Agent tool bypasses all orch spawn infrastructure (verified: hook source only matches Bash `orch spawn`)
- ✅ Worktree edits invisible to Docker containers (verified: bind mounts use relative paths from main repo)
- ✅ Orchestrator skill has no Agent tool prohibition (verified: grep for "Agent tool" returns 0 matches)
- ✅ Spawn flag contract not documented in orchestrator skill (verified: spawning section covers model/modes, not flags)

**What's untested:**

- ⚠️ Whether a PreToolUse hook on the Agent tool would reliably fire (need to verify Claude Code hook matcher supports "Agent" tool name)
- ⚠️ Whether cross-project knowledge loss is the primary cause vs. simple omission from skill content
- ⚠️ Frequency of Agent-tool-as-spawn across other orchestrator sessions (this may be the first or nth time)

**What would change this:**

- If Agent tool hooks cannot be implemented in Claude Code, recommendation 3 (gate) is infeasible — would need skill-level prohibition only
- If other orchestrator sessions show Agent tool spawning with no issues, the problem may be worktree-specific rather than Agent-tool-as-spawn

---

## Implementation Recommendations

**Purpose:** Prevent recurrence of Agent-tool-as-spawn and worktree-for-Docker-projects patterns.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| R1: Add Agent tool prohibition to orchestrator skill | implementation | Tactical fix within existing skill content |
| R2: Add "manual spawn" vocabulary to orchestrator skill | implementation | Tactical vocabulary addition |
| R3: Gate Agent tool in orchestrator context via hook | architectural | New hook covering new tool surface — changes infrastructure |
| R4: Add spawn flag cheat sheet to orchestrator skill | implementation | Documentation within existing patterns |
| R5: Document Docker-incompatible worktree constraint | implementation | Constraint addition to existing skill content |

### Recommended Approach ⭐

**Multi-layer defense: skill prohibition + vocabulary + hook gate**

The Agent tool bypass is a Filter Amnesia instance. The fix follows the canonical prevention pattern: add the filter to the missing path. But because hooks may not support Agent tool matching, a two-layer approach is warranted.

**Why this approach:**
- Layer 1 (skill content) works immediately, requires no code changes
- Layer 2 (hook gate) provides enforcement even when skill content is ignored under pressure
- Vocabulary fix addresses the root misinterpretation, not just the symptom

**Trade-offs accepted:**
- Hook may not be technically feasible for Agent tool (Claude Code hook matcher may not support it)
- Skill-only prohibition relies on model compliance, not enforcement — but this is the same layer where all behavioral norms operate

**Implementation sequence:**

1. **R1 + R2 + R4 + R5: Orchestrator skill updates** (single PR, immediate)
   - Add to Behavioral Norms or new "Tool Boundaries" section:
     ```
     **Agent tool:** The Claude Code Agent tool is NOT a spawn mechanism.
     Never use the Agent tool to create worker agents. Always use orch spawn.
     The Agent tool bypasses all spawn infrastructure: hooks, skill loading,
     workspace creation, beads tracking, and completion pipeline.
     ```
   - Add to Fast Path table:
     ```
     | **Daemon down, need to spawn** | `orch spawn` directly (not Agent tool) | Daemon automates spawn; manual = CLI, not built-in tools |
     ```
   - Add spawn flag cheat sheet:
     ```
     ### Spawn Command Template
     orch spawn --bypass-triage --issue <ID> --intent <TYPE> --reason "<why>" <SKILL> "task"
     # --issue: beads ID (required unless --no-track)
     # --intent: build|fix|investigate|explore|produce|compare|experience
     # --reason: required with --bypass-triage (min 10 chars)
     ```
   - Add Docker/worktree constraint:
     ```
     **Worktree isolation:** Never use worktree isolation for Docker-dependent projects.
     Docker bind mounts resolve from the main repo, not worktrees — edits are invisible to containers.
     ```

2. **R3: Investigate Agent tool hook feasibility** (follow-up investigation)
   - Test whether Claude Code hook PreToolUse matcher supports "Agent" as a tool name
   - If feasible: create `gate-orchestrator-agent-tool.py` that denies Agent tool usage in orchestrator context
   - If infeasible: document limitation, rely on skill-level prohibition

### Alternative Approaches Considered

**Option B: Remove Agent tool from orchestrator entirely via --disallowedTools**
- **Pros:** Absolute prevention — impossible to use
- **Cons:** Agent tool has legitimate orchestrator uses (Explore for codebase search, research agents). Removing it cripples valid workflows.
- **When to use instead:** If Agent-tool-as-spawn recurs despite skill prohibition

**Option C: Skill-only fix (no hook investigation)**
- **Pros:** Simplest, fastest, no infrastructure work
- **Cons:** Skill-level norms are ignored under cognitive pressure (proven by this incident — the orchestrator "knew" about orch spawn but chose Agent tool anyway). Gate Over Remind says enforcement > guidance.
- **When to use instead:** If hook investigation proves Agent tool isn't hookable

**Rationale for recommendation:** The multi-layer approach follows the proven Gate Over Remind principle. Skill content provides fast coverage; hook investigation provides durable enforcement. The two layers are independent — either alone reduces risk, both together minimize it.

---

### Implementation Details

**What to implement first:**
- R1-R5 (orchestrator skill updates) — zero code, immediate deployment via `skillc deploy`
- These can be done by a feature-impl agent with skillc access

**Things to watch out for:**
- ⚠️ The orchestrator skill is policy-type (always loaded). Content additions increase token cost per turn. Keep additions minimal.
- ⚠️ Agent tool hook feasibility is genuinely unknown — the Claude Code hook system may not expose this tool to PreToolUse matchers
- ⚠️ The "worktree + Docker" constraint applies to ANY Docker-dependent project, not just price-watch. The constraint should be general.

**Areas needing further investigation:**
- Claude Code hook system: which tool names are hookable via PreToolUse matcher?
- Frequency analysis: has Agent-tool-as-spawn occurred in other sessions? (`grep -r "Agent tool" ~/.orch/events.jsonl` or session transcripts)
- Whether `--disallowedTools Agent` is a valid Claude Code flag for orchestrator spawns

**Success criteria:**
- ✅ Orchestrator skill explicitly prohibits Agent tool as spawn mechanism
- ✅ "Manual spawn" vocabulary defined in skill
- ✅ Spawn flag template available in skill
- ✅ Docker/worktree constraint documented
- ✅ (Stretch) Hook gates Agent tool in orchestrator context

---

## Defect Class Analysis

| Class | Applies? | Manifestation | Prevention |
|-------|----------|---------------|------------|
| **Class 1: Filter Amnesia** | ✅ PRIMARY | Spawn safety filters on orch spawn path, absent on Agent tool path | Add filter to Agent tool path (hook or skill prohibition) |
| **Class 4: Cross-Project Boundary Bleed** | ✅ CONTRIBUTING | Knowledge about worktree failures in orch-go not available in price-watch context | Worktree constraint in orchestrator skill (project-agnostic) |
| **Class 5: Contradictory Authority Signals** | ✅ CONTRIBUTING | Agent tool offers worktree isolation; prior knowledge says worktrees are broken with Docker | Single authority: skill explicitly prohibits worktree for Docker projects |

---

## Connection to Intent Spiral (Feb 28)

This incident shares the same root mechanism as the Feb 28 intent spiral: **intent displacement across a translation boundary**.

| Feb 28 | Mar 5 |
|--------|-------|
| "Evaluate" (experience tool) → "audit" (produce findings) | "Manually" (use CLI) → "manually" (use built-in tools) |
| Displacement at: orchestrator → skill selection | Displacement at: Dylan → orchestrator instruction parsing |
| Amplified by: skill gravity overriding intent | Amplified by: Agent tool offering convenient features (worktree) |
| Fix: intent declaration at spawn boundary (--intent flag) | Fix: vocabulary precision at instruction boundary |

The spawn hook added after Feb 28 (`gate-spawn-context-validation.py`) prevents intent displacement across the spawn boundary — but only on the `orch spawn` path. The Agent tool is a new spawn boundary that the hook doesn't cover.

---

## References

**Files Examined:**
- `price-watch/2026-03-05-122852-pw-orch-session-for-pm.txt` — Full session transcript (327 lines)
- `~/.claude/skills/meta/orchestrator/SKILL.md` — Orchestrator skill (437 lines)
- `~/.orch/hooks/gate-spawn-context-validation.py` — Spawn intent validation hook (163 lines)
- `~/.orch/hooks/gate-orchestrator-code-access.py` — Code access coaching hook (125 lines)
- `.kb/investigations/2026-02-27-inv-claude-code-worktree-agent-isolation.md` — Worktree investigation
- `.kb/investigations/2026-02-28-investigation-orchestrator-intent-spiral.md` — Intent spiral analysis

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-28-investigation-orchestrator-intent-spiral.md` — Same root mechanism (intent displacement)
- **Investigation:** `.kb/investigations/2026-02-27-inv-claude-code-worktree-agent-isolation.md` — Worktree + Docker known-bad
- **Model:** `.kb/models/defect-class-taxonomy/model.md` — Class 1 (Filter Amnesia) is the primary defect class

---

## Investigation History

**2026-03-05 12:40:** Investigation started
- Initial question: Why did orchestrator use Agent tool instead of orch spawn?
- Context: Post-mortem requested by Dylan after price-watch orchestration session

**2026-03-05 13:00:** Full context gathered
- Read session transcript, orchestrator skill, spawn hooks, worktree investigation, intent spiral investigation
- Identified 4 root causes and 3 systemic patterns

**2026-03-05 13:15:** Investigation completed
- Status: Complete
- Key outcome: Agent tool is an ungated shadow spawn path; vocabulary gap enabled misinterpretation; known-bad pattern recurred because knowledge is passive not gated
