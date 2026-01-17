<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Five platform-level bugs caused orchestrator session ses_4325 to fail, driving Dylan to 'wits end' - the primary issues are Task tool confusion (agent used Task instead of orch spawn), unhelpful CLI error messages, and orchestrator skill verbosity diluting critical guidance.

**Evidence:** Session log shows 15+ user corrections, explicit frustration ("WHAT ARE YOU DOING"), and repeated cycles of the same mistakes. Orchestrator skill is 1,193 lines with delegation rule buried at line 373.

**Knowledge:** Model capability + skill template design are tightly coupled - verbose templates fail on smaller models like Gemini Flash; critical rules must be at the TOP, not buried.

**Next:** Implement 3 structural fixes: (1) Add CLI argument validation with helpful examples, (2) Create orchestrator skill "quick reference" header with top 10 rules, (3) Add model-skill compatibility check before spawn.

**Promote to Decision:** recommend-yes - This establishes the principle that skill templates must be model-aware and critical guidance must appear in first 100 lines.

---

# Investigation: Deep Analysis Post-Mortem - Orchestrator Session ses_4325 Failure

**Question:** What platform-level bugs in orch-go caused the orchestrator failure in session ses_4325, and what structural fixes prevent recurrence?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - implement recommendations
**Status:** Complete

<!-- Lineage -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Task Tool vs orch spawn Confusion

**Evidence:** The orchestrator (Gemini Flash) repeatedly used the `Task` tool instead of `orch spawn` CLI command. Dylan had to abort tool calls and explicitly correct the agent multiple times:
- "you need to spawn agents using orch spawn"
- "WHAT ARE YOU DOING. i said to use orch spawn"
- "you still are thinking like a worker"

The session log shows at least 5 instances of the agent trying to use the Task tool when it should have used `orch spawn`.

**Source:** Session log lines showing:
```
Tool: Task (aborted)
...
Dylan: use orch spawn please
```

**Significance:** The orchestrator skill explicitly states at line 428-443:
> "⚠️ Use `orch spawn`, NOT the Task tool"

But this guidance is buried 428 lines into a 1,193 line document. Gemini Flash (smaller context quality than Opus) failed to maintain this instruction.

---

### Finding 2: CLI Argument Validation Error Messages Are Unhelpful

**Evidence:** When the orchestrator tried `orch spawn architect <<'EOF'...` (wrong syntax), it received:
```
Error: requires at least 2 arg(s), only received 1
```

This error message doesn't tell the user:
1. What the correct syntax is
2. What the 2 required arguments are
3. An example of proper usage

**Source:** `cmd/orch/spawn_cmd.go:160` uses `cobra.MinimumNArgs(2)` which produces a generic error.

**Significance:** An agent struggling with CLI syntax gets no help from the error message. A better error would be:
```
Error: spawn requires SKILL and TASK arguments
Usage: orch spawn <skill> "task description"
Example: orch spawn architect "design auth system"
```

---

### Finding 3: Orchestrator Skill Template is Too Verbose (1,193 lines)

**Evidence:** The orchestrator skill at `~/.claude/skills/meta/orchestrator/SKILL.md` is 1,193 lines. Key guidance is distributed throughout:
- "Pre-Response Gates" at line 56-66
- "Context Detection" at line 70-84
- "ABSOLUTE DELEGATION RULE" at line 373-489
- "Orchestrator Autonomy" at line 514-533
- "Skill Selection Guide" at line 759-818

**Source:** `~/.claude/skills/meta/orchestrator/SKILL.md` (1,193 lines)

**Significance:** Smaller models like Gemini Flash cannot maintain attention to rules buried hundreds of lines deep. The "Fast Path (Surface Table)" at line 36-52 is the right idea but doesn't include the most critical rule: "Use orch spawn, NOT Task tool."

---

### Finding 4: Model-Skill Mismatch (Gemini Flash for Orchestration)

