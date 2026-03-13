# Probe: Hook Infrastructure Audit — Cost, Coverage, Precision, Relevance

**Model:** claude-code-agent-configuration
**Date:** 2026-03-12
**Status:** Complete

---

## Question

The model claims settings.json hooks are one of four configuration layers, and notes Failure Mode 2 (configuration drift across layers) and the open question "Is the Stop hook safe for production?" This probe audits the actual hook infrastructure: what exists, does it fire, what does it cost, and is it still relevant?

---

## What I Tested

Read all 11 hook scripts in `~/.orch/hooks/`. Tested each denial hook with synthetic JSON input via subprocess to avoid self-triggering. Measured passthrough latency for all hooks. Checked for invocation logging, duplicate registrations, and dead dependencies.

```bash
# Benchmark all hooks passthrough latency
for hook in ~/.orch/hooks/*.py; do
  start=$(python3 -c "import time; print(time.time())")
  echo '{"tool_name":"Bash","tool_input":{"command":"echo hello"}}' | python3 "$hook" > /dev/null 2>&1
  end=$(python3 -c "import time; print(time.time())")
  elapsed=$(python3 -c "print(f'{($end - $start)*1000:.0f}ms')")
  echo "$(basename $hook): $elapsed"
done

# Test each denial hook with matching input
echo '{"tool_name":"Bash","tool_input":{"command":"bd close orch-go-1234"}}' \
  | CLAUDE_CONTEXT=worker python3 ~/.orch/hooks/gate-bd-close.py

# Check for invocation logs
find ~/.orch -name "*hook*log*" -o -name "*log*hook*"
grep -r "log\|logging" ~/.orch/hooks/*.py

# Check kn system status (guarded by pre-commit-knowledge-gate.py)
which kn  # not found
tail -1 .kn/entries.jsonl  # last entry: 2025-12-25

# Check duplicate registration
python3 -c "..." # parsed settings.json
```

---

## What I Observed

### Hook Inventory (11 unique scripts, 12 registrations in settings.json)

| # | Hook | Event | Matcher | Type | Context Gate | Can Deny? | Passthrough Latency |
|---|------|-------|---------|------|-------------|-----------|-------------------|
| 1 | gate-bd-close.py | PreToolUse | Bash | deny | worker (always), orchestrator (agent-worked) | Yes | 46ms |
| 2 | gate-orchestrator-bash-write.py | PreToolUse | Bash | deny | orchestrator, meta-orchestrator | Yes | 42ms |
| 3 | gate-orchestrator-git-remote.py | PreToolUse | Bash | deny | orchestrator (all remote), worker (push only) | Yes | 42ms |
| 4 | gate-spawn-context-validation.py | PreToolUse | Bash | deny | orchestrator, meta-orchestrator | Yes | 41ms |
| 5 | gate-worker-bd-dep-add.py | PreToolUse | Bash | deny | worker | Yes | 42ms |
| 6 | gate-worker-git-add-all.py | PreToolUse | Bash | deny | worker | Yes | 41ms |
| 7 | nudge-orchestrator-spawn-context.py | PreToolUse | Bash | nudge | orchestrator, meta-orchestrator | No | 46ms |
| 8 | pre-commit-knowledge-gate.py | PreToolUse | Bash | deny/nudge | hard-gate skills (deny), others (nudge) | Yes (conditional) | 49ms |
| 9 | nudge-orchestrator-investigation-drift.py | PreToolUse | Read | nudge | orchestrator, meta-orchestrator | No | 45ms |
| 10 | gate-governance-file-protection.py | PreToolUse | Edit\|Write | deny | worker | Yes | 43ms |
| 11 | enforce-phase-complete.py | Stop | (all) | block | worker + ORCH_SPAWNED=1 | Yes (block) | 74ms |

### Critical Findings

**F1: Zero observability.** No hook logs invocations. No counters for deny/allow/error. No dashboards. The ONLY evidence of hook activity is:
- `~/.orch/hook-state/spawn-ceremony-*.json` (2 state files for nudge hooks)
- No deny counts, no error counts, no latency tracking
- **Verdict: Theological enforcement** — we believe hooks work because we wrote them, not because we measured them.

