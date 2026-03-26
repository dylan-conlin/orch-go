# Model: Spawn Architecture

**Domain:** Agent Spawning / Workspace Creation
**Last Updated:** 2026-03-06
**Synthesized From:** 36 investigations (Dec 2025 - Jan 2026) into spawn implementation, context generation, tier system, and triage friction. Updated Feb 28 via drift probes and model drift agent. Updated 2026-03-06 by merging 24 probes (Feb 15 - Mar 3, 2026).

---

## Summary (30 seconds)

Spawn evolved through 9 phases from basic CLI integration to a modular, gate-driven architecture with capacity-aware account routing, verification levels, and cross-repo support. The architecture creates a workspace with SPAWN_CONTEXT.md embedding skill content + task description + kb context + orientation frame, then launches a session via two-phase atomic spawn with rollback. Spawn settings are resolved via `pkg/spawn/resolve.go` with 6-level precedence and per-setting provenance tracking. The spawn pipeline is split across three packages: `pkg/spawn/` (config, resolution, context generation), `pkg/spawn/gates/` (pre-spawn validation including hotspot, triage, ratelimit, concurrency, verification, and agreements gates), `pkg/spawn/backends/` (backend abstraction), and `pkg/orch/` (pipeline orchestration and mode dispatch). The V0-V3 verification level system (replacing light/full tier) determines completion gate requirements. Claude CLI is the default backend since Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026). Skill content is processed through a template engine (`ProcessSkillContentTemplate`) before injection so `{{.BeadsID}}` and tier conditionals resolve correctly.

---

## Core Mechanism

### The Spawn Flow

```
orch spawn <skill> "task"
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. SETTINGS RESOLUTION (pkg/spawn/resolve.go)                  │
│     Resolve backend, model, tier, spawn mode, MCP, mode, etc.  │
│     Precedence: CLI > beads labels > project config >           │
│                 user config > heuristics > defaults              │
│     Each setting tracked with SettingSource provenance          │
│     Model-aware backend routing: Anthropic→claude, others→OC   │
│     Account routing: capacity-aware primary/spillover           │
│     user-config default_model treated as explicit (not CLI)     │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. SKILL RESOLUTION (pkg/skills/loader.go)                     │
│     Load ~/.claude/skills/{category}/{skill}/SKILL.md           │
│     Load dependencies (e.g., worker-base)                       │
│     Extract phases, constraints, requirements                   │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. SPAWN GATES (pkg/spawn/gates/)                              │
│     Hotspot check: block spawns targeting CRITICAL files        │
│       - All 4 types must run: bloat-size, fix-density,          │
│         investigation-cluster, coupling-cluster                 │
│       - --force-hotspot requires --architect-ref <closed issue> │
│       - Daemon spawns skip hotspot gate entirely (known gap)    │
│     Triage gate: require --bypass-triage for manual spawns      │
│     Rate limit gate: check account capacity                     │
│     Concurrency gate: limit concurrent agents                   │
│     Verification gate: check verification requirements          │
│     Agreements gate: warn on kb agreement violations (non-block)│
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. BEADS ISSUE CREATION (unless --no-track)                    │
│     bd create "{task}" --type {inferred-from-skill}             │
│     Returns beads ID (e.g., orch-go-abc1)                       │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. KB CONTEXT GATHERING (pkg/spawn/kbcontext.go)               │
│     ExtractKeywordsWithContext(task, orientationFrame, 5)       │
│     Query derived from task TITLE only (not OrientationFrame)   │
│     kb context "{keywords}" --global                            │
│     Local first; global only if <5 local matches               │
│     Local search uses cmd.Dir=projectDir (NOT spawner CWD)      │
│     Scoped tasks: filter to constraints/decisions only (15k cap)│
│     Gap analysis with wrong-project detection (pkg/spawn/gap.go)│
│       - Scoring penalizes when cross-repo matches dominate      │
│       - Global ~/.kb/ correctly excluded from wrong-project     │
│     MinMatchesForLocalSearch=5 before expanding to global       │
│     5-second timeout per query to prevent hangs                 │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  6. ATOMIC SPAWN PHASE 1 (pkg/spawn/atomic.go)                  │
│     Tag beads issue with orch:agent label (via beads socket)    │
│     Create workspace + AGENT_MANIFEST.json + dotfiles           │
│     (Rollback all writes on failure)                            │
│                                                                  │
│     .orch/workspace/{name}/                                     │
│     ├── SPAWN_CONTEXT.md      (skill + task + context)          │
│     ├── AGENT_MANIFEST.json   (canonical agent identity)        │
│     ├── .tier                 (light/full)                      │
│     ├── .beads_id             (beads issue ID)                  │
│     ├── .spawn_time           (timestamp)                       │
│     └── .spawn_mode           (headless/tmux/claude)            │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  7. MODE DISPATCH (pkg/orch/spawn_modes.go)                     │
│     Claude:   SpawnClaude() → tmux + Claude CLI                 │
│     Headless: OpenCode HTTP API (startHeadlessSession)          │
│     Tmux:     OpenCode TUI in tmux window                       │
│     Inline:   OpenCode TUI blocking in current terminal         │
│     Default: Claude backend → tmux; OpenCode backend → headless │
│     Note: opencode attach does NOT support --model flag;        │
│       use opencode run --attach for model-aware TUI spawns      │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  8. ATOMIC SPAWN PHASE 2 (pkg/spawn/atomic.go)                  │
│     Write .session_id                                           │
│     Update AGENT_MANIFEST.json with session ID                  │
│     (Best-effort: session already running)                      │
│     Note: Claude backend cannot write session metadata          │
│       (no OpenCode session; atomic contract is beads+workspace) │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

**Workspace name format:**
```
{project-prefix}-{skill-prefix}-{task-slug}-{date}-{unique}
# e.g., og-inv-analyze-spawn-workflow-15feb-1424
# project-prefix: first letter of each word (2-part names)
# skill-prefix: from SkillTierDefaults map
# task-slug: generateSlug(task, 3) — stop words filtered, 3 meaningful words
# date: 02Jan format, lowercased
# unique: 4-char hex (2 random bytes) for collision prevention
```

**SPAWN_CONTEXT.md structure:**
```markdown
TASK: {task title}

