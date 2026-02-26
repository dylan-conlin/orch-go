<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Orchestration session hit context limit prematurely because 67% of the 56K-token context was consumed by startup content before any work began - primarily the embedded orchestrator skill (2,385 lines / ~28K tokens).

**Evidence:** Analyzed transcript showing lines 1-2,840 were setup context; `orch status` output alone was 437 lines showing 182 agents; actual user conversation was only ~70 lines before export.

**Knowledge:** Skills designed for comprehensive reference become context hogs when embedded in spawn prompts; tool outputs listing 100+ items need pagination/limits by default.

**Next:** Implement skill size reduction (core vs full variants) and add `--compact`/`--limit` flags to `orch status`.

**Promote to Decision:** recommend-yes - This reveals a systemic constraint: skill content + tool output must fit within context budget alongside actual work.

---

# Investigation: Analyze Orchestration Session Hit Context

**Question:** Why did the orchestration session hit context limit prematurely, and what consumed the most context?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Worker agent (spawned)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Embedded Orchestrator Skill = 2,385 Lines (~50% of Setup)

**Evidence:** The orchestrator skill content spans lines 315-2700 of the transcript - approximately 2,385 lines. At ~12 tokens/line average, this is ~28,600 tokens, representing the single largest context consumer.

**Source:**
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/oddly-short-orch-session-full.txt:315-2700`
- Skill contains: delegation rules, synthesis protocols, daemon workflow, strategic alignment, models architecture, command reference, etc.

**Significance:** The orchestrator skill is comprehensive (designed as a reference document) but when embedded verbatim in spawn context, it consumes half the available context before work begins. Skills optimized for reference are not optimized for embedding.

---

### Finding 2: `orch status` Output = 437 Lines (182 Agents Listed)

**Evidence:** A single `orch status` command produced 437 lines of output (lines 2846-3283), listing 182 agents with full details including beads ID, mode, model, status, phase, task, skill, runtime, tokens, and risk flags.

**Source:**
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/oddly-short-orch-session-full.txt:2846-3283`
- Plus 26 orchestrator sessions listed before the agent table

**Significance:** Tool outputs with unbounded listings become context consumers. The orchestrator only needed to see active/running agents (~11), not all 182. Default output should be compact; full listing should require `--all` flag.

---

### Finding 3: Command Help Repetition = 3x ~30 Lines Each

**Evidence:** When `orch complete` failed, the full help text was printed 3 times (lines 4428-4442, 4460-4475, 4506-4541) as the orchestrator tried different flags. Each failure printed the same ~30 lines of flag documentation.

**Source:**
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/oddly-short-orch-session-full.txt:4428-4541`

**Significance:** CLI tools printing full help on errors compound context consumption. A single error can add 90+ lines of repetitive content.

---

### Finding 4: Startup Hooks + Session Handoff = 220 Lines

**Evidence:** Lines 1-220 contain beads workflow context, session handoff from prior session (126 lines with full epic description), and hook success messages.

**Source:**
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/oddly-short-orch-session-full.txt:1-220`

**Significance:** Session handoffs designed for human readability become context overhead when injected as startup hooks. A handoff that helps a human context-switch becomes bloat for a fresh Claude session.

---

### Finding 5: Actual Work = ~1,362 Lines (~24% of Total)

**Evidence:** After all setup (lines 1-2840) and the first `orch status` call (lines 2846-3283), actual work content was lines 3300-4662 (~1,362 lines). But much of this was tool output, not user turns. The user made only ~4 requests before exporting.

**Source:**
- User turns: lines 4586 ("ok, so are we all synced up"), 4657 ("/export")
- Total transcript: 4,662 lines at 56,414 tokens

**Significance:** Only ~24% of context was available for actual work. With 67% consumed by setup and 9% by the first status check, the session was destined to be short.

---

## Synthesis

**Key Insights:**

1. **Skill Design Trade-off** - The orchestrator skill was designed as comprehensive reference documentation (~2,385 lines). This is excellent for reference but catastrophic for embedding. Skills need "core" (embeddable) and "full" (reference) variants.

2. **Tool Output Unboundedness** - `orch status` and CLI help output have no limits. When 182 agents exist, listing them all by default wastes context. Pagination/limits should be defaults, not opt-in.

3. **Startup Context Accumulation** - Multiple systems inject content at startup (beads hooks, session handoffs, kb context, skill embedding, project list). Each seems reasonable alone; together they consume 67% of context.

**Answer to Investigation Question:**