**F2: Duplicate registration.** `gate-worker-git-add-all.py` appears at settings.json indices [5] and [10]. Runs twice per Bash call for workers. No functional harm (idempotent denial) but wastes one Python process per Bash call.

**F3: Dead system guard.** `pre-commit-knowledge-gate.py` guards the `kn` (knowledge notes) system:
- `kn` binary not on PATH (`which kn` returns nothing)
- Last `.kn/entries.jsonl` entry: 2025-12-25 (2.5 months stale)
- The hard-gate path (deny git commit for architect/investigation skills without kn entries) can never succeed — agents can't run `kn` even if they wanted to
- The soft-nudge path still fires but references a tool that doesn't exist

**F4: Stop hook uses non-standard output format.** `enforce-phase-complete.py` outputs `{"decision": "block", "reason": "..."}` — this differs from the PreToolUse `{"hookSpecificOutput": {"permissionDecision": "deny"}}` format. The Stop hook API may accept this, but it's undocumented and untested in the same way.

**F5: 1 of 11 hooks has tests.** Only `gate-orchestrator-bash-write.py` has a test file (`tests/test_gate_orchestrator_bash_write.py`). The other 10 hooks are untested beyond manual verification.

**F6: All denial hooks fire correctly.** Tested each with matching synthetic input:
- gate-bd-close: denies workers always, denies orchestrators on agent-worked issues (tested with real orch-go-ix6np)
- gate-orchestrator-bash-write: denies `echo hello > file.txt` for orchestrators
- gate-orchestrator-git-remote: denies `git push` for workers and orchestrators
- gate-spawn-context-validation: denies `orch spawn` without --issue/--intent
- gate-governance-file-protection: denies Edit on `~/.orch/hooks/*.py`
- gate-worker-bd-dep-add: denies `bd dep add` for workers
- gate-worker-git-add-all: denies `git add -A` and `git add .` for workers
- enforce-phase-complete: blocks exit without Phase: Complete

**F7: Context gating is sound.** All hooks check `CLAUDE_CONTEXT` env var and are no-ops for interactive sessions (no CLAUDE_CONTEXT set). This means hooks impose zero cost on Dylan's direct Claude Code sessions.

**F8: Escape hatches exist.** Most deny hooks have env var bypass: `SKIP_BD_CLOSE_GATE=1`, `SKIP_GOVERNANCE_PROTECTION=1`, `SKIP_GIT_ADD_ALL_GATE=1`, `SKIP_BD_DEP_ADD_GATE=1`, `SKIP_KN_GATE=1`.

### Cost Model

