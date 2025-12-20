**TLDR:** Question: What CLI commands and usage should be documented in the README? Answer: The orch-go binary currently provides six commands: spawn, ask (alias send), send, monitor, status, complete, each with specific flags and behaviors documented in cmd/orch/main.go. The existing README only covers spawn, monitor, and ask, missing status and complete commands. High confidence (90%) - based on reading source code and comparing with README.

<!--
Example TLDR:
"Question: Why aren't worker agents running tests? Answer: Agents follow documentation literally but test-running guidance isn't in spawn prompts or CLAUDE.md, only buried in separate docs. High confidence (85%) - validated across 5 agent sessions but small sample size."

Guidelines:
- Keep to 2-3 sentences maximum
- Answer: What question? What's the answer? How confident?
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Update README with current CLI commands and usage

**Question:** What CLI commands and usage should be documented in the README for orch-go, and how does the existing README compare?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (80-94%)

---

## Findings

### Finding 1: orch-go provides six CLI commands with specific functionality

**Evidence:** Reading cmd/orch/main.go reveals command definitions: spawn (spawn new OpenCode session with skill context), ask (alias for send), send (send message to existing session), monitor (monitor SSE events for session completion), status (list active OpenCode sessions), complete (complete an agent and close beads issue). Each command has flags and usage documented in code comments.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:37-177

**Significance:** These are the commands that need to be documented in README. The existing README only mentions spawn, monitor, and ask, missing status and complete.

---

### Finding 2: Command flags and usage patterns are defined in main.go with examples

**Evidence:** Each command has flags and usage examples in its Long field. For example, spawn command has flags --issue, --phases, --mode, --validation, --inline. The send command expects session ID and message. The complete command has --force and --reason flags.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:68-177

**Significance:** README should include flag descriptions and example usage for each command, reflecting actual implementation.

---

### Finding 3: Existing README is outdated and incomplete

**Evidence:** Comparing current README.md with source code shows README only documents spawn, monitor, and ask commands. Missing status and complete commands. Also missing flag documentation and updated examples. The README mentions 'orch-go ask' but not 'orch-go send'.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/README.md:1-67

**Significance:** README needs to be updated to reflect current CLI functionality to ensure users have accurate documentation.

---

## Synthesis

**Key Insights:**

1. **orch-go CLI has six commands with specific flags and usage patterns** - The spawn, ask, send, monitor, status, and complete commands each serve distinct orchestration functions. Flags like --issue, --phases, --inline for spawn, and --force, --reason for complete enable fine-grained control.

2. **README is missing two commands and lacks flag documentation** - Current README only covers spawn, monitor, and ask, omitting status and complete commands. It also lacks documentation for command flags and updated examples.

3. **Update should follow existing README structure and include examples** - The README should maintain its current sections (Installation, Usage) but expand Usage to include all commands with examples mirroring those in main.go.

**Answer to Investigation Question:**

The README should document all six CLI commands (spawn, ask, send, monitor, status, complete) with their flags and examples. The existing README is missing status and complete commands, and lacks flag documentation. Update should include: Installation instructions, Usage section with each command subheading, examples from main.go, and mention of event logging. Confidence is high as findings are based on direct source code reading.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
Evidence comes from direct source code reading (cmd/orch/main.go) and comparison with existing README. The command definitions are explicit and unambiguous. Minor uncertainty about whether any hidden flags or subcommands exist beyond main.go, but exploration of cmd/orch directory shows only main.go.

**What's certain:**
- ✅ orch-go has six commands as defined in main.go
- ✅ Existing README lacks status and complete commands
- ✅ Command flags are documented in code comments

**What's uncertain:**
- ⚠️ Whether there are any other CLI entry points (e.g., separate binaries) not covered in main.go
- ⚠️ Whether any flags are deprecated or experimental
- ⚠️ Whether the help output matches the code documentation exactly

**What would increase confidence to Very High (95%+):**
- Running `orch-go --help` to verify command listing
- Testing each command with `--help` to verify flags
- Checking if any other documentation exists (e.g., man pages)

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
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
