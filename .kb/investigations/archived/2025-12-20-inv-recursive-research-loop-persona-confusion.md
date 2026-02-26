**TLDR:** Subagents spawned via the `Task` tool incorrectly identify as orchestrators because they lack `SPAWN_CONTEXT.md` and "TASK:" in their prompt, leading to recursive delegation loops. The fix involves updating the `orchestrator` policy to explicitly recognize subagents as workers. High confidence (90%) - the logic in `orchestrator/SKILL.md` directly supports this conclusion.

---

# Investigation: Recursive Research Loop Persona Confusion

**Question:** Why do subagents spawned for research tasks enter a recursive loop, and do they think they are the orchestrator?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** opencode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Orchestrator Detection Logic is Overly Broad

**Evidence:** The `orchestrator` policy uses three indicators to identify the orchestrator persona:
1. Working directory is project root.
2. No `SPAWN_CONTEXT.md` in current workspace.
3. No "TASK:" in initial prompt.

**Source:** `/Users/dylanconlin/.claude/skills/policy/orchestrator/SKILL.md` (Lines 18-21)

**Significance:** Subagents spawned via the `Task` tool satisfy all three criteria:
- They start in the project root.
- `Task` does not create a `SPAWN_CONTEXT.md` (unlike `orch spawn`).
- Their prompt (e.g., "Research X") typically lacks the "TASK:" prefix.
As a result, subagents incorrectly assume the orchestrator persona.

---

### Finding 2: Orchestrator Policy Mandates Delegation

**Evidence:** The policy states: "Decision rule: If orchestrator + non-trivial work → delegate via `orch spawn`. Never invoke worker skills directly." and "⚠️ Use `orch spawn`, NOT the Task tool".

**Source:** `/Users/dylanconlin/.claude/skills/policy/orchestrator/SKILL.md` (Lines 28, 223)

**Significance:** When a subagent (thinking it's an orchestrator) is asked to do research, it follows the policy and attempts to delegate the work. If it uses `Task` again, the loop continues. If it uses `orch spawn`, it creates a new session that *does* have a `SPAWN_CONTEXT.md`, but the initial subagent has already triggered a recursive pattern.

---

### Finding 3: Gemini 2.0 Flash is Highly Literal

**Evidence:** User reports "concerning pattern when using gemini 3 flash" (likely Gemini 2.0 Flash).

**Source:** User report.

**Significance:** Gemini 2.0 Flash is known for its speed and strict adherence to instructions. If the "Always-loaded" orchestrator policy tells it it's an orchestrator and must delegate, it will do so aggressively, leading to the observed recursive loop.

---

## Synthesis

**Key Insights:**

1. **Persona Confusion** - The root cause is that subagents spawned via `Task` lack the markers that distinguish workers from orchestrators in the current policy.
2. **Infinite Delegation Loop** - The "Absolute Delegation Rule" for orchestrators forces these confused subagents to spawn more subagents instead of performing the task themselves.
3. **Tool Mismatch** - While orchestrators are told to use `orch spawn`, workers often use `Task` for sub-tasks. The subagents created by `Task` are not "worker-aware" by default.

**Answer to Investigation Question:**
Yes, each new spawn thinks it is the orchestrator because it satisfies the broad "Orchestrator indicators" in the `orchestrator` policy. This triggers the "Absolute Delegation Rule," causing the agent to delegate its own task to a new subagent, creating a recursive loop.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
The logic in the `orchestrator` skill file perfectly explains the reported behavior. The indicators for being an orchestrator are exactly what a `Task`-spawned subagent would see.

**What's certain:**
- ✅ `orchestrator/SKILL.md` contains the broad detection logic.
- ✅ `Task` subagents do not get `SPAWN_CONTEXT.md`.
- ✅ Orchestrators are strictly forbidden from doing "spawnable work."

**What's uncertain:**
- ⚠️ Whether other factors (like specific model system prompts) also contribute to the persona assumption.

---

## Implementation Recommendations

### Recommended Approach ⭐

**Update Orchestrator Policy** - Refine the context detection logic to explicitly handle subagents and provide a "Subagent" indicator.

**Why this approach:**
- Directly addresses the root cause of persona confusion.
- Prevents the recursive loop by ensuring subagents know they are workers.
- Maintains the "Always-loaded" nature of the policy while making it safer.

**Implementation sequence:**
1. Add "Subagent indicators" to the `orchestrator` policy.
2. Update "Orchestrator indicators" to exclude subagents.
3. Add a specific rule for subagents to perform their assigned tasks instead of delegating.

---

### Implementation Details

**What to implement first:**
- Edit `/Users/dylanconlin/.claude/skills/policy/orchestrator/SKILL.md` to update the `Context Detection` section.

**Things to watch out for:**
- ⚠️ Ensure that the real orchestrator still correctly identifies itself.
- ⚠️ Verify that subagents still have access to the `Task` tool for legitimate parallel work if needed, but with clear guidance to avoid loops.

**Success criteria:**
- ✅ Subagents spawned via `Task` identify as workers.
- ✅ Recursive delegation loops are eliminated.
- ✅ Orchestrator still identifies as orchestrator in the primary session.

---

## References

**Files Examined:**
- `/Users/dylanconlin/.claude/skills/policy/orchestrator/SKILL.md` - Orchestrator policy and detection logic.
- `/Users/dylanconlin/.claude/skills/worker/research/SKILL.md` - Research skill definition.
- `pkg/spawn/context.go` - How `SPAWN_CONTEXT.md` is generated.

**Investigation History:**
- **2025-12-20 10:00:** Investigation started.
- **2025-12-20 10:15:** Root cause identified in `orchestrator/SKILL.md`.
- **2025-12-20 10:30:** Investigation completed.