**Evidence:** The session used Gemini Flash (`gemini-3-flash-preview`) for orchestration work. The orchestrator skill was designed with Claude Opus in mind - the quality of instruction following, context retention, and judgment expected in the skill template exceeds Flash's capabilities.

**Source:** Session was on price-watch project which may have had different model defaults. The orchestrator skill has no model compatibility guidance.

**Significance:** The skill template assumes a high-capability model that can:
- Retain complex multi-step rules over 1000+ lines
- Exercise judgment about when to delegate
- Catch subtle delegation boundaries

Gemini Flash repeatedly fell into "worker mode" despite explicit role instructions.

---

### Finding 5: Concurrency Limit Counted Stale Sessions as Active

**Evidence:** The orchestrator hit a concurrency error showing "265 idle agents" blocking spawns. The error message was:
```
9 active agents (max 5)
```

But `orch status` showed all agents were "idle, Complete" - not truly active.

**Source:** `cmd/orch/spawn_cmd.go:445-475` - the concurrency check uses a 10-minute threshold for "running" vs "idle", but completed agents that haven't been cleaned up still count against the limit.

**Significance:** Ghost agents from previous sessions block new work. The concurrency check should exclude agents with Phase: Complete (which it does at line 467-468), but the threshold logic may be counting too many sessions as active.

---

## Synthesis

**Key Insights:**

1. **Skill Template Verbosity is Model-Dependent** - A 1,193 line skill template may work for Opus but fails catastrophically for Flash. Critical guidance must appear in the first 100 lines, not buried at line 373+.

2. **CLI Error Messages are User Experience** - When an agent (or human) struggles with syntax, the error message is their only help. Generic "requires N args" messages provide zero actionable guidance.

3. **Delegation Rules Need Structural Enforcement** - The "Task tool vs orch spawn" confusion keeps recurring because it's guidance, not a gate. Consider adding validation that warns when an orchestrator-type session uses Task tool for spawning.

4. **Model-Skill Compatibility is a First-Class Concern** - Skills designed for high-capability models will fail on smaller models. The spawn system should warn or block incompatible model-skill combinations.

**Answer to Investigation Question:**

The orchestrator failure was caused by five platform-level bugs working together:

1. **Task tool confusion** - Guidance exists but is buried and not enforced
2. **CLI error messages** - Unhelpful syntax errors compound confusion
3. **Skill verbosity** - 1,193 lines dilutes critical rules for smaller models
4. **Model mismatch** - Flash can't follow complex orchestrator skill
5. **Stale session counting** - Ghost agents block new spawns

The root cause is that the platform assumes high-capability models and doesn't degrade gracefully. Fixes must address both the immediate usability issues (better error messages, skill restructuring) and the systemic model-skill compatibility problem.

---

## Structured Uncertainty

**What's tested:**

- ✅ Orchestrator skill is 1,193 lines (verified: read file, counted lines)
- ✅ Delegation rule is at line 373 (verified: grep search)
- ✅ CLI uses cobra.MinimumNArgs(2) (verified: read spawn_cmd.go:160)
- ✅ Concurrency check threshold is 10 minutes (verified: read spawn_cmd.go:449)

**What's untested:**

- ⚠️ Whether restructuring skill template with top-loaded rules improves Flash behavior (not tested)
- ⚠️ Whether better CLI errors reduce agent confusion (not tested)
- ⚠️ Whether the 265 idle agents were truly blocking or a separate issue (session log incomplete)

**What would change this:**

- If Flash with restructured skill performs well, the model-skill compatibility check may be unnecessary
- If Opus also falls into Task tool confusion, the problem is template design not model capability
- If the concurrency issue was transient (OpenCode restart), the counting logic may be correct

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: Layered Fix

**Three-Phase Implementation** - Address immediate usability, then structural issues, then systemic model compatibility.

**Why this approach:**
- Phase 1 (CLI errors) provides immediate improvement with low risk
- Phase 2 (skill restructure) addresses the primary failure mode
- Phase 3 (model compatibility) prevents future class of failures

