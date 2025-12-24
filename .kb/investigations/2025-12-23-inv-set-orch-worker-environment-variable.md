<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** ORCH_WORKER=1 environment variable successfully added to all spawn modes (headless, inline, tmux).

**Evidence:** Tests pass for all 4 command builders; variable set via cmd.Env for exec.Cmd and via command prefix for shell strings.

**Knowledge:** Headless spawn uses HTTP API but the OpenCode server inherits env from parent; setting env on server start propagates to all spawned agents.

**Next:** Close - implementation complete with tests.

**Confidence:** High (90%) - All spawn paths covered and tested.

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

# Investigation: Set Orch Worker Environment Variable

**Question:** How to set ORCH_WORKER=1 environment variable in headless/inline/tmux spawns to prevent orchestrator skill loading?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Three distinct spawn modes exist

**Evidence:** 
- `runSpawnInline`: Uses exec.Command directly with TUI (main.go:1079-1142)
- `runSpawnHeadless`: Uses HTTP API to existing OpenCode server (main.go:1147-1207)
- `runSpawnTmux`: Sends command string to tmux window via send-keys (main.go:1209-1334)

**Source:** cmd/orch/main.go:1079-1334

**Significance:** Each mode requires different approach to set environment variable.

---

### Finding 2: OpenCode server inherits environment from parent process

**Evidence:** When orch starts OpenCode via `ensureOpenCodeRunning`, it runs `sh -c "opencode serve ..."`. Setting ORCH_WORKER=1 on this command will be inherited by all agents spawned by that server.

**Source:** cmd/orch/main.go:866-873

**Significance:** Headless spawn works by setting env on the server process start command.

---

### Finding 3: Four command builders need updates

**Evidence:** 
- `ensureOpenCodeRunning`: Shell command for starting server
- `runSpawnInline`: Uses cmd.Env
- `BuildOpencodeAttachCommand`: Returns command string for tmux
- `BuildStandaloneCommand`: Returns command string for tmux
- `BuildSpawnCommand`: Uses cmd.Env
- `BuildRunCommand`: Uses cmd.Env

**Source:** cmd/orch/main.go:870, pkg/tmux/tmux.go:43-106, 185-195

**Significance:** All paths now set ORCH_WORKER=1.

---

## Synthesis

**Key Insights:**

1. **Environment inheritance** - Setting env var on OpenCode server process propagates to all agents it spawns for headless mode.

2. **Different mechanisms** - exec.Cmd uses cmd.Env; shell command strings use prefix `ORCH_WORKER=1 command`.

3. **Comprehensive coverage** - Updated 4 places: ensureOpenCodeRunning, runSpawnInline, BuildOpencodeAttachCommand, BuildStandaloneCommand, BuildSpawnCommand, BuildRunCommand.

**Answer to Investigation Question:**

ORCH_WORKER=1 is now set in all spawn paths:
1. Headless: Prefix `opencode serve` command with `ORCH_WORKER=1` in ensureOpenCodeRunning
2. Inline: Add `ORCH_WORKER=1` to cmd.Env
3. Tmux: Prefix shell command strings with `ORCH_WORKER=1`

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

All spawn paths identified and modified. Tests written and passing. Build compiles without errors.

**What's certain:**

- ✅ All 4 spawn command builders updated
- ✅ Tests verify ORCH_WORKER=1 is set in all cases
- ✅ Build passes with no errors

**What's uncertain:**

- ⚠️ OpenCode server may already have been started without env var (existing servers won't restart automatically)

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
