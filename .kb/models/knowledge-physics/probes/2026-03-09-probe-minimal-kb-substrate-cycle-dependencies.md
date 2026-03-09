# Probe: Minimal KB Substrate — Investigation/Probe/Model Cycle Dependencies

**Model:** Knowledge Physics
**Date:** 2026-03-09
**Status:** Complete

---

## Question

The knowledge-physics model claims the investigation/probe/model cycle operates on a "shared mutable substrate" requiring three mechanisms: attractors (models), gates (enforcement), and entropy measurement. What are the actual tooling dependencies of this cycle, and what's the smallest system that runs it without the full orch stack?

Specifically testing:
1. Model claim: "Attention-primed attractors become structurally-coupled via the probe system" — does the structural coupling require orch, or just `.kb/` directory conventions?
2. Model claim: "Every convention without a gate will eventually be violated — in knowledge too" — what gates exist at the tooling level, and which ones require orch?
3. Implicit claim: The investigation/probe/model cycle requires orchestration infrastructure (orch spawn, beads, daemon) to function.

---

## What I Tested

### Test 1: kb CLI standalone context retrieval

```bash
kb context "system learning loop" --format json
```

**Result:** Returns 10 model entries (paths + match lines), but `full_content` field is EMPTY for model.md files. kb context returns **paths and keyword matches**, not extracted model sections (Summary, Critical Invariants, Why This Fails).

The model section extraction is done by orch-go's `extractModelSectionsForSpawn()` in `pkg/spawn/kbcontext.go` (1,496 lines). This is the critical function that reads model files, extracts specific markdown headings, truncates to 2,500 chars, and formats for injection into SPAWN_CONTEXT.md.

### Test 2: kb create investigation standalone

```bash
kb create investigation --help
```

**Result:** Works completely independently of orch. Creates investigation files with `--model` flag for model coupling. `--orphan` flag for explicit decoupling. No orch dependency.

### Test 3: Probe directory structure

```bash
ls .kb/models/knowledge-physics/probes/
ls .kb/models/system-learning-loop/probes/
```

**Result:** Directories exist, contain probes. Probe creation is pure file system — write a markdown file to `.kb/models/{name}/probes/`. No orch involvement in the write itself.

### Test 4: Model content accessibility

```bash
head -20 .kb/models/knowledge-physics/model.md | grep -E "^## |^\*\*"
```

**Result:** Model files are plain markdown, fully readable. Summary, Critical Invariants, Why This Fails sections use standard `##` headings. Any agent with file system access can read and parse them.

### Test 5: Investigation skill availability

```bash
ls ~/.claude/skills/worker/investigation/SKILL.md
```

**Result:** Skill file deployed and available. Claude Code can load it via `/investigation` slash command. Skill contains full probe mode instructions (template, where to write, merge protocol).

### Test 6: Session hooks analysis

```bash
cat ~/.claude/hooks/load-skill-identity.sh
cat ~/.claude/hooks/orient-hook.sh
```

**Result:**
- `load-skill-identity.sh` loads **orchestrator** skill for interactive sessions (not investigation skill)
- `orient-hook.sh` calls `orch orient --hook` which surfaces "relevant models matching ready work" and "stale model warnings"
- Both skip for workers (`CLAUDE_CONTEXT=worker`)
- Neither injects model claims into agent context for investigation purposes

### Test 7: CLAUDE.md kb coverage

```bash
grep -n "kb\|\.kb\|knowledge" CLAUDE.md
```

**Result:** CLAUDE.md references `.kb/guides/` and `.kb/decisions/` for debugging context, but does NOT describe:
- The investigation/probe/model cycle
- How to run `kb context` for prior knowledge
- Probe creation conventions
- Model merge protocol

### Test 8: End-to-end minimal cycle simulation

Simulated the cycle without orch:
1. `kb context "topic"` → returns 20 items including model paths
2. Agent reads `model.md` → full content accessible
3. Agent writes to `.kb/models/{name}/probes/` → directory exists
4. Agent edits `model.md` → standard file edit

**Result:** All four steps work. No orch binary invoked at any point.

### Test 9: Code size analysis

