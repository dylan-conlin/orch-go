## Summary (D.E.K.N.)

**Delta:** Designed a three-layer hook testing and observability system: `orch hook test` for outside-session testing, `HOOK_TRACE=1` for runtime tracing, and `orch hook validate` for configuration linting — solving the four pain points of invisible env vars, silent success, wrong output format with no feedback, and requiring new sessions per iteration.

**Evidence:** Analyzed 10+ production hooks (Python/Bash), Claude Code hooks API specification (17 event types, full JSON I/O schema), existing test infrastructure (pytest suite in ~/.orch/hooks/tests/), and prior decisions (role-aware filtering, observation/intervention separation).

**Knowledge:** Hook development friction is a configuration-drift-class problem — hooks drift from the API contract without feedback. The fix is making the contract testable outside Claude Code. The existing Python unit tests validate logic but can't catch the integration-level issues (wrong JSON output format, missing env vars, incorrect response field names).

**Next:** Create 3 implementation issues: `orch hook test` (primary), `HOOK_TRACE` runtime tracing (secondary), `orch hook validate` config linting (tertiary).

**Authority:** architectural - Cross-component design affecting hooks, CLI, and settings infrastructure

---

# Investigation: Hook Testing and Observability System

**Question:** How should we enable testing hooks outside of Claude Code sessions and observing their behavior during development?

**Defect-Class:** configuration-drift

**Started:** 2026-02-28
**Updated:** 2026-02-28
**Owner:** architect (orch-go-zrrb)
**Phase:** Complete
**Next Step:** None — implementation issues to be created
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-24-spike-claude-code-hooks-orchestrator-guard.md | extends | Yes - confirmed hook JSON I/O format | None |
| 2026-01-17-role-aware-hook-filtering (decision) | extends | Yes - CLAUDE_CONTEXT detection pattern confirmed | None |
| 2026-01-14-separate-observation-from-intervention (decision) | extends | Yes - tracing layer follows this pattern | None |

---

## Findings

### Finding 1: Four Distinct Pain Points with Different Solutions

**Evidence:** From the spawn context and analysis of today's debugging session:
1. **Invisible env vars** — `CLAUDE_CONTEXT` wasn't set, no way to know from inside session. Hooks silently did nothing.
2. **Wrong output format, no feedback** — Hook returned `permissionDecisionReason` but Claude Code expected it nested in `hookSpecificOutput`. The hook appeared to work (exit 0) but Claude never saw the context.
3. **Silent success** — Hooks that match and allow produce zero visible output. No way to confirm a hook fired at all.
4. **Session-per-iteration** — Every code change to a hook requires a new Claude Code session to test, because hooks snapshot at startup.

**Source:** Spawn context pain points; confirmed by code review of gate-orchestrator-code-access.py (originally used wrong output format), load-orchestration-context.py (reads CLAUDE_CONTEXT).

**Significance:** Each pain point maps to a different tool in the solution:
- Pain 1 & 2 → `orch hook test` (simulate invocation with controlled env/input)
- Pain 3 → `HOOK_TRACE` env var (runtime visibility)
- Pain 4 → `orch hook test` eliminates need for live session during development

---

### Finding 2: Hook I/O Contract is Fully Documented but Easy to Get Wrong

**Evidence:** Claude Code hooks API has:
- 17 hook event types with different matchers and input schemas
- 3 different decision control patterns: top-level `decision`, `hookSpecificOutput.permissionDecision`, and exit codes
- `PreToolUse` uses `hookSpecificOutput` with nested `permissionDecision`, while `PostToolUse` uses top-level `decision: "block"`
- `additionalContext` lives inside `hookSpecificOutput` for PreToolUse, but directly in the root for `UserPromptSubmit`
- `permissionDecisionReason` behavior differs: for "deny" it's shown to Claude, for "allow" it's shown to user only

**Source:** Claude Code hooks reference at code.claude.com/docs/en/hooks (full specification read).

