---
name: orchestration-reference
description: On-demand reference for orchestrator agents. Load when spawning, completing, or needing operational details. Not always-loaded.
---

# Orchestration Reference

## 1. Skill Selection

```
BUILD something    → feature-impl (configure phases)
DESIGN decisions   → architect (trade-offs, recommendations)
FIX broken thing   → systematic-debugging (cause clear)
UNDERSTAND         → probe (model exists) | investigation (no model) | research (external)
TRY/EXPERIENCE     → experiential-eval (agent uses the tool, reports qualitatively)
COMPARE approaches → head-to-head (structured comparison, same task both ways)
EVALUATE UI/UX     → ux-audit (requires --mcp playwright)
```

**TRY vs EVALUATE:** "Evaluate Playwright CLI" could mean either. `ux-audit` produces structured findings about a page. `experiential-eval` has the agent use a tool interactively and report on friction, capability, and tradeoffs. The verb "evaluate" is ambiguous — clarify intent before routing.

**The intent persistence problem:** Powerful skills override weak spawn prompts. When a skill has structured methodology (like ux-audit's audit framework), it will dominate agent behavior regardless of what the spawn prompt says. The more structured the skill, the more important it is that routing was correct. If you're unsure, the spawn prompt needs an explicit `INTENT_TYPE:` to counterbalance skill gravity.

**Probe vs Investigation:** Model exists in `.kb/models/{name}/` → spawn probe into `.kb/models/{name}/probes/`. No model exists → spawn investigation into `.kb/investigations/`.

**Strategic-first rule:** Area has 5+ fixes → default to `architect`. Tactical debugging in hotspot areas requires explicit justification.

## 2. Spawn Methods

| Method | When |
|--------|------|
| `bd create "desc" --type TYPE -l triage:ready` | **Default.** Daemon auto-spawns. |
| `orch spawn --bypass-triage SKILL "task" --issue <id>` | Exception: urgency or custom context. From existing issue. |
| `orch spawn --bypass-triage SKILL "task"` | Exception: auto-creates issue. |
| `orch spawn --bypass-triage SKILL "task" --no-track` | Throwaway work only. |

**Never use Task tool. Never use `bd close` — use `orch complete`.**

### Model Selection

| Flag | Result |
|------|--------|
| (none) | Sonnet + claude backend (tmux) |
| `--model opus` | Opus + claude backend (tmux) |
| `--backend opencode` | Sonnet + headless (HTTP API) |
| `--backend opencode --model codex` | Codex-mini (OpenAI) |

**Use Opus for:** systematic-debugging, investigation, architect, codebase-audit, research, design-session.
**Sonnet suffices for:** single-file fixes, config, formatting, well-scoped implementation.
**OpenAI models** (`--model codex`, `--model gpt-5`) require `--backend opencode`.

### Spawn Context Checklist

Every spawn prompt must include:

- [ ] `ORIENTATION_FRAME:` — Why Dylan cares (his words)
- [ ] `INTENT_TYPE:` — What kind of work: `produce` (code/artifact), `experience` (use a tool, report back), or `compare` (try both, report tradeoffs). This counterbalances skill gravity — a structured skill will dominate agent behavior, so intent must be explicit.
- [ ] `PROJECT_DIR:` — Absolute path
- [ ] `SESSION SCOPE:` — What the agent is doing
- [ ] Scope boundaries — Explicit IN and OUT lists
- [ ] Authority levels — What the agent may and may not do
- [ ] Deliverables — With exact file paths
- [ ] Prior artifacts — Exact paths to relevant investigations, models, probes
- [ ] Verification requirements — Concrete command or observable to confirm done

Add `--mcp playwright` for any UI work (required for `ux-audit`).

**Surgical prompts:** Exact files/lines, single objective, explicit ONLY/DO NOT, concrete verification command.

**Pre-spawn knowledge:** `orch spawn` auto-runs `kb context`. Manual `kb context` before spawn is for orchestrator comprehension, not a gate.

## 3. Triage Criteria

Before labeling `triage:ready`, verify:

- [ ] Type is clear (bug, feature, investigation, question)
- [ ] Intent type is explicit — agent will USE the thing, not BUILD around it (for experiential work)
- [ ] No blocking dependencies (`bd show <id>`)
- [ ] Strategic premise validated — no open questions blocking this
- [ ] Scope well-defined — agent can complete without clarification
- [ ] NOT in hotspot area (or architect already completed for this area)
- [ ] Target file is not CRITICAL (>1,500 lines) — or extraction spawned first

**Strategic questions:** `bd create --type question` as first-class entities. `bd ready` excludes issues blocked by open questions.

## 4. Labels & Work Grouping

**Label types:** `area:` (domain), `effort:` (tier), `triage:` (daemon flow).

Prefer `area:` + `triage:` on every issue.

**Work grouping:** Use labels, not epics. Group with `area:` labels, sequence with `blocks` edges, checkpoint with `--type gate -l triage:review`.

**Gate checkpoints:** `bd create "Gate: description" --type gate -l triage:review` — orchestrator reviews before downstream work proceeds.

## 5. Completion Gates (Detailed)

### Behavioral Verification by Work Type

| Work Type | Verification |
|-----------|-------------|
| Features | `orch complete` passes + try the feature (CLI, UI, endpoint) |
| Bug fixes | `orch complete` passes + SYNTHESIS.md confirms reproduction + fix |
| Investigations | Spot-check one claim via `kb context` or running a command |
| Decisions | Comprehension IS the gate — orchestrator understanding suffices |

**Orchestrators verify through automated gates and synthesis review, not by reading code.** If automated gates aren't sufficient, spawn a verification probe.

### Investigation & Probe Outcomes

Before `orch complete`, classify the outcome:

**Investigations** → one of:
- Promote to decision (create follow-up decision issue)
- Create follow-up work (new issues from findings)
- Establish or update a model in `.kb/models/`

**Probes** → act on verdict:
- `confirms` → keep model claim as-is
- `extends` → merge new evidence into model (single writer, or defer via follow-up)
- `contradicts` → update model or create blocking follow-up

For probe merges: evidence must be from executed commands, not code review alone.

Check SYNTHESIS.md for unexplored questions before closing.

### Architect Handoff Gate (HARD — BLOCKS `orch complete`)

Applies only to `architect` skill completions.

1. Review SYNTHESIS.md for recommended changes
2. Verify implementation issues were auto-created by the architect agent (check beads comments for `Created:` lines or `bd list --status=open`)
3. **If no implementation issues exist → BLOCK.** Create them from architect recommendations, then proceed.

Context: Three times in Jan-Feb 2026, architect analysis evaporated because no follow-up issues were created.

### Knowledge Capture (Every Completion)

At least one per completion:

| Situation | Command |
|-----------|---------|
| Pattern failed | `kb quick tried "X" --failed "Y"` |
| New constraint discovered | `kb quick constrain "X" --reason "Y"` |
| Routing decision made | `kb quick decide "X" --reason "Y"` |
| Open question surfaced | `kb quick question "X"` |
| Model claim validated/changed | Write probe in `.kb/models/{name}/probes/` |

## 6. Session Protocols (Detailed)

### Session Start

1. **Surface fires:** "What's broken or blocking you?"
2. **Surface nagging:** "What's been on your mind?"
3. **Backlog age check:** `orch backlog cull --dry-run` — triage stale P3+ items (>14 days). Each one: close, reprioritize, or acknowledge with reason. Clear before starting new work.
4. **Propose threads:** `bd ready` + `orch status`
5. **Drain triage:review:** `orch review triage` — every item gets relabeled `triage:ready`, closed, or acknowledged. Don't spawn new work over unreviewed agent output.
6. Confirm focus with Dylan.

### Session End

1. `orch debrief` — generate session debrief from events, git, in-flight work
2. Commit all work (including the debrief)
3. `bd sync`
4. Ensure git clean
5. `orch review` — confirm no dangling completions
6. `bd list --status=in_progress` — confirm no stale in-progress work

### Cross-Session

**Home repo:** orch-go (has `.beads/`, `.kb/`, `CLAUDE.md`).
**Cross-repo:** `orch spawn --workdir ~/other-repo SKILL "task"`.
**Workspaces:** `.orch/workspace/{name}/` — tiers: light | full (SYNTHESIS.md) | orchestrator.
**Beads scope:** Issues are per-repo. `orch spawn --issue <id>` only in current repo.

## 7. Daemon & Monitoring

**Daemon runs via launchd.**
- Preview: `orch daemon preview`
- Restart: `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`

**Pipeline:** `triage:ready` → daemon auto-spawns → agent completes → orchestrator reviews via `orch complete`

`triage:review` → orchestrator reviews, relabels to `triage:ready` or closes.

**Protections:** TTL dedup cache, concurrency cap (5 agents), rate-limit gating (warn at 80%, block at 95%).

## 8. Commands Quick Reference

| Category | Commands |
|----------|----------|
| Lifecycle | `orch spawn`, `orch complete`, `orch review` |
| Monitoring | `orch wait`, `orch monitor`, `orch send`, `orch resume`, `orch abandon` |
| Strategic | `orch focus`, `kb reflect`, `kb archive` |
| Beads/Knowledge | `bd ready`, `bd create`, `bd sync`, `kb context`, `kb quick ...`, `orch doctor` |
| Dashboard | http://localhost:5188 or `orch monitor` (live) |

Full tool documentation: `.skillc/reference/tools-and-commands.md`

## 9. Allowed Orchestrator Tools

| USE (meta-actions) | NEVER USE (worker actions) |
|--------------------|---------------------------|
| `orch spawn` / `orch complete` / `orch review` | ❌ Task tool |
| `bd create` / `bd show` / `bd ready` | ❌ Edit / Write tools |
| `kb context` / `kb quick ...` | ❌ Read code files (.go, .ts, .css) |
| `git status` (read-only) | ❌ Most bash commands |
| Read: CLAUDE.md, .kb/*.md, SYNTHESIS.md | |
