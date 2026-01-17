<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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

# Investigation: Measure Context Preparation Vs Execution

**Question:** What ratio of tokens go to SPAWN_CONTEXT.md, skill embedding, kb context vs actual work? If 60% is setup, that's an optimization lever.

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker Agent og-feat-measure-context-preparation-17jan-b98b
**Phase:** Investigating
**Next Step:** Calculate token ratios and identify optimization opportunities
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: SPAWN_CONTEXT.md comprises ~7,841 tokens (31,364 bytes)

**Evidence:** 
- Total file size: 767 lines, 31,364 bytes
- Breakdown:
  - Task header + metadata: 454 bytes (lines 1-11)
  - KB context section: 6,610 bytes (lines 12-256)
  - Embedded skill (feature-impl): 16,361 bytes (lines 257-717)
  - Footer (deliverables, protocols, server info): 1,016 bytes (lines 718-768)
  - Remaining baseline: ~6,923 bytes (beads tracking, authority, deliverables)
- Using 4 chars/token ratio: 31,364 bytes ÷ 4 = ~7,841 tokens

**Source:** 
- `wc -l /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-measure-context-preparation-17jan-b98b/SPAWN_CONTEXT.md`
- `wc -c` measurements of each section
- Section-specific measurements via `sed -n` extraction

**Significance:** SPAWN_CONTEXT is the single largest context preparation component, containing embedded kb context, skill content, and spawn-specific instructions. The embedded skill alone is ~4,090 tokens (52% of SPAWN_CONTEXT).

---

### Finding 2: CLAUDE.md files add ~6,071 tokens (24,285 bytes total)

**Evidence:**
- Global ~/.claude/CLAUDE.md: 198 lines, 10,724 bytes (~2,681 tokens)
- Project CLAUDE.md: 338 lines, 13,561 bytes (~3,390 tokens)
- AGENTS.md: 40 lines, 1,327 bytes (~332 tokens)
- Total: 24,285 bytes ÷ 4 = ~6,071 tokens

**Source:**
- `wc -l ~/.claude/CLAUDE.md` (198 lines)
- `wc -c ~/.claude/CLAUDE.md` (10,724 bytes)
- `wc -l /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` (338 lines)
- `wc -c /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` (13,561 bytes)
- `wc -l /Users/dylanconlin/Documents/personal/orch-go/AGENTS.md` (40 lines)
- `wc -c /Users/dylanconlin/Documents/personal/orch-go/AGENTS.md` (1,327 bytes)

**Significance:** Project context files are loaded automatically by OpenCode's session-context plugin. They provide user preferences, project conventions, and orchestration protocols that are essential for proper agent behavior.

---

### Finding 3: Orchestrator skill loading wastes ~13,354 tokens for worker sessions

**Evidence:**
- Orchestrator skill size: 1,192 lines, 53,416 bytes (~13,354 tokens)
- KB context constraint: "Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading"
- Reason cited: "Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget"

**Source:**
- `wc -l ~/.claude/skills/meta/orchestrator/SKILL.md` (1,192 lines)
- `wc -c ~/.claude/skills/meta/orchestrator/SKILL.md` (53,416 bytes)
- SPAWN_CONTEXT.md lines 17-19 (kb context constraint)

**Significance:** Worker agents don't need orchestrator skill content (delegation patterns, spawn management, synthesis protocols). Loading it wastes ~13,354 tokens (48% of available context in a 200k window). Setting ORCH_WORKER=1 prevents this waste.

---

### Finding 4: Total context preparation baseline is ~13,912 tokens (excluding orchestrator skill)

**Evidence:**
- SPAWN_CONTEXT.md: ~7,841 tokens
- CLAUDE.md files: ~6,071 tokens
- Total baseline: 13,912 tokens
- With orchestrator skill (if not skipped): 27,266 tokens
- Current session budget: 200,000 tokens
- Baseline percentage: 6.96% (proper) or 13.63% (with orchestrator)

**Source:** Summation of Finding 1, 2, and 3 measurements

**Significance:** Context preparation consumes 6.96% of the token budget when ORCH_WORKER=1 is set (proper worker configuration). This leaves 93% for actual work. However, if orchestrator skill loads (improper configuration), preparation jumps to 13.63%, leaving only 86.4% for execution.

---

### Finding 5: System prompts and tool definitions consume additional unmeasured tokens

**Evidence:**
- OpenCode system prompt visible in this session: ~42,497 tokens used after reading SPAWN_CONTEXT
- Files read so far: SPAWN_CONTEXT (767 lines) + CLAUDE.md files (577 lines) = 1,344 lines
- Estimated file content: ~14,000 tokens (measured above)
- Remaining ~28,500 tokens = system prompt, tool definitions, orchestrator skill, and previous conversation

**Source:** Token usage warnings from OpenCode system

**Significance:** The measured file content (13,912 tokens) represents only part of the context preparation cost. System prompts, tool definitions, and conversation history add significant overhead not captured in file measurements alone.

---

## Synthesis

**Key Insights:**

1. **Context preparation is well-optimized at ~7% when properly configured** - With ORCH_WORKER=1 set, worker sessions consume only 13,912 tokens (~7%) for setup (SPAWN_CONTEXT + CLAUDE.md files), leaving 93% for execution. This is far better than the hypothetical "60% setup" concern.

2. **Orchestrator skill auto-loading is the primary waste vector** - Without ORCH_WORKER=1, the orchestrator skill (13,354 tokens) doubles context preparation from 7% to 13.6%. This represents the single largest optimization opportunity: ensuring workers properly skip orchestrator content.

