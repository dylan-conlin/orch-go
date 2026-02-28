# Session Synthesis

**Agent:** og-arch-design-hook-testing-28feb-3a21
**Issue:** orch-go-zrrb
**Duration:** 2026-02-28 10:50 → 2026-02-28 11:35
**Outcome:** success

---

## Plain-Language Summary

Designed a three-tool system for developing and debugging Claude Code hooks without requiring live sessions. The core tool is `orch hook test`, which simulates what Claude Code does when it invokes a hook — it reads settings.json to find matching hooks, constructs the same JSON input, sets the same environment variables, runs the hook, and then critically validates that the hook's output JSON matches the format Claude Code actually expects. This would have caught today's bug instantly: the hook was returning `permissionDecisionReason` at the wrong nesting level, and `orch hook test` would flag "Claude Code will IGNORE this field — it needs to be inside hookSpecificOutput." Two supporting tools complete the picture: `HOOK_TRACE=1` for seeing which hooks fired during a live session (solving the "silent success" problem), and `orch hook validate` for catching configuration errors (missing files, bad matchers) before they reach runtime.

---

## TLDR

Designed `orch hook test` (simulate hook invocations outside sessions), `HOOK_TRACE` (runtime visibility), and `orch hook validate` (config linting) to eliminate the four hook debugging pain points: invisible env vars, wrong output format with no feedback, silent success, and requiring new sessions per iteration.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-28-design-hook-testing-observability-system.md` — Full design investigation with findings, synthesis, and implementation recommendations

### Files Modified
- None (design-only session)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- 10+ production hooks analyzed across Python and Bash — consistent patterns for stdin JSON reading and stdout JSON output
- Claude Code hooks API has 17 event types with 3 different decision control patterns — easy to use the wrong one
- Existing pytest tests in `~/.orch/hooks/tests/` validate logic but cannot catch output format mismatches
- `~/.claude/settings.json` is machine-parseable and contains complete hook registry (event, matcher, command, timeout)
- All hook input fields (session_id, cwd, tool_name, tool_input) are synthetic — no live session needed to construct them
- Key bug pattern: `permissionDecisionReason` at root level vs nested in `hookSpecificOutput` — silently ignored by Claude Code

---

## Architectural Choices

### Contract validation over logic testing
- **What I chose:** Focus on validating hook output format against Claude Code's expected schema, not retesting hook logic
- **What I rejected:** Extending the existing pytest suite with more unit tests
- **Why:** The existing unit tests are good at testing logic. The gap is at the integration boundary — hook output format ↔ Claude Code parser. This is where today's bug lived.
- **Risk accepted:** Schema knowledge must be maintained as Claude Code evolves

### Three independent layers over monolithic tool
- **What I chose:** Separate tools for testing (orch hook test), tracing (HOOK_TRACE), and validation (orch hook validate)
- **What I rejected:** Single tool that does everything
- **Why:** Follows observation/intervention separation principle (2026-01-14 decision). Each layer has different lifecycle: testing is development-time, tracing is runtime, validation is configuration-time.
- **Risk accepted:** Three tools to implement instead of one

### Go implementation over Python
- **What I chose:** Implement `orch hook test` in orch-go (Go)
- **What I rejected:** Standalone Python script
- **Why:** orch-go is the CLI tool for all orchestration commands. Hook testing is orchestration infrastructure. Natural home.
- **Risk accepted:** Go needs to parse Python hook output and handle subprocess execution — but orch-go already does this extensively (pkg/spawn, pkg/tmux)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-28-design-hook-testing-observability-system.md` — Complete design with implementation recommendations

### Decisions Made
- Decision 1: Hook testing is a contract validation problem (hook output ↔ Claude Code parser), not a logic testing problem
- Decision 2: Three independent layers following observation/intervention separation
- Decision 3: `orch hook test` as primary tool (highest pain reduction)

### Constraints Discovered
- Claude Code uses 3 different decision control patterns across event types — the most common mistake is using the wrong pattern
- Hook output JSON parsing only happens on exit code 0 — exit code 2 ignores all JSON
- `CLAUDE_ENV_FILE` is only available in SessionStart hooks — other hooks can't use it to set env vars

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Implementation Issues to Create

**Issue 1: `orch hook test` CLI command**
- **Skill:** feature-impl
- **Priority:** P2 (medium — immediate pain reduction)
- **Context:** Implement `orch hook test` command in cmd/orch/ that reads settings.json, resolves matchers, constructs JSON input, runs hooks, and validates output format. See investigation for full CLI spec.
- **File targets:** `cmd/orch/hook_cmd.go` (new), `pkg/hooks/resolver.go` (new — settings.json parser + matcher resolution), `pkg/hooks/schema.go` (new — output format validation)

**Issue 2: `HOOK_TRACE` runtime tracing**
- **Skill:** feature-impl
- **Priority:** P3 (backlog — useful but less urgent)
- **Context:** Add `hook_trace.py` shared module to `~/.orch/hooks/`, add trace calls to existing hooks, implement `orch hook trace` viewer command.

**Issue 3: `orch hook validate` config linter**
- **Skill:** feature-impl
- **Priority:** P3 (backlog — catch config errors)
- **Context:** Implement `orch hook validate` command that checks settings.json hook configuration for common errors (missing files, bad matchers, missing timeouts).

---

## Unexplored Questions

- **HTTP hooks:** Claude Code now supports HTTP hooks — should `orch hook test` simulate these too?
- **Prompt/agent hooks:** Can LLM-based hooks be meaningfully tested offline? Probably not — but format validation still applies.
- **Hook chains:** Should `orch hook test` support testing multiple hooks in sequence (as Claude Code does)?
- **Cross-project hooks:** Project-level hooks in `.claude/settings.json` vs user-level in `~/.claude/settings.json` — should `orch hook test` resolve both?

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification criteria.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-hook-testing-28feb-3a21/`
**Investigation:** `.kb/investigations/2026-02-28-design-hook-testing-observability-system.md`
**Beads:** `bd show orch-go-zrrb`