**Significance:** The API is powerful but the surface area creates integration bugs. A hook author can write correct Python logic but output JSON in the wrong format for the event type, and the hook silently does nothing useful. This is exactly what happened today.

---

### Finding 3: Existing Unit Tests Cover Logic But Miss Integration

**Evidence:** `~/.orch/hooks/tests/` contains pytest suites for 6 hooks:
- `test_gate_bd_close.py` (201 lines, 14 test cases)
- `test_gate_orchestrator_code_access.py` (11KB)
- `test_reflect_suggestions_hook.py`
- `test_orchestrator_session_kn_gate.py`
- `test_load_orchestration_context.py`
- `test_inject_system_context.py`

The tests use `importlib.util.spec_from_file_location` to import hyphenated Python files, mock `os.environ` and subprocess calls, and test individual functions. They validate decision logic (should this be allowed?) but NOT:
- Whether the JSON output matches Claude Code's expected format
- Whether the hook matches the correct tools (matcher regex)
- Whether env vars will be available at runtime
- Whether the hook's timeout is appropriate

**Source:** `~/.orch/hooks/tests/test_gate_bd_close.py` (full read), directory listing of tests/

**Significance:** The gap is at the integration layer — the contract between hook output and Claude Code's parser. `orch hook test` fills this gap by simulating the full invocation pipeline and validating the output against the expected schema.

---

### Finding 4: settings.json Contains Complete Hook Configuration

**Evidence:** `~/.claude/settings.json` has all hook registrations in a structured format:
```json
{
  "hooks": {
    "PreToolUse": [
      { "matcher": "Bash", "hooks": [{ "type": "command", "command": "..." }] },
      { "matcher": "Read|Edit", "hooks": [{ "type": "command", "command": "..." }] },
      { "matcher": "Task", "hooks": [{ "type": "command", "command": "..." }] }
    ],
    "PostToolUse": [...],
    "SessionStart": [...],
    "SessionEnd": [...],
    "PreCompact": [...]
  }
}
```

This is machine-parseable and contains everything needed to simulate hook resolution: event type, matcher regex, command path, and timeout.

**Source:** `~/.claude/settings.json` (full read — 13+ hooks across 6 event types)

**Significance:** `orch hook test` can read this file directly to resolve which hooks would fire for a given event+tool combination, without any additional configuration.

---

### Finding 5: Hook Common Input Fields are Reproducible

**Evidence:** All hooks receive these common fields via stdin JSON:
```json
{
  "session_id": "abc123",
  "transcript_path": "/path/to/transcript.jsonl",
  "cwd": "/current/working/directory",
  "permission_mode": "default",
  "hook_event_name": "PreToolUse"
}
```

Plus event-specific fields like `tool_name`, `tool_input`, `tool_response`. All of these can be synthesized from CLI arguments — no live Claude Code session needed.

Additionally, env vars set by Claude Code:
- `CLAUDE_CONTEXT` (set by orch spawn, not Claude Code itself)
- `CLAUDE_PROJECT_DIR` (project root)
- `CLAUDE_ENV_FILE` (SessionStart only)
- `CLAUDE_CODE_REMOTE` (true in remote environments)
- `CLAUDE_WORKSPACE` (set by orch spawn)

**Source:** Claude Code hooks reference (common input fields section), existing hook implementations that read these env vars.

**Significance:** We can construct valid hook input without a live session. The `orch hook test` command just needs to synthesize the JSON and set the env vars.

---

## Synthesis

**Key Insights:**

1. **The problem is contract validation, not logic testing** — Existing pytest tests validate hook logic correctly. The gap is validating that hook output conforms to Claude Code's expected format for the specific event type. This is a schema validation problem.

2. **Observation and intervention separation applies** — Per the 2026-01-14 decision, the tracing system (observation) should be independent from the testing tool (intervention). Tracing writes to a persistent log; the testing tool reads configuration and simulates.

3. **The settings.json is the hook registry** — No need to build a separate hook discovery mechanism. settings.json already contains the complete hook configuration with matchers, commands, and timeouts.

