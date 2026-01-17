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

# Investigation: CI Implement Lazy Loading Orchestrator

**Question:** How can we implement lazy-loading for the orchestrator skill (52KB) to load it only for orchestrator sessions, not worker sessions?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent orch-go-y1ikp
**Phase:** Complete
**Next Step:** None (implementation proceeding based on recommendations)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Config Hook Always Loads Orchestrator Skill for All Sessions

**Evidence:** The orchestrator-session.ts plugin's config hook (lines 137-154) unconditionally adds the orchestrator skill path to instructions for ALL sessions in orch projects. There is no worker detection in the config hook.

**Source:**
- `.opencode/plugins/orchestrator-session.ts:137-154` - Config hook implementation
- Line 141-144: "Skip if skill doesn't exist" but no worker detection
- Line 150: Always pushes skill path to instructions array

**Significance:** This means the 52KB orchestrator skill loads for both orchestrator AND worker sessions, wasting context budget for workers who don't need orchestration guidance.

---

### Finding 2: Config Hook Runs Too Early for Worker Detection

**Evidence:** The config hook signature is `config?: (input: Config) => Promise<void>` and only receives the Config object (instructions array, etc). It doesn't have access to sessionID, environment variables, or session metadata needed to detect workers.

**Source:**
- `.opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:112` - Config hook type definition
- `orchestrator-session.ts:70` - Comment: "Plugin runs in OpenCode server process which never sees ORCH_WORKER env var"
- Plugin architecture: config hook runs at session initialization, before any tools execute

**Significance:** The config hook is the wrong place for per-session worker detection. We need a hook that runs later and has access to session context.

---

### Finding 3: Worker Detection Requires Session-Specific Information

**Evidence:** Workers are detected via three signals: (1) ORCH_WORKER env var set during spawn (passed as x-opencode-env-ORCH_WORKER header), (2) reading SPAWN_CONTEXT.md file, (3) working in .orch/workspace/ directory. The coaching plugin successfully detects workers using tool.execute.before hook which has sessionID and args.

**Source:**
- `plugins/coaching.ts:1319-1360` - detectWorkerSession() function using tool args
- Lines 1328-1334: Detects SPAWN_CONTEXT.md reads
- Lines 1337-1350: Detects .orch/workspace/ file paths
- `cmd/orch/spawn_cmd.go:401-402` - Sets ORCH_WORKER=1 env var
- `pkg/opencode/client.go:553-555` - Sets x-opencode-env-ORCH_WORKER header

**Significance:** Worker detection is possible but requires a hook that runs after session creation and has access to tool arguments or session metadata.

---

### Finding 4: Tool Hooks Have Session Context and Can Modify Behavior

**Evidence:** The coaching plugin uses tool.execute.before hook which receives {tool, sessionID, callID} and can modify args. This hook runs for every tool call and successfully tracks worker sessions using a Map<sessionID, boolean>.

**Source:**
- `plugins/coaching.ts:1305-1307` - workerSessions Map for tracking
- `.opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:151-157` - tool.execute.before hook definition
- Lines 151: "tool.execute.before" has sessionID and args access

**Significance:** Tool hooks can detect workers but can't retroactively remove the orchestrator skill that was already loaded by the config hook.

---

### Finding 5: experimental.chat.system.transform Hook Can Modify System Prompt

**Evidence:** The plugin API includes an experimental.chat.system.transform hook that receives sessionID and can modify the system array. This hook runs when constructing the system prompt for each chat interaction.

**Source:**
- `.opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:173-177` - Hook definition
- Lines 173-177: Hook has sessionID input and system[] output
- System array is used to construct the system prompt sent to the LLM

**Significance:** This hook could potentially be used to inject orchestrator skill content conditionally based on worker detection, running after we have session context.

---

## Synthesis

**Key Insights:**

1. **Config Hook is Eager, System Transform Hook is Lazy** - The config hook loads instructions at session init (too early for worker detection), while experimental.chat.system.transform runs when constructing prompts (after we have session context). This timing difference is the key to lazy-loading.

2. **Worker Detection Requires Progressive Discovery** - We can't detect workers at plugin init or config time. We must use a two-phase approach: (1) track sessions as they interact via tool hooks, (2) conditionally inject orchestrator skill based on tracked worker status.

3. **Coaching Plugin Pattern is Proven** - The coaching.ts plugin already implements per-session worker detection via tool.execute.before hook, caching results in a Map<sessionID, boolean>. We can reuse this pattern.