ORIENTATION_FRAME: {issue description / strategic framing}
  (ORIENTATION_FRAME is NOT in template; lives only in beads comments for completion review)

SESSION SCOPE: {Small/Medium/Large — parsed from task or --scope flag}

SPAWN TIER: {light/full}

CONFIG RESOLUTION: {backend, model, tier, spawn mode, MCP, mode, validation}

SKILL GUIDANCE: {full SKILL.md content, section-filtered by phases/mode;
                 template variables {{.BeadsID}} and tier conditionals resolved
                 via ProcessSkillContentTemplate before injection}

PRIOR KNOWLEDGE (from kb context): {constraints, decisions, models, investigations}

HOTSPOT AREA WARNING: {if targeting hotspot files}

VERIFICATION REQUIREMENTS: {V0-V3 level and gate requirements}

DELIVERABLES: {workspace files, commits}
```

**Workspace metadata (canonical: AGENT_MANIFEST.json):**
- `AGENT_MANIFEST.json` - Canonical source of agent identity and spawn-time context
- Includes `verify_level` field (V0-V3) for completion gate selection
- Read path: `ReadAgentManifestWithFallback()` → OpenCode session metadata → AGENT_MANIFEST.json → dotfiles (legacy)
- `git_baseline` field: git commit SHA at spawn time for verification

**Legacy dotfiles (still written for backward compatibility):**
- `.tier` - "light" or "full" (determines SYNTHESIS.md requirement)
- `.session_id` - OpenCode session ID for `orch send`
- `.beads_id` - Issue tracking ID for `orch complete`
- `.spawn_time` - Timestamp for age calculations
- `.spawn_mode` - Which spawn backend was used

### Tier Inference

Tier defaults are skill-based (SkillTierDefaults map in `pkg/spawn/config.go`), but the inference also considers **task scope signals**:
- If task text signals session scope or new package/module/test requirements, tier upgrades from light to full even when skill default is light.
- Scope extraction is canonical in `pkg/spawn` (`ParseScopeFromTask`, `ResolveScope`).
- `--scope` flag (small/medium/large) overrides task-parsed scope.
- Tier-based verify level capping: light tier caps at V0 (via `VerifyLevelForTier()`); only inferred levels are capped, explicit `--verify-level` overrides respected.

**Full tier** (requires SYNTHESIS.md): investigation, architect, research, codebase-audit, design-session, systematic-debugging

**Light tier** (no SYNTHESIS.md): feature-impl, reliability-testing, issue-creation

**Default:** TierFull for unknown skills (conservative)

### Skill Content Template Processing

Skill content (SKILL.md) is injected into SPAWN_CONTEXT.md via `{{.SkillContent}}`, which is a literal substitution. Template variables inside skill content (`{{.BeadsID}}`, `{{if eq .Tier "full"}}` conditionals) are NOT automatically processed by the outer template engine.

**Fix (Feb 20, 2026):** `ProcessSkillContentTemplate(content, beadsID, tier)` processes skill content through the template engine before injection. Called after `StripBeadsInstructions()` in:
- `GenerateContext()` (worker spawns)
- `GenerateOrchestratorContext()`
- `GenerateMetaOrchestratorContext()`

`skillContentData` struct intentionally exposes only `BeadsID` and `Tier`. Adding new template variables in skill content requires updating this struct.

### State Transitions

**Normal spawn lifecycle:**

```
Command invoked (orch spawn)
    ↓
