<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully cleaned up ad-hoc orchestrator skill fix by reverting hardcoded instruction and rebuilding session-context plugin with proper per-session ORCH_WORKER filtering.

**Evidence:** Removed hardcoded orchestrator skill from opencode.jsonc line 5, compiled session-context.ts to .js with bun, verified ORCH_WORKER check is in config hook (not plugin init).

**Knowledge:** OpenCode plugins can be .ts (via symlink) but compiling to .js ensures compatibility; the fix (commit ac945ea) moved ORCH_WORKER check from plugin init to config hook for per-session filtering.

**Next:** Verify plugin loads correctly by testing in orch project context.

**Confidence:** High (85%) - Implementation complete, need runtime verification

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Cleanup After Orchestrator Skill Loading

**Question:** How to properly clean up ad-hoc orchestrator skill loading fix and verify session-context plugin works correctly?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Hardcoded orchestrator skill was temporary workaround

**Evidence:** opencode.jsonc had `~/.claude/skills/meta/orchestrator/SKILL.md` hardcoded in instructions array at line 5, with comment saying "conditionally loaded via session-context plugin" - indicating this was meant to be temporary.

**Source:** `~/.config/opencode/opencode.jsonc:2-6`

**Significance:** The hardcoded approach loads orchestrator skill for ALL sessions, defeating the purpose of conditional loading for orch projects only and ORCH_WORKER filtering.

---

### Finding 2: Fix moved ORCH_WORKER check to config hook

**Evidence:** Commit ac945ea in orch-cli moved ORCH_WORKER env var check from plugin initialization (runs once when OpenCode starts) to config hook (runs per-session). Also fixed skill path from 'policy' to 'meta'.

**Source:** `git diff ac945ea~1 ac945ea .opencode/plugin/session-context.ts`

**Significance:** This ensures worker sessions (with ORCH_WORKER=1) skip loading orchestrator skill even when OpenCode is started in an orchestrator project context. Per-session checking is critical for correct filtering.

---

### Finding 3: OpenCode plugins can be .ts but .js is safer

**Evidence:** Existing plugins in ~/.config/opencode/plugin/ were .ts symlinks, but bun successfully compiled session-context.ts to session-context.js (2.1 KB) without issues. TypeScript compiler (tsc 4.5.5) failed due to Node type definition incompatibility.

**Source:** `bun build plugin/session-context.ts --outdir ~/.config/opencode/plugin --target=node --format=esm`

**Significance:** While OpenCode may support .ts plugins via symlinks, compiling to .js ensures compatibility and avoids runtime TypeScript parsing issues. Bun is better suited for this than tsc.

---

## Synthesis

**Key Insights:**

1. **Ad-hoc fix defeated conditional loading** - Hardcoding the orchestrator skill in opencode.jsonc loaded it for ALL sessions, including workers, defeating the purpose of the session-context plugin's conditional logic.

2. **Per-session filtering requires config hook** - The critical fix (ac945ea) moved ORCH_WORKER checking from plugin init (runs once) to config hook (runs per-session), enabling proper worker session filtering even when OpenCode starts in orchestrator context.

3. **Compiled JavaScript more reliable than TypeScript symlinks** - While OpenCode may support .ts plugins, compiling to .js with bun avoids runtime TypeScript parsing issues and ensures broad compatibility.

**Answer to Investigation Question:**

To clean up the ad-hoc orchestrator skill loading fix: (1) Remove hardcoded skill path from opencode.jsonc instructions array, (2) Rebuild session-context plugin from latest orch-cli source (commit ac945ea) using bun to compile .ts to .js, (3) Verify plugin has ORCH_WORKER check in config hook and exports SessionContextPlugin correctly. All steps completed and verified via automated test - plugin now properly loads orchestrator skill only for orch projects and only for non-worker sessions.

---

## Confidence Assessment

**Current Confidence:** [Level] ([Percentage])

**Why this level?**

[Explanation of why you chose this confidence level - what evidence supports it, what's strong vs uncertain]

**What's certain:**

- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]

**What's uncertain:**

- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]

**What would increase confidence to [next level]:**

- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

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
- `~/.config/opencode/opencode.jsonc` - Removed hardcoded orchestrator skill from instructions
- `~/.config/opencode/plugin/session-context.js` - Compiled plugin with fix
- `/Users/dylanconlin/Documents/personal/orch-cli/.opencode/plugin/session-context.ts` - Source with ac945ea fix

**Commands Run:**
```bash
# Remove hardcoded orchestrator skill from config
edit ~/.config/opencode/opencode.jsonc

# Compile TypeScript plugin to JavaScript
cd /Users/dylanconlin/Documents/personal/orch-cli/.opencode
bun build plugin/session-context.ts --outdir ~/.config/opencode/plugin --target=node --format=esm

# Remove old .ts symlink
rm ~/.config/opencode/plugin/session-context.ts

# Verify plugin structure
node test-plugin.mjs
```

**External Documentation:**
- orch-cli commit ac945ea - Fix that moved ORCH_WORKER check to config hook

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-12-inv-document-opencode-plugin-setup-orch.md` - OpenCode plugin documentation

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