**Answer to Investigation Question:**

To implement lazy-loading for the orchestrator skill, we should:
1. Remove skill injection from the config hook (Finding 1, 2)
2. Add tool.execute.before hook to detect worker sessions progressively (Finding 3, 4)
3. Add experimental.chat.system.transform hook to inject orchestrator skill ONLY for non-worker sessions (Finding 5)
4. Cache worker detection results in a Map<sessionID, boolean> to avoid repeated checks (Finding 4)

This approach ensures the orchestrator skill loads on-demand: only when sessions are confirmed to be non-workers, and only when the system prompt is being constructed (lazy evaluation).

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Progressive Worker Detection + Lazy System Transform** - Replace config hook skill injection with experimental.chat.system.transform hook that conditionally injects orchestrator skill only for non-worker sessions after progressive worker detection.

**Why this approach:**
- Runs at the right time: system.transform hook runs when building prompts, after tools have executed (Finding 2, 5)
- Reuses proven pattern: coaching.ts already implements this successfully (Finding 4)
- True lazy-loading: orchestrator skill content only added to system prompt when needed
- Progressive detection: worker status determined incrementally as session interacts

**Trade-offs accepted:**
- Requires two hooks (tool.execute.before + experimental.chat.system.transform) instead of one simple config hook
- Worker detection happens after first tool call, so orchestrator skill might load for first interaction then get removed (minor UX issue)
- Relies on experimental API (system.transform hook), but coaching plugin shows it's stable in practice

**Implementation sequence:**
1. Add workerSessions Map<sessionID, boolean> for caching detection results
2. Add tool.execute.before hook with detectWorkerSession() logic (check SPAWN_CONTEXT.md reads, .orch/workspace/ paths)
3. Add experimental.chat.system.transform hook to read orchestrator skill file and inject into system[] array only for non-workers
4. Remove orchestrator skill injection from config hook
5. Add debug logging to track which sessions load vs skip the skill

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
- Copy detectWorkerSession() from coaching.ts to orchestrator-session.ts (proven implementation)
- Add workerSessions Map for caching (prevents redundant detection)
- Wire up tool.execute.before hook to populate the cache

**Things to watch out for:**
- ⚠️ First tool call might not trigger worker detection (if worker's first action isn't reading SPAWN_CONTEXT.md) - solution: check multiple signals (file paths, tool args)
- ⚠️ Racing condition if system.transform runs before tool.execute.before - solution: default to loading skill unless explicitly detected as worker (safe fallback)
- ⚠️ Reading orchestrator skill file on every system prompt construction could be slow - solution: read once at plugin init, cache content in memory
- ⚠️ experimental.chat.system.transform hook might not support file path injection, only string content - need to test if we need to read the file or can inject the path

**Areas needing further investigation:**
- Does system.transform hook accept file paths or require full content? (test with small experiment)
- Performance impact of reading 52KB file on every prompt vs caching in memory
- Whether first-tool-call detection delay causes issues for worker sessions

**Success criteria:**
- ✅ Worker sessions DO NOT load orchestrator skill (verify via debug logs and context usage metrics)
- ✅ Orchestrator sessions DO load orchestrator skill (verify it still works)
- ✅ No regression in existing behavior (orchestrator sessions function identically)
- ✅ Performance improvement measurable (track avg context size for worker sessions before/after)

---

## References

**Files Examined:**
- `.opencode/plugins/orchestrator-session.ts` - Current implementation with config hook
- `.opencode/plugins/coaching.ts:1319-1360` - Proven worker detection pattern
- `.opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts` - Plugin API type definitions
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator skill file (52KB)
- `cmd/orch/spawn_cmd.go:401-402` - ORCH_WORKER env var setting
- `pkg/opencode/client.go:553-555` - x-opencode-env-ORCH_WORKER header

**Commands Run:**
```bash
# Check orchestrator skill file size
ls -lh ~/.claude/skills/meta/orchestrator/SKILL.md
# Result: 52KB

# Find worker detection references
rg "ORCH_WORKER" --type ts --type go -l

# Find plugin hook definitions
grep -r "config.*hook" .opencode/node_modules/@opencode-ai/plugin/dist
```

**External Documentation:**
- OpenCode Plugin API - Hook types and timing

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-23-inv-orchestrator-skill-loading-workers-despite.md` - Prior fix for ORCH_WORKER timing in orch-cli
- **Constraint:** "Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading" (from kb context)

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