**Trade-offs accepted:**
- Not implementing a hard gate on Task tool usage (would break legitimate uses)
- Not fully redesigning skill loading (would be high risk)

**Implementation sequence:**

1. **Improve CLI error messages** (spawn_cmd.go)
   - Add custom Args function with helpful error including example
   - Applies to all CLI users, not just agents

2. **Restructure orchestrator skill template**
   - Add "CRITICAL RULES" section at top (first 50 lines)
   - Include: "Use orch spawn NOT Task tool", delegation boundary, role detection
   - Keep detailed reference sections below

3. **Add model-skill compatibility check**
   - Create skill metadata field: `min-capability: opus | sonnet | any`
   - Warn (not block) when spawning orchestrator skill with non-Opus model
   - Document the capability requirements

### Alternative Approaches Considered

**Option B: Hard gate on Task tool in orchestrator context**
- **Pros:** Prevents the confusion entirely
- **Cons:** Task tool has legitimate uses (quick research tasks); would require complex context detection
- **When to use instead:** If the soft guidance continues to fail after skill restructure

**Option C: Create separate "lite" orchestrator skill for Flash**
- **Pros:** Optimized for smaller models
- **Cons:** Maintenance burden of two skills; capability gap may be too large for any Flash orchestrator
- **When to use instead:** If orchestration on Flash is a hard requirement

**Rationale for recommendation:** The primary issue is guidance placement, not capability - even Opus would struggle with rules buried at line 373. The restructure is the highest-leverage fix.

---

### Implementation Details

**What to implement first:**
1. CLI error message improvement - lowest risk, immediate value
2. Skill template restructure - highest leverage for the failure mode
3. Model compatibility check - prevents future failures

**Things to watch out for:**
- ⚠️ Skill template changes require skillc recompile and redeploy
- ⚠️ CLI changes require `make install` to take effect
- ⚠️ Model compatibility check may need capability testing to calibrate thresholds

**Areas needing further investigation:**
- Why did the session use Flash instead of Opus for orchestration?
- Are there other skills with similar verbosity problems?
- Is the concurrency counting bug a separate issue or related to session cleanup?

**Success criteria:**
- ✅ CLI error shows example usage when args are wrong
- ✅ Orchestrator skill has critical rules in first 50 lines
- ✅ Spawn with Flash model for orchestrator skill shows warning
- ✅ No "Task tool vs orch spawn" confusion in next 10 orchestrator sessions

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/session-ses_4325.md` - Full session log showing failure pattern
- `~/.claude/skills/meta/orchestrator/SKILL.md` - 1,193 line orchestrator skill template
- `cmd/orch/spawn_cmd.go` - CLI argument validation and concurrency checking
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template generation

**Commands Run:**
```bash
# Count lines in orchestrator skill
wc -l ~/.claude/skills/meta/orchestrator/SKILL.md

# Search for delegation rule location
grep -n "ABSOLUTE DELEGATION" ~/.claude/skills/meta/orchestrator/SKILL.md

# Search for Task tool warning
grep -n "Task tool" ~/.claude/skills/meta/orchestrator/SKILL.md
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md` - Prior post-mortem with similar themes
- **Decision:** `.kb/decisions/2025-12-04-orchestrator-delegates-all-investigations.md` - Established delegation rule

---

## Investigation History

**2026-01-17 14:30:** Investigation started
- Initial question: What platform bugs caused session ses_4325 to fail?
- Context: Spawned by orchestrator to analyze root cause of "wits end" frustration

**2026-01-17 15:00:** Analyzed session log
- Found 5 distinct failure patterns
- Identified Task tool confusion as primary issue

**2026-01-17 15:30:** Examined skill template and CLI code
- Confirmed 1,193 line skill with buried rules
- Confirmed unhelpful CLI error messages

**2026-01-17 16:00:** Investigation completed
- Status: Complete
- Key outcome: 5 platform bugs identified with 3-phase fix recommended