Settings resolved (backend, model, tier, spawn mode)
    ↓
Skill loaded + beads issue created
    ↓
KB context gathered
    ↓
Atomic Phase 1: Tag beads + write workspace (rollback on failure)
    ↓
Session created (OpenCode API or Claude CLI)
    ↓
Atomic Phase 2: Write session ID + update manifest
    ↓
Agent starts working
```

**Cross-project spawn:**

```
cd ~/orchestrator-project
    ↓
orch spawn --workdir ~/target-project investigation "task"
    ↓
Workspace created in: ~/target-project/.orch/workspace/
Beads DefaultDir set to: ~/orchestrator-project/.beads/
projectDir threaded through kb context for correct resolution
  (runKBContextQuery sets cmd.Dir=projectDir for both local and global search)
Agent works in: ~/target-project/
```

**Atomic spawn backend matrix:**

| Backend | Creates Session Via | Can Write OC Metadata? | Can Capture SessionID? |
|---------|--------------------|-----------------------|----------------------|
| Headless | OpenCode HTTP API | Yes (at creation) | Yes (immediate) |
| Inline | OpenCode HTTP API | Yes (at creation) | Yes (immediate) |
| Tmux | `opencode attach` TUI | No (TUI creates it) | Maybe (retry-based discovery) |
| Claude | Claude CLI binary | No (no OpenCode session) | No (no session at all) |

### Critical Invariants

1. **Workspace name = `{project-prefix}-{skill-prefix}-{task-slug}-{date}-{unique}`** — kebab-case, generated by `generateSlug(task, 3)` with stop word filtering
2. **Beads ID required for phase reporting** - `--no-track` creates untracked IDs that can't report to beads, are excluded from `orch status`, orphan GC, and daemon active count; falls entirely outside both lanes of two-lane architecture
3. **KB context uses --global flag** — as fallback when local returns <5 matches; local-first strategy means global is rarely triggered for projects with rich knowledge bases
4. **Skill content stripped for --no-track** - Beads instructions removed when not tracking
5. **Session scoping is per-project** - `orch send` only works within same directory hash
6. **Token estimation at 4 chars/token** - Warning at 100k, error at 150k
7. **Model-aware backend routing** - Backend determined by model provider unless CLI overrides (Decision: kb-2d62ef)
8. **Claude backend implies tmux** - Claude CLI physically requires tmux window; headless + claude auto-switches to tmux
9. **Account routing is capacity-aware** - `resolveAccount()` scores every account with capacity data by tier-weighted effective headroom (`min(5h*tier, 7d*tier)`), tie-breaks on weighted 5-hour headroom, then name; roles only shape fallback behavior when heuristic routing is unavailable
10. **V0-V3 verification levels are strict subsets** - V0⊂V1⊂V2⊂V3; level set at spawn, enforced at completion. **Tier-based capping:** Light tier caps verify level at V0 regardless of skill default (via `VerifyLevelForTier()`). Only applies to inferred levels; explicit `--verify-level` overrides are respected.
11. **Cross-repo spawns inject BEADS_DIR** - Without this, `bd comment` in cross-repo agents targets wrong project
12. **Orientation frame is separate from task title** - Title drives workspace name slug; frame provides strategic context. ORIENTATION_FRAME is NOT in SPAWN_CONTEXT.md template (removed Feb 20) — it lives only in beads comments for orchestrator completion review.
13. **CLAUDE_CONTEXT env var set explicitly on all spawn paths** - Workers get "worker", orchestrators get "orchestrator", meta-orchestrators get "meta-orchestrator". Prevents inherited env from triggering wrong hooks.
14. **Safety-override flags require --reason** - `--bypass-triage`, `--force-hotspot`, `--no-track` require `--reason` with min 10 chars (daemon-driven spawns exempt). Reasons persisted in events.jsonl.
15. **Concurrency gate counts only running agents** - Idle agents (>10 min since last message) don't count against the cap. `--max-agents 0` means unlimited (flag default -1 = "not set").
16. **Concurrency gate includes tmux agents** - Claude CLI agents in tmux windows (no OpenCode session) are counted via `daemon.CountActiveTmuxAgents()` to prevent invisible agents from being uncapped.
17. **RunHotspotCheckForSpawn must include all 4 hotspot types** - fix-density, investigation-cluster, coupling-cluster, bloat-size. Omitting bloat-size means CRITICAL file detection by name in task fails silently.
18. **--force-hotspot requires --architect-ref** - Flag must reference a closed architect issue. Converting escape hatch from "bypass with flag" to "bypass with proof of architectural review."
19. **Skill content must be processed through template engine** - `ProcessSkillContentTemplate()` must run before skill content is injected into SPAWN_CONTEXT.md. Failure results in literal `{{.BeadsID}}` in agent context.
20. **user-config default_model is treated as explicit for backend selection** - Setting `default_model` in `~/.orch/config.yaml` prevents infra detection from forcing the claude backend.
21. **Malformed user config emits a warning at spawn** - Warning includes config path and `backend`/`default_model` hint; spawn continues with defaults.
22. **Skills are procedural guidance, not runtime tooling** - When an agent can accomplish a task with existing tools (Bash + Write) but produces poor results, the solution is a skill (knowledge), not an MCP plugin (new tools). MCP is warranted only when the existing tool surface literally cannot accomplish the task.

---

## Why This Fails

### Failure Mode 1: Cross-Project Spawn Injects Wrong-Project KB Context

**Symptom:** Agent spawned for toolshed receives orch-go infrastructure knowledge instead of toolshed architecture/pricing knowledge.

**Root cause:** `runKBContextQuery` does not set `cmd.Dir`. The local kb search runs from the daemon/orchestrator's CWD (orch-go), not the target project's directory. With a rich local knowledge base (280+ investigations), the local search trivially hits the ≥5-match threshold and global search is never reached, bypassing even the group-filtering fix from Feb 25.

**Why it happens:**
- `projectDir` is threaded into `gatherKBContext()` but not into `runKBContextQuery()`
- Local-first search uses spawner CWD → orch-go matches on generic terms ("architect", "redesign")
- ≥5 orch-go matches → global search skipped entirely
- Group-filtering fix (Feb 25) only applies in Step 2 (global), which is never reached

**Fix (Feb 25-27, 2026):** Thread `projectDir` through full call chain AND set `cmd.Dir=projectDir` in `runKBContextQuery`.

**Residual gap:** Gap analysis scores category population, not content relevance — spawns can score 90-95% quality while containing 100% wrong-project knowledge. Fix: `AnalyzeGaps` now accepts `projectDir` and uses path-based checking to detect wrong-project injection; >50% wrong = critical gap.

### Failure Mode 2: KB Context Query in Wrong Semantic Domain

**Symptom:** Architect agent misplaces implementation (e.g., skill tooling in orch-go instead of skillc) because critical boundary decisions are not surfaced.

**Root cause:** KB context query is derived exclusively from task title keywords. When the relevant decision lives in a different semantic domain ("skillc is the correct home for skill authoring tools"), it isn't surfaced by title-derived keywords ("design", "infrastructure", "orchestrator").

**Compounding factor:** The orchestrator's framing can pre-commit to the wrong repo by asking "should there be a skill linter?" within an orch-go context, before the architect considers tool placement.

**No current fix** — structural limitation of keyword-based retrieval.

### Failure Mode 3: Daemon Spawns Bypass Hotspot Gate

**Symptom:** Daemon spawns feature-impl for an issue targeting CRITICAL hotspot files without architectural review.

**Root cause:** `pkg/spawn/gates/hotspot.go` returns early for daemon-driven spawns (`if daemonDriven { return result, nil }`). The daemon has no hotspot awareness in skill inference — feature/task issues always map to `feature-impl`.

**No current fix** — known architectural gap.

### Failure Mode 4: Token Limit Exceeded on Spawn

**Symptom:** Spawn fails with "context too large" error

**Root cause:** SPAWN_CONTEXT.md exceeds 150k token limit

**Why it happens:**
- Skill content dominates (70% of total prompt) — ~8,000 tokens typical
- KB context can be large (30-50k tokens)
- Unused phases/modes still inlined (section filtering can save ~1,400-2,400 tokens)

**Fix (Dec 22):** Warning at 100k tokens, hard error at 150k with guidance

### Failure Mode 5: Daemon Spawns Blocked Issues

**Symptom:** Daemon spawns issue that has blockers

**Root cause:** Dependency checking missing in triage workflow

**Fix (Jan 3):** Dependency gating with `--force` override flag

---

## Constraints

### Why Can't We Infer Skill from Task Description?

**Constraint:** Natural language is ambiguous - "fix bug" could be systematic-debugging or feature-impl

**Implication:** Must explicitly specify skill in spawn command

**Workaround:** Daemon infers skill from beads issue type

**This enables:** Precise skill selection for complex tasks
**This constrains:** Manual spawns require explicit skill argument

### Why Require --bypass-triage for Manual Spawns?

**Constraint:** Design choice to make manual spawns intentionally harder

**Implication:** Friction encourages daemon-driven workflow

**Workaround:** Use `--bypass-triage` flag for urgent/exceptional spawns

**This enables:** Scalable automation via daemon
**This constrains:** Ad-hoc spawning is discouraged

### Why V0-V3 Verification Levels Instead of Binary Tier?

**Constraint:** Binary light/full tier was too coarse — SYNTHESIS.md requirement doesn't capture the spectrum of verification needs (test evidence, visual verification, explain-back)

**Implication:** V0-V3 levels determine which completion gates fire. Defaults derived from skill + issue type (max of both). `--verify-level` flag overrides.

**V0 (Acknowledge):** Phase Complete only
**V1 (Artifacts):** V0 + deliverable/constraint checks
**V2 (Evidence):** V1 + test evidence, build logs, git diff
**V3 (Behavioral):** V2 + visual verification, explain-back

**Skill defaults:** issue-creation/capture-knowledge→V0, investigation/architect/research/codebase-audit/design-session/probe/ux-audit→V1, feature-impl/systematic-debugging/reliability-testing→V2, debug-with-playwright→V3
**Issue type minimums:** feature/bug/decision→V2, investigation/probe→V1
**Tier capping:** Light tier→max V0 (via `TierMaxVerifyLevel`); full tier→no cap

**This enables:** Graduated verification matching work complexity
**This constrains:** Must decide level at spawn (persisted in AGENT_MANIFEST.json)

### Why Can't --no-track Agents Be Managed Normally?

**Constraint:** `--no-track` generates synthetic beads IDs that don't exist in the beads database. These agents fall outside both lanes of the two-lane architecture: not visible in `orch status` (tracked lane) and not visible in `orch sessions` (OpenCode lane, only works for non-Claude-CLI backends).

**Blast radius:** 5 distinct `isUntrackedBeadsID()` guards have accumulated across the codebase to prevent crashes.

**Recommended fix:** Replace `--no-track` with lightweight tracking (real beads issue with no-track label).

---

## Evolution

**Phase 1: Initial Implementation (Dec 19, 2025)**
- CLI command structure with Cobra
- Skill loading from `~/.claude/skills/`
- SPAWN_CONTEXT.md template generation
- Beads integration for tracking

**Phase 2: Tmux Visual Mode (Dec 20-21, 2025)**
- Per-project workers sessions (`workers-orch-go`)
- Window naming with skill emojis
- `opencode attach` for TUI + API dual access
- Readiness detection via pane content polling

**Phase 3: Headless Default (Dec 22, 2025)**
- Flipped default from tmux to headless (HTTP API)
- `--tmux` became opt-in
- Enabled daemon automation
- SSE monitoring via `orch monitor`

**Phase 4: Tiered Completion (Dec 22, 2025)**
- Light tier for implementation (no SYNTHESIS.md required)
- Full tier for knowledge work (SYNTHESIS.md required)
- Skill-based defaults
- `.tier` file in workspace

**Phase 5: Triage Friction (Jan 3-6, 2026)**
- Dependency gating (`--force` to override)
- `--bypass-triage` flag to discourage manual spawns
- Daemon-driven workflow as default
- Event logging for bypass analysis

**Phase 6: Atomic Spawn + Resolved Settings (Jan-Feb 2026)**
- Registry removed; AGENT_MANIFEST.json replaces dotfiles as canonical metadata
- `pkg/spawn/resolve.go` centralizes all settings resolution with provenance
- Two-phase atomic spawn with rollback on failure (`pkg/spawn/atomic.go`)
- `--backend claude` implies tmux spawn mode (derived setting)
- Flash models blocked entirely at resolve layer
- Context file variants: SPAWN_CONTEXT.md, ORCHESTRATOR_CONTEXT.md, META_ORCHESTRATOR_CONTEXT.md
- Hotspot gating blocks spawns targeting CRITICAL files (>1500 lines)

**Phase 7: Modular Extraction + Account Distribution (Feb 2026)**
- Extracted `pkg/orch/spawn_modes.go` + `pkg/orch/spawn_helpers.go` from `extraction.go` (-644 lines)
- New `pkg/spawn/gates/` subdirectory: hotspot, triage, ratelimit, concurrency, verification gates
- New `pkg/spawn/backends/` subdirectory: backend interface + common/headless/inline/tmux implementations
- Account distribution with capacity-aware routing (3 phases: schema+CLI+env → cache+heuristic → logging)
- `resolveAccount()` uses tier-weighted effective headroom when capacity fetches are available, then falls back to primary/empty-role or alphabetical defaults when they are not
- Cross-project spawn fixes: `beads.DefaultDir` set correctly, `projectDir` threaded through kb context
- Bug-type issues now route to `systematic-debugging` skill (was `architect`)
- `--force-hotspot` requires `--architect-ref` with verified closed architect issue
- `--disallowedTools` enforcement + PreToolUse hook for `bd close` gating
- Claude CLI became default backend (Anthropic banned subscription OAuth in third-party tools Feb 19)
- Pre-create session for tmux spawns with non-default models
- GPT-5 alias remapped to `gpt-5.2` to prevent zombie sessions

**Phase 8: Verification Levels + Cross-Repo Quality + Context Intelligence (Feb 25-27, 2026)**
- V0-V3 verification levels replace binary light/full tier (`pkg/spawn/verify_level.go`)
  - Defaults from skill + issue type (max of both); `--verify-level` flag overrides
  - Persisted in AGENT_MANIFEST.json; completion gates check via `ShouldRunGate()`
- Agreements gate added to spawn pipeline (`pkg/spawn/gates/agreements.go`)
  - Runs `kb agreements check --json`; warning-only (non-blocking), graduated to blocking after 30 days
- Wrong-project knowledge detection in gap analysis (`pkg/spawn/gap.go`)
  - `countWrongProjectMatches()` penalizes quality score when cross-repo matches dominate
  - Global `~/.kb/` correctly excluded from wrong-project classification
- Orientation frame separates task title from strategic context
  - Issue title → TASK (concise, drives workspace name); description → ORIENTATION_FRAME
  - ORIENTATION_FRAME removed from SPAWN_CONTEXT.md template (Feb 20) — beads comment only
- `ExtractKeywordsWithContext()` dual-source keyword extraction (`pkg/spawn/kbcontext.go`)
  - Title keywords get priority; frame keywords provide domain disambiguation
  - Skill-prefix stripping prevents infrastructure terms from polluting queries
- Scope-appropriate kb context filtering (`pkg/spawn/kbcontext.go`)
  - File-targeted tasks get constraints/decisions only (15k char cap vs 80k default)
  - `TaskIsScoped()` detects directory-qualified file paths in task
- `MinMatchesForLocalSearch` raised from 3 to 5
  - Rich KBs (280+ investigations) trivially hit low threshold on generic terms
- BEADS_DIR env var injection for cross-repo Claude CLI spawns (`pkg/spawn/claude.go`)
  - Enables `bd comment` to target correct project in cross-repo agents
- `RunKBContextCheckForDir` fixes: `projectDir` threaded through full call chain + `cmd.Dir` set
- Cross-repo group resolution bug fixed (Feb 25): `resolveProjectAllowlistForDir(projectDir)`
- SESSION SCOPE in template now reflects actual task scope; `--scope` flag added
- Tier inference upgrades light-tier skills when task signals session scope or new package/tests
- `ProcessSkillContentTemplate` added; all three context generators call it
- AUTHORITY section deduplication: duplicated core (worker-base) removed from spawn template; unique content preserved ("Surface Before Circumvent")
- `RunHotspotCheckForSpawn` fixed to include bloat-size analysis (all 4 hotspot types)
- Spawn-time model staleness detection confirmed working in production (detects changed+deleted files)

**Phase 9: Safety Gates + Environment Isolation (Feb 27-28, 2026)**
- `--reason` flag required for safety-override flags (`--bypass-triage`, `--force-hotspot`, `--no-track`)
  - Min 10 chars, persisted in events.jsonl alongside existing events
  - Daemon-driven spawns exempt (daemon has its own triage logic)
- Concurrency gate fixes:
  - Only counts running agents (idle >10min excluded) — prevents 15 idle agents from blocking new spawns
  - `--max-agents 0` means unlimited; flag default changed to -1 as sentinel for "not set"
  - Tmux agents (Claude CLI backend) now counted via `daemon.CountActiveTmuxAgents()`
  - Batch beads-closed check prevents counting completed agents
- `CLAUDE_CONTEXT` env var explicitly set on all spawn paths (`pkg/spawn/config.go:ClaudeContext()`)
  - Workers get "worker", orchestrators get "orchestrator", meta-orchestrators get "meta-orchestrator"
  - Fixed bug where OpenCode backend spawns (tmux, inline) inherited parent's CLAUDE_CONTEXT
  - Claude CLI path already had this; now all backends aligned
- Tier-based verify level capping (`pkg/spawn/verify_level.go:VerifyLevelForTier()`)
  - Light tier caps at V0 (acknowledge only), full tier uncapped
  - Applied to inferred levels in `BuildSpawnConfig()`; explicit `--verify-level` overrides respected
- Expanded skill verify level defaults:
  - Added: capture-knowledge→V0, probe→V1, ux-audit→V1, debug-with-playwright→V3

---

## References

**Key Investigations:**
- `2025-12-19-inv-spawn-agent-tmux-implementation.md` - Initial tmux implementation
- `2025-12-22-inv-flip-default-spawn-mode-headless.md` - Headless as default
- `2025-12-22-inv-implement-tiered-spawn-protocol.md` - Tier system design
- `2026-01-03-inv-spawn-dependency-gating-design.md` - Dependency checking
- `2026-01-06-inv-add-bypass-triage-friction-manual.md` - Triage friction
- ...and 31 others

**Decisions Informed by This Model:**
- Headless default (enables daemon automation)
- Tier system (appropriate docs for work complexity)
- Triage friction (encourage daemon workflow)
- KB context gathering (prevent duplicate work)

**Related Models:**
- `.kb/models/model-access-spawn-paths/model.md` - Model selection, backend routing, escape hatch
- `.kb/models/agent-lifecycle-state-model/model.md` - How spawned agents' status is calculated

**Related Guides:**
- `.kb/guides/spawn.md` - How to use spawn command (procedural)
- `.kb/guides/daemon.md` - How daemon auto-spawns (procedural)

**Primary Evidence (Verify These):**
- `cmd/orch/spawn_cmd.go` - Main spawn command + infrastructure detection (~952 lines)
- `pkg/orch/extraction.go` - Spawn pipeline types and functions (~1619 lines)
- `pkg/orch/spawn_modes.go` - Mode dispatch: inline/headless/tmux/claude (~530 lines)
- `pkg/orch/spawn_helpers.go` - Helper utilities for spawn pipeline (~148 lines)
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md generation (~1418 lines; was 1315 Feb 20, growing)
- `pkg/spawn/kbcontext.go` - KB context gathering, keyword extraction, scoped filtering (~1485 lines)
- `pkg/spawn/config.go` - Config struct, tier defaults, skill mappings, CLAUDE_CONTEXT (~519 lines)
- `pkg/spawn/resolve.go` - Settings resolution with 6-level precedence, account routing (~661 lines)
- `pkg/spawn/atomic.go` - Two-phase atomic spawn with rollback (~113 lines)
- `pkg/spawn/claude.go` - Claude CLI backend (tmux spawn, MCP wiring, BEADS_DIR injection) (~165 lines)
- `pkg/spawn/gap.go` - Context gap analysis, quality scoring, wrong-project detection
- `pkg/spawn/session.go` - Session management, AGENT_MANIFEST.json with verify_level field
- `pkg/spawn/verify_level.go` - V0-V3 level definitions, defaults, tier-based capping, comparison functions (~130 lines)
- `pkg/spawn/errors.go` - Error type definitions and handling (~261 lines)
- `pkg/spawn/orchestrator_context.go` - Orchestrator SPAWN_CONTEXT generation (~656 lines)
- `pkg/spawn/meta_orchestrator_context.go` - Meta-orchestrator context generation (~415 lines)
- `pkg/spawn/skill_requires.go` - Skill requirement validation (~634 lines)
- `pkg/spawn/learning.go` - Learning/knowledge context injection (~975 lines)
- `pkg/spawn/verification_spec.go` - VERIFICATION_SPEC.yaml handling (~377 lines)
- `pkg/spawn/tokens.go` - Token estimation and management (~236 lines)
- `pkg/spawn/staleness_events.go` - Model staleness event emission at spawn time (~191 lines)
- `pkg/spawn/rework.go` - Rework/retry logic (~174 lines)
- `pkg/spawn/probes.go` - Probe generation/management (~203 lines)
- `pkg/spawn/ecosystem.go` - Ecosystem/environment setup (~112 lines)
- `pkg/spawn/resolve_format.go` - Format resolution helpers (~74 lines)
- `pkg/spawn/opencode_mcp.go` - OpenCode MCP integration (~73 lines)
- `pkg/spawn/gates/` - Pre-spawn validation gates (hotspot, triage, ratelimit, concurrency, verification, agreements)
- `pkg/spawn/backends/` - Backend abstraction layer (backend interface, common, headless, inline, tmux)
- `pkg/skills/loader.go` - Skill discovery, loading, dependency composition (~334 lines)

**Merged Probes:**
- `2026-02-15-spawn-workflow-mechanics-analysis` — Confirmed workspace name format; contradicted cross-project session directory failure mode (already fixed via x-opencode-directory header); documented 14-step spawn workflow sequence
- `2026-02-15-spawn-time-staleness-detection-behavioral-verification` — Confirmed staleness detection fires in production; detects changed and deleted files across multiple models per spawn
- `2026-02-18-tier-inference-scope-signals` — Extended: tier inference now upgrades light-tier skills when task signals session scope or new package/test requirements
- `2026-02-18-tmux-readiness-timeout` — Extended: `opencode attach` does not support `--model` flag (causes silent TUI startup failure); socket awareness gap in tmux package
- `2026-02-18-warn-malformed-userconfig` — Confirmed: spawn pipeline warns on malformed `~/.orch/config.yaml` with path and hint
- `2026-02-18-probe-default-model-explicit-backend` — Extended: user-config `default_model` is treated as explicit for backend selection, preventing infra detection from forcing claude
- `2026-02-19-atomic-spawn-architecture-readiness` — Extended: Claude backend cannot participate in OpenCode session metadata (no session); atomic contract is beads+workspace only for claude mode
- `2026-02-19-probe-runwork-default-model-precedence` — Confirmed: `runWork` does not inject `default_model` into CLI precedence
- `2026-02-20-spawn-architecture-structural-drift` — Contradictions resolved: registry removed; context.go now ~1315+ lines; AGENT_MANIFEST.json is canonical (not 5 dotfiles); spawn is atomic two-phase not linear
- `2026-02-20-probe-orientation-frame-dedup-verification` — Confirmed: ORIENTATION_FRAME removed from SPAWN_CONTEXT.md template (Feb 20 fix); lives only in beads comments
- `2026-02-20-spawn-bloat-analysis-gap` — Extended: `RunHotspotCheckForSpawn` was missing bloat-size analysis; all 4 hotspot types must be included
- `2026-02-20-probe-authority-section-dedup` — Extended: AUTHORITY section in spawn template deduplicated; worker-base core authority removed, unique "Surface Before Circumvent" preserved
- `2026-02-20-skill-content-template-processing` — Extended: skill content template variables (`{{.BeadsID}}`, tier conditionals) were not processed before injection; `ProcessSkillContentTemplate` added
- `2026-02-20-probe-skill-content-template-fix-verification` — Confirmed fix works on real worker-base SKILL.md (14,600 bytes, 13 references → 0 remaining)
- `2026-02-20-probe-progressive-skill-disclosure-design` — Extended: skill content dominates SPAWN_CONTEXT.md (~70%); section filtering feasible via heading patterns; saves ~1,400-2,400 tokens (not 2,000-6,000 for skill filtering alone)
- `2026-02-20-probe-session-scope-template-honor` — Extended: SESSION SCOPE in template previously hardcoded "Medium"; now resolved from task or `--scope` flag; scope resolution canonical in `pkg/spawn`
- `2026-02-24-probe-architect-gate-hotspot-enforcement` — Extended: `--force-hotspot` was unconditional bypass with no accountability; daemon spawns skip hotspot gate entirely; `--force-hotspot` now requires `--architect-ref`
- `2026-02-25-probe-cross-project-kb-context-group-resolution` — Extended: `resolveProjectAllowlist()` used spawner CWD not workdir for group resolution; fixed by threading `projectDir` through 4-function chain
- `2026-02-27-probe-kb-context-query-derivation-and-assembly` — Extended: query derived from task title only (not OrientationFrame); OR-based stemmed matching with no minimum relevance threshold; beads comments excluded from default spawn path
- `2026-02-27-probe-cross-repo-spawn-context-quality-audit` — Extended: `runKBContextQuery` never sets `cmd.Dir`; local search runs from spawner CWD; confirmed real toolshed spawns received 100% orch-go knowledge; gap analysis scores category population not content relevance
- `2026-02-27-probe-gap-analysis-wrong-project-false-positive` — Extended: `AnalyzeGaps` now accepts `projectDir` and uses path-based wrong-project detection; >50% wrong-project → critical gap; quality score drops to 0 for all-wrong-project context
- `2026-03-01-probe-architect-missed-skillc-kb-context-gap` — Extended: keyword-based retrieval has structural blindspot when critical decision is in different semantic domain; orchestrator framing can pre-commit to wrong repo before architect considers tool placement
- `2026-03-03-probe-no-track-invisible-agent-operational-cost` — Extended: `--no-track` creates third invisible agent class outside both two-lane architecture lanes; 5 `isUntrackedBeadsID()` guards accumulated; should be replaced with lightweight tracking
- `2026-03-03-probe-spreadsheet-generation-skill-vs-mcp` — Extended: confirmed skill/MCP boundary principle: when existing tool surface (Bash+Write) is sufficient, solution is a skill not an MCP plugin; MCP warranted only when tool surface cannot accomplish the task

## Auto-Linked Investigations

- .kb/investigations/archived/epic-management-deprecated/2026-01-13-inv-analyze-understanding-artifact-architecture-epic.md
- .kb/investigations/2026-03-04-inv-design-system-prompt-skill-injection.md
- .kb/investigations/archived/2025-12-23-debug-headless-spawn-model-format.md
- .kb/investigations/archived/2025-12-25-inv-orchestrator-pre-spawn-context-gathering.md
- .kb/investigations/archived/2025-12-25-inv-extend-skill-yaml-schema-spawn.md
- .kb/investigations/archived/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md
- .kb/investigations/archived/2026-01-17-inv-analyze-spawn-value-ratio.md
- .kb/investigations/archived/2025-12-26-design-multi-project-orchestration-architecture.md
- .kb/investigations/archived/2026-01-04-inv-phase-skill-type-detection-spawn.md
- .kb/investigations/2026-03-16-inv-design-minimal-harness-openscad-agent.md
- .kb/investigations/archived/2025-12-22-inv-headless-spawn-registers-wrong-project.md