```
spawn_cmd.go:     1,171 lines  (spawn orchestration)
kbcontext.go:     1,496 lines  (kb context query + model extraction + formatting)
context.go:       1,495 lines  (SPAWN_CONTEXT.md template + generation)
config.go:          569 lines  (spawn config struct)
Total:            4,731 lines
```

Of 4,731 lines of spawn infrastructure, the kb-cycle-relevant code is primarily in `kbcontext.go` (model section extraction and formatting). The rest is orchestration overhead (tmux, beads, workspace creation, skill injection, completion tracking).

---

## What I Observed

### The Five Components of the Full Cycle

| Step | What Happens | Full Stack Tool | Minimal Substrate Tool |
|------|-------------|-----------------|----------------------|
| 1. Context retrieval | Find existing models for the topic | `orch spawn` → `kbcontext.go` → `kb context` → extract sections → SPAWN_CONTEXT.md | `kb context "topic"` → agent reads model files |
| 2. Skill injection | Tell agent about probe vs investigation mode | orch injects investigation skill via SPAWN_CONTEXT.md | Claude Code `/investigation` slash command |
| 3. Artifact creation | Create probe or investigation file | Agent follows skill instructions | Same — agent follows skill instructions |
| 4. Empirical work | Run commands, observe behavior | Agent has codebase access via tmux/claude | Agent has codebase access via Claude Code |
| 5. Model merge | Update model.md with probe findings | Skill instruction (advisory) | Same — skill instruction (advisory) |

### The Critical Gap: Context Injection

The one function the full orch stack provides that doesn't have a minimal equivalent is **pre-computed model section extraction**:

- `kb context` returns file paths + keyword match lines
- orch's `extractModelSectionsForSpawn()` reads those files and extracts Summary/Critical Invariants/Why This Fails as structured sections
- These structured sections are injected into SPAWN_CONTEXT.md with the line: "Your findings should confirm, contradict, or extend the claims above"
- This is what triggers the investigation skill's "Probe Mode" detection

Without orch, the agent must:
1. Run `kb context "topic"` themselves (1 bash call)
2. Read model.md files from the returned paths (1-3 read calls)
3. Identify claims to probe (cognitive work, not tooling)

This adds 2-4 tool calls but is fully functional. The gap is convenience, not capability.

### What Orch Adds Beyond the Cycle

| Orch Feature | Purpose | Required for Cycle? |
|-------------|---------|-------------------|
| `orch spawn` | Create workspace, inject context, start tmux | NO — agent can run in Claude Code directly |
| beads (bd) | Issue tracking, phase reporting | NO — knowledge cycle doesn't need issue tracking |
| daemon | Autonomous spawn of `triage:ready` issues | NO — human can invoke cycle manually |
| OpenCode server | Multi-model headless backend | NO — Claude Code is the agent runtime |
| tmux | Visual monitoring of agents | NO — Claude Code has its own UI |
| Accounts | Multi-account OAuth management | NO — single Claude Code session |
| Completion verification | Gate on deliverables, test evidence | NO — the cycle is self-contained |
| SPAWN_CONTEXT.md | Pre-formatted context document | NO — agent can query kb context directly |
| Workspace management | .orch/workspace/ lifecycle | NO — knowledge artifacts live in .kb/ |

### The Minimal Substrate

**Required (cannot remove):**

1. **Claude Code** (`claude` CLI) — Agent runtime. The entity that asks questions, runs tests, observes behavior, and writes findings.
2. **kb CLI** (`kb`) — Knowledge artifact management. `kb context` for retrieval, `kb create investigation` for artifact creation. Without this, agents must know `.kb/` directory conventions by heart.
3. **Git** — Version control for `.kb/` artifacts. Without this, no audit trail, no collaboration between agents.
4. **`.kb/` directory** — The shared mutable substrate itself. Models, investigations, probes, decisions.
5. **Investigation skill** (`~/.claude/skills/worker/investigation/SKILL.md`) — Tells agent about probe vs investigation mode, template format, merge protocol. Without this, agent doesn't know the cycle's conventions.

**Optional but high-value:**