The session hit context limit prematurely because **67% of the 56,414-token context was consumed before any work began**:
- Orchestrator skill embedding: ~50% of setup (~28K tokens)
- `orch status` output: ~9% of total (~5K tokens)
- Startup hooks + handoff: ~5% of total (~3K tokens)

The culprit is primarily **skill content** (not spawn context structure, not conversation bloat). The orchestrator skill is designed as comprehensive documentation but embedded verbatim, making spawned orchestrator sessions context-starved from the start.

---

## Structured Uncertainty

**What's tested:**

- ✅ Orchestrator skill is ~2,385 lines (verified: read transcript lines 315-2700)
- ✅ `orch status` outputs 182 agents in one call (verified: read transcript lines 2846-3283)
- ✅ Total transcript is 56,414 tokens, 4,662 lines (verified: file read error showed token count)

**What's untested:**

- ⚠️ Token-per-line estimate (~12 tokens/line) is approximate
- ⚠️ Other orchestrator skill variants may be smaller (not checked)
- ⚠️ Impact of reducing skill size on orchestrator effectiveness (not measured)

**What would change this:**

- Finding would be wrong if skill content is actively used/referenced during session (then it's not waste)
- Finding would be wrong if context limit wasn't actually the cause of session brevity

---

## Implementation Recommendations

### Recommended Approach: Tiered Skill Content

**Create "core" and "reference" skill variants** - Embed minimal core guidance (500-800 lines) in spawn context; keep full reference (2,385 lines) as linked documentation.

**Why this approach:**
- Reduces spawn context by ~1,600 lines (~19K tokens)
- Leaves more context for actual work (~50% improvement)
- Core contains: delegation rules, pre-response gates, command syntax
- Reference contains: detailed protocols, rationale, edge cases

**Trade-offs accepted:**
- Orchestrator may need to reference external docs for edge cases
- Requires maintaining two versions (or generating core from full)

**Implementation sequence:**
1. Identify which orchestrator skill sections are used in every session (core)
2. Extract detailed protocols/rationale to reference file
3. Update skillc to support `--tier core|full` embedding option

### Alternative Approaches Considered

**Option B: Pagination in `orch status`**
- **Pros:** Immediate win, no skill changes needed
- **Cons:** Addresses symptom (one command) not root cause (skill size)
- **When to use instead:** As a complementary fix alongside skill tiering

**Option C: Lazy skill loading via references**
- **Pros:** Maximum flexibility, load sections on-demand
- **Cons:** Complex implementation, may cause mid-session friction
- **When to use instead:** If tiered approach proves insufficient

**Rationale for recommendation:** Skill content is 50% of setup context. Even with perfect tool output limits, the skill alone would consume 50% of context. Must address the largest contributor first.

---

### Implementation Details

**What to implement first:**
- Add `--compact` flag to `orch status` (show only running/recent agents)
- Make compact the default; `--all` for full listing

**Things to watch out for:**
- ⚠️ "Core" skill may be too minimal, causing orchestrator confusion
- ⚠️ Splitting skill creates maintenance burden
- ⚠️ Tool output limits may hide important information

**Areas needing further investigation:**
- What's the actual token budget for a productive orchestrator session?
- Which skill sections are referenced vs ignored during typical sessions?
- Can we auto-generate core from full based on usage patterns?

**Success criteria:**
- ✅ Orchestrator sessions can complete 10+ spawns before context limit
- ✅ Setup context < 30% of total (vs current 67%)
- ✅ `orch status` output < 50 lines by default

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/oddly-short-orch-session-full.txt` - Full transcript (56,414 tokens, 4,662 lines)
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/oddly-short-orchestration-session.txt` - Summary transcript (47,349 tokens, 3,481 lines)

**Commands Run:**
```bash
# Get line counts
wc -l oddly-short-orch-session-full.txt oddly-short-orchestration-session.txt

# Search for context-related patterns
grep -n "context limit\|compaction\|SKILL.md\|truncated" oddly-short-orch-session-full.txt
```

**Related Artifacts:**
- **Decision:** (recommended) Create skill tiering decision after orchestrator review
- **Investigation:** This is a novel finding - no prior investigations on skill context consumption

---

## Investigation History

**2026-01-16 11:30:** Investigation started
- Initial question: Why did orchestration session hit context limit prematurely?
- Context: Dylan reported oddly short session, provided two transcript files

**2026-01-16 11:45:** Major finding - skill content dominates
- Discovered orchestrator skill is 2,385 lines, 50% of setup context
- Identified `orch status` output as secondary contributor (437 lines)

**2026-01-16 12:00:** Investigation completed
- Status: Complete
- Key outcome: 67% of context consumed by setup; skill embedding is primary culprit; recommend tiered skill approach