**Answer to Investigation Question:**

Hook testing and observability requires three complementary tools:
1. **`orch hook test`** — Simulate hook invocations outside Claude Code. Constructs valid JSON input, sets env vars, runs the hook, validates output format against the expected schema for that event type. This is the primary tool that addresses all four pain points.
2. **`HOOK_TRACE=1`** — Runtime tracing during live sessions. When set, hooks log their invocations (input received, output produced, decision made) to `~/.orch/hooks/trace.jsonl`. Addresses "silent success" during live debugging.
3. **`orch hook validate`** — Static validation of settings.json hook configuration. Checks that commands exist, matchers are valid regex, and timeouts are reasonable. Catches configuration errors before they reach runtime.

---

## Structured Uncertainty

**What's tested:**

- ✅ Hook JSON input format is fully documented and reproducible (verified: read Claude Code hooks reference)
- ✅ Existing hooks follow consistent patterns for reading stdin and producing output (verified: read 7 hook implementations)
- ✅ settings.json is parseable and contains complete hook configuration (verified: read and analyzed the file)
- ✅ Existing pytest infrastructure demonstrates hook testability (verified: read test_gate_bd_close.py)

**What's untested:**

- ⚠️ Whether simulated env vars perfectly match what Claude Code sets at runtime (some vars may be set by Claude Code internals we can't see)
- ⚠️ Whether hook snapshot behavior (hooks frozen at session start) affects tracing (tracing should still work because hooks are commands, not in-memory)
- ⚠️ Whether Go-based `orch hook test` can correctly replicate Python subprocess invocation patterns
- ⚠️ Performance impact of HOOK_TRACE on hook execution time (should be minimal — append to JSONL file)

**What would change this:**

- If Claude Code changes its hook I/O format, the schema validation in `orch hook test` would need updating
- If Claude Code adds new env vars or input fields, the simulation would be incomplete
- If hooks start requiring session-level state (e.g., transcript content), simulation would need to be more complex

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| `orch hook test` CLI command | architectural | New CLI command + hook I/O schema knowledge — crosses orch and hook boundaries |
| `HOOK_TRACE` env var convention | implementation | Hooks opt-in to tracing by checking env var — each hook is independent |
| `orch hook validate` CLI command | implementation | Static analysis of settings.json — contained within orch |

### Recommended Approach ⭐

**Three-Layer Hook Development Toolkit** — Build `orch hook test` as the primary tool, `HOOK_TRACE` as a runtime debug aid, and `orch hook validate` as a configuration linter.

**Why this approach:**
- Directly addresses all four pain points from today's debugging session
- Builds on existing infrastructure (settings.json parsing, hook I/O format knowledge)
- Follows observation/intervention separation principle (tracing is observation, test is intervention)
- Each layer can be implemented independently and incrementally

**Trade-offs accepted:**
- Simulated env vars may not perfectly match Claude Code's runtime — we document known vars and allow overrides
- `orch hook test` can't test hooks that require live session state (transcript content) — but none of the current hooks do
- Schema validation requires maintaining knowledge of Claude Code's expected formats — worth it given the pain of wrong-format bugs

**Implementation sequence:**

#### Layer 1: `orch hook test` (Primary — highest pain reduction)

```
orch hook test PreToolUse --tool Bash --input '{"command": "bd close orch-go-1234"}' \
  --env CLAUDE_CONTEXT=orchestrator
```

**What it does:**
1. Reads `~/.claude/settings.json` to find matching hooks for event+tool
2. Constructs the full JSON input (common fields + event-specific fields from --input)
3. Sets env vars (CLAUDE_CONTEXT, CLAUDE_PROJECT_DIR, etc. + overrides from --env)
4. Runs each matching hook, capturing stdout, stderr, and exit code
5. Parses the output and shows:
   - Raw JSON output
   - Interpreted decision (ALLOW / DENY / ASK / NUDGE)
   - Whether the output format matches the expected schema for the event type
   - Any format warnings (e.g., "used `permissionDecisionReason` outside `hookSpecificOutput` — Claude Code won't see this")

**Flags:**
- `--event` or positional: Hook event name (PreToolUse, PostToolUse, SessionStart, etc.)
- `--tool`: Tool name for matcher resolution (Bash, Read, Edit, Task, etc.)
- `--input`: JSON string or @file for event-specific input fields
- `--env KEY=VALUE`: Override/set env vars (repeatable)
- `--hook PATH`: Test a specific hook file directly (skip settings.json resolution)
- `--dry-run`: Show which hooks would fire and what input they'd receive, without executing
- `--verbose`: Show full JSON input/output
- `--all-hooks`: Run all registered hooks for this event, not just matching ones (for discovery)

**Example outputs:**

```
$ orch hook test PreToolUse --tool Task --env CLAUDE_CONTEXT=orchestrator

Matching hooks for PreToolUse (tool: Task):
  1. ~/.orch/hooks/gate-orchestrator-task-tool.py (matcher: Task)

Running hook 1: gate-orchestrator-task-tool.py
  Exit code: 0
  Decision: DENY
  Reason: ⚠️ ORCHESTRATOR SPAWN GUARD: Task tool blocked. [...]
  Format: ✅ Valid (hookSpecificOutput.permissionDecision = "deny")

$ orch hook test PreToolUse --tool Bash --input '{"command": "git commit -m test"}' \
    --env CLAUDE_CONTEXT=worker

Matching hooks for PreToolUse (tool: Bash):
  1. ~/.orch/hooks/gate-bd-close.py (matcher: Bash)
  2. ~/.orch/hooks/pre-commit-knowledge-gate.py (matcher: Bash)

Running hook 1: gate-bd-close.py
  Exit code: 0
  Decision: ALLOW (no output)
  Format: ✅ Valid (exit 0, no JSON = allow)

Running hook 2: pre-commit-knowledge-gate.py
  Exit code: 0
  Decision: DENY
  Reason: ❌ Cannot commit without knowledge capture (investigation session).
  Format: ✅ Valid (hookSpecificOutput.permissionDecision = "deny")

$ orch hook test PreToolUse --tool Read --input '{"file_path": "/path/to/main.go"}' \
    --env CLAUDE_CONTEXT=orchestrator

Matching hooks for PreToolUse (tool: Read):
  1. ~/.orch/hooks/gate-orchestrator-code-access.py (matcher: Read|Edit)

Running hook 1: gate-orchestrator-code-access.py
  Exit code: 0
  Decision: ALLOW (with context)
  Context: 📋 Orchestrator coaching nudge: You are reading a code file (.go). [...]
  Format: ✅ Valid (hookSpecificOutput.permissionDecision = "allow" + additionalContext)
```

**Schema validation catches the bug from today:**
```
$ orch hook test PreToolUse --hook ~/.orch/hooks/buggy-hook.py --tool Bash

Running hook: buggy-hook.py
  Exit code: 0
  Decision: ??? (unrecognized)
  Format: ⚠️ WARNING: Output has 'permissionDecisionReason' at root level.
          For PreToolUse, this must be inside 'hookSpecificOutput'.
          Claude Code will IGNORE this field.
          Expected format:
            {"hookSpecificOutput": {"hookEventName": "PreToolUse",
             "permissionDecision": "deny",
             "permissionDecisionReason": "..."}}
```

#### Layer 2: `HOOK_TRACE` Runtime Tracing

**Convention:** Hooks opt-in to tracing by checking the `HOOK_TRACE` env var. A shared Python module provides the tracing function.

**Shared module:** `~/.orch/hooks/hook_trace.py`
```python
import json, os, time
from pathlib import Path

TRACE_FILE = Path.home() / ".orch" / "hooks" / "trace.jsonl"

def trace(hook_name: str, event: str, input_data: dict, output: dict | None, decision: str, duration_ms: float):
    if os.environ.get("HOOK_TRACE") != "1":
        return
    entry = {
        "ts": time.time(),
        "hook": hook_name,
        "event": event,
        "tool": input_data.get("tool_name", ""),
        "decision": decision,
        "duration_ms": round(duration_ms, 1),
        "context": os.environ.get("CLAUDE_CONTEXT", ""),
        "session": input_data.get("session_id", ""),
    }
    if output:
        entry["output_preview"] = json.dumps(output)[:200]
    try:
        TRACE_FILE.parent.mkdir(parents=True, exist_ok=True)
        with open(TRACE_FILE, "a") as f:
            f.write(json.dumps(entry) + "\n")
    except Exception:
        pass  # Never block on tracing
```

**Enabling in SessionStart hook:**
```bash
# In session-start.sh or via CLAUDE_ENV_FILE:
if [ -n "$CLAUDE_ENV_FILE" ]; then
    echo 'export HOOK_TRACE=1' >> "$CLAUDE_ENV_FILE"
fi
```

**Viewing traces:**
```bash
orch hook trace          # Show recent trace entries (last 50)
orch hook trace --tail   # Follow trace file
orch hook trace --session abc123  # Filter by session
orch hook trace --hook gate-bd-close  # Filter by hook name
```

#### Layer 3: `orch hook validate` Configuration Linter

```bash
$ orch hook validate

Checking hook configuration in ~/.claude/settings.json...

PreToolUse:
  ✅ gate-bd-close.py (Bash matcher) — exists, executable, timeout: 10s
  ✅ pre-commit-knowledge-gate.py (Bash matcher) — exists, executable, timeout: 10s
  ✅ gate-orchestrator-code-access.py (Read|Edit matcher) — exists, executable, timeout: 10s
  ✅ gate-orchestrator-task-tool.py (Task matcher) — exists, executable, timeout: 10s

PostToolUse:
  ✅ post-tool-use.sh (Bash matcher) — exists, executable, timeout: default(600)
  ⚠️ check-workspace-complete.py (Edit|Write matcher) — timeout: 10s
  ⚠️ log-tool-outcomes.py (Read|Glob|Grep matcher) — timeout: 5s

SessionStart:
  ✅ session-start.sh — exists, executable, timeout: 10s
  ⚠️ load-orchestration-context.py — timeout: 30s (>15s — may slow startup)

SessionEnd:
  ✅ tmux_cleanup.sh — exists, executable
  ✅ cleanup-agent-on-exit.py — exists, executable, timeout: 30s

Validation summary: 13 hooks, 0 errors, 3 warnings
```

**Checks performed:**
- Command file exists and is executable
- `$HOME` and other env vars in path are resolvable
- Matcher is valid regex
- Timeout is set (warns if default 600s used — likely unintentional)
- Duplicate matchers (same event+matcher pointing to same command)
- Hook file has correct shebang line
- No orphaned hook files (hooks in directory but not in settings.json)

### Alternative Approaches Considered

**Option B: pytest-only approach (extend existing test suite)**
- **Pros:** Leverages existing test infrastructure; familiar pytest patterns
- **Cons:** Still requires Python knowledge; can't validate hook resolution (matcher → command); doesn't address runtime tracing; can't test shell hooks (post-tool-use.sh)
- **When to use instead:** For unit testing individual hook functions (existing pattern is fine for this)

**Option C: Claude Code native testing (propose upstream feature)**
- **Pros:** Would be authoritative; could test with real session state
- **Cons:** Requires modifying Claude Code (out of scope per task definition); long lead time; doesn't help today
- **When to use instead:** If Anthropic decides to build hook testing into Claude Code itself

**Option D: Docker-based hook testing (containerized simulation)**
- **Pros:** Perfect env isolation; reproducible
- **Cons:** Massive over-engineering for shell scripts; slow iteration; the hooks depend on local file system state (settings.json, .orch/, .beads/) that's hard to replicate
- **When to use instead:** Never for this use case

**Rationale for recommendation:** Option A (three-layer toolkit) addresses all four pain points with minimal infrastructure. It builds on existing patterns (settings.json parsing in orch, pytest for logic tests) and follows the observation/intervention separation principle. Each layer is independently valuable and incrementally implementable.

---

### Implementation Details

**What to implement first:**
- `orch hook test` — highest pain reduction, needed most urgently
- Schema validation logic for hook output format — this is what would have caught today's bug
- `--dry-run` flag — useful even without executing hooks (shows which hooks match)

**Things to watch out for:**
- ⚠️ Hook commands use `$HOME` in settings.json — must expand before execution
- ⚠️ Some hooks shell out to other tools (bd, orch, tmux) — may fail in testing context without those tools
- ⚠️ Python hooks with `#!/opt/homebrew/bin/python3.12` hardcoded shebang — may not match testing environment
- ⚠️ CLAUDE_ENV_FILE is only available in SessionStart — HOOK_TRACE needs to be set via a different mechanism for non-SessionStart hooks

**Areas needing further investigation:**
- Whether `orch hook test` should support testing hook *chains* (multiple hooks firing in sequence, as Claude Code runs them)
- Whether HTTP hooks (new in Claude Code) need simulation support
- Whether prompt/agent hooks (LLM-based) can be meaningfully tested offline

**Success criteria:**
- ✅ Can reproduce today's bug outside Claude Code: `orch hook test PreToolUse --hook buggy-hook.py --tool Bash` shows format warning
- ✅ Can confirm a hook fires for a given tool: `orch hook test PreToolUse --tool Task --env CLAUDE_CONTEXT=orchestrator` shows DENY
- ✅ Can see hook execution during live session: `HOOK_TRACE=1` + `orch hook trace --tail`
- ✅ Can validate hook configuration without starting a session: `orch hook validate` shows all hooks and their status

---

## References

**Files Examined:**
- `~/.claude/settings.json` — Complete hook configuration (13+ hooks, 6 event types)
- `~/.orch/hooks/gate-orchestrator-task-tool.py` — PreToolUse deny pattern (65 lines)
- `~/.orch/hooks/gate-orchestrator-code-access.py` — PreToolUse allow+context pattern (124 lines)
- `~/.orch/hooks/gate-bd-close.py` — Complex PreToolUse with subprocess calls (317 lines)
- `~/.orch/hooks/check-workspace-complete.py` — PostToolUse with tmux integration (69 lines)
- `~/.orch/hooks/log-tool-outcomes.py` — PostToolUse observation pattern (157 lines)
- `~/.orch/hooks/pre-commit-knowledge-gate.py` — PreToolUse deny+allow dual pattern (221 lines)
- `~/.claude/hooks/post-tool-use.sh` — Bash PostToolUse with jq (56 lines)
- `~/.orch/hooks/tests/test_gate_bd_close.py` — Pytest unit test pattern (204 lines)

**External Documentation:**
- Claude Code Hooks Reference (code.claude.com/docs/en/hooks) — Full API specification

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-14-separate-observation-from-intervention.md` — Tracing layer follows observation/intervention separation
- **Decision:** `.kb/decisions/2026-01-17-role-aware-hook-filtering.md` — CLAUDE_CONTEXT detection pattern used by hooks
- **Investigation:** `.kb/investigations/2026-02-24-spike-claude-code-hooks-orchestrator-guard.md` — Prior spike on hook capabilities

---

## Investigation History

**2026-02-28 10:51:** Investigation started
- Initial question: How to test hooks outside Claude Code sessions?
- Context: Dylan spent time debugging hook that wasn't firing due to wrong JSON output format

**2026-02-28 11:05:** Exploration complete
- Read 10+ hook implementations, Claude Code hooks API, existing test suite
- Identified 4 decision forks

**2026-02-28 11:30:** Synthesis complete
- Three-layer design: orch hook test, HOOK_TRACE, orch hook validate
- Key insight: problem is contract validation (hook output ↔ Claude Code parser), not logic testing
