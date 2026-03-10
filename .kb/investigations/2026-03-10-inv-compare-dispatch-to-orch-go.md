## Summary (D.E.K.N.)

**Delta:** Dispatch and orch-go solve the same core problem (multi-agent orchestration) with radically different complexity envelopes — Dispatch is a ~615-line prompt-only skill using file-based IPC, while orch-go is a 77K+ line compiled binary with 13+ verification gates, autonomous daemon, and knowledge base infrastructure.

**Evidence:** Read every file in Dispatch (SKILL.md v2.0.0, config-example.yaml, docs/*.md, CLAUDE.md, README.md) and all cmd/orch/*.go + pkg/**/*.go in orch-go. Compared spawn flows, state management, coordination primitives, verification, and evaluation approaches.

**Knowledge:** Dispatch's file-based IPC (worker writes question → monitor detects → dispatcher answers) solves context-preserving human-in-loop without orch's beads-comment channel. Orch's verification gates, accretion control, and autonomous daemon are capabilities Dispatch hasn't attempted. Each system's coordination model reflects its author's operating mode: Dispatch optimizes for a user running from a single Claude Code session; orch optimizes for overnight autonomous fleets.

**Next:** Close. No implementation needed — this is a comparative analysis. Specific borrowable patterns identified in SYNTHESIS.md.

**Authority:** strategic - Cross-project architectural comparison; any adoption decisions are Dylan's call.

---

# Investigation: Compare Dispatch To Orch Go

**Question:** What architectural decisions differ between Dispatch and orch-go, where is each ahead, and what can orch learn from Dispatch's approach?

**Started:** 2026-03-10
**Updated:** 2026-03-10
**Owner:** spawned agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/archived/2025-12-20-inv-compare-orch-cli-python-orch.md | extends | yes | none — that was Python vs Go rewrite; this is orch vs external tool |

---

## Findings

### Finding 1: Dispatch is a Pure Prompt Skill, Not a Binary

**Evidence:** Dispatch's entire implementation lives in `skills/dispatch/SKILL.md` (615 lines of markdown). No compiled code. No server process. No database. The dispatcher runs as behavior injected into a Claude Code session. Workers are spawned via `bash` tool calls to `claude -p`, `agent -p`, or `codex exec`.

**Source:** `~/Documents/personal/dispatch/skills/dispatch/SKILL.md` (canonical v2.0.0); `~/Documents/personal/dispatch/CLAUDE.md` (conventions)

**Significance:** Dispatch is zero-infrastructure. A user types `/dispatch "do X"` and gets multi-agent orchestration with no daemon, no server, no build step. Orch requires `make install`, an OpenCode server, a beads daemon, and tmux. The friction-to-first-agent ratio is dramatically lower for Dispatch.

---

### Finding 2: File-Based IPC Preserves Worker Context

**Evidence:** Dispatch's IPC protocol: worker writes `001.question` atomically (temp → mv), monitor script (bash polling loop) detects the file and exits, which sends a `<task-notification>` to the dispatcher, dispatcher reads question and surfaces to user, writes `001.answer` atomically, respawns monitor. Worker polls for answer file, writes `001.done`, continues. **Worker never exits during Q&A** — context is preserved.

**Source:** `~/Documents/personal/dispatch/docs/ipc-protocol.md`; `~/Documents/personal/dispatch/skills/dispatch/SKILL.md` (IPC section, worker prompt template)

**Significance:** Orch's Q&A flow uses `orch send <session-id> "message"` which sends into an existing OpenCode session. For Claude CLI agents in tmux, the equivalent is `orch question <agent-id>` which extracts questions from tmux output. Both are more complex than Dispatch's approach. But Dispatch's advantage is that the worker can ask questions **without the orchestrator needing to manually check** — the monitor pattern auto-surfaces questions.

---

### Finding 3: Dispatch's Plan-as-State vs Orch's Beads-as-State

**Evidence:** Dispatch tracks progress via checklist markers in `.dispatch/tasks/<id>/plan.md`: `[ ]` pending, `[x]` done, `[?]` blocked, `[!]` error. The dispatcher reads this file to report progress. No separate status tracking system.

Orch tracks progress via beads issues (`.beads/issues.jsonl`), phase comments (`bd comments add <id> "Phase: Planning"`), workspace manifests (`.orch/workspace/<name>/AGENT_MANIFEST.json`), and event logs (`~/.orch/events.jsonl`). Status is derived at query time from multiple authoritative sources.

**Source:** Dispatch: `skills/dispatch/SKILL.md` (plan markers); Orch: `pkg/session/session.go` (GetLiveness), `pkg/beads/client.go` (RPC client), `cmd/orch/status_cmd.go`

**Significance:** Dispatch's single-file state is simpler but supports fewer queries. Orch's multi-source derivation enables richer status views (dashboard, daemon decisions, completion verification) but creates the "Multi-Backend Blindness" defect class when sources disagree.

---

### Finding 4: Dispatch Supports Multi-CLI Backends Natively

**Evidence:** Dispatch config (`~/.dispatch/config.yaml`) defines backends as CLI command templates: `claude -p --dangerously-skip-permissions`, `agent -p --force`, `codex exec --full-auto`. Models map to backends. Auto-detection via `which agent/claude/codex`. Model routing: Claude models → claude backend (no `--model` flag), GPT models → codex backend, others → cursor backend.

**Source:** `~/Documents/personal/dispatch/skills/dispatch/references/config-example.yaml`; `~/Documents/personal/dispatch/docs/config.md`

**Significance:** Orch has multi-model support via `pkg/model/model.go` (resolves aliases to provider/model format) but routes Anthropic → Claude CLI, non-Anthropic → OpenCode HTTP API. Dispatch's approach is more generic — any CLI that accepts a prompt argument can be a backend. Orch's approach is more integrated (tmux windows, session tracking, completion verification).

---

### Finding 5: Orch Has 13+ Verification Gates; Dispatch Has None

**Evidence:** Orch's completion pipeline (`cmd/orch/complete_cmd.go`, `pkg/verify/check.go`) validates: phase_complete, synthesis, test_evidence, visual_verification, git_diff, build, constraint, explain_back, verified, skill_output, decision_patch, accretion, self_review. Each gate can be individually skipped with `--skip-<gate> --skip-reason "..."`.

Dispatch's completion: dispatcher reads plan.md, checks if all items are `[x]`. No validation of work quality, no test evidence, no diff checking.

**Source:** Orch: `pkg/verify/check.go` lines 13-32; `cmd/orch/complete_cmd.go` lines 33-49. Dispatch: `skills/dispatch/SKILL.md` (checking progress section)

**Significance:** Orch has invested heavily in "trust but verify" — autonomous agents need verification to catch drift, hallucinated completions, and quality decay. Dispatch trusts the worker completely. For a user-supervised tool, this is fine. For overnight autonomous operation, it's insufficient.

---

### Finding 6: Orch's Autonomous Daemon vs Dispatch's User-Driven Model

**Evidence:** Orch's daemon (`pkg/daemon/daemon.go`) polls beads for `triage:ready` issues, infers skills, checks hotspot gates, spawns agents, auto-completes `review_tier:auto` work, detects orphans, and manages health signals. It runs continuously via launchd or foreground.

Dispatch has no daemon. The dispatcher session IS the user interaction — you ask it to dispatch work, it does. There's no autonomous overnight mode.

**Source:** Orch: `pkg/daemon/daemon.go` lines 46-150; `cmd/orch/daemon.go`. Dispatch: no equivalent.

**Significance:** This is the clearest capability gap. Dispatch is designed for interactive use ("I'm working and want to parallelize"). Orch is designed for autonomous operation ("spawn agents while I sleep, verify their work, spawn follow-ups").

---

### Finding 7: Dispatch's Fresh Context Per Subtask

**Evidence:** Dispatch explicitly calls out that each worker gets a **separate CLI session** with its own full context window. The dispatcher decomposes work into independent checklist items, each handled by a fresh worker. This prevents context window exhaustion.

Orch spawns agents with full SPAWN_CONTEXT.md (which can be 10-50KB of KB context, skill content, prior art, hotspot warnings) in a single session that persists until completion.

**Source:** Dispatch: `README.md` (before-after comparison); Orch: `pkg/spawn/context.go` (SpawnContextTemplate)

**Significance:** Dispatch's model naturally avoids context exhaustion by decomposing upfront. Orch mitigates with session checkpoints (agent context thresholds in `pkg/session/session.go`) but doesn't decompose — one agent handles the full task.

---

## Synthesis

**Key Insights:**

1. **Complexity envelope reflects operating model** - Dispatch optimizes for a human sitting at a terminal who wants to parallelize their current session's work. Orch optimizes for fleet management where agents run unsupervised. These are fundamentally different products, not just different implementations of the same idea.

2. **File-based IPC is underrated** - Dispatch's question/answer protocol via filesystem is elegant: atomic writes, no server dependency, no socket management, crash-recoverable. Orch's beads-comment channel requires a running daemon and socket connection. For agent-to-orchestrator communication specifically, the file approach has fewer failure modes.

3. **Verification is orch's moat** - The 13+ gate completion pipeline with tier-aware escalation is where orch invested that Dispatch didn't. This matters because autonomous agents need external validation — the work isn't done just because the agent says it is.

**Answer to Investigation Question:**

Dispatch and orch-go are optimized for different user profiles (interactive parallelizer vs autonomous fleet operator). Dispatch is ahead on: zero-infrastructure setup, multi-CLI backend generality, file-based IPC, and fresh-context-per-subtask decomposition. Orch is ahead on: verification gates, autonomous daemon, knowledge base integration, accretion control, and cross-project orchestration. The most borrowable pattern from Dispatch is the file-based IPC for question surfacing, which could reduce orch's dependency on beads-daemon availability for agent communication.

---

## Structured Uncertainty

**What's tested:**
- ✅ Dispatch codebase is entirely prompt-based (verified: read all files, no compiled code exists)
- ✅ Dispatch IPC uses filesystem atomicity (verified: read ipc-protocol.md, worker prompt template)
- ✅ Orch has 13+ verification gates (verified: read pkg/verify/check.go, complete_cmd.go)
- ✅ Dispatch supports Claude/Cursor/Codex backends (verified: read config-example.yaml, SKILL.md)

**What's untested:**
- ⚠️ Dispatch's monitor pattern reliability under concurrent workers (not tested — only read docs)
- ⚠️ How Dispatch handles worker failures mid-checklist (described in docs but not observed)
- ⚠️ Whether Dispatch's plan-as-state scales beyond ~10 concurrent tasks (no evidence either way)

**What would change this:**
- Running Dispatch on a real multi-task workload would validate IPC reliability claims
- Seeing Dispatch failure modes in production would clarify whether zero-verification is acceptable
- A user who runs both tools on the same project would provide direct comparison data

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Consider file-based IPC for agent question surfacing | architectural | Cross-component pattern change (beads → filesystem); multiple valid approaches |
| No implementation changes recommended from this analysis | strategic | Adopting patterns from external tool is a direction/value choice |

### Recommended Approach ⭐

**No immediate implementation** - This is an analytical comparison. Specific patterns worth considering for future architect sessions:

1. File-based IPC as fallback when beads daemon is unavailable
2. Task decomposition pattern (split one task into N subtasks with fresh contexts)
3. Auto-detection of available CLI backends

---

## References

**Files Examined:**

Dispatch:
- `~/Documents/personal/dispatch/skills/dispatch/SKILL.md` - Core implementation (v2.0.0, 615 lines)
- `~/Documents/personal/dispatch/CLAUDE.md` - Development conventions
- `~/Documents/personal/dispatch/README.md` - Project overview
- `~/Documents/personal/dispatch/docs/architecture.md` - Flow diagrams and components
- `~/Documents/personal/dispatch/docs/config.md` - Config schema and model detection rules
- `~/Documents/personal/dispatch/docs/ipc-protocol.md` - IPC technical specification
- `~/Documents/personal/dispatch/docs/getting-started.md` - Quick start
- `~/Documents/personal/dispatch/docs/development.md` - Dev setup
- `~/Documents/personal/dispatch/skills/dispatch/references/config-example.yaml` - Example config

Orch-go:
- `cmd/orch/spawn_cmd.go` - Spawn pipeline entry point
- `cmd/orch/complete_cmd.go` - Completion verification pipeline
- `cmd/orch/daemon.go` - Daemon command
- `cmd/orch/status_cmd.go` - Status aggregation
- `cmd/orch/serve.go` - HTTP API server
- `pkg/spawn/claude.go` - Claude CLI backend
- `pkg/opencode/client.go` - OpenCode HTTP client
- `pkg/daemon/daemon.go` - Daemon core
- `pkg/verify/check.go` - Verification gates
- `pkg/model/model.go` - Model resolution
- `pkg/beads/client.go` - Beads RPC client
- `pkg/session/session.go` - Session management
- `pkg/skills/loader.go` - Skill discovery

---

## Investigation History

**2026-03-10 16:18:** Investigation started
- Question: How does Dispatch compare architecturally to orch-go?
- Context: Eledath's Level 7 orchestration tool; need concrete code-level comparison

**2026-03-10 16:30:** Both codebases fully explored via parallel agents
- Dispatch: 615-line prompt skill, file IPC, multi-CLI
- Orch: 77K+ lines Go, 13+ verification gates, autonomous daemon

**2026-03-10 16:35:** Investigation completed
- Status: Complete
- Key outcome: Different complexity envelopes for different operating models; borrowable patterns identified