Hooks run in **parallel** per matcher group (confirmed via Claude Code docs and feature request #21533). Each entry is its own matcher, so all 9 Bash hooks run concurrently.

| Scenario | Hooks Fired | Wall-Clock Cost | Process Cost |
|----------|-------------|-----------------|--------------|
| Worker Bash call | 9 (incl. duplicate) | ~50ms (slowest hook) | 9 Python processes |
| Worker Read call | 1 | ~45ms | 1 Python process |
| Worker Edit/Write call | 1 | ~43ms | 1 Python process |
| Orchestrator Bash call | 9 | ~50ms | 9 Python processes |
| Interactive Bash call | 9 (all no-op) | ~50ms | 9 Python processes |
| Worker Stop | 1 | ~74ms (may shell out to `bd show`) | 1 Python process |

**Per-session estimate (worker, ~75 Bash + 100 Read + 30 Edit calls):**
- Wall-clock: ~75 * 50ms + 100 * 45ms + 30 * 43ms = **~9.5s total latency overhead**
- Process spawns: 75 * 9 + 100 * 1 + 30 * 1 = **805 Python processes per session**

**Per-session for interactive (no CLAUDE_CONTEXT):**
- Same latency (~9.5s) but ALL hooks exit immediately after JSON parse + env check
- Zero denials produced — pure waste for interactive sessions

### Cost/Coverage/Precision/Relevance Assessment

| Hook | Cost (per call) | Coverage | Precision | Relevance | Verdict |
|------|----------------|----------|-----------|-----------|---------|
| gate-bd-close | 46ms (+ bd show on match) | High — catches all `bd close` | High — checks labels/status | **High** — core verification bypass | Keep |
| gate-orchestrator-bash-write | 42ms | Medium — regex-based detection | Medium — allowlist may have gaps | **High** — defense-in-depth for tool restriction | Keep |
| gate-orchestrator-git-remote | 42ms | High — covers push + remote ops | High — pattern set is comprehensive | **High** — prevents accidental deploys | Keep |
| gate-spawn-context-validation | 41ms | High — catches orch spawn | High — flag parsing is solid | **High** — prevents intent displacement | Keep |
| gate-worker-bd-dep-add | 42ms | High — simple regex | High — no false positives likely | **Medium** — how often do workers try this? | Keep (low cost) |
| gate-worker-git-add-all | 41ms (x2!) | High — catches -A, ., --all | High — pattern set is tight | **High** — prevents dirty commits | Keep (fix duplicate) |
| nudge-orchestrator-spawn-context | 46ms | Medium — tracks state across calls | Medium — may nudge unnecessarily | **Medium** — coaching value unverified | Keep (no cost: nudge only) |
| pre-commit-knowledge-gate | 49ms | **Dead** — guards nonexistent `kn` CLI | N/A | **None** — kn system abandoned Dec 2025 | **Remove** |
| nudge-investigation-drift | 45ms | Medium — counts investigation reads | Medium | **Medium** — coaching value unverified | Keep (no cost: nudge only) |
| gate-governance-file-protection | 43ms | High — regex patterns on file paths | High | **High** — prevents self-modification | Keep |
| enforce-phase-complete | 74ms | High — checks message + beads | Medium — output format uncertain | **High** — prevents silent agent death | Keep (verify format) |

---

## Model Impact

- [x] **Confirms** invariant: "Configuration Drift Across Layers" (Failure Mode 2) — the `pre-commit-knowledge-gate` guards a system that no longer exists, demonstrating exactly this drift pattern
- [x] **Confirms** invariant: "hooks and skill text must not overlap" constraint — hooks like gate-bd-close and gate-worker-git-add-all enforce rules that are also stated in skill text (worker-base), creating the dual-authority pattern the model warns about
- [x] **Extends** model with: Zero observability finding — the model discusses hook failure modes but doesn't mention that hooks have no logging/metrics, making it impossible to measure effectiveness
- [x] **Extends** model with: Cost quantification — 9 Python processes per Bash call, ~50ms wall-clock per call, ~9.5s per session
- [x] **Extends** model with: Stop hook IS in production (answering Open Question 2), uses `{"decision": "block"}` format, and has a `stop_hook_active` escape hatch (second attempt allows exit)
- [x] **Extends** model with: Duplicate hook registration as a concrete drift example

---

## Notes

### Recommended Actions
1. **Remove** `pre-commit-knowledge-gate.py` from settings.json and hooks/ — guarding a dead system
2. **Remove duplicate** gate-worker-git-add-all.py registration (index [10] in settings.json)
3. **Add invocation logging** — even a simple append to `~/.orch/logs/hooks.jsonl` with timestamp, hook name, decision, context, latency
4. **Consider consolidation** — 9 Bash hooks could be a single dispatcher that routes internally, avoiding 9 Python startups
5. **Verify Stop hook format** — confirm `{"decision": "block"}` is actually the correct API

### Interactive Session Tax
All 9 Bash hooks fire on every Bash call even in interactive sessions (no CLAUDE_CONTEXT). They all exit after ~2 lines of code (JSON parse + env check), but still spawn 9 Python processes. If interactive performance matters, consider adding a fast-path: a single dispatcher that checks CLAUDE_CONTEXT once.