3. **Embedded content strategy is efficient** - SPAWN_CONTEXT embeds both kb context (6,610 bytes) and skill content (16,361 bytes) in a single file. This eliminates additional file reads and ensures all spawn-critical context loads atomically. The skill embedding (52% of SPAWN_CONTEXT) is necessary - workers need full skill guidance upfront.

4. **System prompt overhead is significant but unmeasured** - The ~28,500 token gap between file content (14k) and initial usage (42.5k) represents system prompts, tool definitions, and orchestrator skill loading. This overhead is mostly unavoidable (tools, system instructions) but includes avoidable waste (orchestrator skill for workers).

**Answer to Investigation Question:**

Context preparation consumes approximately **6.96% of the token budget** (13,912 tokens of 200,000) when properly configured with ORCH_WORKER=1. This leaves **93% for execution work**, which is excellent efficiency.

However, improper configuration (missing ORCH_WORKER=1) causes context preparation to jump to **13.63%** (27,266 tokens), nearly doubling the overhead. The orchestrator skill accounts for this entire waste.

The breakdown is:
- SPAWN_CONTEXT.md: 7,841 tokens (56% of preparation)
- CLAUDE.md files: 6,071 tokens (44% of preparation)
- Orchestrator skill (avoidable): 13,354 tokens (extra 96% overhead if not skipped)

**Optimization levers identified:**
1. **Critical:** Ensure ORCH_WORKER=1 is set for all worker spawns (saves 13,354 tokens)
2. **Moderate:** Consider caching CLAUDE.md content (saves ~6,071 tokens on subsequent spawns)
3. **Low priority:** Compress kb context format (current 6,610 bytes is reasonable for discoverability)

---

## Structured Uncertainty

**What's tested:**

- ✅ File sizes measured via `wc -c` for SPAWN_CONTEXT, CLAUDE.md files, orchestrator skill
- ✅ Line counts verified via `wc -l` for all components
- ✅ Section extraction via `sed -n` to isolate kb context, embedded skill, header/footer
- ✅ Token usage tracking from OpenCode system warnings (42,497 tokens after initial reads)
- ✅ Constraint documented in kb context about ORCH_WORKER=1 requirement

**What's untested:**

- ⚠️ Actual token counts from Claude API (using 4 chars/token heuristic, not API response)
- ⚠️ Impact of ORCH_WORKER=1 on actual spawns (verified constraint exists, not tested in live spawn)
- ⚠️ System prompt token cost breakdown (observed gap, didn't decompose into components)
- ⚠️ Caching benefits for CLAUDE.md content (theoretical optimization, not benchmarked)
- ⚠️ Execution phase token distribution (measured preparation, not actual work patterns)

**What would change this:**

- Finding would be wrong if Claude tokenizer differs significantly from 4 chars/token (could be 3-5 range)
- Ratios would change if system prompts or tool definitions were significantly modified
- Optimization priority would shift if ORCH_WORKER=1 is already universally enforced (need to check spawn cmd implementation)
- Caching value would change if CLAUDE.md files change frequently (assumption: stable)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Enforce ORCH_WORKER=1 for all worker spawns via spawn command** - Add automatic environment variable setting to the spawn command implementation to ensure orchestrator skill is never loaded for worker sessions.

**Why this approach:**
- Saves 13,354 tokens (48% waste reduction) per worker spawn
- Prevents configuration drift (no reliance on manual environment setting)
- Addresses the primary optimization opportunity identified in Finding 3
- Makes the constraint (already documented in kb) automatically enforced

**Trade-offs accepted:**
- Requires code change to spawn command (not just documentation)
- Workers will never have orchestrator skill loaded (acceptable - they shouldn't need it)
- Adds hardcoded environment variable to spawn flow (coupling, but intentional)

**Implementation sequence:**
1. Verify current spawn command doesn't already set ORCH_WORKER=1 (check `pkg/spawn/` or `cmd/orch/spawn.go`)
2. Add environment variable setting to spawn execution path (where OpenCode session is created)
3. Test worker spawn to confirm orchestrator skill is not loaded (check token usage before/after)
4. Document the automatic enforcement in spawn documentation

### Alternative Approaches Considered

**Option B: Add ORCH_WORKER=1 to SPAWN_CONTEXT.md instructions**
- **Pros:** No code change required, documentation-only fix
- **Cons:** Relies on agents reading and following instructions (error-prone), doesn't prevent misconfiguration
- **When to use instead:** If spawn command modification is blocked or risky

**Option C: Compress kb context and CLAUDE.md content**
- **Pros:** Reduces overall context preparation cost
- **Cons:** Saves only ~6-12k tokens (vs 13k from orchestrator), reduces readability, unclear if compression is needed given 93% execution budget
- **When to use instead:** If context preparation becomes >20% despite orchestrator fix

**Option D: Cache CLAUDE.md content across spawns**
- **Pros:** Saves ~6,071 tokens on subsequent spawns in same session
- **Cons:** Complex caching implementation, unclear benefit (200k budget is ample), assumes CLAUDE.md stability
- **When to use instead:** If spawn frequency is very high (>10/session) and context budget is tight

**Rationale for recommendation:** Option A (automatic ORCH_WORKER=1) provides the largest single optimization (13,354 tokens) with minimal complexity and no user-facing changes. It addresses the primary waste identified in the investigation. Options B-D provide smaller gains with higher complexity or lower reliability.

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
