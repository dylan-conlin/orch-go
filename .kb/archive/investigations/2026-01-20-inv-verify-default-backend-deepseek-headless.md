<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Default backend is DeepSeek headless (OpenCode HTTP API with DeepSeek model) due to project config, though documentation says Claude is default.

**Evidence:** Config file has `spawn_mode: opencode` and `opencode.model: deepseek`; test program confirms this results in OpenCode backend with DeepSeek model.

**Knowledge:** Default depends on config - without config, default is Claude Opus; documentation mismatch creates confusion.

**Next:** Close investigation - default verified as DeepSeek headless with current config.

**Promote to Decision:** recommend-no (verification complete, no architectural change needed)

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

# Investigation: Verify Default Backend Deepseek Headless

**Question:** Is the default backend DeepSeek headless (OpenCode HTTP API with DeepSeek model)?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Code analysis shows default backend logic

**Evidence:** The spawn command has `spawnBackend := "claude"` hardcoded at line 1143, but checks project config for `SpawnMode` setting. Config file at `.orch/config.yaml` has `spawn_mode: opencode` and `opencode.model: deepseek`.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1143`, `/Users/dylanconlin/Documents/personal/orch-go/.orch/config.yaml:1-7`

**Significance:** The code suggests default should be "opencode" (OpenCode HTTP API) with DeepSeek model due to config file, but need to test actual behavior.

---

### Finding 2: Test confirms default is DeepSeek headless

**Evidence:** Created test program that simulates spawn backend logic. Output shows: "Final spawn backend: opencode" and "Resolved model: deepseek/deepseek-chat". Test concludes "✅ DEFAULT IS DEEPSEEK HEADLESS".

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/test_default_backend.go` (test program), command output

**Significance:** Actual behavior matches config file settings - default is OpenCode HTTP API with DeepSeek model.

---

### Finding 3: Documentation discrepancy

**Evidence:** Help text says "claude: Uses Claude Code CLI in tmux (Max subscription, unlimited Opus) (default)" but actual behavior with config file is OpenCode with DeepSeek. The code has hardcoded `spawnBackend := "claude"` but config overrides it.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:87` (help text), line 1143 (hardcoded default), line 1193 (config override)

**Significance:** Documentation is misleading - should indicate config can override default.

---

## Synthesis

**Key Insights:**

1. **Default depends on config** - With config file (`spawn_mode: opencode`, `opencode.model: deepseek`), default is DeepSeek headless. Without config, default is Claude Opus.

2. **Code vs documentation mismatch** - Help text says Claude is default, but config overrides this. Documentation should be clearer about config precedence.

3. **DeepSeek is configured default** - Current project config intentionally sets OpenCode with DeepSeek as default, making DeepSeek headless the operational default.

**Answer to Investigation Question:**

Yes, the default backend is DeepSeek headless (OpenCode HTTP API with DeepSeek model) **when using the current project configuration**. The config file (`.orch/config.yaml`) has `spawn_mode: opencode` and `opencode.model: deepseek`, which overrides the hardcoded default of Claude Opus. Without this config file, the default would be Claude Opus.

---

## Structured Uncertainty

**What's tested:**

- ✅ Default with config is OpenCode + DeepSeek (verified: wrote test program that simulates spawn logic)
- ✅ Default without config is Claude + Opus (verified: moved config file and tested)
- ✅ Config file contains expected settings (verified: read `.orch/config.yaml`)

**What's untested:**

- ⚠️ Actual spawn behavior (tested logic simulation, not actual spawn command)
- ⚠️ Edge cases with other config combinations
- ⚠️ Behavior when OpenCode server is not running

**What would change this:**

- If config file changes (`spawn_mode` or `opencode.model`)
- If spawn command logic changes (hardcoded default or config precedence)
- If `DefaultModel` in model.go changes

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
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
