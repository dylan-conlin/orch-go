# Session Synthesis

**Agent:** og-inv-investigate-orchestrator-context-27mar-27e6
**Issue:** orch-go-ev88j
**Duration:** 2026-03-27 10:14 → 2026-03-27 10:35
**Outcome:** success

---

## Plain-Language Summary

We tested whether a GPT-5.4 orchestrator could receive the same context that Claude Code gets — the orchestrator skill, dynamic orientation, runtime governance, and spawn context — through either Codex CLI or OpenCode TUI. The answer: OpenCode TUI is the clear winner because it already has 80% of the infrastructure wired (backends, model routing, hook system, project context loading), while Codex CLI lacks hooks entirely and would need a new spawn backend built from scratch. The critical unknown remains whether GPT-5.4 can actually follow the ~37k-token orchestrator protocol without stalling.

---

## TLDR

Investigated Codex CLI and OpenCode TUI as frontends for a GPT-5.4 orchestrator. OpenCode is the recommended path — it already has plugin hooks (model-independent), existing orch-go backends, and GPT-5.4 in its model snapshot. Codex CLI works for simple worker spawns but can't do runtime governance.

---

## Delta (What Changed)

### Files Created
- `.kb/models/context-injection/probes/2026-03-27-probe-non-claude-frontend-context-loading.md` — Full probe with test evidence
- `.orch/workspace/og-inv-investigate-orchestrator-context-27mar-27e6/VERIFICATION_SPEC.yaml` — Verification contract
- `.orch/workspace/og-inv-investigate-orchestrator-context-27mar-27e6/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-investigate-orchestrator-context-27mar-27e6/BRIEF.md` — Comprehension artifact

### Files Modified
- `.kb/models/context-injection/model.md` — Added "Frontend Portability" section, updated SPAWN_CONTEXT delivery paths, new open question on GPT-5.4 protocol compliance, new evolution entry, new probe reference

---

## Evidence (What Was Observed)

### Codex CLI (v0.116.0) Tested Directly

- **AGENTS.md loaded fully:** `codex exec` reproduced complete 1,327-byte AGENTS.md verbatim
- **Stdin context works:** `echo context | codex exec` — GPT-5.4 correctly extracted BeadsID, role, and structured fields from piped markdown
- **Env vars inherited:** `ORCH_ROLE=orchestrator codex exec` — visible to shell commands inside session
- **No file tools:** Only `exec_command` (bash) and `apply_patch` — no Read/Write/Edit/Grep equivalents
- **No hooks:** No SessionStart, PreToolUse, PostToolUse — confirmed via help output
- **Sub-agents exist:** spawn_agent, send_input, resume_agent, wait_agent, close_agent
- **Session resume:** `codex resume <session-id>` and `codex exec resume`

### OpenCode Fork Verified via Code

- **GPT-5.4 in snapshot:** `models-snapshot.ts` contains `gpt-5.4` with `reasoning:true, tool_call:true`
- **Model routing:** `isGpt5OrLater()` routes GPT-5.4 to responses API
- **Plugin hooks model-independent:** `experimental.chat.system.transform` fires for all models, no model gating
- **Instruction loading:** `instruction.ts` loads AGENTS.md, CLAUDE.md, CONTEXT.md from project root — model-independent
- **Worker role detection:** `session.ts:245` reads `x-opencode-env-orch_worker` header
- **Backend routing:** `resolve.go:615` routes `openai` provider → `BackendOpenCode`

### Architecture Layer Portability

| Layer | Claude Code | Codex CLI | OpenCode+GPT-5.4 |
|-------|-------------|-----------|-------------------|
| Project instructions | ✅ | ✅ | ✅ |
| Orchestrator skill injection | ✅ (hook) | ⚠️ (expand AGENTS.md) | 🔧 (plugin or instructions config) |
| Dynamic orientation | ✅ (hook) | ⚠️ (shell wrapper) | 🔧 (plugin) |
| Runtime governance | ✅ (hooks) | ❌ | 🔧 (feasible) |
| Spawn context | ✅ | ✅ | ✅ |
| File operation tools | ✅ (dedicated) | ⚠️ (bash only) | ✅ (dedicated) |
| Existing orch-go backend | ✅ | ❌ | ✅ |

---

## Architectural Choices

### OpenCode TUI over Codex CLI for orchestrator role
- **What I chose:** Recommend OpenCode TUI as primary path
- **What I rejected:** Codex CLI as orchestrator frontend
- **Why:** OpenCode has hook infrastructure (plugin system), existing backends in orch-go, dedicated file tools, and Dylan maintains the fork. Codex lacks all of these.
- **Risk accepted:** GPT-5.4 protocol compliance is untested — previous GPT models stalled 67-87% on protocol-heavy skills

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Codex CLI has NO hooks system at all — governance must come from elsewhere
- Codex CLI file operations are bash-only (`exec_command`), no structured tool calls for files
- OpenCode's `experimental.chat.system.transform` hook is the equivalent of Claude Code's SessionStart for context injection

### Key Insight
The context-injection architecture has a clean frontend-agnostic/frontend-locked boundary. Everything in `pkg/spawn/` is generic. Only `~/.claude/` hooks are locked. This was implicit in the code but not documented — now captured in the model's "Frontend Portability" section.

---

## Next (What Should Happen)

**Recommendation:** close

### Follow-up Work (if pursuing GPT-5.4 orchestrator)
1. **Test GPT-5.4 protocol compliance** — Load full orchestrator skill (~37k tokens) in OpenCode session with GPT-5.4, run a realistic orchestration task, measure stall rate
2. **Write OpenCode orchestrator plugin** — Plugin that hooks `experimental.chat.system.transform` to inject orchestrator skill for non-worker sessions
3. **Or simpler: use `instructions` config** — Point OpenCode config to orchestrator skill file path

### If Close
- [x] All deliverables complete
- [x] Probe file created with test evidence
- [x] Model updated with Frontend Portability section
- [x] Ready for `orch complete orch-go-ev88j`

---

## Unexplored Questions

- **GPT-5.4 stall rate on orchestrator protocol** — The single biggest unknown. All infrastructure is viable, but if GPT-5.4 can't follow the protocol, the model quality blocks the path regardless of frontend.
- **MCP as context injection path for Codex** — `codex mcp add` exists but untested for orchestrator context injection. Could an MCP server serve orchestrator skill content?
- **Codex CLI sub-agents for worker spawning** — Codex has `spawn_agent`/`send_input` — could these replace `orch spawn` for Codex-native orchestration?

---

## Friction

- No friction — smooth session

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-orchestrator-context-27mar-27e6/`
**Probe:** `.kb/models/context-injection/probes/2026-03-27-probe-non-claude-frontend-context-loading.md`
**Beads:** `bd show orch-go-ev88j`