6. **CLAUDE.md** — Project instructions. Currently doesn't describe the kb cycle, but could. Adding 10-20 lines about "before investigating, run `kb context` to find existing models" would make the cycle self-starting.
7. **Probe template** (`.orch/templates/PROBE.md`) — Structure for probe files. Agent could write probes without this (skill describes the format), but the template reduces errors.
8. **`orch orient`** — Session orientation surfaces relevant models and stale warnings. Partially bridges the context injection gap for interactive sessions, but doesn't inject model claims.

### The Substrate Independence Finding

The investigation/probe/model cycle is substrate-independent in a second sense: it doesn't depend on the orchestration substrate (orch). The cycle's actual dependencies are:

| Dependency | Category | Can Be Replaced? |
|-----------|----------|-----------------|
| Agent runtime | Essential | Any LLM CLI (Claude Code, Aider, Cursor) |
| Knowledge store | Essential | Any `.kb/`-like directory + git |
| Context retrieval | Essential | `kb context` or manual file reading |
| Cycle conventions | Essential | Skill file, CLAUDE.md, or inline prompt |
| Model section extraction | Convenience | Agent reads files directly (2-4 extra tool calls) |

The orch stack is **orchestration infrastructure** — it makes the cycle run reliably at scale across multiple concurrent agents with tracking. But the cycle itself needs only: an agent, a knowledge store, a way to find prior knowledge, and conventions for how artifacts relate.

---

## Model Impact

- [x] **Confirms** invariant: "Models are the fundamental unit of knowledge organization" — the probe system's structural coupling (probes live in model directories) works purely through filesystem conventions, no orch needed
- [x] **Confirms** invariant: "Attention-primed attractors become structurally-coupled via the probe system" — this structural coupling is directory-level (`.kb/models/{name}/probes/`), independent of orch
- [x] **Confirms** invariant: "Every convention without a gate will eventually be violated" — the probe-to-model merge is advisory at both the orch level and the minimal level; no tooling adds a hard gate
- [x] **Extends** model with: The cycle has a clear separation between **substrate** (the knowledge system itself) and **orchestration** (the infrastructure that drives the cycle at scale). The minimal substrate is 5 components: agent runtime + kb CLI + git + .kb/ directory + investigation skill. Everything else in the orch stack is orchestration overhead — valuable for scale/reliability but not for the cycle's core mechanism.
- [x] **Extends** model with: The context injection gap — `kb context` returns paths but not extracted model sections — is the single tooling gap between "minimal substrate" and "self-starting cycle." Bridging this (a `kb context --extract-models` flag or a Claude Code hook) would make the cycle fully autonomous without orch.
- [x] **Extends** model with: The cycle is agent-runtime-independent. Any LLM CLI that can read files, run bash commands, and follow markdown instructions could run it. Claude Code is one instance, not a requirement.

---

## Notes

### Three Tiers of Knowledge Infrastructure

The investigation reveals a natural three-tier architecture:

1. **Substrate tier** (minimal, ~5 components): agent + kb + git + .kb/ + skill. Runs the cycle for a single agent working on a single question.

2. **Convenience tier** (adds ~3 components): CLAUDE.md kb instructions + probe templates + `orch orient`. Makes the cycle self-starting and reduces errors.

3. **Orchestration tier** (full orch stack): spawn + beads + daemon + completion verification + dashboard. Makes the cycle run reliably across many concurrent agents with tracking, routing, and quality gates.

### Recommended Follow-up

1. **kb CLI enhancement**: Add `kb context --extract-models` flag that reads model files and returns Summary/Critical Invariants/Why This Fails sections in the JSON output. This would bridge the context injection gap (~50-100 lines of Go in kb-cli).

2. **CLAUDE.md addition**: Add 10-20 lines describing the kb cycle conventions for agents in interactive sessions. This makes the cycle discoverable without orch spawn or `/investigation`.

3. **Claude Code hook**: A session-start hook that runs `kb context` based on the user's first message and surfaces relevant models. This would approximate orch's context injection for the minimal substrate.

### What This Means for Publishability

The knowledge physics model's substrate generalization ("the physics hold for any shared mutable substrate") gets stronger evidence: the cycle itself doesn't depend on the orchestration substrate it was built in. You could run investigation/probe/model in a fresh repo with just `kb init` + Claude Code + the investigation skill. The substrate IS the `.kb/` directory and the conventions around it, not the orch binary.
